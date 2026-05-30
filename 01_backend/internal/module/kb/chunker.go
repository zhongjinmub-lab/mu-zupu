package kb

import "strings"

func SplitTextChunks(text string, maxChars, overlapChars int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if maxChars <= 0 {
		maxChars = 1200
	}
	if overlapChars < 0 {
		overlapChars = 0
	}
	if overlapChars >= maxChars {
		overlapChars = maxChars / 10
	}

	runes := []rune(text)
	out := make([]string, 0, len(runes)/maxChars+1)
	start := 0
	for start < len(runes) {
		end := start + maxChars
		if end > len(runes) {
			end = len(runes)
		}
		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			out = append(out, chunk)
		}
		if end == len(runes) {
			break
		}
		start = end - overlapChars
		if start < 0 {
			start = 0
		}
	}
	return out
}
