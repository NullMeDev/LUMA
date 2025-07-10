package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"universal-checker/internal/checker"
	"universal-checker/pkg/types"
	"universal-checker/pkg/utils"

	"github.com/spf13/cobra"
)

var (
	// Command line flags for global checker
	globalConfigPaths    []string
	globalComboPath      string
	globalProxyPath      string
	globalOutputDir      string
	globalMaxWorkers     int
	globalAutoScrape     bool
	globalSaveValidOnly  bool
	globalRequestTimeout int
	globalProxyTimeout   int
	globalLogLevel       string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "universal-checker-global",
		Short: "Enhanced Universal Account Checker with Global Config Processing",
		Long: `A high-performance enhanced universal account checker that supports:
- OpenBullet (.opk) configurations
- SilverBullet (.svb) configurations  
- Loli (.loli) configurations
- Global processing: tests each combo against ALL configs simultaneously
- Enhanced logging and error reporting
- Real-time GUI with live logs and statistics
- Automatic proxy scraping (SOCKS4, SOCKS5, HTTP, HTTPS)
- Drag-and-drop config file support
- Superior performance compared to SilverBullet and OpenBullet`,
		Run: runGlobalChecker,
	}

	// Add flags
	rootCmd.Flags().StringSliceVarP(&globalConfigPaths, "configs", "c", []string{}, "Config file paths (supports .opk, .svb, .loli)")
	rootCmd.Flags().StringVarP(&globalComboPath, "combos", "l", "", "Combo list file path")
	rootCmd.Flags().StringVarP(&globalProxyPath, "proxies", "p", "", "Proxy list file path")
	rootCmd.Flags().StringVarP(&globalOutputDir, "output", "o", "results", "Output directory for results")
	rootCmd.Flags().IntVarP(&globalMaxWorkers, "workers", "w", 50, "Maximum number of workers")
	rootCmd.Flags().BoolVar(&globalAutoScrape, "auto-scrape", false, "Automatically scrape proxies")
	rootCmd.Flags().BoolVar(&globalSaveValidOnly, "valid-only", true, "Save only valid results")
	rootCmd.Flags().IntVar(&globalRequestTimeout, "request-timeout", 30000, "Request timeout in milliseconds")
	rootCmd.Flags().IntVar(&globalProxyTimeout, "proxy-timeout", 5000, "Proxy validation timeout in milliseconds")
	rootCmd.Flags().StringVar(&globalLogLevel, "log-level", "info", "Log level (debug, info, warning, error)")

	// Handle drag-and-drop arguments
	if len(os.Args) > 1 {
		handleGlobalDragAndDrop()
	}

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func handleGlobalDragAndDrop() {
	// Process command line arguments for drag-and-drop files
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-") {
			continue // Skip flags
		}

		if !utils.FileExists(arg) {
			continue // Skip non-existent files
		}

		ext := strings.ToLower(filepath.Ext(arg))
		switch ext {
		case ".opk", ".svb", ".loli":
			globalConfigPaths = append(globalConfigPaths, arg)
		case ".txt":
			// Determine if it's a combo or proxy file based on content
			if isGlobalComboFile(arg) {
				if globalComboPath == "" {
					globalComboPath = arg
				}
			} else {
				if globalProxyPath == "" {
					globalProxyPath = arg
				}
			}
		}
	}
}

func isGlobalComboFile(filePath string) bool {
	// Enhanced heuristic to determine if a file contains combos or proxies
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	// Read first few lines to analyze
	buffer := make([]byte, 2048)
	n, err := file.Read(buffer)
	if err != nil {
		return false
	}

	content := string(buffer[:n])
	lines := strings.Split(content, "\n")
	
	comboScore := 0
	proxyScore := 0
	
	// Analyze multiple lines for better accuracy
	for i, line := range lines {
		if i >= 10 { // Check first 10 lines max
			break
		}
		
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) >= 2 {
			// If first part looks like an IP, it's probably a proxy file
			if utils.IsValidIP(parts[0]) {
				proxyScore += 2
			}
			// If it contains @ symbol, it's likely a combo file
			if strings.Contains(parts[0], "@") {
				comboScore += 3
			}
			// If second part is a number and short (port), it's likely a proxy file
			if utils.IsNumeric(parts[1]) && len(parts[1]) <= 5 {
				proxyScore += 1
			} else if len(parts[1]) > 5 {
				// Likely a password
				comboScore += 2
			}
			// Check for email patterns
			if strings.Contains(line, "@") && strings.Contains(line, ".") {
				comboScore += 2
			}
		}
	}

	return comboScore > proxyScore
}

