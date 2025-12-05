package trackanalysisjob

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/navidrome/navidrome/adapters/taglib"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/trackanalysis"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/utils/singleton"
	"golang.org/x/time/rate"
)

var (
	ErrAlreadyRunning = errors.New("track analysis job already running")
	ErrNotAvailable   = errors.New("track analysis API not available")
)

// Stats holds the job statistics
type Stats struct {
	Total     int64 `json:"total"`
	Processed int64 `json:"processed"`
	Fetched   int64 `json:"fetched"`
	Failed    int64 `json:"failed"`
	Skipped   int64 `json:"skipped"`
}

// Status represents the current job status
type Status struct {
	Running     bool          `json:"running"`
	StartTime   time.Time     `json:"startTime,omitempty"`
	ElapsedTime time.Duration `json:"elapsedTime,omitempty"`
	Stats       Stats         `json:"stats"`
	Error       string        `json:"error,omitempty"`
}

type Job struct {
	ds      model.DataStore
	broker  events.Broker
	limiter *rate.Sometimes

	running  atomic.Bool
	cancelFn context.CancelFunc
}

func GetInstance(ds model.DataStore, broker events.Broker) *Job {
	return singleton.GetInstance(func() *Job {
		return &Job{
			ds:      ds,
			broker:  broker,
			limiter: &rate.Sometimes{Interval: 2 * time.Second},
		}
	})
}

// Start begins the track analysis job
func (j *Job) Start(ctx context.Context) error {
	if !trackanalysis.IsAvailable() {
		return ErrNotAvailable
	}

	if !j.running.CompareAndSwap(false, true) {
		return ErrAlreadyRunning
	}

	// Create an independent context for the job - don't use the request context
	// as it will be cancelled when the HTTP request completes
	jobCtx, cancelFn := context.WithCancel(context.Background())
	j.cancelFn = cancelFn
	jobCtx = auth.WithAdminUser(jobCtx, j.ds)

	go j.run(jobCtx)
	return nil
}

// Stop stops the running job
func (j *Job) Stop() {
	if j.cancelFn != nil {
		j.cancelFn()
	}
}

// Status returns the current job status
func (j *Job) Status(ctx context.Context) (*Status, error) {
	status := &Status{
		Running: j.running.Load(),
	}

	// Load stats from database
	statsStr, _ := j.ds.Property(ctx).DefaultGet(consts.TrackAnalysisJobStats, "{}")
	_ = json.Unmarshal([]byte(statsStr), &status.Stats)

	if status.Running {
		startTimeStr, _ := j.ds.Property(ctx).DefaultGet(consts.TrackAnalysisJobStartTime, "")
		if startTimeStr != "" {
			if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
				status.StartTime = startTime
				status.ElapsedTime = time.Since(startTime)
			}
		}
	}

	return status, nil
}

