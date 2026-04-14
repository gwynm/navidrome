package nativeapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/aiplaylists"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

func (api *Router) addAIPlaylistRoute(r chi.Router) {
	r.Route("/ai/playlist", func(r chi.Router) {
		r.Post("/preview", aiPlaylistPreview(api.ds))
		r.Post("/confirm", aiPlaylistConfirm(api.ds))
	})
}

func newAIService(ds model.DataStore) *aiplaylists.Service {
	llm := aiplaylists.NewLLMClient(conf.Server.OpenRouter.APIKey, conf.Server.OpenRouter.Model)
	return aiplaylists.NewService(ds, llm)
}

func aiPlaylistPreview(ds model.DataStore) http.HandlerFunc {
	type previewRequest struct {
		Name      string `json:"name"`
		Criteria  string `json:"criteria"`
		Mode      string `json:"mode"`
		Confirmed bool   `json:"confirmed"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if conf.Server.OpenRouter.APIKey == "" {
			http.Error(w, "OpenRouter API key not configured (set ND_OPENROUTER_APIKEY)", http.StatusServiceUnavailable)
			return
		}

		var req previewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}
		if req.Criteria == "" {
			http.Error(w, "criteria is required", http.StatusBadRequest)
			return
		}
		if req.Mode == "" {
			req.Mode = "strict"
		}

		svc := newAIService(ds)
		result, err := svc.Preview(r.Context(), req.Name, req.Criteria, req.Mode, req.Confirmed)
		if err != nil {
			log.Error(r.Context(), "AI playlist preview failed", err)
			http.Error(w, "AI playlist generation failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(result); err != nil {
			log.Error(r.Context(), "Error encoding response", err)
		}
	}
}

func aiPlaylistConfirm(ds model.DataStore) http.HandlerFunc {
	type confirmRequest struct {
		Name     string   `json:"name"`
		TrackIDs []string `json:"trackIds"`
		Rules    string   `json:"rules"`
		Criteria string   `json:"criteria"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req confirmRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}
		if req.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}
		if len(req.TrackIDs) == 0 {
			http.Error(w, "trackIds is required", http.StatusBadRequest)
			return
		}

		svc := newAIService(ds)
		result, err := svc.Confirm(r.Context(), req.Name, req.TrackIDs, req.Rules, req.Criteria)
		if err != nil {
			log.Error(r.Context(), "AI playlist confirm failed", err)
			http.Error(w, "Failed to create playlist: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(result); err != nil {
			log.Error(r.Context(), "Error encoding response", err)
		}
	}
}
