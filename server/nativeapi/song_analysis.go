package nativeapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core/audio"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
)

type suggestedEnergyResponse struct {
	ID              string               `json:"id"`
	SuggestedEnergy string               `json:"suggestedEnergy"`
	Metrics         audio.AudioMetrics   `json:"metrics"`
	Score           audio.ScoreBreakdown `json:"score"`
}

func (api *Router) addSongAnalysisRoute(r chi.Router) {
	r.With(server.URLParamsMiddleware).Get("/song/{id}/suggested-energy", getSuggestedEnergy(api.ds, api.analyzer))
}

func getSuggestedEnergy(ds model.DataStore, analyzer audio.Analyzer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id := chi.URLParam(r, "id")

		// Check if analyzer is available
		if analyzer == nil || !analyzer.IsAvailable() {
			log.Warn(ctx, "Audio analyzer not available")
			http.Error(w, "Audio analyzer not available. Please install essentia_streaming_extractor_music.", http.StatusServiceUnavailable)
			return
		}

		// Get the media file
		mf, err := ds.MediaFile(ctx).Get(id)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				http.Error(w, "Song not found", http.StatusNotFound)
				return
			}
			log.Error(ctx, "Failed to get media file", "id", id, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Build the full path to the file
		fullPath := filepath.Join(mf.LibraryPath, mf.Path)

		// Run analysis
		log.Debug(ctx, "Analyzing audio for energy suggestion", "id", id, "path", fullPath)
		result, err := analyzer.Analyze(ctx, fullPath)
		if err != nil {
			if errors.Is(err, audio.ErrAnalyzerNotAvailable) {
				http.Error(w, "Audio analyzer not available", http.StatusServiceUnavailable)
				return
			}
			log.Error(ctx, "Failed to analyze audio", "id", id, "path", fullPath, err)
			http.Error(w, "Failed to analyze audio: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Return response
		resp := suggestedEnergyResponse{
			ID:              id,
			SuggestedEnergy: result.SuggestedEnergy,
			Metrics:         result.Metrics,
			Score:           result.Score,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error(ctx, "Failed to encode response", err)
		}

		log.Info(ctx, "Suggested energy for song", "id", id, "suggestedEnergy", result.SuggestedEnergy, "bpm", result.Metrics.BPM)
	}
}
