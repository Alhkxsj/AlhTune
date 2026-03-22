package utils

import (
	"strings"
	"unicode"
)

// NormalizeText removes special characters and converts to lowercase
// Keeps only letters, numbers, and Chinese characters
func NormalizeText(s string) string {
	if s == "" {
		return ""
	}
	s = toLower(s)
	var b stringsBuilder
	b.grow(len(s))
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.In(r, unicode.Han) {
			b.writeRune(r)
		}
	}
	return b.string()
}

// SimilarityScore calculates the similarity between two strings using Levenshtein distance
// Returns a value between 0.0 (no similarity) and 1.0 (identical)
func SimilarityScore(a, b string) float64 {
	if a == b {
		return 1.0
	}
	if a == "" || b == "" {
		return 0.0
	}

	ra := []rune(a)
	rb := []rune(b)
	la := len(ra)
	lb := len(rb)

	maxLen := la
	if lb > maxLen {
		maxLen = lb
	}
	if maxLen == 0 {
		return 0.0
	}

	dist := levenshteinDistance(ra, rb)
	if dist >= maxLen {
		return 0.0
	}

	return 1.0 - float64(dist)/float64(maxLen)
}

// CalcSongSimilarity calculates similarity between two songs
// Weight: 70% for name, 30% for artist
func CalcSongSimilarity(name, artist, candName, candArtist string) float64 {
	nameA := NormalizeText(name)
	nameB := NormalizeText(candName)
	if nameA == "" || nameB == "" {
		return 0.0
	}
	nameSim := SimilarityScore(nameA, nameB)

	artistA := NormalizeText(artist)
	artistB := NormalizeText(candArtist)
	if artistA == "" || artistB == "" {
		return nameSim
	}

	artistSim := SimilarityScore(artistA, artistB)
	return nameSim*0.7 + artistSim*0.3
}

// levenshteinDistance calculates the edit distance between two rune slices
func levenshteinDistance(ra, rb []rune) int {
	la := len(ra)
	lb := len(rb)

	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	cur := make([]int, lb+1)

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		cur[0] = i
		for j := 1; j <= lb; j++ {
			cost := 0
			if ra[i-1] != rb[j-1] {
				cost = 1
			}

			del := prev[j] + 1
			ins := cur[j-1] + 1
			sub := prev[j-1] + cost

			cur[j] = minInt(del, minInt(ins, sub))
		}
		prev, cur = cur, prev
	}

	return prev[lb]
}

// IsDurationClose checks if two durations are similar enough
// Returns true if difference is <= 10 seconds or within 15% of the first duration
func IsDurationClose(a, b int) bool {
	if a <= 0 || b <= 0 {
		return true
	}

	diff := intAbs(a - b)
	if diff <= 10 {
		return true
	}

	maxAllowed := int(float64(a) * 0.15)
	if maxAllowed < 10 {
		maxAllowed = 10
	}

	return diff <= maxAllowed
}

// intAbs returns the absolute value of an integer
func intAbs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// stringsBuilder is a minimal wrapper around strings.Builder for performance
type stringsBuilder struct {
	b strings.Builder
}

func (sb *stringsBuilder) grow(n int) {
	sb.b.Grow(n)
}

func (sb *stringsBuilder) writeRune(r rune) {
	sb.b.WriteRune(r)
}

func (sb *stringsBuilder) string() string {
	return sb.b.String()
}

// toLower converts a string to lowercase (using strings.ToLower)
func toLower(s string) string {
	return strings.ToLower(s)
}
