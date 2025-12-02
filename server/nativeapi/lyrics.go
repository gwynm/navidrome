package nativeapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/adapters/taglib"
	"github.com/navidrome/navidrome/core/lyrics"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
)

type fetchLyricsResponse struct {
	Total   int `json:"total"`
	Fetched int `json:"fetched"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

func (api *Router) addAlbumLyricsRoute(r chi.Router) {
	r.With(server.URLParamsMiddleware).Post("/album/{id}/fetch-lyrics", fetchAlbumLyrics(api.ds))
}

func fetchAlbumLyrics(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		albumID := chi.URLParam(r, "id")

		// Check if Genius is available
		if !lyrics.IsGeniusAvailable() {
			log.Warn(ctx, "Genius lyrics fetcher not available")
			http.Error(w, "Genius lyrics fetcher not available. Please set GENIUS_CLIENT_ACCESS_TOKEN environment variable and install lyricsgenius.", http.StatusServiceUnavailable)
			return
		}

		// Get all songs in the album
		songs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
			Filters: squirrel.Eq{"album_id": albumID},
		})
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				http.Error(w, "Album not found", http.StatusNotFound)
				return
			}
			log.Error(ctx, "Failed to get songs for album", "albumId", albumID, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if len(songs) == 0 {
			http.Error(w, "No songs found in album", http.StatusNotFound)
			return
		}

		resp := fetchLyricsResponse{
			Total: len(songs),
		}

		// Process each song
		for _, song := range songs {
			// Check if the audio file has CORRECTLY stored embedded lyrics.
			// Properly parsed lyrics appear as "lyrics:xxx" or "lyrics:eng" keys
			// (from USLT frames for MP3, or LYRICS Vorbis comments for FLAC/OGG).
			// Tags like "unsyncedlyrics" or "txxx:unsyncedlyrics" are incorrectly stored
			// and should be re-fetched and stored properly.
			hasCorrectlyStoredLyrics := false
			filePath := song.AbsolutePath()
			tags, err := taglib.Read(filePath)
			if err == nil {
				// Check for properly formatted lyrics tags (lyrics:xxx, lyrics:eng, etc.)
				for key := range tags {
					keyLower := strings.ToLower(key)
					if strings.HasPrefix(keyLower, "lyrics:") {
						hasCorrectlyStoredLyrics = true
						break
					}
				}
			}

			if hasCorrectlyStoredLyrics {
				resp.Skipped++
				log.Debug(ctx, "Skipping song with correctly embedded lyrics", "title", song.Title, "artist", song.Artist)
				continue
			}

			// Fetch lyrics from Genius
			log.Info(ctx, "Fetching lyrics for song", "title", song.Title, "artist", song.Artist)
			lyricsList, err := lyrics.FetchFromGenius(ctx, &song)
			if err != nil {
				log.Error(ctx, "Failed to fetch lyrics", "title", song.Title, "artist", song.Artist, err)
				resp.Failed++
				continue
			}

			if len(lyricsList) == 0 {
				log.Debug(ctx, "No lyrics found for song", "title", song.Title, "artist", song.Artist)
				resp.Failed++
				continue
			}

			// Update the database with the new lyrics
			lyricsJSON, err := json.Marshal(lyricsList)
			if err != nil {
				log.Error(ctx, "Failed to marshal lyrics", "title", song.Title, err)
				resp.Failed++
				continue
			}

			song.Lyrics = string(lyricsJSON)
			if err := ds.MediaFile(ctx).Put(&song); err != nil {
				log.Error(ctx, "Failed to save lyrics to database", "title", song.Title, err)
				resp.Failed++
				continue
			}

			resp.Fetched++
			log.Info(ctx, "Successfully fetched and saved lyrics", "title", song.Title, "artist", song.Artist)
		}

		// Return response
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error(ctx, "Failed to encode response", err)
		}

		log.Info(ctx, "Finished fetching lyrics for album", "albumId", albumID,
			"total", resp.Total, "fetched", resp.Fetched, "failed", resp.Failed, "skipped", resp.Skipped)
	}
}

