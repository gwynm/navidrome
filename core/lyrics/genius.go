package lyrics

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/navidrome/navidrome/adapters/taglib"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

var (
	ErrGeniusNotAvailable = errors.New("genius lyrics fetcher not available")
	ErrNoLyricsFound      = errors.New("no lyrics found on Genius")
)

var (
	geniusOnce      sync.Once
	geniusScriptPath string
	geniusPythonPath string
	geniusErr       error
)

func geniusCmd() (pythonPath, scriptPath string, err error) {
	geniusOnce.Do(func() {
		// Check if GENIUS_CLIENT_ACCESS_TOKEN is set
		if os.Getenv("GENIUS_CLIENT_ACCESS_TOKEN") == "" {
			geniusErr = errors.New("GENIUS_CLIENT_ACCESS_TOKEN environment variable not set")
			return
		}

		// Find Python
		geniusPythonPath, _ = exec.LookPath("python3")
		if geniusPythonPath == "" {
			geniusPythonPath, _ = exec.LookPath("python")
		}
		if geniusPythonPath == "" {
			geniusErr = errors.New("python not found")
			return
		}

		// Check if lyricsgenius module is installed
		cmd := exec.Command(geniusPythonPath, "-c", "import lyricsgenius") // #nosec
		if err := cmd.Run(); err != nil {
			geniusErr = errors.New("lyricsgenius python module not installed (run: pip install lyricsgenius)")
			return
		}

		// Find the script in common locations
		scriptLocations := []string{
			"scripts/genius_lyrics.py",
			"./scripts/genius_lyrics.py",
			"/workspaces/navidrome/scripts/genius_lyrics.py",
			"/app/scripts/genius_lyrics.py",
		}

		// Also try relative to executable
		if execPath, err := os.Executable(); err == nil {
			execDir := filepath.Dir(execPath)
			scriptLocations = append(scriptLocations,
				filepath.Join(execDir, "scripts", "genius_lyrics.py"),
				filepath.Join(execDir, "..", "scripts", "genius_lyrics.py"),
			)
		}

		// Also try relative to working directory
		if wd, err := os.Getwd(); err == nil {
			scriptLocations = append(scriptLocations,
				filepath.Join(wd, "scripts", "genius_lyrics.py"),
			)
		}

		for _, loc := range scriptLocations {
			if _, err := os.Stat(loc); err == nil {
				geniusScriptPath = loc
				geniusErr = nil
				log.Info("Found Genius lyrics script", "path", geniusScriptPath, "python", geniusPythonPath)
				return
			}
		}

		log.Warn("Genius lyrics script not found", "searchedPaths", scriptLocations)
		geniusErr = errors.New("genius_lyrics.py script not found")
	})
	return geniusPythonPath, geniusScriptPath, geniusErr
}

// IsGeniusAvailable checks if Genius lyrics fetching is available
func IsGeniusAvailable() bool {
	_, _, err := geniusCmd()
	return err == nil
}

// FetchFromGenius fetches lyrics from Genius.com for the given media file
// This is exported so it can be called directly from the API endpoint
func FetchFromGenius(ctx context.Context, mf *model.MediaFile) (model.LyricList, error) {
	return fromGenius(ctx, mf)
}

// fromGenius fetches lyrics from Genius.com for the given media file
func fromGenius(ctx context.Context, mf *model.MediaFile) (model.LyricList, error) {
	pythonPath, scriptPath, err := geniusCmd()
	if err != nil {
		log.Debug(ctx, "Genius not available", err)
		return nil, nil
	}

	artist := mf.Artist
	title := mf.Title

	if artist == "" || title == "" {
		log.Debug(ctx, "Cannot fetch from Genius: missing artist or title", "artist", artist, "title", title)
		return nil, nil
	}

	log.Debug(ctx, "Fetching lyrics from Genius", "artist", artist, "title", title)

	cmd := exec.CommandContext(ctx, pythonPath, scriptPath, artist, title) // #nosec
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			log.Debug(ctx, "No lyrics found on Genius", "artist", artist, "title", title, "stderr", string(exitErr.Stderr))
		} else {
			log.Error(ctx, "Failed to fetch lyrics from Genius", "artist", artist, "title", title, err)
		}
		return nil, nil
	}

	lyricsText := strings.TrimSpace(string(output))
	if lyricsText == "" {
		log.Debug(ctx, "Empty lyrics returned from Genius", "artist", artist, "title", title)
		return nil, nil
	}

	log.Info(ctx, "Fetched lyrics from Genius", "artist", artist, "title", title)

	// Parse the lyrics into structured format
	lyrics, err := model.ToLyrics("xxx", lyricsText)
	if err != nil {
		log.Error(ctx, "Failed to parse Genius lyrics", "artist", artist, "title", title, err)
		return nil, err
	}

	if lyrics == nil || lyrics.IsEmpty() {
		return nil, nil
	}

	// Save lyrics to the audio file
	if err := saveLyricsToFile(ctx, mf, lyricsText); err != nil {
		log.Warn(ctx, "Failed to save lyrics to audio file", "path", mf.AbsolutePath(), err)
		// Don't fail - we still return the lyrics even if saving failed
	}

	return model.LyricList{*lyrics}, nil
}

// saveLyricsToFile writes lyrics to the audio file using the appropriate format.
// For MP3: Creates a proper USLT (UnsynchronizedLyricsFrame) ID3v2 frame
// For FLAC/OGG: Uses LYRICS Vorbis comment
// For M4A: Uses the ©lyr atom
func saveLyricsToFile(ctx context.Context, mf *model.MediaFile, lyricsText string) error {
	path := mf.AbsolutePath()
	
	log.Debug(ctx, "Saving lyrics to audio file", "path", path)
	
	// Use "xxx" as the language code (unspecified) since Genius doesn't provide language info
	err := taglib.WriteLyrics(path, "xxx", lyricsText)
	if err != nil {
		return err
	}

	log.Info(ctx, "Saved lyrics to audio file", "path", path)
	return nil
}