func runGlobalChecker(cmd *cobra.Command, args []string) {
	fmt.Println("🚀 Enhanced Universal Checker - Global Mode Starting...")
	
	// Validate inputs
	if len(globalConfigPaths) == 0 {
		log.Fatal("❌ No config files provided. Drag and drop .opk, .svb, or .loli files onto the executable.")
	}

	if globalComboPath == "" {
		log.Fatal("❌ No combo file provided. Please specify with --combos flag or drag and drop a combo file.")
	}

	// Create enhanced checker configuration
	checkerConfig := &types.CheckerConfig{
		MaxWorkers:        globalMaxWorkers,
		ProxyTimeout:      globalProxyTimeout,
		RequestTimeout:    globalRequestTimeout,
		RetryCount:        3,
		ProxyRotation:     true,
		AutoScrapeProxies: globalAutoScrape,
		SaveValidOnly:     globalSaveValidOnly,
		OutputFormat:      "txt",
		OutputDirectory:   globalOutputDir,
	}

	// Create global checker instance
	gc := checker.NewGlobalChecker(checkerConfig)

	// Load configurations
	fmt.Printf("📁 Loading %d configuration(s) for global processing...\n", len(globalConfigPaths))
	if err := gc.LoadConfigs(globalConfigPaths); err != nil {
		log.Fatalf("❌ Failed to load configs: %v", err)
	}

	for _, configPath := range globalConfigPaths {
		fmt.Printf("   ✅ Loaded: %s\n", filepath.Base(configPath))
	}

	// Load combos
	fmt.Printf("📋 Loading combos from: %s\n", filepath.Base(globalComboPath))
	if err := gc.LoadCombos(globalComboPath); err != nil {
		log.Fatalf("❌ Failed to load combos: %v", err)
	}
	fmt.Printf("   ✅ Loaded %d combos\n", len(gc.Combos))

	// Load proxies
	if globalAutoScrape {
		fmt.Println("🌐 Auto-scraping and validating proxies...")
		if err := gc.LoadProxies(""); err != nil {
			log.Printf("⚠️  Warning: Failed to scrape proxies: %v", err)
		} else {
			fmt.Printf("   ✅ Scraped and validated %d working proxies\n", len(gc.Proxies))
		}
	} else if globalProxyPath != "" {
		fmt.Printf("🌐 Loading proxies from: %s\n", filepath.Base(globalProxyPath))
		if err := gc.LoadProxies(globalProxyPath); err != nil {
			log.Printf("⚠️  Warning: Failed to load proxies: %v", err)
		} else {
			fmt.Printf("   ✅ Loaded %d proxies\n", len(gc.Proxies))
		}
	}

	// Create output directory
	if err := os.MkdirAll(globalOutputDir, 0755); err != nil {
		log.Fatalf("❌ Failed to create output directory: %v", err)
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start global checker
	fmt.Printf("⚡ Starting global checker with %d workers...\n", globalMaxWorkers)
	fmt.Printf("🔄 Each combo will be tested against ALL %d configs\n", len(gc.Configs))
	if err := gc.Start(); err != nil {
		log.Fatalf("❌ Failed to start global checker: %v", err)
	}

	// Display live statistics
	go displayGlobalStats(gc)

	// Display live logs
	go displayGlobalLogs(gc)

	fmt.Println("✨ Global checker started! Press Ctrl+C to stop.")
	
	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\n🛑 Shutdown signal received, stopping global checker...")
	gc.Stop()
	
	// Display final statistics
	finalStats := gc.GetGlobalStats()
	displayFinalGlobalStats(finalStats)
	
	fmt.Println("✅ Global checker stopped successfully")
}

func displayGlobalStats(gc *checker.GlobalChecker) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stats := gc.GetGlobalStats()
			
			// Clear screen and show stats
			fmt.Print("\033[2J\033[H") // Clear screen
			
			fmt.Println("🔥 Enhanced Universal Checker - Global Mode - Live Statistics")
			fmt.Println("===========================================================")
			fmt.Printf("⏱️  Elapsed Time:       %s\n", formatDuration(stats.ElapsedTime))
			fmt.Printf("📊 Total Combos:        %d\n", stats.TotalCombos)
			fmt.Printf("📁 Total Configs:       %d\n", stats.TotalConfigs)
			fmt.Printf("📈 Tasks Processed:     %d/%d\n", stats.ProcessedTasks, stats.TotalTasks)
			fmt.Printf("✅ Valid Combos:        %d\n", stats.ValidCombos)
			fmt.Printf("❌ Invalid Combos:      %d\n", stats.InvalidCombos)
			fmt.Printf("⚠️  Error Combos:       %d\n", stats.ErrorCombos)
			fmt.Printf("🚀 Current CPM:         %.1f\n", stats.CurrentCPM)
			fmt.Printf("👥 Active Workers:      %d\n", stats.ActiveWorkers)
			fmt.Printf("🌐 Working Proxies:     %d/%d\n", stats.WorkingProxies, stats.TotalProxies)
			
			// Progress bar
			if stats.TotalTasks > 0 {
				progress := float64(stats.ProcessedTasks) / float64(stats.TotalTasks) * 100
				fmt.Printf("📈 Progress:           %.1f%%\n", progress)
				fmt.Println(createProgressBar(int(progress), 50))
			}
			
			// Efficiency metrics
			if stats.ProcessedTasks > 0 {
				totalChecks := stats.ProcessedTasks * stats.TotalConfigs
				fmt.Printf("🔢 Total Config Checks: %d\n", totalChecks)
				
				if stats.ValidCombos > 0 {
					hitRate := float64(stats.ValidCombos) / float64(stats.ProcessedTasks) * 100
					fmt.Printf("🎯 Hit Rate:           %.2f%%\n", hitRate)
				}
			}
			
			fmt.Println("\n💡 Press Ctrl+C to stop checking")
		}
	}
}

