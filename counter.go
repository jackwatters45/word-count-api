// wordcounter/counter.go
package wordcounter

import (
	"regexp"
	"sort"
	"strings"
)

type WordFrequency struct {
	Word      string `json:"word"`
	Frequency int    `json:"frequency"`
}

var wordRegex = regexp.MustCompile(`\b[\p{L}]+\b`)

// ProcessText analyzes text and returns word frequencies sorted by count
func ProcessText(text string) []WordFrequency {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Extract words using regex
	words := wordRegex.FindAllString(text, -1)

	// Count frequencies
	frequencies := make(map[string]int)
	for _, word := range words {
		frequencies[word]++
	}

	// Convert to slice and sort
	var result []WordFrequency
	for word, freq := range frequencies {
		result = append(result, WordFrequency{
			Word:      word,
			Frequency: freq,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Frequency > result[j].Frequency
	})

	return result
}