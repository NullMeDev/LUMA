package tests

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"universal-checker/internal/checker"
	"universal-checker/internal/logger"
	"universal-checker/internal/reporting"
	"universal-checker/pkg/types"
)

func TestIntegrationCheckerWorkflow(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "luma_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test configuration
	config := &types.CheckerConfig{
		MaxWorkers:        2,
		ProxyTimeout:      5000,
		RequestTimeout:    10000,
		RetryCount:        1,
		ProxyRotation:     true,
		AutoScrapeProxies: false,
		SaveValidOnly:     false,
		OutputFormat:      "json",
		OutputDirectory:   tempDir,
	}

	// Initialize checker
	checker := checker.NewChecker(config)

	// Create test combo file
	comboFile := filepath.Join(tempDir, "test_combos.txt")
	comboContent := "user1:pass1\nuser2:pass2\ntest@example.com:testpass\n"
	if err := os.WriteFile(comboFile, []byte(comboContent), 0644); err != nil {
		t.Fatalf("Failed to create combo file: %v", err)
	}

	// Create test config file (OpenBullet format)
	configFile := filepath.Join(tempDir, "test_config.opk")
	configContent := `{
		"name": "Test Config",
		"url": "https://httpbin.org/post",
		"method": "POST",
		"headers": {
			"User-Agent": "LUMA-Test"
		},
		"data": {
			"username": "<USER>",
			"password": "<PASS>"
		},
		"conditions": {
			"success": ["json"],
			"failure": ["error", "invalid"]
		}
	}`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Load configurations and combos
	if err := checker.LoadConfigs([]string{configFile}); err != nil {
		t.Fatalf("Failed to load configs: %v", err)
	}

	if err := checker.LoadCombos(comboFile); err != nil {
		t.Fatalf("Failed to load combos: %v", err)
	}

	// Verify loaded data
	stats := checker.GetStats()
	if stats.TotalCombos != 3 {
		t.Errorf("Expected 3 combos, got %d", stats.TotalCombos)
	}

	t.Logf("Integration test completed successfully")
}

func TestLoggerIntegration(t *testing.T) {
	// Create temporary directory for logs
	tempDir, err := os.MkdirTemp("", "luma_log_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create logger configuration
	logFile := filepath.Join(tempDir, "test.log")
	loggerConfig := logger.LoggerConfig{
		Level:      logger.INFO,
		JSONFormat: true,
		OutputFile: logFile,
		BufferSize: 100,
		Component:  "test",
	}

	// Initialize logger
	structuredLogger, err := logger.NewStructuredLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer structuredLogger.Close()

	// Test logging different levels
	structuredLogger.Info("Test info message", map[string]interface{}{
		"test_field": "test_value",
		"number":     42,
	})

	structuredLogger.Warn("Test warning message")

	// Verify log file was created
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Log file was not created")
	}

	// Test log buffer functionality
	recentLogs := structuredLogger.GetRecentLogs(10)
	if len(recentLogs) != 2 {
		t.Errorf("Expected 2 recent logs, got %d", len(recentLogs))
	}

	// Test log export
	exportFile := filepath.Join(tempDir, "exported_logs.json")
	if err := structuredLogger.ExportLogs(exportFile, 10); err != nil {
		t.Errorf("Failed to export logs: %v", err)
	}

	// Verify export file exists
	if _, err := os.Stat(exportFile); os.IsNotExist(err) {
		t.Errorf("Exported log file was not created")
	}

	t.Logf("Logger integration test completed successfully")
}

func TestReportingIntegration(t *testing.T) {
	// Create temporary directory for reports
	tempDir, err := os.MkdirTemp("", "luma_report_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test data
	checkerStats := types.CheckerStats{
		TotalCombos:   10,
		ValidCombos:   3,
		InvalidCombos: 5,
		ErrorCombos:   2,
		CurrentCPM:    150.5,
		AverageCPM:    145.2,
		StartTime:     time.Now().Add(-time.Hour),
		ElapsedTime:   3600,
	}

	testResults := []types.CheckResult{
		{
			Combo: types.Combo{
				Username: "test1",
				Password: "pass1",
			},
			Config:    "test_config",
			Status:    types.BotStatusSuccess,
			Response:  "success response",
			Timestamp: time.Now(),
			Latency:   250,
		},
		{
			Combo: types.Combo{
				Username: "test2",
				Password: "pass2",
			},
			Config:    "test_config",
			Status:    types.BotStatusFail,
			Response:  "failure response",
			Timestamp: time.Now(),
			Latency:   180,
		},
	}

	// Generate report
	reportFile := filepath.Join(tempDir, "test_report.json")
	if err := reporting.GenerateReport(reportFile, "test_session", checkerStats, testResults); err != nil {
		t.Fatalf("Failed to generate report: %v", err)
	}

	// Verify report file exists
	if _, err := os.Stat(reportFile); os.IsNotExist(err) {
		t.Errorf("Report file was not created")
	}

	t.Logf("Reporting integration test completed successfully")
}

func TestProxyManagerIntegration(t *testing.T) {
	// Create advanced proxy manager
	proxyManager := checker.NewAdvancedProxyManager(checker.StrategyBestScore)

	// Add test proxies
	testProxies := []types.Proxy{
		{Host: "127.0.0.1", Port: 8080, Type: types.ProxyTypeHTTP},
		{Host: "127.0.0.1", Port: 8081, Type: types.ProxyTypeHTTPS},
		{Host: "127.0.0.1", Port: 8082, Type: types.ProxyTypeSOCKS5},
	}

	for _, proxy := range testProxies {
		if err := proxyManager.AddProxy(proxy); err != nil {
			t.Errorf("Failed to add proxy: %v", err)
		}
	}

	// Test proxy selection
	selectedProxy, err := proxyManager.GetBestProxy()
	if err != nil {
		t.Errorf("Failed to get best proxy: %v", err)
	}

	if selectedProxy == nil {
		t.Errorf("No proxy was selected")
	}

	// Test proxy statistics
	stats := proxyManager.GetProxyStats()
	if stats["total_proxies"].(int) != 3 {
		t.Errorf("Expected 3 total proxies, got %d", stats["total_proxies"])
	}

	// Test geo-preferred selection
	proxyManager.SetPreferredCountries([]string{"US", "CA"})
	
	geoProxy, err := proxyManager.GetBestProxy()
	if err != nil {
		t.Errorf("Failed to get geo-preferred proxy: %v", err)
	}

	if geoProxy == nil {
		t.Errorf("No geo-preferred proxy was selected")
	}

	t.Logf("Proxy manager integration test completed successfully")
}
