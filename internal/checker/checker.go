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
	"universal-checker/internal/proxy"
	"universal-checker/pkg/types"
)

// Checker represents the main checker engine
type Checker struct {
	Config      *types.CheckerConfig
	Stats       *types.CheckerStats
	Proxies     []types.Proxy
	Configs     []types.Config
	Combos      []types.Combo
	
	// Channels for communication
	taskChan   chan types.WorkerTask
	resultChan chan types.WorkerResult
	
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
	
	// Enhanced parsing and variable systems
	workflowEngine *WorkflowEngine
	varManipulator *VariableManipulator
}

// NewChecker creates a new checker instance
func NewChecker(config *types.CheckerConfig) *Checker {
	ctx, cancel := context.WithCancel(context.Background())
	
	workflowEngine := NewWorkflowEngine()
	varManipulator := NewVariableManipulator(workflowEngine.variables)
	
	return &Checker{
		Config:         config,
		Stats:          &types.CheckerStats{},
		Proxies:        make([]types.Proxy, 0),
		Configs:        make([]types.Config, 0),
		Combos:         make([]types.Combo, 0),
		taskChan:       make(chan types.WorkerTask, config.MaxWorkers*2),
		resultChan:     make(chan types.WorkerResult, config.MaxWorkers*2),
		ctx:            ctx,
		cancel:         cancel,
		exporter:       NewResultExporter(config.OutputDirectory, config.OutputFormat),
		workflowEngine: workflowEngine,
		varManipulator: varManipulator,
	}
}

// LoadConfigs loads configuration files
func (c *Checker) LoadConfigs(configPaths []string) error {
	parser := config.NewParser()
	
	for _, configPath := range configPaths {
		cfg, err := parser.ParseConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to parse config %s: %v", configPath, err)
		}
		c.Configs = append(c.Configs, *cfg)
	}
	
	return nil
}

// LoadCombos loads combos from a file
func (c *Checker) LoadCombos(comboPath string) error {
	file, err := os.Open(comboPath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		combo := c.parseCombo(line)
		if combo != nil {
			c.Combos = append(c.Combos, *combo)
		}
	}

	c.Stats.TotalCombos = len(c.Combos)
	return scanner.Err()
}

// LoadProxies loads proxies from file or auto-scrapes them
func (c *Checker) LoadProxies(proxyPath string) error {
	if c.Config.AutoScrapeProxies {
		scraper := proxy.NewScraper(c.Config)
		proxies, err := scraper.ScrapeAndValidate()
		if err != nil {
			return err
		}
		c.Proxies = proxies
	} else if proxyPath != "" {
		file, err := os.Open(proxyPath)
		if err != nil {
			return err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			proxy := c.parseProxy(line)
			if proxy != nil {
				c.Proxies = append(c.Proxies, *proxy)
			}
		}
	}

	c.Stats.TotalProxies = len(c.Proxies)
	return nil
}

// Start starts the checking process
func (c *Checker) Start() error {
	c.Stats.StartTime = time.Now()
	
	// Start workers
	for i := 0; i < c.Config.MaxWorkers; i++ {
		c.wg.Add(1)
		go c.worker()
	}

	// Start result processor
	go c.processResults()

	// Generate tasks
	go c.generateTasks()

	return nil
}

// Stop stops the checking process
func (c *Checker) Stop() {
	c.cancel()
	close(c.taskChan)
	c.wg.Wait()
	close(c.resultChan)
}

// worker is the main worker function that processes tasks
func (c *Checker) worker() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		case task, ok := <-c.taskChan:
			if !ok {
				return
			}

			result := c.checkCombo(task)
			c.resultChan <- result
		}
	}
}

// generateTasks generates tasks for all combo/config combinations
func (c *Checker) generateTasks() {
	for _, combo := range c.Combos {
		for _, config := range c.Configs {
			var proxy *types.Proxy
			if config.UseProxy && len(c.Proxies) > 0 {
				proxy = c.getNextProxy()
			}

			task := types.WorkerTask{
				Combo:  combo,
				Config: config,
				Proxy:  proxy,
			}

			select {
			case <-c.ctx.Done():
				return
			case c.taskChan <- task:
			}
		}
	}
}

