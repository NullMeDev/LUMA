package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"universal-checker/internal/checker"
	"universal-checker/pkg/types"
	"universal-checker/pkg/utils"

	"github.com/spf13/cobra"
)

var (
	// Command line flags
	configPaths    []string
	comboPath      string
	proxyPath      string
	outputDir      string
	maxWorkers     int
	autoScrape     bool
	saveValidOnly  bool
	requestTimeout int
	proxyTimeout   int
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "universal-checker",
		Short: "Universal account checker compatible with opk, svb, and loli configs",
		Long: `A high-performance universal account checker that supports:
- OpenBullet (.opk) configurations
- SilverBullet (.svb) configurations  
- Loli (.loli) configurations
- Automatic proxy scraping (SOCKS4, SOCKS5, HTTP, HTTPS)
- Drag-and-drop config file support
- High CPM optimization`,
		Run: runChecker,
	}

	// Add flags
	rootCmd.Flags().StringSliceVarP(&configPaths, "configs", "c", []string{}, "Config file paths (supports .opk, .svb, .loli)")
	rootCmd.Flags().StringVarP(&comboPath, "combos", "l", "", "Combo list file path")
	rootCmd.Flags().StringVarP(&proxyPath, "proxies", "p", "", "Proxy list file path")
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", "results", "Output directory for results")
	rootCmd.Flags().IntVarP(&maxWorkers, "workers", "w", 100, "Maximum number of workers")
	rootCmd.Flags().BoolVar(&autoScrape, "auto-scrape", true, "Automatically scrape proxies")
	rootCmd.Flags().BoolVar(&saveValidOnly, "valid-only", true, "Save only valid results")
	rootCmd.Flags().IntVar(&requestTimeout, "request-timeout", 30000, "Request timeout in milliseconds")
	rootCmd.Flags().IntVar(&proxyTimeout, "proxy-timeout", 5000, "Proxy validation timeout in milliseconds")

	// Handle drag-and-drop arguments
	if len(os.Args) > 1 {
		handleDragAndDrop()
	}

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func handleDragAndDrop() {
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
			configPaths = append(configPaths, arg)
		case ".txt":
			// Determine if it's a combo or proxy file based on content
			if isComboFile(arg) {
				if comboPath == "" {
					comboPath = arg
				}
			} else {
				if proxyPath == "" {
					proxyPath = arg
				}
			}
		}
	}
}

func isComboFile(filePath string) bool {
	// Simple heuristic to determine if a file contains combos or proxies
	// Combo files typically have email:password or username:password format
	// Proxy files typically have ip:port format
	
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	// Read first few lines to analyze
	buffer := make([]byte, 1024)
	n, err := file.Read(buffer)
	if err != nil {
		return false
	}

	content := string(buffer[:n])
	lines := strings.Split(content, "\n")
	
	// Check first valid line
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) >= 2 {
			// If first part looks like an IP, it's probably a proxy file
			if utils.IsValidIP(parts[0]) {
				return false
			}
			// If it contains @ symbol, it's likely a combo file
			if strings.Contains(parts[0], "@") {
				return true
			}
			// If second part is a number (port), it's likely a proxy file
			if utils.IsNumeric(parts[1]) && len(parts[1]) <= 5 {
				return false
			}
			// Otherwise assume it's a combo file
			return true
		}
	}

	return true // Default to combo file
}

