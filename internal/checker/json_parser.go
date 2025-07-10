package checker

import (
	"encoding/json"
	"fmt"
)

type JSONParser struct{}

func (j *JSONParser) Parse(input string, field string) ([]string, error) {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(input), &result); err != nil {
		return nil, fmt.Errorf("JSON parse error: %v", err)
	}
	
	value, ok := result[field]
	if !ok {
		return nil, fmt.Errorf("field %s not found in JSON", field)
	}
	
	return []string{fmt.Sprintf("%v", value)}, nil
}
