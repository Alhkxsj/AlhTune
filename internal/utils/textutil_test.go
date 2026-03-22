package utils

import (
	"testing"
)

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", ""},
		{"lowercase", "Hello World", "helloworld"},
		{"with numbers", "Song 123", "song123"},
		{"with chinese", "周杰伦 123", "周杰伦123"},
		{"with_special_chars", "Hello, World!", "helloworld"},
		{"mixed", "周杰伦 feat. Taylor Swift", "周杰伦feattaylorswift"},
		{"only_special", "!@#$%", ""},
		{"only letters", "ABC", "abc"},
		{"only numbers", "123", "123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeText(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeText(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSimilarityScore(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected float64
	}{
		{"identical", "hello", "hello", 1.0},
		{"empty a", "", "hello", 0.0},
		{"empty b", "hello", "", 0.0},
		{"both empty", "", "", 1.0}, // Empty strings are considered equal
		{"one_char_diff", "hello", "hellp", 0.8},
		{"two_char_diff", "hello", "heilo", 0.8},
		{"completely_different", "hello", "world", 0.2},
		{"chinese identical", "周杰伦", "周杰伦", 1.0},
		{"chinese similar", "周杰伦", "周杰论", 0.6666666666666666},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SimilarityScore(tt.a, tt.b)
			// Allow small floating point differences
			if diff := result - tt.expected; diff < -0.01 || diff > 0.01 {
				t.Errorf("SimilarityScore(%q, %q) = %f, want %f", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestCalcSongSimilarity(t *testing.T) {
	tests := []struct {
		name       string
		name1      string
		artist1    string
		name2      string
		artist2    string
		minScore   float64
		maxScore   float64
	}{
		{"identical_song", "hello", "taylor", "hello", "taylor", 0.99, 1.0},
		{"same_name_diff_artist", "hello", "taylor", "hello", "adele", 0.7, 0.8},
		{"diff_name_same_artist", "hello", "taylor", "world", "taylor", 0.3, 0.6},
		{"completely_different", "hello", "taylor", "goodbye", "adele", 0.0, 0.3},
		{"empty_name", "", "taylor", "hello", "taylor", 0.0, 0.0},
		{"empty_artist", "hello", "", "hello", "taylor", 0.99, 1.0},
		{"both_empty", "", "", "", "", 0.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalcSongSimilarity(tt.name1, tt.artist1, tt.name2, tt.artist2)
			if result < tt.minScore || result > tt.maxScore {
				t.Errorf("CalcSongSimilarity(%q, %q, %q, %q) = %f, want [%f, %f]",
					tt.name1, tt.artist1, tt.name2, tt.artist2, result, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestIsDurationClose(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected bool
	}{
		{"identical", 180, 180, true},
		{"close_5s", 180, 185, true},
		{"close_10s", 180, 190, true},
		{"within_15_percent", 100, 114, true},
		{"not_close_big_diff", 180, 220, false},
		{"zero a", 0, 180, true},
		{"zero b", 180, 0, true},
		{"both zero", 0, 0, true},
		{"negative a", -10, 180, true},
		{"negative b", 180, -10, true},
		{"exactly_15_percent", 100, 115, true},
		{"just_over_15_percent", 100, 116, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDurationClose(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("IsDurationClose(%d, %d) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestIntAbs(t *testing.T) {
	tests := []struct {
		name     string
		x        int
		expected int
	}{
		{"positive", 5, 5},
		{"negative", -5, 5},
		{"zero", 0, 0},
		{"large positive", 1000000, 1000000},
		{"large negative", -1000000, 1000000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := intAbs(tt.x)
			if result != tt.expected {
				t.Errorf("intAbs(%d) = %d, want %d", tt.x, result, tt.expected)
			}
		})
	}
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		{"identical", "hello", "hello", 0},
		{"empty a", "", "hello", 5},
		{"empty b", "hello", "", 5},
		{"both empty", "", "", 0},
		{"one_diff", "hello", "hellp", 1},
		{"two_diffs", "hello", "hallo", 1},
		{"completely_different", "abc", "xyz", 3},
		{"chinese identical", "周杰伦", "周杰伦", 0},
		{"chinese_different", "周杰伦", "周润发", 2},
		{"insertion", "abc", "abxc", 1},
		{"deletion", "abc", "ac", 1},
		{"substitution", "abc", "axc", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := levenshteinDistance([]rune(tt.a), []rune(tt.b))
			if result != tt.expected {
				t.Errorf("levenshteinDistance(%q, %q) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestMinInt(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{"a less", 1, 2, 1},
		{"b less", 2, 1, 1},
		{"equal", 1, 1, 1},
		{"negative", -1, 1, -1},
		{"both negative", -2, -1, -2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := minInt(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("minInt(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}