func runChecker(cmd *cobra.Command, args []string) {
	fmt.Println("🚀 Universal Checker - Starting...")
	
	// Validate inputs
	if len(configPaths) == 0 {
		log.Fatal("❌ No config files provided. Drag and drop .opk, .svb, or .loli files onto the executable.")
	}

	if comboPath == "" {
		log.Fatal("❌ No combo file provided. Please specify with --combos flag or drag and drop a combo file.")
	}

	// Create checker configuration
	checkerConfig := &types.CheckerConfig{
		MaxWorkers:        maxWorkers,
		ProxyTimeout:      proxyTimeout,
		RequestTimeout:    requestTimeout,
		RetryCount:        3,
		ProxyRotation:     true,
		AutoScrapeProxies: autoScrape,
		SaveValidOnly:     saveValidOnly,
		OutputFormat:      "txt",
		OutputDirectory:   outputDir,
	}

	// Create checker instance
	c := checker.NewChecker(checkerConfig)

	// Load configurations
	fmt.Printf("📁 Loading %d configuration(s)...\n", len(configPaths))
	if err := c.LoadConfigs(configPaths); err != nil {
		log.Fatalf("❌ Failed to load configs: %v", err)
	}

	for _, configPath := range configPaths {
		fmt.Printf("   ✅ Loaded: %s\n", filepath.Base(configPath))
	}

	// Load combos
	fmt.Printf("📋 Loading combos from: %s\n", filepath.Base(comboPath))
	if err := c.LoadCombos(comboPath); err != nil {
		log.Fatalf("❌ Failed to load combos: %v", err)
	}
	fmt.Printf("   ✅ Loaded %d combos\n", len(c.Combos))

	// Load proxies
	if autoScrape {
		fmt.Println("🌐 Auto-scraping and validating proxies...")
		if err := c.LoadProxies(""); err != nil {
			log.Printf("⚠️  Warning: Failed to scrape proxies: %v", err)
		} else {
			fmt.Printf("   ✅ Scraped and validated %d working proxies\n", len(c.Proxies))
		}
	} else if proxyPath != "" {
		fmt.Printf("🌐 Loading proxies from: %s\n", filepath.Base(proxyPath))
		if err := c.LoadProxies(proxyPath); err != nil {
			log.Printf("⚠️  Warning: Failed to load proxies: %v", err)
		} else {
			fmt.Printf("   ✅ Loaded %d proxies\n", len(c.Proxies))
		}
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("❌ Failed to create output directory: %v", err)
	}

	// Start checker
	fmt.Println("⚡ Starting checker with", maxWorkers, "workers...")
	if err := c.Start(); err != nil {
		log.Fatalf("❌ Failed to start checker: %v", err)
	}

	// Display live statistics
	go displayStats(c)

	// Wait for completion (in a real implementation, you'd handle signals)
	fmt.Println("✨ Checker started! Press Ctrl+C to stop.")
	
	// Keep the program running
	select {}
}

func displayStats(c *checker.Checker) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stats := c.GetStats()
			
			// Clear screen and show stats
			fmt.Print("\033[2J\033[H") // Clear screen
			
			fmt.Println("🔥 Universal Checker - Live Statistics")
			fmt.Println("=====================================")
			fmt.Printf("⏱️  Elapsed Time:    %s\n", formatDuration(stats.ElapsedTime))
			fmt.Printf("📊 Total Combos:     %d\n", stats.TotalCombos)
			fmt.Printf("✅ Valid:           %d\n", stats.ValidCombos)
			fmt.Printf("❌ Invalid:         %d\n", stats.InvalidCombos)
			fmt.Printf("⚠️  Errors:          %d\n", stats.ErrorCombos)
			fmt.Printf("🚀 Current CPM:     %.1f\n", stats.CurrentCPM)
			fmt.Printf("👥 Active Workers:   %d\n", stats.ActiveWorkers)
			fmt.Printf("🌐 Working Proxies:  %d/%d\n", stats.WorkingProxies, stats.TotalProxies)
			
			// Progress bar
			totalProcessed := stats.ValidCombos + stats.InvalidCombos + stats.ErrorCombos
			totalTasks := stats.TotalCombos * len(c.Configs) // Each combo tested against each config
			if totalTasks > 0 {
				progress := float64(totalProcessed) / float64(totalTasks) * 100
				fmt.Printf("📈 Progress:        %.1f%%\n", progress)
				fmt.Println(createProgressBar(int(progress), 50))
			}
			
			fmt.Println("\n💡 Press Ctrl+C to stop checking")
		}
	}
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
