package checker

import (
	"fmt"
	"strings"
)

type LRParser struct{}

// Parse extracts content between left and right delimiters.
func (l *LRParser) Parse(input string, leftDelim string, rightDelim string) (string, error) {
	start := strings.Index(input, leftDelim)
	if start == -1 {
		return "", fmt.Errorf("left delimiter not found")
	}

	start += len(leftDelim)
	end := strings.Index(input[start:], rightDelim)
	if end == -1 {
		return "", fmt.Errorf("right delimiter not found")
	}

	return input[start : start+end], nil
}
