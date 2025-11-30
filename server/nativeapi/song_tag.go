package nativeapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"slices"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/adapters/taglib"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
)

// Valid energy values
var validEnergyValues = []string{"", "low", "medium", "high"}

// Valid mood values
var validMoodValues = []string{"", "negative", "neutral", "positive"}

type setEnergyRequest struct {
	Value string `json:"value"`
}

type setEnergyResponse struct {
	ID     string `json:"id"`
	Energy string `json:"energy"`
}

type setMoodRequest struct {
	Value string `json:"value"`
}

type setMoodResponse struct {
	ID   string `json:"id"`
	Mood string `json:"mood"`
}

func (api *Router) addSongTagRoute(r chi.Router) {
	r.With(server.URLParamsMiddleware).Put("/song/{id}/energy", setEnergy(api.ds))
	r.With(server.URLParamsMiddleware).Put("/song/{id}/mood", setMood(api.ds))
}

func setEnergy(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id := chi.URLParam(r, "id")

		// Parse request body
		var req setEnergyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error(ctx, "Failed to decode request body", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate energy value
		if !slices.Contains(validEnergyValues, req.Value) {
			log.Warn(ctx, "Invalid energy value", "value", req.Value)
			http.Error(w, "Invalid energy value. Must be one of: '', 'low', 'medium', 'high'", http.StatusBadRequest)
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

		// Write the tag to the file
		if err := taglib.WriteTag(fullPath, "ENERGY", req.Value); err != nil {
			log.Error(ctx, "Failed to write energy tag", "path", fullPath, err)
			http.Error(w, "Failed to write tag to file: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Update the tags in memory
		if mf.Tags == nil {
			mf.Tags = make(model.Tags)
		}
		if req.Value == "" {
			delete(mf.Tags, model.TagName("energy"))
		} else {
			mf.Tags[model.TagName("energy")] = []string{req.Value}
		}

		// Update the media file in the database
		mf.UpdatedAt = time.Now()
		if err := ds.MediaFile(ctx).Put(mf); err != nil {
			log.Error(ctx, "Failed to update media file", "id", id, err)
			http.Error(w, "Failed to update database", http.StatusInternalServerError)
			return
		}

		// Return success response
		resp := setEnergyResponse{
			ID:     id,
			Energy: req.Value,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error(ctx, "Failed to encode response", err)
		}

		log.Info(ctx, "Set energy tag", "id", id, "path", mf.Path, "value", req.Value)
	}
}

func setMood(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id := chi.URLParam(r, "id")

		// Parse request body
		var req setMoodRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error(ctx, "Failed to decode request body", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate mood value
		if !slices.Contains(validMoodValues, req.Value) {
			log.Warn(ctx, "Invalid mood value", "value", req.Value)
			http.Error(w, "Invalid mood value. Must be one of: '', 'negative', 'neutral', 'positive'", http.StatusBadRequest)
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

		// Write the tag to the file
		if err := taglib.WriteTag(fullPath, "MOOD", req.Value); err != nil {
			log.Error(ctx, "Failed to write mood tag", "path", fullPath, err)
			http.Error(w, "Failed to write tag to file: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Update the tags in memory
		if mf.Tags == nil {
			mf.Tags = make(model.Tags)
		}
		if req.Value == "" {
			delete(mf.Tags, model.TagName("mood"))
		} else {
			mf.Tags[model.TagName("mood")] = []string{req.Value}
		}

		// Update the media file in the database
		mf.UpdatedAt = time.Now()
		if err := ds.MediaFile(ctx).Put(mf); err != nil {
			log.Error(ctx, "Failed to update media file", "id", id, err)
			http.Error(w, "Failed to update database", http.StatusInternalServerError)
			return
		}

		// Return success response
		resp := setMoodResponse{
			ID:   id,
			Mood: req.Value,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error(ctx, "Failed to encode response", err)
		}

		log.Info(ctx, "Set mood tag", "id", id, "path", mf.Path, "value", req.Value)
	}
}