// checkCombo checks a single combo against a config
func (c *Checker) checkCombo(task types.WorkerTask) types.WorkerResult {
	start := time.Now()
	
	// Create HTTP client
	client := c.createHTTPClient(task.Proxy)
	
	// Build request
	req, err := c.buildRequest(task.Combo, task.Config)
	if err != nil {
		return types.WorkerResult{
			Result: types.CheckResult{
				Combo:     task.Combo,
				Config:    task.Config.Name,
				Status:    "error",
				Error:     err.Error(),
				Timestamp: time.Now(),
			},
			Error: err,
		}
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return types.WorkerResult{
			Result: types.CheckResult{
				Combo:     task.Combo,
				Config:    task.Config.Name,
				Status:    "error",
				Error:     err.Error(),
				Timestamp: time.Now(),
				Latency:   int(time.Since(start).Milliseconds()),
			},
			Error: err,
		}
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.WorkerResult{
			Result: types.CheckResult{
				Combo:     task.Combo,
				Config:    task.Config.Name,
				Status:    "error",
				Error:     err.Error(),
				Timestamp: time.Now(),
				Latency:   int(time.Since(start).Milliseconds()),
			},
			Error: err,
		}
	}

	// Analyze response
	status := c.analyzeResponse(string(body), resp.StatusCode, task.Config)
	
	return types.WorkerResult{
		Result: types.CheckResult{
			Combo:     task.Combo,
			Config:    task.Config.Name,
			Status:    status,
			Response:  string(body),
			Proxy:     task.Proxy,
			Timestamp: time.Now(),
			Latency:   int(time.Since(start).Milliseconds()),
		},
		Error: nil,
	}
}

// createHTTPClient creates an HTTP client with optional proxy
func (c *Checker) createHTTPClient(proxy *types.Proxy) *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	if proxy != nil {
		proxyURL, err := url.Parse(fmt.Sprintf("%s://%s:%d", string(proxy.Type), proxy.Host, proxy.Port))
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   time.Duration(c.Config.RequestTimeout) * time.Millisecond,
	}
}

// buildRequest builds an HTTP request from combo and config
func (c *Checker) buildRequest(combo types.Combo, config types.Config) (*http.Request, error) {
	// Replace variables in URL
	url := c.replaceVariables(config.URL, combo)
	
	// Create request
	var req *http.Request
	var err error

	if config.Method == "GET" {
		req, err = http.NewRequest("GET", url, nil)
	} else {
		// Build form data
		formData := c.buildFormData(config.Data, combo)
		req, err = http.NewRequest(config.Method, url, strings.NewReader(formData))
	}

	if err != nil {
		return nil, err
	}

	// Set headers
	for key, value := range config.Headers {
		req.Header.Set(key, c.replaceVariables(value, combo))
	}

	// Set content type for POST requests
	if config.Method == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	return req, nil
}

// buildFormData builds form data from config data and combo
func (c *Checker) buildFormData(data map[string]interface{}, combo types.Combo) string {
	var formData []string
	
	for key, value := range data {
		valueStr := fmt.Sprintf("%v", value)
		valueStr = c.replaceVariables(valueStr, combo)
		formData = append(formData, fmt.Sprintf("%s=%s", key, url.QueryEscape(valueStr)))
	}
	
	return strings.Join(formData, "&")
}

// replaceVariables replaces variables in strings with combo values and dynamic variables
func (c *Checker) replaceVariables(text string, combo types.Combo) string {
	// Set combo variables in the variable manipulator
	c.varManipulator.SetVariable("USER", combo.Username, false)
	c.varManipulator.SetVariable("PASS", combo.Password, false)
	c.varManipulator.SetVariable("EMAIL", combo.Email, false)
	c.varManipulator.SetVariable("username", combo.Username, false)
	c.varManipulator.SetVariable("password", combo.Password, false)
	c.varManipulator.SetVariable("email", combo.Email, false)
	
	// Use the variable manipulator for enhanced variable replacement
	return c.varManipulator.ReplaceVariables(text)
}