func (j *Job) run(ctx context.Context) {
	defer func() {
		j.running.Store(false)
		j.cancelFn = nil
		_ = j.ds.Property(ctx).Put(consts.TrackAnalysisJobInProgress, "false")
		j.sendStatus(ctx, false, Stats{}, "")
	}()

	log.Info(ctx, "Track analysis job starting")

	// Mark job as in progress
	startTime := time.Now()
	_ = j.ds.Property(ctx).Put(consts.TrackAnalysisJobInProgress, "true")
	_ = j.ds.Property(ctx).Put(consts.TrackAnalysisJobStartTime, startTime.Format(time.RFC3339))

	// Load previous stats (for resumption)
	stats := j.loadStats(ctx)

	// Get total count of songs needing processing
	total, err := j.countSongsNeedingAnalysis(ctx)
	if err != nil {
		log.Error(ctx, "Failed to count songs needing analysis", err)
		return
	}
	stats.Total = total + stats.Processed
	j.saveStats(ctx, stats)

	log.Info(ctx, "Track analysis job started", "songsRemaining", total, "previouslyProcessed", stats.Processed)

	// Get last processed ID for resumption
	lastProcessedID, _ := j.ds.Property(ctx).DefaultGet(consts.TrackAnalysisJobLastProcessedID, "")

	// Send initial status
	j.sendStatus(ctx, true, stats, "")

	for {
		select {
		case <-ctx.Done():
			log.Info(ctx, "Track analysis job cancelled")
			return
		default:
		}

		// Get next song to process
		song, err := j.getNextSong(ctx, lastProcessedID)
		if err != nil {
			log.Error(ctx, "Error getting next song", err)
			j.sendStatus(ctx, true, stats, err.Error())
			return
		}
		if song == nil {
			// No more songs to process
			log.Info(ctx, "Track analysis job completed", "stats", stats)
			// Clear progress markers on completion
			_ = j.ds.Property(ctx).Delete(consts.TrackAnalysisJobLastProcessedID)
			return
		}

		// Process the song
		result := j.processSong(ctx, *song)
		stats.Processed++
		if result.fetched {
			stats.Fetched++
		} else if result.failed {
			stats.Failed++
		} else if result.skipped {
			stats.Skipped++
		}

		// Persist progress
		lastProcessedID = song.ID
		_ = j.ds.Property(ctx).Put(consts.TrackAnalysisJobLastProcessedID, lastProcessedID)
		j.saveStats(ctx, stats)

		// Send status update (rate limited)
		j.limiter.Do(func() {
			j.sendStatus(ctx, true, stats, "")
		})

		// Small delay to avoid hammering the API
		select {
		case <-ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (j *Job) loadStats(ctx context.Context) Stats {
	var stats Stats
	statsStr, _ := j.ds.Property(ctx).DefaultGet(consts.TrackAnalysisJobStats, "{}")
	_ = json.Unmarshal([]byte(statsStr), &stats)
	return stats
}

func (j *Job) saveStats(ctx context.Context, stats Stats) {
	statsBytes, _ := json.Marshal(stats)
	_ = j.ds.Property(ctx).Put(consts.TrackAnalysisJobStats, string(statsBytes))
}

func (j *Job) sendStatus(ctx context.Context, running bool, stats Stats, errMsg string) {
	startTimeStr, _ := j.ds.Property(ctx).DefaultGet(consts.TrackAnalysisJobStartTime, "")
	var elapsed time.Duration
	if startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			elapsed = time.Since(startTime)
		}
	}

	j.broker.SendBroadcastMessage(ctx, &events.TrackAnalysisStatus{
		Running:     running,
		Total:       stats.Total,
		Processed:   stats.Processed,
		Fetched:     stats.Fetched,
		Failed:      stats.Failed,
		Skipped:     stats.Skipped,
		Error:       errMsg,
		ElapsedTime: elapsed,
	})
}

func (j *Job) countSongsNeedingAnalysis(ctx context.Context) (int64, error) {
	lastProcessedID, _ := j.ds.Property(ctx).DefaultGet(consts.TrackAnalysisJobLastProcessedID, "")
	return j.ds.MediaFile(ctx).CountWithoutTrackAnalysis(lastProcessedID)
}

func (j *Job) getNextSong(ctx context.Context, afterID string) (*model.MediaFile, error) {
	return j.ds.MediaFile(ctx).GetNextWithoutTrackAnalysis(afterID)
}

// songResult holds the result of processing a single song
type songResult struct {
	fetched bool
	failed  bool
	skipped bool
}

func (j *Job) processSong(ctx context.Context, song model.MediaFile) songResult {
	// Check if the song already has track data or was previously attempted
	filePath := song.AbsolutePath()
	tags, err := taglib.Read(filePath)
	if err == nil {
		for key := range tags {
			keyLower := strings.ToLower(key)
			// Skip if already has track data
			if keyLower == "energyval" || keyLower == "happiness" || keyLower == "instrumentalness" {
				log.Debug(ctx, "Skipping song with existing track data", "title", song.Title, "artist", song.Artist)
				return songResult{skipped: true}
			}
			// Skip if previously attempted and not found (prevents burning API credits on songs not in the database)
			if keyLower == "trackanalysis_notfound" {
				log.Debug(ctx, "Skipping song previously not found in API", "title", song.Title, "artist", song.Artist)
				return songResult{skipped: true}
			}
		}
	}

	// Fetch track data from API
	log.Info(ctx, "Fetching track data for song", "title", song.Title, "artist", song.Artist)
	data, err := trackanalysis.FetchTrackData(ctx, song.Artist, song.Title)
	if err != nil {
		if errors.Is(err, trackanalysis.ErrTrackNotFound) {
			log.Debug(ctx, "Track not found in analysis API", "title", song.Title, "artist", song.Artist)
			// Mark the song so we don't retry it - this saves API credits
			if writeErr := markTrackAnalysisNotFound(ctx, filePath); writeErr != nil {
				log.Warn(ctx, "Failed to mark song as not found", "path", filePath, writeErr)
			}
		} else if errors.Is(err, trackanalysis.ErrRateLimited) {
			log.Warn(ctx, "Rate limited by track analysis API", "title", song.Title, "artist", song.Artist)
			// Don't mark as failed - we'll retry later
		} else {
			log.Error(ctx, "Failed to fetch track data", "title", song.Title, "artist", song.Artist, err)
			// Transient errors (network, etc.) - don't mark, allow retry
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

// markTrackAnalysisNotFound writes a marker tag to the file so we don't retry fetching
// data for songs that aren't in the API database. This saves expensive API credits.
func markTrackAnalysisNotFound(ctx context.Context, filePath string) error {
	if err := taglib.WriteTag(filePath, "TRACKANALYSIS_NOTFOUND", "1"); err != nil {
		return fmt.Errorf("failed to write TRACKANALYSIS_NOTFOUND tag: %w", err)
	}
	log.Debug(ctx, "Marked song as not found in track analysis API", "path", filePath)
	return nil
}
