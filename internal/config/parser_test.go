package config

import (
	"os"
	"path/filepath"
	"testing"

	"universal-checker/pkg/types"
)

func TestParseOPK(t *testing.T) {
	// Create temporary OPK file
	tmpDir := t.TempDir()
	opkFile := filepath.Join(tmpDir, "test.opk")
	
	opkContent := `{
		"name": "Test Config",
		"url": "https://example.com/login",
		"method": "POST",
		"headers": {
			"User-Agent": "Test Agent"
		},
		"data": {
			"username": "<USER>",
			"password": "<PASS>"
		},
		"conditions": {
			"success": ["welcome"],
			"failure": ["error"]
		},
		"timeout": 30,
		"cpm": 300
	}`
	
	err := os.WriteFile(opkFile, []byte(opkContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	parser := NewParser()
	config, err := parser.ParseConfig(opkFile)
	if err != nil {
		t.Fatalf("Failed to parse OPK config: %v", err)
	}
	
	// Validate parsed config
	if config.Name != "Test Config" {
		t.Errorf("Expected name 'Test Config', got '%s'", config.Name)
	}
	
	if config.Type != types.ConfigTypeOPK {
		t.Errorf("Expected type OPK, got %s", config.Type)
	}
	
	if config.URL != "https://example.com/login" {
		t.Errorf("Expected URL 'https://example.com/login', got '%s'", config.URL)
	}
	
	if config.Method != "POST" {
		t.Errorf("Expected method POST, got %s", config.Method)
	}
	
	if config.CPM != 300 {
		t.Errorf("Expected CPM 300, got %d", config.CPM)
	}
	
	if len(config.SuccessStrings) != 1 || config.SuccessStrings[0] != "welcome" {
		t.Errorf("Expected success string 'welcome', got %v", config.SuccessStrings)
	}
	
	if len(config.FailureStrings) != 1 || config.FailureStrings[0] != "error" {
		t.Errorf("Expected failure string 'error', got %v", config.FailureStrings)
	}
}

func TestDetectConfigType(t *testing.T) {
	tests := []struct {
		filename string
		expected types.ConfigType
	}{
		{"test.opk", types.ConfigTypeOPK},
		{"config.svb", types.ConfigTypeSVB},
		{"checker.loli", types.ConfigTypeLoli},
		{"unknown.txt", ""},
	}
	
	for _, test := range tests {
		result := DetectConfigType(test.filename)
		if result != test.expected {
			t.Errorf("For file %s, expected %s, got %s", test.filename, test.expected, result)
		}
	}
}
