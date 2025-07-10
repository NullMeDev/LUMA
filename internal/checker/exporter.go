package checker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"universal-checker/pkg/types"
	"universal-checker/pkg/utils"
)

// ResultExporter handles exporting results in various formats
type ResultExporter struct {
	OutputDir string
	Format    string
}

// NewResultExporter creates a new result exporter
func NewResultExporter(outputDir, format string) *ResultExporter {
	return &ResultExporter{
		OutputDir: outputDir,
		Format:    format,
	}
}

// ExportResult exports a single result
func (e *ResultExporter) ExportResult(result types.CheckResult) error {
	// Create output directory if it doesn't exist
	if err := utils.CreateDirectory(e.OutputDir); err != nil {
		return err
	}

	// Create config-specific directory
	configDir := filepath.Join(e.OutputDir, utils.SanitizeFilename(result.Config))
	if err := utils.CreateDirectory(configDir); err != nil {
		return err
	}

	// Determine file path based on status
	var filename string
	switch result.Status {
	case "valid":
		filename = "valid.txt"
	case "invalid":
		filename = "invalid.txt"
	case "error":
		filename = "errors.txt"
	default:
		filename = "unknown.txt"
	}

	filePath := filepath.Join(configDir, filename)

	// Format the result line
	var line string
	switch e.Format {
	case "json":
		jsonData, err := json.Marshal(result)
		if err != nil {
			return err
		}
		line = string(jsonData) + "\n"
	case "csv":
		line = fmt.Sprintf("%s,%s,%s,%s,%d,%s\n",
			result.Combo.Username,
			result.Combo.Password,
			result.Status,
			result.Config,
			result.Latency,
			result.Timestamp.Format("2006-01-02 15:04:05"))
	default: // txt format
		if result.Combo.Email != "" {
			line = fmt.Sprintf("%s:%s\n", result.Combo.Email, result.Combo.Password)
		} else {
			line = fmt.Sprintf("%s:%s\n", result.Combo.Username, result.Combo.Password)
		}
	}

	// Append to file
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(line)
	return err
}

// ExportGlobalResult exports a global result (combo tested against all configs)
func (e *ResultExporter) ExportGlobalResult(result types.GlobalWorkerResult) error {
	// Create output directory if it doesn't exist
	if err := utils.CreateDirectory(e.OutputDir); err != nil {
		return err
	}

	// Create global results directory
	globalDir := filepath.Join(e.OutputDir, "global")
	if err := utils.CreateDirectory(globalDir); err != nil {
		return err
	}

	// Determine file path based on overall status
	var filename string
	switch result.OverallStatus {
	case "valid":
		filename = "valid.txt"
	case "invalid":
		filename = "invalid.txt"
	case "error":
		filename = "errors.txt"
	default:
		filename = "unknown.txt"
	}

	filePath := filepath.Join(globalDir, filename)

	// Format the result line
	var line string
	switch e.Format {
	case "json":
		jsonData, err := json.Marshal(result)
		if err != nil {
			return err
		}
		line = string(jsonData) + "\n"
	case "csv":
		// Create CSV line with global result info
		line = fmt.Sprintf("%s,%s,%s,%d,%d,%d,%s\n",
			result.Combo.Username,
			result.Combo.Password,
			result.OverallStatus,
			result.ValidConfigCount,
			len(result.Results),
			result.Latency,
			result.Timestamp.Format("2006-01-02 15:04:05"))
	default: // txt format
		if result.Combo.Email != "" {
			line = fmt.Sprintf("%s:%s\n", result.Combo.Email, result.Combo.Password)
		} else {
			line = fmt.Sprintf("%s:%s\n", result.Combo.Username, result.Combo.Password)
		}
	}

	// Append to file
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(line)
	if err != nil {
		return err
	}

	// Also export detailed results for each config if valid
	if result.OverallStatus == "valid" {
		return e.exportDetailedGlobalResult(result)
	}

	return nil
}

// exportDetailedGlobalResult exports detailed results for each config
func (e *ResultExporter) exportDetailedGlobalResult(result types.GlobalWorkerResult) error {
	detailedDir := filepath.Join(e.OutputDir, "detailed")
	if err := utils.CreateDirectory(detailedDir); err != nil {
		return err
	}

	// Create a detailed result file for this combo
	filename := fmt.Sprintf("combo_%d_%s.json", result.TaskID, utils.SanitizeFilename(result.Combo.Username))
	filePath := filepath.Join(detailedDir, filename)

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, jsonData, 0644)
}

// ExportStats exports final statistics
func (e *ResultExporter) ExportStats(stats types.CheckerStats, configs []types.Config) error {
	if err := utils.CreateDirectory(e.OutputDir); err != nil {
		return err
	}

	statsFile := filepath.Join(e.OutputDir, "stats.json")
	
	// Create detailed stats
	detailedStats := map[string]interface{}{
		"summary": stats,
		"configs": configs,
		"export_time": time.Now().Format("2006-01-02 15:04:05"),
	}

	jsonData, err := json.MarshalIndent(detailedStats, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(statsFile, jsonData, 0644)
}

// GetResultsSummary returns a summary of exported results
func (e *ResultExporter) GetResultsSummary() (map[string]map[string]int, error) {
	summary := make(map[string]map[string]int)

	// Walk through the output directory
	err := filepath.Walk(e.OutputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Get relative path from output directory
		relPath, err := filepath.Rel(e.OutputDir, path)
		if err != nil {
			return err
		}

		// Extract config name and result type
		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) >= 2 {
			configName := parts[0]
			fileName := parts[len(parts)-1]
			
			if summary[configName] == nil {
				summary[configName] = make(map[string]int)
			}

			// Count lines in file
			count, err := e.countLinesInFile(path)
			if err != nil {
				return err
			}

			resultType := strings.TrimSuffix(fileName, filepath.Ext(fileName))
			summary[configName][resultType] = count
		}

		return nil
	})

	return summary, err
}

// countLinesInFile counts the number of lines in a file
func (e *ResultExporter) countLinesInFile(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	count := 0
	buffer := make([]byte, 32*1024)
	
	for {
		n, err := file.Read(buffer)
		if n == 0 {
			break
		}
		
		for i := 0; i < n; i++ {
			if buffer[i] == '\n' {
				count++
			}
		}
		
		if err != nil {
			break
		}
	}

	return count, nil
}
