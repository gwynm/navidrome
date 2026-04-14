package aiplaylists

import (
	"encoding/json"
	"testing"

	"github.com/navidrome/navidrome/model/criteria"
)

func init() {
	criteria.AddTagNames([]string{"genre", "mood"})
	criteria.AddNumericTags([]string{"bpm"})
}

func TestExtractJSON_PlainJSON(t *testing.T) {
	input := `{"all":[{"contains":{"genre":"rock"}}],"limit":50}`
	got := ExtractJSON(input)
	if got != input {
		t.Errorf("ExtractJSON plain JSON:\ngot:  %s\nwant: %s", got, input)
	}
}

func TestExtractJSON_WithMarkdownFences(t *testing.T) {
	input := "Here is the result:\n```json\n{\"all\":[{\"contains\":{\"genre\":\"rock\"}}],\"limit\":50}\n```\nDone."
	want := `{"all":[{"contains":{"genre":"rock"}}],"limit":50}`
	got := ExtractJSON(input)
	if got != want {
		t.Errorf("ExtractJSON with fences:\ngot:  %s\nwant: %s", got, want)
	}
}

func TestExtractJSON_WithSurroundingText(t *testing.T) {
	input := `Sure! Here's the criteria:
{"all":[{"is":{"year":1995}}],"limit":25}
Let me know if you need changes.`
	want := `{"all":[{"is":{"year":1995}}],"limit":25}`
	got := ExtractJSON(input)
	if got != want {
		t.Errorf("ExtractJSON with surrounding text:\ngot:  %s\nwant: %s", got, want)
	}
}

func TestExtractJSON_InvalidJSON(t *testing.T) {
	input := "I don't understand the request."
	got := ExtractJSON(input)
	if got != "" {
		t.Errorf("ExtractJSON should return empty for non-JSON, got: %s", got)
	}
}

func TestExtractTrackIDs_PlainArray(t *testing.T) {
	input := `["abc123", "def456", "ghi789"]`
	got := ExtractTrackIDs(input)
	want := []string{"abc123", "def456", "ghi789"}
	assertStringSliceEqual(t, got, want)
}

func TestExtractTrackIDs_WithFences(t *testing.T) {
	input := "```json\n[\"abc123\", \"def456\"]\n```"
	got := ExtractTrackIDs(input)
	want := []string{"abc123", "def456"}
	assertStringSliceEqual(t, got, want)
}

func TestExtractTrackIDs_WithSurroundingText(t *testing.T) {
	input := `Based on your criteria, here are the tracks:
["id1", "id2", "id3"]
Enjoy your playlist!`
	got := ExtractTrackIDs(input)
	want := []string{"id1", "id2", "id3"}
	assertStringSliceEqual(t, got, want)
}

func TestExtractTrackIDs_NoArray(t *testing.T) {
	input := "Sorry, I couldn't find any matching tracks."
	got := ExtractTrackIDs(input)
	if len(got) != 0 {
		t.Errorf("Expected empty slice, got: %v", got)
	}
}

func TestBuildStrictPrompt_ContainsFields(t *testing.T) {
	prompt := BuildStrictPrompt("Sample genre values: Rock, Jazz, Pop\n")
	for _, expected := range []string{
		"genre", "mood", "year", "bpm", "artist", "title", "album",
		"contains", "is", "isNot", "gt", "lt", "inTheRange",
		"Sample genre values: Rock, Jazz, Pop",
	} {
		if !contains(prompt, expected) {
			t.Errorf("Prompt should contain %q", expected)
		}
	}
}

func TestCriteriaParseRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{
			name: "simple genre filter",
			json: `{"all":[{"contains":{"genre":"rock"}}],"limit":50}`,
		},
		{
			name: "year range with sort",
			json: `{"all":[{"inTheRange":{"year":[1990,1999]}},{"contains":{"genre":"rock"}}],"sort":"random","limit":30}`,
		},
		{
			name: "any with mood and genre",
			json: `{"any":[{"contains":{"genre":"jazz"}},{"contains":{"mood":"chill"}}],"limit":40}`,
		},
		{
			name: "nested all/any",
			json: `{"all":[{"contains":{"genre":"electronic"}},{"any":[{"gt":{"bpm":120}},{"contains":{"mood":"energetic"}}]}],"sort":"random","limit":25}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c criteria.Criteria
			if err := json.Unmarshal([]byte(tt.json), &c); err != nil {
				t.Fatalf("Failed to parse criteria JSON: %v\nInput: %s", err, tt.json)
			}

			marshaled, err := json.Marshal(c)
			if err != nil {
				t.Fatalf("Failed to marshal criteria: %v", err)
			}

			var c2 criteria.Criteria
			if err := json.Unmarshal(marshaled, &c2); err != nil {
				t.Fatalf("Failed to re-parse marshaled criteria: %v\nMarshaled: %s", err, string(marshaled))
			}

			sql, args, err := c.ToSql()
			if err != nil {
				t.Fatalf("Failed to generate SQL: %v", err)
			}
			if sql == "" {
				t.Error("Generated SQL is empty")
			}
			_ = args
		})
	}
}

func TestExtractJSON_NestedCriteria(t *testing.T) {
	llmResponse := `Based on your request for "upbeat rock songs from the 90s", here are the filters:

` + "```json" + `
{
  "all": [
    {"contains": {"genre": "rock"}},
    {"inTheRange": {"year": [1990, 1999]}},
    {"gt": {"bpm": 100}}
  ],
  "sort": "random",
  "limit": 50
}
` + "```" + `

This will find rock songs from 1990-1999 with a BPM over 100.`

	jsonStr := ExtractJSON(llmResponse)
	if jsonStr == "" {
		t.Fatal("Failed to extract JSON from LLM response")
	}

	var c criteria.Criteria
	if err := json.Unmarshal([]byte(jsonStr), &c); err != nil {
		t.Fatalf("Failed to parse extracted JSON as criteria: %v\nExtracted: %s", err, jsonStr)
	}

	sql, _, err := c.ToSql()
	if err != nil {
		t.Fatalf("Failed to generate SQL from criteria: %v", err)
	}
	if sql == "" {
		t.Error("Generated SQL should not be empty")
	}
}

func assertStringSliceEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %d, want %d\ngot:  %v\nwant: %v", len(got), len(want), got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("index %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && searchSubstring(s, substr))
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
