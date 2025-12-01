package audio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/navidrome/navidrome/log"
)

var (
	ErrAnalyzerNotAvailable = errors.New("audio analyzer not available")
	ErrAnalysisFailed       = errors.New("audio analysis failed")
)

// AudioMetrics contains the extracted audio features
type AudioMetrics struct {
	BPM             float64 `json:"bpm"`
	BeatsLoudness   float64 `json:"beatsLoudness"`
	AverageLoudness float64 `json:"averageLoudness"`
	Danceability    float64 `json:"danceability"`
}

// ScoreBreakdown shows how the energy score was calculated
type ScoreBreakdown struct {
	BPMScore          int `json:"bpmScore"`
	BeatsScore        int `json:"beatsScore"`
	LoudnessScore     int `json:"loudnessScore"`
	DanceabilityScore int `json:"danceabilityScore"`
	TotalScore        int `json:"totalScore"`
}

// AnalysisResult contains the suggested energy and metrics
type AnalysisResult struct {
	SuggestedEnergy string         `json:"suggestedEnergy"`
	Metrics         AudioMetrics   `json:"metrics"`
	Score           ScoreBreakdown `json:"score"`
}

// Analyzer interface for audio analysis
type Analyzer interface {
	Analyze(ctx context.Context, path string) (*AnalysisResult, error)
	IsAvailable() bool
}

// EssentiaAnalyzer implements audio analysis using Essentia CLI
type EssentiaAnalyzer struct{}

func NewAnalyzer() Analyzer {
	return &EssentiaAnalyzer{}
}

var (
	essentiaOnce  sync.Once
	essentiaPath  string
	essentiaErr   error
	pythonPath    string
	useScriptMode bool
)

func essentiaCmd() (string, error) {
	essentiaOnce.Do(func() {
		// First try to find the official CLI tool
		essentiaPath, essentiaErr = exec.LookPath("essentia_streaming_extractor_music")
		if essentiaErr == nil {
			log.Info("Found essentia CLI", "path", essentiaPath)
			return
		}

		// Fall back to our Python script wrapper
		// Check if Python and essentia module are available
		pythonPath, _ = exec.LookPath("python3")
		if pythonPath == "" {
			pythonPath, _ = exec.LookPath("python")
		}
		if pythonPath == "" {
			essentiaErr = errors.New("neither essentia CLI nor python found")
			return
		}

		// Check if essentia module is installed
		cmd := exec.Command(pythonPath, "-c", "import essentia") // #nosec
		if err := cmd.Run(); err != nil {
			essentiaErr = errors.New("essentia python module not installed")
			return
		}

		// Find the script in common locations
		scriptLocations := []string{
			"scripts/essentia_analyze.py",
			"./scripts/essentia_analyze.py",
			"/workspaces/navidrome/scripts/essentia_analyze.py",
			"/app/scripts/essentia_analyze.py",
		}

		// Also try relative to executable
		if execPath, err := os.Executable(); err == nil {
			execDir := filepath.Dir(execPath)
			scriptLocations = append(scriptLocations,
				filepath.Join(execDir, "scripts", "essentia_analyze.py"),
				filepath.Join(execDir, "..", "scripts", "essentia_analyze.py"),
			)
		}

		// Also try relative to working directory
		if wd, err := os.Getwd(); err == nil {
			scriptLocations = append(scriptLocations,
				filepath.Join(wd, "scripts", "essentia_analyze.py"),
			)
		}

		for _, loc := range scriptLocations {
			if _, err := os.Stat(loc); err == nil {
				essentiaPath = loc
				useScriptMode = true
				essentiaErr = nil
				log.Info("Found essentia Python script", "path", essentiaPath, "python", pythonPath)
				return
			}
		}

		log.Warn("Essentia script not found", "searchedPaths", scriptLocations)
		essentiaErr = errors.New("essentia_analyze.py script not found")
	})
	return essentiaPath, essentiaErr
}

func (e *EssentiaAnalyzer) IsAvailable() bool {
	_, err := essentiaCmd()
	return err == nil
}

