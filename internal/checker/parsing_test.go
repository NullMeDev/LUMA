package checker

import (
	"testing"
)

func TestJSONParser(t *testing.T) {
	parser := &JSONParser{}
	
	input := `{"status": "success", "user": "john_doe"}`
	
	result, err := parser.Parse(input, "status")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(result) != 1 || result[0] != "success" {
		t.Fatalf("Expected [success], got %v", result)
	}
}

func TestREGEXParser(t *testing.T) {
	parser := &REGEXParser{}
	
	input := "Hello, my name is John and I am 25 years old"
	pattern := `name is (\w+) and I am (\d+)`
	
	result, err := parser.Parse(input, pattern)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(result) != 2 || result[0] != "John" || result[1] != "25" {
		t.Fatalf("Expected [John, 25], got %v", result)
	}
}

func TestLRParser(t *testing.T) {
	parser := &LRParser{}
	
	input := "Hello [world] how are you?"
	
	result, err := parser.Parse(input, "[", "]")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if result != "world" {
		t.Fatalf("Expected 'world', got '%s'", result)
	}
}

func TestParsingEngine(t *testing.T) {
	engine := NewParsingEngine()
	
	// Test JSON parsing
	jsonInput := `{"message": "Hello World"}`
	result, err := engine.Parse(ParseTypeJSON, jsonInput, "message")
	if err != nil {
		t.Fatalf("JSON parsing failed: %v", err)
	}
	if len(result) != 1 || result[0] != "Hello World" {
		t.Fatalf("Expected [Hello World], got %v", result)
	}
	
	// Test LR parsing
	lrInput := "Start <content> End"
	result, err = engine.Parse(ParseTypeLR, lrInput, "<", ">")
	if err != nil {
		t.Fatalf("LR parsing failed: %v", err)
	}
	if len(result) != 1 || result[0] != "content" {
		t.Fatalf("Expected [content], got %v", result)
	}
}
