package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"universal-checker/internal/config"
	"universal-checker/pkg/types"
)

func main() {
	fmt.Println("Testing Intelligent Config Proxy Requirement Detection")
	fmt.Println("=======================================================")

	parser := config.NewParser()

	// Test configs directory
	configDir := "/home/null/Desktop/checkertools/Configs"

	// Test VPN configs
	vpnConfigs := []string{
		filepath.Join(configDir, "VPN/NordVPN.opk"),
		filepath.Join(configDir, "VPN/TorGuard [VPN].opk"),
		filepath.Join(configDir, "VPN/Tabnine [@TheStarkArmyX].svb"),
	}

	// Test streaming configs (likely require proxies for geo-locking)
	streamingConfigs := []string{
		filepath.Join(configDir, "Streaming/OnlyFans.opk"),
		filepath.Join(configDir, "Streaming/Crunchyroll [Android].opk"),
		filepath.Join(configDir, "Streaming/WWE Network.opk"),
	}

	fmt.Println("\nTesting VPN configs (should require proxies):")
	for _, configPath := range vpnConfigs {
		testConfig(parser, configPath)
	}

	fmt.Println("\nTesting Streaming configs (may require proxies based on content):")
	for _, configPath := range streamingConfigs {
		testConfig(parser, configPath)
	}

	// Test with artificial examples
	fmt.Println("\nTesting artificial examples:")
	testArtificialConfig(parser)
}

func testConfig(parser *config.Parser, configPath string) {
	config, err := parser.ParseConfig(configPath)
	if err != nil {
		fmt.Printf("  ❌ Error parsing %s: %v\n", filepath.Base(configPath), err)
		return
	}

	proxyStatus := "No"
	if config.RequiresProxy {
		proxyStatus = "Yes"
	}

	fmt.Printf("  📁 %s: RequiresProxy=%s\n", filepath.Base(configPath), proxyStatus)
	
	// Show detection reasons
	if config.RequiresProxy {
		showDetectionReasons(configPath, config)
	}
}

func showDetectionReasons(filePath string, config *types.Config) {
	reasons := []string{}

	// Check filename
	if containsIgnoreCase(filePath, "vpn", "proxy") {
		reasons = append(reasons, "filename contains VPN/proxy keywords")
	}

	// Check streaming services
	streamingServices := []string{"streaming", "netflix", "hulu", "disney", "onlyfans"}
	for _, service := range streamingServices {
		if containsIgnoreCase(filePath, service) {
			reasons = append(reasons, fmt.Sprintf("filename contains geo-locked service: %s", service))
			break
		}
	}

	// Check failure strings
	banIndicators := []string{"ban", "forbidden", "403", "captcha", "region", "geo"}
	for _, indicator := range banIndicators {
		for _, failStr := range config.FailureStrings {
			if containsIgnoreCase(failStr, indicator) {
				reasons = append(reasons, fmt.Sprintf("failure string contains '%s'", indicator))
				break
			}
		}
	}

	// Check SVB config flags
	if needsProxies, ok := config.RawConfig["NeedsProxies"].(bool); ok && needsProxies {
		reasons = append(reasons, "explicit NeedsProxies flag in SVB config")
	}

	if len(reasons) > 0 {
		fmt.Printf("    🔍 Detection reasons: %v\n", reasons)
	}
}

func testArtificialConfig(parser *config.Parser) {
	// Create artificial configs to test detection logic
	testCases := []struct {
		name        string
		path        string
		failStrings []string
		rawConfig   map[string]interface{}
		expected    bool
	}{
		{
			name:     "VPN in path",
			path:     "/configs/vpn/test.opk",
			expected: true,
		},
		{
			name:        "Ban in failure",
			path:        "/configs/test.opk",
			failStrings: []string{"You have been banned"},
			expected:    true,
		},
		{
			name:        "403 Forbidden",
			path:        "/configs/test.opk",
			failStrings: []string{"403 Forbidden"},
			expected:    true,
		},
		{
			name:      "SVB NeedsProxies",
			path:      "/configs/test.svb",
			rawConfig: map[string]interface{}{"NeedsProxies": true},
			expected:  true,
		},
		{
			name:     "Regular config",
			path:     "/configs/regular.opk",
			expected: false,
		},
	}

	for _, tc := range testCases {
		config := &types.Config{
			Name:           tc.name,
			FailureStrings: tc.failStrings,
			RawConfig:      tc.rawConfig,
		}
		if config.RawConfig == nil {
			config.RawConfig = make(map[string]interface{})
		}

		result := parser.DetermineProxyRequirement(tc.path, config)
		status := "✅"
		if result != tc.expected {
			status = "❌"
		}

		fmt.Printf("  %s %s: Expected=%v, Got=%v\n", status, tc.name, tc.expected, result)
	}
}

func containsIgnoreCase(str string, substrings ...string) bool {
	for _, substr := range substrings {
		if strings.Contains(strings.ToLower(str), strings.ToLower(substr)) {
			return true
		}
	}
	return false
}
