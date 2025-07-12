package main

import (
	"fmt"
	"os"

	"universal-checker/internal/config"
)

func main() {
	// Test the enhanced parser with real config files
	parser := config.NewParser()

	// Test VPN OPK file
	opkPath := "/home/null/Desktop/checkertools/Configs/VPN/PotatoVPN.opk"
	fmt.Printf("Testing VPN OPK file: %s\n", opkPath)
	
	if _, err := os.Stat(opkPath); os.IsNotExist(err) {
		fmt.Printf("File does not exist: %s\n", opkPath)
		return
	}

	cfg, err := parser.ParseConfig(opkPath)
	if err != nil {
		fmt.Printf("Error parsing OPK: %v\n", err)
	} else {
		fmt.Printf("✓ Success parsing OPK!\n")
		fmt.Printf("  Name: %s\n", cfg.Name)
		fmt.Printf("  Type: %s\n", cfg.Type)
		fmt.Printf("  URL: %s\n", cfg.URL)
		fmt.Printf("  Method: %s\n", cfg.Method)
		fmt.Printf("  RequiresProxy: %t\n", cfg.RequiresProxy)
		fmt.Printf("  Success Strings: %v\n", cfg.SuccessStrings)
		fmt.Printf("  Failure Strings: %v\n", cfg.FailureStrings)
	}

	// Test with ConfigManager for drag-and-drop simulation
	fmt.Println("\n--- Testing ConfigManager ---")
	manager := config.NewConfigManager()
	
	result, err := manager.LoadConfigsFromDrop([]string{opkPath})
	if err != nil {
		fmt.Printf("Error loading configs: %v\n", err)
	} else {
		fmt.Printf("✓ Successfully loaded %d configs\n", result.Stats.SuccessfulLoads)
		fmt.Printf("  Proxy configs: %d\n", result.Stats.ProxyRequired)
		fmt.Printf("  Proxyless configs: %d\n", result.Stats.ProxyOptional)
		fmt.Printf("  Failed loads: %d\n", result.Stats.FailedLoads)
		fmt.Printf("  Format breakdown: %v\n", result.Stats.FormatBreakdown)
		
		if len(result.Errors) > 0 {
			fmt.Println("  Errors:")
			for _, errMsg := range result.Errors {
				fmt.Printf("    - %s\n", errMsg)
			}
		}
	}

	// Test streaming config for geo-lock detection
	fmt.Println("\n--- Testing Streaming Config ---")
	streamingPath := "/home/null/Desktop/checkertools/Configs/Streaming/Crunchyroll [Android].opk"
	fmt.Printf("Testing streaming config: %s\n", streamingPath)
	
	if _, err := os.Stat(streamingPath); os.IsNotExist(err) {
		fmt.Printf("Streaming file does not exist: %s\n", streamingPath)
	} else {
		cfg2, err := parser.ParseConfig(streamingPath)
		if err != nil {
			fmt.Printf("Error parsing streaming config: %v\n", err)
		} else {
			fmt.Printf("✓ Success parsing streaming config!\n")
			fmt.Printf("  Name: %s\n", cfg2.Name)
			fmt.Printf("  RequiresProxy: %t\n", cfg2.RequiresProxy)
			fmt.Printf("  (Should be true due to 'streaming' and 'crunchyroll' in path)\n")
		}
	}

	// Test multiple configs batch processing
	fmt.Println("\n--- Testing Batch Loading ---")
	batchPaths := []string{
		opkPath,
		"/home/null/Desktop/checkertools/Configs/Streaming/OnlyFans.opk",
		"/home/null/Desktop/checkertools/Configs/VPN/NordVPN.opk",
	}
	
	batchResult, err := manager.LoadConfigsFromDrop(batchPaths)
	if err != nil {
		fmt.Printf("Error batch loading: %v\n", err)
	} else {
		fmt.Printf("✓ Batch loaded %d/%d configs successfully\n", batchResult.Stats.SuccessfulLoads, batchResult.Stats.TotalProcessed)
		fmt.Printf("  Proxy required: %d\n", batchResult.Stats.ProxyRequired)
		fmt.Printf("  Proxy optional: %d\n", batchResult.Stats.ProxyOptional)
		fmt.Printf("  Processing time: %v\n", batchResult.Stats.ProcessingTime)
		
		if len(batchResult.Errors) > 0 {
			fmt.Println("  Batch errors:")
			for _, errMsg := range batchResult.Errors {
				fmt.Printf("    - %s\n", errMsg)
			}
		}
	}
}
