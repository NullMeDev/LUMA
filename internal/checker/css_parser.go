package checker

import (
	"fmt"
	"strings"
	
	"github.com/PuerkitoBio/goquery"
)

type CSSParser struct{}

func (c *CSSParser) Parse(input string, selector string, attribute string) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(input))
	if err != nil {
		return nil, fmt.Errorf("CSS parse error: %v", err)
	}

	var results []string
	doc.Find(selector).Each(func(index int, item *goquery.Selection) {
		if attr, exists := item.Attr(attribute); exists {
			results = append(results, attr)
		}
	})

	return results, nil
}
