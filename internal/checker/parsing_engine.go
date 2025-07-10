package checker

import (
	"fmt"
)

// ParseType represents the type of parsing to perform
type ParseType string

const (
	ParseTypeLR    ParseType = "LR"
	ParseTypeCSS   ParseType = "CSS"
	ParseTypeJSON  ParseType = "JSON"
	ParseTypeREGEX ParseType = "REGEX"
)

// Parser interface for all parsing strategies
type Parser interface {
	Parse(input string, params ...string) ([]string, error)
}

// ParsingEngine manages all parsing operations
type ParsingEngine struct {
	jsonParser  *JSONParser
	cssParser   *CSSParser
	regexParser *REGEXParser
	lrParser    *LRParser
}

// NewParsingEngine creates a new parsing engine
func NewParsingEngine() *ParsingEngine {
	return &ParsingEngine{
		jsonParser:  &JSONParser{},
		cssParser:   &CSSParser{},
		regexParser: &REGEXParser{},
		lrParser:    &LRParser{},
	}
}

// Parse executes the appropriate parser based on the parse type
func (pe *ParsingEngine) Parse(parseType ParseType, input string, params ...string) ([]string, error) {
	switch parseType {
	case ParseTypeJSON:
		if len(params) < 1 {
			return nil, fmt.Errorf("JSON parsing requires field parameter")
		}
		return pe.jsonParser.Parse(input, params[0])
	
	case ParseTypeCSS:
		if len(params) < 2 {
			return nil, fmt.Errorf("CSS parsing requires selector and attribute parameters")
		}
		return pe.cssParser.Parse(input, params[0], params[1])
	
	case ParseTypeREGEX:
		if len(params) < 1 {
			return nil, fmt.Errorf("REGEX parsing requires pattern parameter")
		}
		return pe.regexParser.Parse(input, params[0])
	
	case ParseTypeLR:
		if len(params) < 2 {
			return nil, fmt.Errorf("LR parsing requires left and right delimiter parameters")
		}
		result, err := pe.lrParser.Parse(input, params[0], params[1])
		if err != nil {
			return nil, err
		}
		return []string{result}, nil
	
	default:
		return nil, fmt.Errorf("unsupported parse type: %s", parseType)
	}
}
