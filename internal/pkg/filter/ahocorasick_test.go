package filter

import (
	"testing"
)

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase",
			input:    "HELLO WORLD",
			expected: "hello world",
		},
		{
			name:     "leetspeak numbers",
			input:    "h3ll0 w0rld",
			expected: "hello world",
		},
		{
			name:     "leetspeak symbols",
			input:    "he110 wor1d",
			expected: "heiio worid",
		},
		{
			name:     "at sign to a",
			input:    "b@dword",
			expected: "badword",
		},
		{
			name:     "dollar sign to s",
			input:    "a$$hole",
			expected: "asshole",
		},
		{
			name:     "mixed case and leetspeak",
			input:    "B4DW0RD",
			expected: "badword",
		},
		{
			name:     "unicode diacritics",
			input:    "café résumé",
			expected: "cafe resume",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only numbers",
			input:    "12345",
			expected: "i2eas", // 2 is not in leetspeak map
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeText(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeText(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAhoCorasick_Build(t *testing.T) {
	ac := NewAhoCorasick()
	patterns := []PatternInfo{
		{Word: "bad", Category: "profanity", Severity: 3},
		{Word: "word", Category: "spam", Severity: 2},
		{Word: "badword", Category: "profanity", Severity: 5},
	}

	ac.Build(patterns)

	// Verify the automaton was built by searching
	if !ac.HasMatch("this contains bad content") {
		t.Error("Expected to find 'bad' in text")
	}
	if !ac.HasMatch("this contains word") {
		t.Error("Expected to find 'word' in text")
	}
	if !ac.HasMatch("this contains badword") {
		t.Error("Expected to find 'badword' in text")
	}
}

func TestAhoCorasick_Search(t *testing.T) {
	ac := NewAhoCorasick()
	patterns := []PatternInfo{
		{Word: "he", Category: "test", Severity: 1},
		{Word: "she", Category: "test", Severity: 1},
		{Word: "his", Category: "test", Severity: 1},
		{Word: "hers", Category: "test", Severity: 1},
	}
	ac.Build(patterns)

	tests := []struct {
		name          string
		text          string
		expectedCount int
		expectedWords map[string]bool
	}{
		{
			name:          "single match",
			text:          "he is here",
			expectedCount: 2,
			expectedWords: map[string]bool{"he": true},
		},
		{
			name:          "overlapping matches",
			text:          "she",
			expectedCount: 2,
			expectedWords: map[string]bool{"he": true, "she": true},
		},
		{
			name:          "multiple different matches",
			text:          "she said his name",
			expectedCount: 3,
			expectedWords: map[string]bool{"he": true, "she": true, "his": true},
		},
		{
			name:          "no matches",
			text:          "no matches here",
			expectedCount: 2,
			expectedWords: map[string]bool{"he": true},
		},
		{
			name:          "empty text",
			text:          "",
			expectedCount: 0,
			expectedWords: map[string]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := ac.Search(tt.text)
			if len(matches) != tt.expectedCount {
				t.Errorf("Search(%q) returned %d matches; want %d", tt.text, len(matches), tt.expectedCount)
				return
			}
			for _, match := range matches {
				if !tt.expectedWords[match.Word] {
					t.Errorf("Search(%q) found unexpected word %q", tt.text, match.Word)
				}
			}
		})
	}
}

func TestAhoCorasick_HasMatch(t *testing.T) {
	ac := NewAhoCorasick()
	patterns := []PatternInfo{
		{Word: "spam", Category: "spam", Severity: 2},
		{Word: "scam", Category: "fraud", Severity: 5},
	}
	ac.Build(patterns)

	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "has spam",
			text:     "this is spam content",
			expected: true,
		},
		{
			name:     "has scam",
			text:     "this is a scam",
			expected: true,
		},
		{
			name:     "clean text",
			text:     "this is clean text",
			expected: false,
		},
		{
			name:     "empty text",
			text:     "",
			expected: false,
		},
		{
			name:     "partial match not counted",
			text:     "spa",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ac.HasMatch(tt.text)
			if result != tt.expected {
				t.Errorf("HasMatch(%q) = %v; want %v", tt.text, result, tt.expected)
			}
		})
	}
}

