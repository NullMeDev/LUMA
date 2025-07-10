package test

import (
	"testing"

	"universal-checker/internal/config"
	"universal-checker/pkg/types"
)

func TestAllConfigFormats(t *testing.T) {
	parser := config.NewParser()
	
	// Test OpenBullet config
	t.Run("OpenBullet", func(t *testing.T) {
		cfg, err := parser.ParseConfig("../configs/test_openbullet.opk")
		if err != nil {
			t.Fatalf("Failed to parse OpenBullet config: %v", err)
		}
		
		if cfg.Type != types.ConfigTypeOPK {
			t.Errorf("Expected type OPK, got %s", cfg.Type)
		}
		
		if cfg.Name != "OpenBullet Login Test" {
			t.Errorf("Expected name 'OpenBullet Login Test', got '%s'", cfg.Name)
		}
		
		if cfg.URL != "https://httpbin.org/post" {
			t.Errorf("Expected URL 'https://httpbin.org/post', got '%s'", cfg.URL)
		}
		
		if cfg.Method != "POST" {
			t.Errorf("Expected method POST, got %s", cfg.Method)
		}
		
		if len(cfg.SuccessStrings) == 0 {
			t.Error("Expected success strings, got none")
		}
		
		if len(cfg.FailureStrings) == 0 {
			t.Error("Expected failure strings, got none")
		}
		
		t.Logf("OpenBullet config parsed successfully: %+v", cfg)
	})
	
	// Test SilverBullet config
	t.Run("SilverBullet", func(t *testing.T) {
		cfg, err := parser.ParseConfig("../configs/test_silverbullet.svb")
		if err != nil {
			t.Fatalf("Failed to parse SilverBullet config: %v", err)
		}
		
		if cfg.Type != types.ConfigTypeSVB {
			t.Errorf("Expected type SVB, got %s", cfg.Type)
		}
		
		if cfg.Name != "SilverBullet Login Test" {
			t.Errorf("Expected name 'SilverBullet Login Test', got '%s'", cfg.Name)
		}
		
		if cfg.URL != "https://httpbin.org/post" {
			t.Errorf("Expected URL 'https://httpbin.org/post', got '%s'", cfg.URL)
		}
		
		if cfg.Method != "POST" {
			t.Errorf("Expected method POST, got %s", cfg.Method)
		}
		
		if len(cfg.SuccessStrings) != 3 {
			t.Errorf("Expected 3 success strings, got %d", len(cfg.SuccessStrings))
		}
		
		if len(cfg.FailureStrings) != 3 {
			t.Errorf("Expected 3 failure strings, got %d", len(cfg.FailureStrings))
		}
		
		t.Logf("SilverBullet config parsed successfully: %+v", cfg)
	})
	
	// Test Loli config
	t.Run("Loli", func(t *testing.T) {
		cfg, err := parser.ParseConfig("../configs/test_loli.loli")
		if err != nil {
			t.Fatalf("Failed to parse Loli config: %v", err)
		}
		
		if cfg.Type != types.ConfigTypeLoli {
			t.Errorf("Expected type Loli, got %s", cfg.Type)
		}
		
		if cfg.URL != "https://httpbin.org/post" {
			t.Errorf("Expected URL 'https://httpbin.org/post', got '%s'", cfg.URL)
		}
		
		if cfg.CPM != 300 {
			t.Errorf("Expected CPM 300, got %d", cfg.CPM)
		}
		
		if len(cfg.Headers) == 0 {
			t.Error("Expected headers, got none")
		}
		
		if len(cfg.Data) == 0 {
			t.Error("Expected data, got none")
		}
		
		t.Logf("Loli config parsed successfully: %+v", cfg)
	})
}

func TestConfigDetection(t *testing.T) {
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
		result := config.DetectConfigType(test.filename)
		if result != test.expected {
			t.Errorf("For file %s, expected %s, got %s", test.filename, test.expected, result)
		}
	}
}

func TestVariableReplacement(t *testing.T) {
	// This test would be part of the checker package
	// Testing variable replacement like <USER>, <PASS>, etc.
	// Will be implemented when running the full checker
}