func (e *EssentiaAnalyzer) Analyze(ctx context.Context, audioPath string) (*AnalysisResult, error) {
	cmdPath, err := essentiaCmd()
	if err != nil {
		return nil, ErrAnalyzerNotAvailable
	}

	// Check if file exists
	if _, err := os.Stat(audioPath); err != nil {
		return nil, fmt.Errorf("audio file not found: %w", err)
	}

	// Create temp file for output
	tmpFile, err := os.CreateTemp("", "essentia-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	outputPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(outputPath)

	// Run essentia analysis
	log.Debug(ctx, "Running essentia analysis", "input", audioPath, "output", outputPath, "scriptMode", useScriptMode)

	var cmd *exec.Cmd
	if useScriptMode {
		cmd = exec.CommandContext(ctx, pythonPath, cmdPath, audioPath, outputPath) // #nosec
	} else {
		cmd = exec.CommandContext(ctx, cmdPath, audioPath, outputPath) // #nosec
	}
	cmd.Stderr = nil
	cmd.Stdout = nil

	if err := cmd.Run(); err != nil {
		log.Error(ctx, "Essentia analysis failed", "path", audioPath, err)
		return nil, fmt.Errorf("%w: %v", ErrAnalysisFailed, err)
	}

	// Parse output JSON
	metrics, err := parseEssentiaOutput(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse essentia output: %w", err)
	}

	// Classify energy
	suggestedEnergy, scoreBreakdown := classifyEnergy(metrics)

	return &AnalysisResult{
		SuggestedEnergy: suggestedEnergy,
		Metrics:         *metrics,
		Score:           scoreBreakdown,
	}, nil
}

// essentiaOutput represents the structure of Essentia's JSON output
// We only extract the fields we need
type essentiaOutput struct {
	Rhythm struct {
		BPM           float64 `json:"bpm"`
		BeatsLoudness struct {
			Mean float64 `json:"mean"`
		} `json:"beats_loudness"`
	} `json:"rhythm"`
	Lowlevel struct {
		AverageLoudness float64 `json:"average_loudness"`
	} `json:"lowlevel"`
	Highlevel struct {
		Danceability struct {
			All struct {
				Danceable float64 `json:"danceable"`
			} `json:"all"`
		} `json:"danceability"`
	} `json:"highlevel"`
}

func parseEssentiaOutput(outputPath string) (*AudioMetrics, error) {
	data, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, err
	}

	var output essentiaOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, err
	}

	return &AudioMetrics{
		BPM:             output.Rhythm.BPM,
		BeatsLoudness:   output.Rhythm.BeatsLoudness.Mean,
		AverageLoudness: output.Lowlevel.AverageLoudness,
		Danceability:    output.Highlevel.Danceability.All.Danceable,
	}, nil
}

// classifyEnergy determines energy level based on audio metrics
func classifyEnergy(metrics *AudioMetrics) (string, ScoreBreakdown) {
	breakdown := ScoreBreakdown{}

	// BPM contribution
	switch {
	case metrics.BPM >= 130:
		breakdown.BPMScore = 3
	case metrics.BPM >= 110:
		breakdown.BPMScore = 2
	case metrics.BPM >= 85:
		breakdown.BPMScore = 1
	}

	// Beat strength contribution (0-1 scale, higher = punchier)
	switch {
	case metrics.BeatsLoudness >= 0.7:
		breakdown.BeatsScore = 2
	case metrics.BeatsLoudness >= 0.4:
		breakdown.BeatsScore = 1
	}

	// Loudness contribution (0-1 scale)
	if metrics.AverageLoudness >= 0.8 {
		breakdown.LoudnessScore = 1
	}

	// Danceability boost (0-1 scale)
	if metrics.Danceability >= 0.7 {
		breakdown.DanceabilityScore = 1
	}

	breakdown.TotalScore = breakdown.BPMScore + breakdown.BeatsScore + breakdown.LoudnessScore + breakdown.DanceabilityScore

	// Classify based on score
	var energy string
	switch {
	case breakdown.TotalScore >= 5:
		energy = "high"
	case breakdown.TotalScore >= 2:
		energy = "medium"
	default:
		energy = "low"
	}

	return energy, breakdown
}

// GetAudioFilePath returns the full path to an audio file
func GetAudioFilePath(libraryPath, relativePath string) string {
	return filepath.Join(libraryPath, relativePath)
}
