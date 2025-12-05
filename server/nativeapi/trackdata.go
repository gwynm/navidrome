package nativeapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/adapters/taglib"
	"github.com/navidrome/navidrome/core/trackanalysis"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
)

type fetchTrackDataResponse struct {
	Total   int `json:"total"`
	Fetched int `json:"fetched"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

func (api *Router) addAlbumTrackDataRoute(r chi.Router) {
	r.With(server.URLParamsMiddleware).Post("/album/{id}/fetch-trackdata", fetchAlbumTrackData(api.ds))
}

func (api *Router) addPlaylistTrackDataRoute(r chi.Router) {
	r.With(server.URLParamsMiddleware).Post("/playlist/{id}/fetch-trackdata", fetchPlaylistTrackData(api.ds))
}

func fetchAlbumTrackData(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		albumID := chi.URLParam(r, "id")

		// Check if track analysis API is available
		if !trackanalysis.IsAvailable() {
			log.Warn(ctx, "Track analysis API not available")
			http.Error(w, "Track analysis API not available. Please set RAPIDAPI_KEY environment variable.", http.StatusServiceUnavailable)
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

		resp := fetchTrackDataForSongs(ctx, ds, songs, "album", albumID)
		writeTrackDataResponse(w, resp)
	}
}

func fetchPlaylistTrackData(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		playlistID := chi.URLParam(r, "id")

		// Check if track analysis API is available
		if !trackanalysis.IsAvailable() {
			log.Warn(ctx, "Track analysis API not available")
			http.Error(w, "Track analysis API not available. Please set RAPIDAPI_KEY environment variable.", http.StatusServiceUnavailable)
			return
		}

		// Get playlist with tracks
		playlist, err := ds.Playlist(ctx).GetWithTracks(playlistID, true, false)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				http.Error(w, "Playlist not found", http.StatusNotFound)
				return
			}
			log.Error(ctx, "Failed to get playlist", "playlistId", playlistID, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		songs := playlist.MediaFiles()
		if len(songs) == 0 {
			http.Error(w, "No songs found in playlist", http.StatusNotFound)
			return
		}

		resp := fetchTrackDataForSongs(ctx, ds, songs, "playlist", playlistID)
		writeTrackDataResponse(w, resp)
	}
}

func writeTrackDataResponse(w http.ResponseWriter, resp fetchTrackDataResponse) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error(context.Background(), "Failed to encode response", err)
	}
}

// songResult holds the result of processing a single song
type songResult struct {
	fetched bool
	failed  bool
	skipped bool
}

func fetchTrackDataForSongs(ctx context.Context, ds model.DataStore, songs model.MediaFiles, resourceType, resourceID string) fetchTrackDataResponse {
	total := len(songs)
	var fetched, failed, skipped int

	// Process songs sequentially - the upstream API has strict rate limits
	// and doesn't handle concurrent requests well even with retries
	for _, song := range songs {
		result := processSong(ctx, ds, song)
		if result.fetched {
			fetched++
		} else if result.failed {
			failed++
		} else if result.skipped {
			skipped++
		}
	}

	log.Info(ctx, "Finished fetching track data", "resourceType", resourceType, "resourceId", resourceID,
		"total", total, "fetched", fetched, "failed", failed, "skipped", skipped)

	return fetchTrackDataResponse{
		Total:   total,
		Fetched: fetched,
		Failed:  failed,
		Skipped: skipped,
	}
}

func processSong(ctx context.Context, ds model.DataStore, song model.MediaFile) songResult {
	// Check if the song already has track data stored
	filePath := song.AbsolutePath()
	tags, err := taglib.Read(filePath)
	if err == nil {
		hasTrackData := false
		for key := range tags {
			keyLower := strings.ToLower(key)
			if keyLower == "energyval" || keyLower == "happiness" || keyLower == "instrumentalness" {
				hasTrackData = true
				break
			}
		}
		if hasTrackData {
			log.Debug(ctx, "Skipping song with existing track data", "title", song.Title, "artist", song.Artist)
			return songResult{skipped: true}
		}
	}

	// Fetch track data from API
	log.Info(ctx, "Fetching track data for song", "title", song.Title, "artist", song.Artist)
	data, err := trackanalysis.FetchTrackData(ctx, song.Artist, song.Title)
	if err != nil {
		if errors.Is(err, trackanalysis.ErrTrackNotFound) {
			log.Debug(ctx, "Track not found in analysis API", "title", song.Title, "artist", song.Artist)
		} else if errors.Is(err, trackanalysis.ErrRateLimited) {
			log.Warn(ctx, "Rate limited by track analysis API", "title", song.Title, "artist", song.Artist)
		} else {
			log.Error(ctx, "Failed to fetch track data", "title", song.Title, "artist", song.Artist, err)
		}
		return songResult{failed: true}
	}

	// Write tags to the audio file
	if err := writeTrackDataTags(ctx, filePath, data); err != nil {
		log.Error(ctx, "Failed to write track data tags", "title", song.Title, "path", filePath, err)
		return songResult{failed: true}
	}

	log.Info(ctx, "Successfully fetched and saved track data", "title", song.Title, "artist", song.Artist,
		"energyVal", data.EnergyVal, "happiness", data.Happiness, "instrumentalness", data.Instrumentalness)

	return songResult{fetched: true}
}

func writeTrackDataTags(ctx context.Context, filePath string, data *trackanalysis.TrackData) error {
	// Write each tag to the file
	// Use uppercase tag names - TagLib will handle the format correctly for each file type
	// (TXXX frames for MP3, Vorbis comments for FLAC/OGG, freeform atoms for M4A)
	if err := taglib.WriteTag(filePath, "ENERGYVAL", fmt.Sprintf("%d", data.EnergyVal)); err != nil {
		return fmt.Errorf("failed to write ENERGYVAL tag: %w", err)
	}

	if err := taglib.WriteTag(filePath, "HAPPINESS", fmt.Sprintf("%d", data.Happiness)); err != nil {
		return fmt.Errorf("failed to write HAPPINESS tag: %w", err)
	}

	if err := taglib.WriteTag(filePath, "INSTRUMENTALNESS", fmt.Sprintf("%d", data.Instrumentalness)); err != nil {
		return fmt.Errorf("failed to write INSTRUMENTALNESS tag: %w", err)
	}

	log.Debug(ctx, "Wrote track data tags to file", "path", filePath)
	return nil
}
