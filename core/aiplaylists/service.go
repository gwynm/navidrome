package aiplaylists

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
	"github.com/navidrome/navidrome/model/request"
)

type PreviewTrack struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
}

type PreviewResult struct {
	Rules           string         `json:"rules"`
	Tracks          []PreviewTrack `json:"tracks"`
	NeedsConfirm    bool           `json:"needsConfirm,omitempty"`
	CSVRows         int            `json:"csvRows,omitempty"`
	CSVSizeBytes    int            `json:"csvSizeBytes,omitempty"`
	EstimatedTokens int            `json:"estimatedTokens,omitempty"`
}

type ConfirmResult struct {
	PlaylistID string `json:"playlistId"`
}

type Service struct {
	ds  model.DataStore
	llm *LLMClient
}

func NewService(ds model.DataStore, llm *LLMClient) *Service {
	return &Service{ds: ds, llm: llm}
}

func (s *Service) Preview(ctx context.Context, name, userCriteria, mode string, confirmed bool) (*PreviewResult, error) {
	switch mode {
	case "strict":
		return s.previewStrict(ctx, userCriteria)
	case "vibes":
		return s.previewVibes(ctx, userCriteria, confirmed)
	default:
		return nil, fmt.Errorf("unknown mode: %q", mode)
	}
}

