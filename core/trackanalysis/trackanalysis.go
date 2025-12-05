package trackanalysis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/navidrome/navidrome/log"
)

// ErrRateLimited indicates the API rate limit was exceeded
var ErrRateLimited = errors.New("API rate limit exceeded")

var (
	ErrTrackAnalysisNotAvailable = errors.New("track analysis API not available")
	ErrTrackNotFound             = errors.New("track not found in analysis API")
)

// TrackData contains the analysis results we care about
type TrackData struct {
	EnergyVal        int `json:"energy"`
	Happiness        int `json:"happiness"`
	Instrumentalness int `json:"instrumentalness"`
}

// apiResponse represents the full API response
type apiResponse struct {
	ID               string `json:"id"`
	Key              string `json:"key"`
	Mode             string `json:"mode"`
	Camelot          string `json:"camelot"`
	Tempo            int    `json:"tempo"`
	Duration         string `json:"duration"`
	Popularity       int    `json:"popularity"`
	Energy           int    `json:"energy"`
	Danceability     int    `json:"danceability"`
	Happiness        int    `json:"happiness"`
	Acousticness     int    `json:"acousticness"`
	Instrumentalness int    `json:"instrumentalness"`
	Liveness         int    `json:"liveness"`
	Speechiness      int    `json:"speechiness"`
	Loudness         string `json:"loudness"`
}

const (
	apiBaseURL = "https://track-analysis.p.rapidapi.com/pktx/analysis"
	apiHost    = "track-analysis.p.rapidapi.com"
	envAPIKey  = "RAPIDAPI_KEY" //nolint:gosec // This is the env var name, not a credential
)

var (
	apiKeyOnce sync.Once
	apiKey     string
)

func getAPIKey() string {
	apiKeyOnce.Do(func() {
		apiKey = os.Getenv(envAPIKey)
		if apiKey != "" {
			log.Info("Track analysis API key configured")
		}
	})
	return apiKey
}

// IsAvailable checks if the track analysis API is available
func IsAvailable() bool {
	return getAPIKey() != ""
}

// FetchTrackData fetches track analysis data from the RapidAPI service
func FetchTrackData(ctx context.Context, artist, title string) (*TrackData, error) {
	key := getAPIKey()
	if key == "" {
		return nil, ErrTrackAnalysisNotAvailable
	}

	if artist == "" || title == "" {
		return nil, fmt.Errorf("artist and title are required")
	}

	// Build the URL with query parameters
	u, err := url.Parse(apiBaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse API URL: %w", err)
	}

	q := u.Query()
	q.Set("song", title)
	q.Set("artist", artist)
	u.RawQuery = q.Encode()

	// Create the request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-rapidapi-host", apiHost)
	req.Header.Set("x-rapidapi-key", key)

	// Make the request with a timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	log.Debug(ctx, "Fetching track analysis data", "artist", artist, "title", title)

	// Retry logic with exponential backoff for rate limiting
	var resp *http.Response
	maxRetries := 6
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 2s, 4s, 8s, 16s, 32s, 64s
			backoff := time.Duration(1<<attempt) * time.Second
			log.Debug(ctx, "Rate limited, waiting before retry", "artist", artist, "title", title, "attempt", attempt+1, "backoff", backoff)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}

			// Recreate request for retry
			req, err = http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("x-rapidapi-host", apiHost)
			req.Header.Set("x-rapidapi-key", key)
		}

		resp, err = client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("API request failed: %w", err)
		}

		if resp.StatusCode != http.StatusTooManyRequests {
			break
		}
		resp.Body.Close()

		if attempt == maxRetries-1 {
			return nil, ErrRateLimited
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrTrackNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}

	log.Debug(ctx, "Received track analysis data", "artist", artist, "title", title,
		"energy", apiResp.Energy, "happiness", apiResp.Happiness, "instrumentalness", apiResp.Instrumentalness)

	return &TrackData{
		EnergyVal:        apiResp.Energy,
		Happiness:        apiResp.Happiness,
		Instrumentalness: apiResp.Instrumentalness,
	}, nil
}
