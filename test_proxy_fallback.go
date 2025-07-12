package main

import (
	"fmt"
	"log"
	"time"

	"universal-checker/internal/checker"
	"universal-checker/pkg/types"
)

// Test function to demonstrate the fallback and no-proxy logic
func main() {
	// Create test configuration
	config := &types.CheckerConfig{
		MaxWorkers:     2,
		RequestTimeout: 5000, // 5 seconds
		RetryCount:     3,
		ProxyRotation:  true,
		SaveValidOnly:  false,
		OutputFormat:   "txt",
		OutputDirectory: "output",
	}

	// Create checker instance
	c := checker.NewChecker(config)

	// Create test configs - one requires proxy, one doesn't, one uses proxy optionally
	testConfigs := []types.Config{
		{
			Name:          "no-proxy-config",
			Type:          types.ConfigTypeOPK,
			URL:           "https://httpbin.org/get",
			Method:        "GET",
			Headers:       map[string]string{"User-Agent": "test-agent"},
			UseProxy:      false,
			RequiresProxy: false,
		},
		{
			Name:          "optional-proxy-config", 
			Type:          types.ConfigTypeOPK,
			URL:           "https://httpbin.org/get",
			Method:        "GET", 
			Headers:       map[string]string{"User-Agent": "test-agent"},
			UseProxy:      true,
			RequiresProxy: false,
		},
		{
			Name:          "required-proxy-config",
			Type:          types.ConfigTypeOPK,
			URL:           "https://httpbin.org/get",
			Method:        "GET",
			Headers:       map[string]string{"User-Agent": "test-agent"},
			UseProxy:      true,
			RequiresProxy: true,
		},
	}

	// Load test configs into checker
	c.Configs = testConfigs

	// Create test combo
	testCombos := []types.Combo{
		{
			Username: "testuser",
			Password: "testpass",
			Line:     "testuser:testpass",
		},
	}
	c.Combos = testCombos

	// Test scenarios

	fmt.Println("=== Testing Scenario 1: No proxies available ===")
	c.Proxies = []types.Proxy{} // No proxies

	// Test each config type
	for _, config := range testConfigs {
		shouldSkip := c.ShouldSkipTaskDueToProxy(config)
		fmt.Printf("Config %s (RequiresProxy: %v, UseProxy: %v) - Should skip: %v\n", 
			config.Name, config.RequiresProxy, config.UseProxy, shouldSkip)
	}

	fmt.Println("\n=== Testing Scenario 2: Dead proxies available ===")
	// Add some dead proxies
	c.Proxies = []types.Proxy{
		{
			Host:    "127.0.0.1",
			Port:    8080,
			Type:    types.ProxyTypeHTTP,
			Working: false,
		},
		{
			Host:    "127.0.0.1", 
			Port:    8081,
			Type:    types.ProxyTypeHTTP,
			Working: false,
		},
	}

	for _, config := range testConfigs {
		shouldSkip := c.ShouldSkipTaskDueToProxy(config)
		fmt.Printf("Config %s (RequiresProxy: %v, UseProxy: %v) - Should skip: %v\n",
			config.Name, config.RequiresProxy, config.UseProxy, shouldSkip)
	}

	fmt.Println("\n=== Testing Scenario 3: Working proxies available ===")
	// Add some working proxies  
	c.Proxies = []types.Proxy{
		{
			Host:    "127.0.0.1",
			Port:    8080,
			Type:    types.ProxyTypeHTTP,
			Working: true,
		},
		{
			Host:    "127.0.0.1",
			Port:    8081, 
			Type:    types.ProxyTypeHTTP,
			Working: true,
		},
	}

	for _, config := range testConfigs {
		shouldSkip := c.ShouldSkipTaskDueToProxy(config)
		fmt.Printf("Config %s (RequiresProxy: %v, UseProxy: %v) - Should skip: %v\n",
			config.Name, config.RequiresProxy, config.UseProxy, shouldSkip)
	}

	fmt.Println("\n=== Testing proxy selection logic ===")
	for i := 0; i < 5; i++ {
		proxy := c.GetNextProxy()
		healthyProxy := c.GetNextHealthyProxy()
		
		var proxyInfo, healthyInfo string
		if proxy != nil {
			proxyInfo = fmt.Sprintf("%s:%d (Working: %v)", proxy.Host, proxy.Port, proxy.Working)
		} else {
			proxyInfo = "nil"
		}
		
		if healthyProxy != nil {
			healthyInfo = fmt.Sprintf("%s:%d (Working: %v)", healthyProxy.Host, healthyProxy.Port, healthyProxy.Working)
		} else {
			healthyInfo = "nil"
		}
		
		fmt.Printf("Iteration %d - Next proxy: %s, Next healthy proxy: %s\n", i+1, proxyInfo, healthyInfo)
	}

	fmt.Println("\n=== Testing task generation ===")
	// Simulate task generation for a small subset
	go func() {
		time.Sleep(2 * time.Second)
		c.Stop()
	}()

	err := c.Start()
	if err != nil {
		log.Fatalf("Failed to start checker: %v", err)
	}

	fmt.Println("Test completed successfully!")
}