func TestAhoCorasick_CaseInsensitive(t *testing.T) {
	ac := NewAhoCorasick()
	patterns := []PatternInfo{
		{Word: "badword", Category: "profanity", Severity: 3},
	}
	ac.Build(patterns)

	tests := []struct {
		text     string
		expected bool
	}{
		{"this contains badword", true},
		{"this contains BADWORD", true},
		{"this contains BadWord", true},
		{"this contains bAdWoRd", true},
		{"this is clean", false},
	}

	for _, tt := range tests {
		result := ac.HasMatch(tt.text)
		if result != tt.expected {
			t.Errorf("HasMatch(%q) = %v; want %v", tt.text, result, tt.expected)
		}
	}
}

func TestAhoCorasick_Leetspeak(t *testing.T) {
	ac := NewAhoCorasick()
	patterns := []PatternInfo{
		{Word: "badword", Category: "profanity", Severity: 5},
		{Word: "spam", Category: "spam", Severity: 3},
	}
	ac.Build(patterns)

	tests := []struct {
		text     string
		expected bool
	}{
		{"this contains b4dw0rd", true}, // leetspeak
		{"this contains b@dword", true}, // @ -> a
		{"this contains $pam", true},    // $ -> s
		{"this contains 5pam", true},    // 5 -> s
		{"this contains sp4m", true},    // 4 -> a
		{"this is clean text", false},
	}

	for _, tt := range tests {
		result := ac.HasMatch(tt.text)
		if result != tt.expected {
			t.Errorf("HasMatch(%q) = %v; want %v", tt.text, result, tt.expected)
		}
	}
}

func TestAhoCorasick_SeverityInMatches(t *testing.T) {
	ac := NewAhoCorasick()
	patterns := []PatternInfo{
		{Word: "mild", Category: "mild", Severity: 1},
		{Word: "moderate", Category: "moderate", Severity: 3},
		{Word: "severe", Category: "severe", Severity: 5},
	}
	ac.Build(patterns)

	matches := ac.Search("mild and severe content")

	if len(matches) != 2 {
		t.Fatalf("Expected 2 matches, got %d", len(matches))
	}

	// Check severities
	foundMild, foundSevere := false, false
	for _, m := range matches {
		if m.Word == "mild" && m.Severity == 1 {
			foundMild = true
		}
		if m.Word == "severe" && m.Severity == 5 {
			foundSevere = true
		}
	}

	if !foundMild {
		t.Error("Expected to find 'mild' with severity 1")
	}
	if !foundSevere {
		t.Error("Expected to find 'severe' with severity 5")
	}
}

func BenchmarkAhoCorasick_Search(b *testing.B) {
	ac := NewAhoCorasick()
	patterns := make([]PatternInfo, 1000)
	for i := 0; i < 1000; i++ {
		patterns[i] = PatternInfo{
			Word:     "pattern" + string(rune('a'+i%26)),
			Category: "test",
			Severity: 1,
		}
	}
	ac.Build(patterns)

	text := "This is a long text that contains patterna and patternb and some other content that needs to be searched."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ac.Search(text)
	}
}

func BenchmarkAhoCorasick_HasMatch(b *testing.B) {
	ac := NewAhoCorasick()
	patterns := make([]PatternInfo, 1000)
	for i := 0; i < 1000; i++ {
		patterns[i] = PatternInfo{
			Word:     "pattern" + string(rune('a'+i%26)),
			Category: "test",
			Severity: 1,
		}
	}
	ac.Build(patterns)

	text := "This is clean text without any matches"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ac.HasMatch(text)
	}
}