func (s *Service) Confirm(ctx context.Context, name string, trackIDs []string, rules, userCriteria string) (*ConfirmResult, error) {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return nil, fmt.Errorf("no user in context")
	}

	comment := fmt.Sprintf("AI Playlist\nCriteria: %s\nRules: %s", userCriteria, rules)

	pls := &model.Playlist{
		ID:      uuid.NewString(),
		Name:    name,
		Comment: comment,
		OwnerID: user.ID,
		Public:  false,
	}

	err := s.ds.WithTx(func(tx model.DataStore) error {
		if err := tx.Playlist(ctx).Put(pls); err != nil {
			return fmt.Errorf("creating playlist: %w", err)
		}
		tracksRepo := tx.Playlist(ctx).Tracks(pls.ID, false)
		if _, err := tracksRepo.Add(trackIDs); err != nil {
			return fmt.Errorf("adding tracks: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &ConfirmResult{PlaylistID: pls.ID}, nil
}

// --- Strict mode ---

func (s *Service) previewStrict(ctx context.Context, userCriteria string) (*PreviewResult, error) {
	sampleValues, err := s.getSampleTagValues(ctx)
	if err != nil {
		log.Error(ctx, "Error getting sample tag values", err)
		sampleValues = ""
	}

	prompt := buildStrictPrompt(sampleValues)

	response, err := s.llm.Complete(ctx, prompt, userCriteria)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	criteriaJSON := extractJSON(response)
	if criteriaJSON == "" {
		return nil, fmt.Errorf("could not extract criteria JSON from LLM response: %s", response)
	}

	var c criteria.Criteria
	if err := json.Unmarshal([]byte(criteriaJSON), &c); err != nil {
		return nil, fmt.Errorf("parsing criteria JSON: %w (raw: %s)", err, criteriaJSON)
	}

	tracks, err := s.queryWithCriteria(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("querying tracks: %w", err)
	}

	previewTracks := make([]PreviewTrack, len(tracks))
	for i, t := range tracks {
		previewTracks[i] = PreviewTrack{ID: t.ID, Title: t.Title, Artist: t.Artist}
	}

	return &PreviewResult{
		Rules:  criteriaJSON,
		Tracks: previewTracks,
	}, nil
}

func (s *Service) queryWithCriteria(ctx context.Context, c criteria.Criteria) (model.MediaFiles, error) {
	opts := model.QueryOptions{
		Filters: c,
		Max:     500,
	}
	if c.Sort != "" {
		opts.Sort = c.Sort
		opts.Order = c.Order
	}
	if c.Limit > 0 && c.Limit < opts.Max {
		opts.Max = c.Limit
	}

	return s.ds.MediaFile(ctx).GetAll(opts)
}

func (s *Service) getSampleTagValues(ctx context.Context) (string, error) {
	var sb strings.Builder

	tagNames := []model.TagName{model.TagGenre, model.TagMood, model.TagKeyword, model.TagGrouping}

	for _, tagName := range tagNames {
		mfs, err := s.ds.MediaFile(ctx).GetAllByTags(tagName, nil, model.QueryOptions{Max: 50})
		if err != nil {
			continue
		}
		seen := map[string]bool{}
		var values []string
		for _, mf := range mfs {
			for _, v := range mf.Tags[tagName] {
				if !seen[v] {
					seen[v] = true
					values = append(values, v)
				}
				if len(values) >= 20 {
					break
				}
			}
			if len(values) >= 20 {
				break
			}
		}
		if len(values) > 0 {
			sb.WriteString(fmt.Sprintf("Sample %s values: %s\n", tagName, strings.Join(values, ", ")))
		}
	}

	return sb.String(), nil
}

func BuildStrictPrompt(sampleValues string) string {
	return buildStrictPrompt(sampleValues)
}

func buildStrictPrompt(sampleValues string) string {
	return `You are a music playlist filter generator. Given a user's description of the playlist they want,
return a JSON object representing filter criteria that can be used to query a music database.

The JSON format uses this structure:
{
  "all": [ ...conditions that must ALL match... ],
  "any": [ ...conditions where ANY can match... ],
  "sort": "fieldName",
  "order": "asc" or "desc",
  "limit": 50
}

You can nest "all" and "any" blocks. Each condition is an object with one operator key:
- {"is": {"field": "value"}} - exact match
- {"isNot": {"field": "value"}} - not equal
- {"contains": {"field": "value"}} - substring match
- {"notContains": {"field": "value"}} - does not contain
- {"startsWith": {"field": "value"}} - starts with
- {"endsWith": {"field": "value"}} - ends with
- {"gt": {"field": value}} - greater than (numeric/date)
- {"lt": {"field": value}} - less than (numeric/date)
- {"inTheRange": {"field": [min, max]}} - between two values
- {"before": {"field": "date"}} - date before
- {"after": {"field": "date"}} - date after
- {"inTheLast": {"field": numberOfDays}} - within the last N days (for date fields)

Available fields:
- title, album, artist (text fields)
- genre, mood, keyword, grouping (tag fields - use "contains" for partial matching)
- year, originalyear, releaseyear (integer)
- duration (seconds, float)
- bpm (integer)
- bitrate (integer, kbps)
- rating (0-5 integer)
- playcount (integer)
- loved (boolean true/false)
- compilation (boolean)
- comment (text)
- dateadded, datemodified, lastplayed (date fields)
- tracknumber, discnumber (integer)
- filepath, filetype (text)

` + sampleValues + `
IMPORTANT RULES:
1. Return ONLY the JSON object, no explanation or markdown fences.
2. Use "contains" for genre and mood matching since they may be multi-valued tags.
3. Default to limit 50 if the user doesn't specify a count.
4. Use "random" as the sort field if the user wants a random/shuffled selection.
5. Think about what database filters best capture the user's intent.`
}

// --- Vibes mode ---

func (s *Service) previewVibes(ctx context.Context, userCriteria string, confirmed bool) (*PreviewResult, error) {
	csvData, rowCount, err := s.buildCSV(ctx)
	if err != nil {
		return nil, fmt.Errorf("building CSV: %w", err)
	}

	csvSize := len(csvData)
	estimatedTokens := csvSize / 4

	if !confirmed {
		return &PreviewResult{
			Rules:           "(vibes)",
			NeedsConfirm:    true,
			CSVRows:         rowCount,
			CSVSizeBytes:    csvSize,
			EstimatedTokens: estimatedTokens,
		}, nil
	}

	prompt := buildVibesPrompt()

	userMsg := fmt.Sprintf("User's criteria: %s\n\nHere is the complete music library as CSV:\n\n%s", userCriteria, csvData)

	response, err := s.llm.Complete(ctx, prompt, userMsg)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	trackIDs := extractTrackIDs(response)
	if len(trackIDs) == 0 {
		return nil, fmt.Errorf("LLM returned no track IDs. Response: %s", response)
	}

	tracks, err := s.fetchTracksByIDs(ctx, trackIDs)
	if err != nil {
		return nil, fmt.Errorf("fetching tracks: %w", err)
	}

	previewTracks := make([]PreviewTrack, len(tracks))
	for i, t := range tracks {
		previewTracks[i] = PreviewTrack{ID: t.ID, Title: t.Title, Artist: t.Artist}
	}

	return &PreviewResult{
		Rules:  "(vibes)",
		Tracks: previewTracks,
	}, nil
}

func (s *Service) buildCSV(ctx context.Context) (string, int, error) {
	mfs, err := s.ds.MediaFile(ctx).GetAll()
	if err != nil {
		return "", 0, err
	}

	var sb strings.Builder
	w := csv.NewWriter(&sb)

	header := []string{"id", "title", "artist", "album", "album_artist", "genre", "year", "duration_secs", "bpm", "mood", "comment", "compilation", "rating", "play_count", "loved"}
	if err := w.Write(header); err != nil {
		return "", 0, err
	}

	for _, mf := range mfs {
		mood := strings.Join(mf.Tags[model.TagMood], "; ")
		genre := mf.Genre
		if genre == "" {
			genre = strings.Join(mf.Tags[model.TagGenre], "; ")
		}

		loved := "false"
		if mf.Starred {
			loved = "true"
		}

		row := []string{
			mf.ID,
			mf.Title,
			mf.Artist,
			mf.Album,
			mf.AlbumArtist,
			genre,
			fmt.Sprintf("%d", mf.Year),
			fmt.Sprintf("%.0f", mf.Duration),
			fmt.Sprintf("%d", mf.BPM),
			mood,
			mf.Comment,
			fmt.Sprintf("%t", mf.Compilation),
			fmt.Sprintf("%d", mf.Rating),
			fmt.Sprintf("%d", mf.PlayCount),
			loved,
		}
		if err := w.Write(row); err != nil {
			return "", 0, err
		}
	}

	w.Flush()
	return sb.String(), len(mfs), nil
}

func buildVibesPrompt() string {
	return `You are a music curator AI. You will receive a user's description of the playlist they want,
plus a CSV file containing their entire music library.

Your job is to select tracks that best match the user's criteria. Use both the data in the CSV
and your own knowledge about the artists, albums, and songs to make great selections.

Return a JSON array of track IDs (the "id" column from the CSV). Select around 30-50 tracks
unless the user specifies otherwise.

IMPORTANT RULES:
1. Return ONLY a JSON array of ID strings, like: ["id1", "id2", "id3"]
2. No explanation or markdown fences. Just the JSON array.
3. Use your music knowledge - if the user asks for "chill vibes", pick songs you know are mellow,
   even if the metadata doesn't explicitly say so.
4. Consider genre, mood, BPM, artist style, and any other relevant factors.
5. Try to create a cohesive playlist that flows well.`
}

func (s *Service) fetchTracksByIDs(ctx context.Context, ids []string) (model.MediaFiles, error) {
	var result model.MediaFiles
	for _, id := range ids {
		mf, err := s.ds.MediaFile(ctx).Get(id)
		if err != nil {
			log.Warn(ctx, "Track not found, skipping", "id", id, err)
			continue
		}
		result = append(result, *mf)
	}
	return result, nil
}

// --- Helpers ---

var jsonBlockRe = regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.+?\\})\\s*```")

func extractJSON(response string) string {
	response = strings.TrimSpace(response)

	if m := jsonBlockRe.FindStringSubmatch(response); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}

	if strings.HasPrefix(response, "{") {
		var js json.RawMessage
		if json.Unmarshal([]byte(response), &js) == nil {
			return response
		}
	}

	// Try to find a JSON object by scanning for balanced braces
	start := strings.Index(response, "{")
	if start >= 0 {
		if obj := extractBalancedJSON(response[start:]); obj != "" {
			return obj
		}
	}

	return ""
}

func extractBalancedJSON(s string) string {
	depth := 0
	inString := false
	escape := false
	for i, ch := range s {
		if escape {
			escape = false
			continue
		}
		if ch == '\\' && inString {
			escape = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 {
				candidate := s[:i+1]
				var js json.RawMessage
				if json.Unmarshal([]byte(candidate), &js) == nil {
					return candidate
				}
				return ""
			}
		}
	}
	return ""
}

func ExtractJSON(response string) string {
	return extractJSON(response)
}

var jsonArrayBlockRe = regexp.MustCompile("(?s)```(?:json)?\\s*(\\[.*?\\])\\s*```")

func extractTrackIDs(response string) []string {
	response = strings.TrimSpace(response)

	jsonStr := ""
	if m := jsonArrayBlockRe.FindStringSubmatch(response); len(m) > 1 {
		jsonStr = m[1]
	} else if strings.HasPrefix(response, "[") {
		jsonStr = response
	} else {
		re := regexp.MustCompile("(?s)(\\[\\s*\"[^\"]*\"(?:\\s*,\\s*\"[^\"]*\")*\\s*\\])")
		if m := re.FindString(response); m != "" {
			jsonStr = m
		}
	}

	if jsonStr == "" {
		return nil
	}

	var ids []string
	if err := json.Unmarshal([]byte(jsonStr), &ids); err != nil {
		return nil
	}
	return ids
}

func ExtractTrackIDs(response string) []string {
	return extractTrackIDs(response)
}
