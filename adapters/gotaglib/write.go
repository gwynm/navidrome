package gotaglib

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/navidrome/navidrome/log"
	"go.senan.xyz/taglib"
)

// WriteTag writes a single tag to an audio file. If tagValue is empty, the tag is cleared.
// tagName should be the PropertyMap key (e.g. "ENERGY", "MOOD", "KEYWORDS").
func WriteTag(path string, tagName string, tagValue string) (err error) {
	debug.SetPanicOnFault(true)
	defer func() {
		if r := recover(); r != nil {
			log.Error("gotaglib: recovered from panic when writing tag", "file", path, "tag", tagName, "error", r)
			err = fmt.Errorf("gotaglib: recovered from panic: %s", r)
		}
	}()

	var values []string
	if tagValue != "" {
		values = []string{tagValue}
	}

	tags := map[string][]string{
		strings.ToUpper(tagName): values,
	}

	log.Debug("gotaglib: writing tag", "path", path, "tag", tagName, "value", tagValue)
	if err := taglib.WriteTags(path, tags, 0); err != nil {
		return fmt.Errorf("failed to write tag to %s: %w", path, err)
	}
	return nil
}

// WriteLyrics writes lyrics with a language code to an audio file.
// The language should be an ISO 639-2 code (e.g. "eng", "xxx" for unspecified).
// TagLib's PropertyMap uses "LYRICS:lang" as the key for language-specific lyrics.
func WriteLyrics(path string, language string, lyricsText string) (err error) {
	debug.SetPanicOnFault(true)
	defer func() {
		if r := recover(); r != nil {
			log.Error("gotaglib: recovered from panic when writing lyrics", "file", path, "error", r)
			err = fmt.Errorf("gotaglib: recovered from panic: %s", r)
		}
	}()

	key := "LYRICS:" + language
	var values []string
	if lyricsText != "" {
		values = []string{lyricsText}
	}

	tags := map[string][]string{key: values}

	log.Debug("gotaglib: writing lyrics", "path", path, "language", language)
	if err := taglib.WriteTags(path, tags, 0); err != nil {
		return fmt.Errorf("failed to write lyrics to %s: %w", path, err)
	}
	return nil
}

// Read reads all tags from an audio file and returns them as a map.
// This is a convenience wrapper for cases that need direct file path access
// rather than going through the fs.FS-based extractor.
func Read(path string) (tags map[string][]string, err error) {
	debug.SetPanicOnFault(true)
	defer func() {
		if r := recover(); r != nil {
			log.Error("gotaglib: recovered from panic when reading tags", "file", path, "error", r)
			err = fmt.Errorf("gotaglib: recovered from panic: %s", r)
		}
	}()

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tf, err := taglib.OpenStream(f, taglib.WithFilename(path))
	if err != nil {
		return nil, fmt.Errorf("failed to open tags for %s: %w", path, err)
	}
	defer tf.Close()

	allTags := tf.AllTags()
	result := make(map[string][]string, len(allTags.Tags))
	for key, values := range allTags.Tags {
		result[strings.ToLower(key)] = values
	}
	processRawTags(allTags, result)
	return result, nil
}
