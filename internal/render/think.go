package render

import (
	"regexp"
	"strings"
)

var thinkBlockRe = regexp.MustCompile(`(?s)<think>(.*?)</think>`)

// SplitThink separates think block content from the main response.
// Returns (think, response, found) - if no think block found, think is empty and found is false.
func SplitThink(content string) (think, response string, found bool) {
	matches := thinkBlockRe.FindStringSubmatch(content)
	if len(matches) > 1 {
		think = strings.TrimSpace(matches[1])
		response = strings.TrimSpace(thinkBlockRe.ReplaceAllString(content, ""))
		return think, response, true
	}
	return "", content, false
}