// analyzeResponse analyzes the response to determine success/failure
func (c *Checker) analyzeResponse(body string, statusCode int, config types.Config) types.BotStatus {
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

	// Check success strings
	for _, successStr := range config.SuccessStrings {
		if strings.Contains(body, successStr) {
			return types.BotStatusSuccess
		}
	}

	// Check failure strings
	for _, failureStr := range config.FailureStrings {
		if strings.Contains(body, failureStr) {
			return types.BotStatusFail
		}
	}

	// Default to invalid if no specific conditions match
	return types.BotStatusFail
}

// processResults processes results from workers
func (c *Checker) processResults() {
	for result := range c.resultChan {
		c.updateStats(result.Result)
		
		// Save result if needed
		if !c.Config.SaveValidOnly || result.Result.Status == types.BotStatusSuccess {
			c.saveResult(result.Result)
		}
	}
}

// updateStats updates checker statistics
func (c *Checker) updateStats(result types.CheckResult) {
	c.statsMutex.Lock()
	defer c.statsMutex.Unlock()

	switch result.Status {
	case types.BotStatusSuccess:
		c.Stats.ValidCombos++
	case types.BotStatusFail:
		c.Stats.InvalidCombos++
	case types.BotStatusError:
		c.Stats.ErrorCombos++
	}

	// Update CPM
	elapsed := time.Since(c.Stats.StartTime).Minutes()
	if elapsed > 0 {
		totalChecks := c.Stats.ValidCombos + c.Stats.InvalidCombos + c.Stats.ErrorCombos
		c.Stats.CurrentCPM = float64(totalChecks) / elapsed
	}
}

// saveResult saves a result to file
func (c *Checker) saveResult(result types.CheckResult) {
	if err := c.exporter.ExportResult(result); err != nil {
	log.Printf("[ERROR] Failed to export result: %v", err)
	}
}

// getNextProxy returns the next proxy in rotation
func (c *Checker) getNextProxy() *types.Proxy {
	c.proxyMutex.Lock()
	defer c.proxyMutex.Unlock()

	if len(c.Proxies) == 0 {
		return nil
	}

	if c.Config.ProxyRotation {
		proxy := &c.Proxies[c.proxyIndex]
		c.proxyIndex = (c.proxyIndex + 1) % len(c.Proxies)
		return proxy
	}

	// Random proxy selection
	return &c.Proxies[rand.Intn(len(c.Proxies))]
}

// parseCombo parses a combo line into a Combo struct
func (c *Checker) parseCombo(line string) *types.Combo {
	// Support different formats: username:password, email:password
	parts := strings.Split(line, ":")
	if len(parts) < 2 {
		return nil
	}

	combo := &types.Combo{
		Line:     line,
		Username: parts[0],
		Password: parts[1],
	}

	// Check if username looks like an email
	if strings.Contains(combo.Username, "@") {
		combo.Email = combo.Username
	}

	return combo
}

// parseProxy parses a proxy line into a Proxy struct
func (c *Checker) parseProxy(line string) *types.Proxy {
	parts := strings.Split(line, ":")
	if len(parts) < 2 {
		return nil
	}

	proxy := &types.Proxy{
		Host: parts[0],
		Port: c.parseInt(parts[1]),
		Type: types.ProxyTypeHTTP, // Default to HTTP
	}

	// Try to detect proxy type from line
	if len(parts) > 2 {
		switch strings.ToLower(parts[2]) {
		case "socks4":
			proxy.Type = types.ProxyTypeSOCKS4
		case "socks5":
			proxy.Type = types.ProxyTypeSOCKS5
		case "https":
			proxy.Type = types.ProxyTypeHTTPS
		}
	}

	return proxy
}

// parseInt parses a string to integer
func (c *Checker) parseInt(s string) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return 0
}

// GetStats returns current statistics
func (c *Checker) GetStats() types.CheckerStats {
	c.statsMutex.RLock()
	defer c.statsMutex.RUnlock()
	
	stats := *c.Stats
	stats.ElapsedTime = int(time.Since(c.Stats.StartTime).Seconds())
	stats.ActiveWorkers = c.Config.MaxWorkers
	stats.WorkingProxies = len(c.getWorkingProxies())
	
	return stats
}

// getWorkingProxies returns only working proxies
func (c *Checker) getWorkingProxies() []types.Proxy {
	var working []types.Proxy
	for _, proxy := range c.Proxies {
		if proxy.Working {
			working = append(working, proxy)
		}
	}
	return working
}
