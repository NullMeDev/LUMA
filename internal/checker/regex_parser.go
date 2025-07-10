package checker

import (
	"regexp"
	"fmt"
)

type REGEXParser struct{}

func (r *REGEXParser) Parse(input string, pattern string) ([]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("REGEX compile error: %v", err)
	}

	matches := re.FindStringSubmatch(input)
	if matches == nil {
		return nil, fmt.Errorf("no matches found")
	}

	return matches[1:], nil // Skip full match at index 0
}
