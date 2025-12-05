package nativeapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core/trackanalysis"
	"github.com/navidrome/navidrome/core/trackanalysisjob"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/events"
)

func (api *Router) addTrackAnalysisJobRoutes(r chi.Router) {
	r.Route("/trackanalysis", func(r chi.Router) {
		r.Get("/status", getTrackAnalysisStatus(api.ds, api.broker))
		r.Post("/start", startTrackAnalysisJob(api.ds, api.broker))
		r.Post("/stop", stopTrackAnalysisJob(api.ds, api.broker))
	})
}

func getTrackAnalysisStatus(ds model.DataStore, broker events.Broker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Check if API is available
		available := trackanalysis.IsAvailable()

		job := trackanalysisjob.GetInstance(ds, broker)
		status, err := job.Status(ctx)
		if err != nil {
			log.Error(ctx, "Failed to get track analysis job status", err)
			http.Error(w, "Failed to get status", http.StatusInternalServerError)
			return
		}

		response := struct {
			Available bool                     `json:"available"`
			Status    *trackanalysisjob.Status `json:"status"`
		}{
			Available: available,
			Status:    status,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error(ctx, "Failed to encode response", err)
		}
	}
}

func startTrackAnalysisJob(ds model.DataStore, broker events.Broker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		job := trackanalysisjob.GetInstance(ds, broker)
		err := job.Start(ctx)
		if err != nil {
			log.Warn(ctx, "Failed to start track analysis job", err)
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"started"}`))
	}
}

func stopTrackAnalysisJob(ds model.DataStore, broker events.Broker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		job := trackanalysisjob.GetInstance(ds, broker)
		job.Stop()

		log.Info(ctx, "Track analysis job stop requested")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"stopping"}`))
	}
}