func displayGlobalLogs(gc *checker.GlobalChecker) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	lastLogCount := 0

	for {
		select {
		case <-ticker.C:
			logs := gc.GetLogs()
			
			// Display only new logs
			if len(logs) > lastLogCount {
				newLogs := logs[lastLogCount:]
				for _, logEntry := range newLogs {
					color := getLogColor(logEntry.Level)
					fmt.Printf("%s[%s] %s: %s%s\n", 
						color,
						logEntry.Timestamp.Format("15:04:05"),
						strings.ToUpper(logEntry.Level),
						logEntry.Message,
						"\033[0m") // Reset color
				}
				lastLogCount = len(logs)
			}
		}
	}
}

func getLogColor(level string) string {
	switch level {
	case "error":
		return "\033[31m" // Red
	case "warning":
		return "\033[33m" // Yellow
	case "success":
		return "\033[32m" // Green
	case "info":
		return "\033[36m" // Cyan
	case "debug":
		return "\033[37m" // White
	default:
		return "\033[0m" // Default
	}
}

func displayFinalGlobalStats(stats types.GlobalCheckerStats) {
	fmt.Println("\n📊 Final Global Checker Statistics")
	fmt.Println("=====================================")
	fmt.Printf("⏱️  Total Runtime:      %s\n", formatDuration(stats.ElapsedTime))
	fmt.Printf("📊 Total Combos:        %d\n", stats.TotalCombos)
	fmt.Printf("📁 Total Configs:       %d\n", stats.TotalConfigs)
	fmt.Printf("✅ Valid Combos:        %d\n", stats.ValidCombos)
	fmt.Printf("❌ Invalid Combos:      %d\n", stats.InvalidCombos)
	fmt.Printf("⚠️  Error Combos:       %d\n", stats.ErrorCombos)
	fmt.Printf("🚀 Average CPM:         %.1f\n", stats.AverageCPM)
	
	if stats.TotalCombos > 0 {
		successRate := float64(stats.ValidCombos) / float64(stats.TotalCombos) * 100
		fmt.Printf("🎯 Success Rate:       %.2f%%\n", successRate)
	}
	
	totalChecks := stats.ProcessedTasks * stats.TotalConfigs
	fmt.Printf("🔢 Total Config Checks: %d\n", totalChecks)
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
