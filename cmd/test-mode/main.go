package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"universal-checker/internal/checker"
	"universal-checker/pkg/types"
)

func main() {
	fmt.Println("🚀 Universal Checker - TEST MODE")
	fmt.Println("=================================")
	fmt.Printf("⏰ Started at: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("📊 Live monitoring enabled")
	fmt.Println("🔧 Detailed logging enabled")
	fmt.Println()

	// Create test configuration
	config := &types.CheckerConfig{
		MaxWorkers:        5,  // Lower for easier monitoring
		ProxyTimeout:      3000,
		RequestTimeout:    10000,
		RetryCount:        2,
		ProxyRotation:     true,
		AutoScrapeProxies: false, // Use manual proxies for testing
		SaveValidOnly:     false, // Save all results for testing
		OutputFormat:      "txt",
		OutputDirectory:   "test_results",
	}

	// Create checker
	c := checker.NewChecker(config)

	// Load test configurations
	fmt.Println("📁 Loading test configurations...")
	configPaths := []string{
		"test_data/Configs/Streaming/Crunchyroll CYBER v3.svb", 
		"test_data/Configs/VPN/TunnelBear VPN.loli",
		"test_data/Configs/VPN/Strong VPN.loli",
	}

	if err := c.LoadConfigs(configPaths); err != nil {
		log.Fatalf("❌ Failed to load configs: %v", err)
	}

	fmt.Printf("✅ Loaded %d configurations:\n", len(c.Configs))
	for i, cfg := range c.Configs {
		fmt.Printf("   %d. %s (%s) -> %s\n", i+1, cfg.Name, cfg.Type, cfg.URL)
		fmt.Printf("      Headers: %d, Data fields: %d\n", len(cfg.Headers), len(cfg.Data))
		fmt.Printf("      Success conditions: %v\n", cfg.SuccessStrings)
		fmt.Printf("      Failure conditions: %v\n", cfg.FailureStrings)
		fmt.Println()
	}

	// Load test combos
	fmt.Println("📋 Loading test combos...")
	if err := c.LoadCombos("test_data/Combos/40 valids.txt"); err != nil {
		log.Fatalf("❌ Failed to load combos: %v", err)
	}
	fmt.Printf("✅ Loaded %d combos\n", len(c.Combos))
	for i, combo := range c.Combos {
		fmt.Printf("   %d. %s\n", i+1, combo.Line)
		if i >= 9 { // Only show first 10 combos
			fmt.Println("   ... and more")
			break
		}
	}
	fmt.Println()

	// Load test proxies
	fmt.Println("🌐 Loading test proxies...")
	if err := c.LoadProxies("test_data/proxies/SOCKS4_proxies (13).txt"); err != nil {
		fmt.Printf("⚠️  Warning: Failed to load proxies: %v\n", err)
		fmt.Println("📡 Proceeding without proxies for this test...")
	} else {
		fmt.Printf("✅ Loaded %d proxies\n", len(c.Proxies))
		for i, proxy := range c.Proxies {
			fmt.Printf("   %d. %s:%d (%s)\n", i+1, proxy.Host, proxy.Port, proxy.Type)
			if i >= 9 { // Only show first 10 proxies
				fmt.Println("   ... and more")
				break
			}
		}
	}
	fmt.Println()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start checker
	fmt.Println("⚡ Starting checker in test mode...")
	fmt.Printf("👥 Workers: %d\n", config.MaxWorkers)
	fmt.Printf("⏱️  Request timeout: %dms\n", config.RequestTimeout)
	fmt.Printf("🎯 Total tasks: %d (combos) × %d (configs) = %d\n", 
		len(c.Combos), len(c.Configs), len(c.Combos)*len(c.Configs))
	fmt.Println()

	if err := c.Start(); err != nil {
		log.Fatalf("❌ Failed to start checker: %v", err)
	}

	// Live monitoring
	go func() {
		ticker := time.NewTicker(1 * time.Second) // More frequent updates for testing
		defer ticker.Stop()

		lastValid := 0
		lastInvalid := 0
		lastErrors := 0

		for {
			select {
			case <-ticker.C:
				stats := c.GetStats()
				
				// Clear screen and display live stats
				fmt.Print("\033[2J\033[H") // Clear screen
				
				fmt.Println("🔥 UNIVERSAL CHECKER - LIVE TEST MODE")
				fmt.Println("=====================================")
				fmt.Printf("⏰ Current Time: %s\n", time.Now().Format("15:04:05"))
				fmt.Printf("⏱️  Elapsed: %s\n", formatDuration(stats.ElapsedTime))
				fmt.Println()
				
				// Progress
				totalTasks := stats.TotalCombos * len(c.Configs)
				processed := stats.ValidCombos + stats.InvalidCombos + stats.ErrorCombos
				progress := float64(processed) / float64(totalTasks) * 100
				
				fmt.Printf("📈 Progress: %.1f%% (%d/%d tasks)\n", progress, processed, totalTasks)
				fmt.Println(createProgressBar(int(progress), 40))
				fmt.Println()
				
				// Statistics
				fmt.Printf("📊 Results:\n")
				fmt.Printf("   ✅ Valid: %d", stats.ValidCombos)
				if stats.ValidCombos > lastValid {
					fmt.Printf(" (+%d)", stats.ValidCombos-lastValid)
				}
				fmt.Println()
				
				fmt.Printf("   ❌ Invalid: %d", stats.InvalidCombos)
				if stats.InvalidCombos > lastInvalid {
					fmt.Printf(" (+%d)", stats.InvalidCombos-lastInvalid)
				}
				fmt.Println()
				
				fmt.Printf("   ⚠️  Errors: %d", stats.ErrorCombos)
				if stats.ErrorCombos > lastErrors {
					fmt.Printf(" (+%d)", stats.ErrorCombos-lastErrors)
				}
				fmt.Println()
				fmt.Println()
				
				// Performance
				fmt.Printf("🚀 Performance:\n")
				fmt.Printf("   CPM: %.1f\n", stats.CurrentCPM)
				fmt.Printf("   Workers: %d/%d active\n", stats.ActiveWorkers, config.MaxWorkers)
				fmt.Printf("   Proxies: %d/%d working\n", stats.WorkingProxies, stats.TotalProxies)
				fmt.Println()
				
				// Status
				if progress >= 100 {
					fmt.Println("🎉 TEST COMPLETED!")
					fmt.Println("📁 Check 'test_results/' directory for output files")
				} else {
					fmt.Println("🔄 Testing in progress... Press Ctrl+C to stop")
				}
				
				// Update last values
				lastValid = stats.ValidCombos
				lastInvalid = stats.InvalidCombos
				lastErrors = stats.ErrorCombos
			}
		}
	}()

	// Wait for completion or signal
	select {
	case <-sigChan:
		fmt.Println("\n🛑 Received interrupt signal. Stopping checker...")
		c.Stop()
		fmt.Println("✅ Checker stopped gracefully.")
	case <-time.After(2 * time.Minute): // Auto-stop after 2 minutes for safety
		fmt.Println("\n⏰ Test duration limit reached. Stopping checker...")
		c.Stop()
		fmt.Println("✅ Test completed.")
	}

	// Final statistics
	finalStats := c.GetStats()
	fmt.Println("\n📊 FINAL TEST RESULTS")
	fmt.Println("=====================")
	fmt.Printf("⏱️  Total runtime: %s\n", formatDuration(finalStats.ElapsedTime))
	fmt.Printf("📊 Total combos tested: %d\n", finalStats.TotalCombos)
	fmt.Printf("✅ Valid results: %d\n", finalStats.ValidCombos)
	fmt.Printf("❌ Invalid results: %d\n", finalStats.InvalidCombos)
	fmt.Printf("⚠️  Error results: %d\n", finalStats.ErrorCombos)
	fmt.Printf("🚀 Average CPM: %.1f\n", finalStats.CurrentCPM)
	fmt.Printf("📁 Results saved to: test_results/\n")

	fmt.Println("\n🎯 TEST MODE COMPLETE")
}

func formatDuration(seconds int) string {
	duration := time.Duration(seconds) * time.Second
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	secs := int(duration.Seconds()) % 60
	
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, secs)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, secs)
	} else {
		return fmt.Sprintf("%ds", secs)
	}
}

func createProgressBar(progress, width int) string {
	if progress > 100 {
		progress = 100
	}
	
	filled := progress * width / 100
	bar := "["
	
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	
	bar += "]"
	return bar
}
