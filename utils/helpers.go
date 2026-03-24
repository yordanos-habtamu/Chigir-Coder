package utils

import (
	"regexp"
	"strings"
)

// ExtractKeywords pulls simple lowercase tokens from user input.
// Used for tagging records and matching learning patterns.
func ExtractKeywords(input string) []string {
	// lowercase + strip punctuation
	clean := strings.ToLower(input)
	re := regexp.MustCompile(`[^a-z0-9\s]`)
	clean = re.ReplaceAllString(clean, "")

	words := strings.Fields(clean)
	// deduplicate and filter short/stop words
	seen := make(map[string]bool)
	stop := map[string]bool{
		"a": true, "an": true, "the": true, "is": true, "it": true,
		"to": true, "for": true, "and": true, "or": true, "in": true,
		"on": true, "of": true, "with": true, "that": true, "this": true,
		"i": true, "me": true, "my": true, "we": true, "you": true,
		"do": true, "be": true, "am": true, "are": true, "was": true,
		"can": true, "will": true, "should": true, "would": true,
		"have": true, "has": true, "had": true, "not": true,
	}

	var keywords []string
	for _, w := range words {
		if len(w) < 2 || stop[w] || seen[w] {
			continue
		}
		seen[w] = true
		keywords = append(keywords, w)
	}
	return keywords
}

// Truncate shortens a string to maxLen characters, appending "…" if truncated.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}

// ContainsAny checks if the text contains any of the given substrings.
func ContainsAny(text string, subs []string) bool {
	lower := strings.ToLower(text)
	for _, sub := range subs {
		if strings.Contains(lower, strings.ToLower(sub)) {
			return true
		}
	}
	return false
}
