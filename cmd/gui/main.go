package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"fyne.io/fyne/v2"

	"universal-checker/internal/checker"
	"universal-checker/internal/config"
	"universal-checker/pkg/types"
)

type GUI struct {
	app    fyne.App
	window fyne.Window
	
	// File paths
	configPaths []string
	comboPath   string
	proxyPath   string
	
	// GUI components
	configList     *widget.List
	comboEntry     *widget.Entry
	proxyEntry     *widget.Entry
	selectAllCheck *widget.Check
	workersEntry   *widget.Entry
	timeoutEntry   *widget.Entry
	
	// Status components
	statusLabel    *widget.Label
	progressBar    *widget.ProgressBar
	statsLabel     *widget.RichText
	logArea        *widget.RichText
	
	// Control buttons
	startBtn       *widget.Button
	stopBtn        *widget.Button
	clearBtn       *widget.Button
	
	// Checker instance
	checker        *checker.Checker
	isRunning      bool
	mutex          sync.RWMutex
	
	// Configuration data
	configs        []types.Config
	selectedConfigs map[int]bool
}

func main() {
	gui := NewGUI()
	gui.Run()
}

func NewGUI() *GUI {
	myApp := app.New()
	myApp.SetIcon(nil) // You can set a custom icon here
	
	window := myApp.NewWindow("Universal Checker - GUI")
	window.Resize(fyne.NewSize(800, 600))
	
	gui := &GUI{
		app:             myApp,
		window:          window,
		configPaths:     make([]string, 0),
		selectedConfigs: make(map[int]bool),
		isRunning:       false,
	}
	
	gui.setupUI()
	return gui
}

func (g *GUI) setupUI() {
	// Main container
	content := container.NewVBox()
	
	// Header
	title := widget.NewLabelWithStyle("Universal Checker", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	
	// File selection section
	fileSection := g.createFileSection()
	
	// Configuration selection section
	configSection := g.createConfigSection()
	
	// Settings section
	settingsSection := g.createSettingsSection()
	
	// Control buttons
	controlSection := g.createControlSection()
	
	// Status and progress section
	statusSection := g.createStatusSection()
	
	// Add all sections to main container
	content.Add(title)
	content.Add(widget.NewSeparator())
	content.Add(fileSection)
	content.Add(widget.NewSeparator())
	content.Add(configSection)
	content.Add(widget.NewSeparator())
	content.Add(settingsSection)
	content.Add(widget.NewSeparator())
	content.Add(controlSection)
	content.Add(widget.NewSeparator())
	content.Add(statusSection)
	
	// Set up drag and drop
	g.setupDragAndDrop()
	
	g.window.SetContent(container.NewScroll(content))
}

func (g *GUI) createFileSection() *fyne.Container {
	section := container.NewVBox()
	
	// Combo file selection
	comboLabel := widget.NewLabel("Combo File:")
	g.comboEntry = widget.NewEntry()
	g.comboEntry.SetPlaceHolder("Select or drag combo file (.txt)")
	comboBtn := widget.NewButton("Browse", func() {
		g.selectComboFile()
	})
	comboRow := container.NewBorder(nil, nil, comboLabel, comboBtn, g.comboEntry)
	
	// Proxy file selection (optional)
	proxyLabel := widget.NewLabel("Proxy File:")
	g.proxyEntry = widget.NewEntry()
	g.proxyEntry.SetPlaceHolder("Optional: Select or drag proxy file (.txt)")
	proxyBtn := widget.NewButton("Browse", func() {
		g.selectProxyFile()
	})
	proxyRow := container.NewBorder(nil, nil, proxyLabel, proxyBtn, g.proxyEntry)
	
	// Auto-scrape option
	autoScrapeCheck := widget.NewCheck("Auto-scrape proxies", nil)
	autoScrapeCheck.SetChecked(true)
	
	section.Add(comboRow)
	section.Add(proxyRow)
	section.Add(autoScrapeCheck)
	
	return section
}

func (g *GUI) createConfigSection() *fyne.Container {
	section := container.NewVBox()
	
	// Config files header
	configHeader := container.NewHBox(
		widget.NewLabelWithStyle("Configuration Files", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewButton("Add Config", func() {
			g.selectConfigFiles()
		}),
		widget.NewButton("Clear All", func() {
			g.clearConfigs()
		}),
	)
	
	// Select all checkbox
	g.selectAllCheck = widget.NewCheck("Select All Configs", func(checked bool) {
		g.toggleAllConfigs(checked)
	})
	
	// Config list
	g.configList = widget.NewList(
		func() int {
			return len(g.configs)
		},
		func() fyne.CanvasObject {
			check := widget.NewCheck("", nil)
			label := widget.NewLabel("Config Name")
			return container.NewHBox(check, label)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			container := obj.(*fyne.Container)
			check := container.Objects[0].(*widget.Check)
			label := container.Objects[1].(*widget.Label)
			
			if id < len(g.configs) {
				config := g.configs[id]
				label.SetText(fmt.Sprintf("%s (%s)", config.Name, strings.ToUpper(string(config.Type))))
				check.SetChecked(g.selectedConfigs[id])
				check.OnChanged = func(checked bool) {
					g.selectedConfigs[id] = checked
					g.updateSelectAllCheck()
				}
			}
		},
	)
	g.configList.Resize(fyne.NewSize(400, 150))
	
	section.Add(configHeader)
	section.Add(g.selectAllCheck)
	section.Add(g.configList)
	
	return section
}

func (g *GUI) createSettingsSection() *fyne.Container {
	section := container.NewVBox()
	
	settingsLabel := widget.NewLabelWithStyle("Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	
	// Workers setting
	workersLabel := widget.NewLabel("Workers:")
	g.workersEntry = widget.NewEntry()
	g.workersEntry.SetText("100")
	workersRow := container.NewBorder(nil, nil, workersLabel, nil, g.workersEntry)
	
	// Timeout setting
	timeoutLabel := widget.NewLabel("Timeout (ms):")
	g.timeoutEntry = widget.NewEntry()
	g.timeoutEntry.SetText("30000")
	timeoutRow := container.NewBorder(nil, nil, timeoutLabel, nil, g.timeoutEntry)
	
	settingsGrid := container.NewGridWithColumns(2, workersRow, timeoutRow)
	
	section.Add(settingsLabel)
	section.Add(settingsGrid)
	
	return section
}

func (g *GUI) createControlSection() *fyne.Container {
	g.startBtn = widget.NewButton("Start Checking", func() {
		g.startChecking()
	})
	g.startBtn.Importance = widget.HighImportance
	
	g.stopBtn = widget.NewButton("Stop", func() {
		g.stopChecking()
	})
	g.stopBtn.Disable()
	
	g.clearBtn = widget.NewButton("Clear Results", func() {
		g.clearResults()
	})
	
	return container.NewHBox(g.startBtn, g.stopBtn, g.clearBtn)
}

func (g *GUI) createStatusSection() *fyne.Container {
	section := container.NewVBox()
	
	// Status label
	g.statusLabel = widget.NewLabelWithStyle("Ready", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	
	// Progress bar
	g.progressBar = widget.NewProgressBar()
	g.progressBar.Hide()
	
	// Statistics
	g.statsLabel = widget.NewRichTextFromMarkdown("")
	g.statsLabel.Resize(fyne.NewSize(400, 100))
	
	// Log area
	logLabel := widget.NewLabelWithStyle("Log Output", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	g.logArea = widget.NewRichText()
	g.logArea.Resize(fyne.NewSize(400, 150))
	logScroll := container.NewScroll(g.logArea)
	logScroll.SetMinSize(fyne.NewSize(400, 150))
	
	section.Add(g.statusLabel)
	section.Add(g.progressBar)
	section.Add(g.statsLabel)
	section.Add(logLabel)
	section.Add(logScroll)
	
	return section
}

func (g *GUI) setupDragAndDrop() {
	// Note: Fyne doesn't have built-in drag and drop for files yet
	// This is a placeholder for when that feature is available
	// For now, users will use the browse buttons
}

func (g *GUI) selectComboFile() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()
		
		g.comboPath = reader.URI().Path()
		g.comboEntry.SetText(filepath.Base(g.comboPath))
		g.logMessage(fmt.Sprintf("Loaded combo file: %s", filepath.Base(g.comboPath)))
	}, g.window)
}

func (g *GUI) selectProxyFile() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()
		
		g.proxyPath = reader.URI().Path()
		g.proxyEntry.SetText(filepath.Base(g.proxyPath))
		g.logMessage(fmt.Sprintf("Loaded proxy file: %s", filepath.Base(g.proxyPath)))
	}, g.window)
}

func (g *GUI) selectConfigFiles() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()
		
		configPath := reader.URI().Path()
		ext := strings.ToLower(filepath.Ext(configPath))
		
		if ext != ".opk" && ext != ".svb" && ext != ".loli" {
			dialog.ShowError(fmt.Errorf("unsupported config format: %s", ext), g.window)
			return
		}
		
		// Parse the config
		parser := config.NewParser()
		cfg, err := parser.ParseConfig(configPath)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to parse config: %v", err), g.window)
			return
		}
		
		g.configs = append(g.configs, *cfg)
		g.configPaths = append(g.configPaths, configPath)
		g.selectedConfigs[len(g.configs)-1] = true
		
		g.configList.Refresh()
		g.updateSelectAllCheck()
		g.logMessage(fmt.Sprintf("Loaded config: %s (%s)", cfg.Name, strings.ToUpper(string(cfg.Type))))
	}, g.window)
}

func (g *GUI) clearConfigs() {
	g.configs = make([]types.Config, 0)
	g.configPaths = make([]string, 0)
	g.selectedConfigs = make(map[int]bool)
	g.configList.Refresh()
	g.selectAllCheck.SetChecked(false)
	g.logMessage("Cleared all configurations")
}

func (g *GUI) toggleAllConfigs(checked bool) {
	for i := range g.configs {
		g.selectedConfigs[i] = checked
	}
	g.configList.Refresh()
}

func (g *GUI) updateSelectAllCheck() {
	allSelected := true
	for i := range g.configs {
		if !g.selectedConfigs[i] {
			allSelected = false
			break
		}
	}
	g.selectAllCheck.SetChecked(allSelected && len(g.configs) > 0)
}

func (g *GUI) startChecking() {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	
	if g.isRunning {
		return
	}
	
	// Validate inputs
	if g.comboPath == "" {
		dialog.ShowError(fmt.Errorf("please select a combo file"), g.window)
		return
	}
	
	selectedConfigs := g.getSelectedConfigs()
	if len(selectedConfigs) == 0 {
		dialog.ShowError(fmt.Errorf("please select at least one configuration"), g.window)
		return
	}
	
	// Parse settings
	workers := 100
	if w := g.workersEntry.Text; w != "" {
		if parsed, err := fmt.Sscanf(w, "%d", &workers); err != nil || parsed != 1 {
			workers = 100
		}
	}
	
	timeout := 30000
	if t := g.timeoutEntry.Text; t != "" {
		if parsed, err := fmt.Sscanf(t, "%d", &timeout); err != nil || parsed != 1 {
			timeout = 30000
		}
	}
	
	// Create checker configuration
	checkerConfig := &types.CheckerConfig{
		MaxWorkers:        workers,
		ProxyTimeout:      5000,
		RequestTimeout:    timeout,
		RetryCount:        3,
		ProxyRotation:     true,
		AutoScrapeProxies: g.proxyPath == "",
		SaveValidOnly:     true,
		OutputFormat:      "txt",
		OutputDirectory:   "results",
	}
	
	// Create checker instance
	g.checker = checker.NewChecker(checkerConfig)
	
	// Set only selected configs
	g.checker.Configs = selectedConfigs
	
	// Start checking in goroutine
	go g.runChecker()
	
	g.isRunning = true
	g.startBtn.Disable()
	g.stopBtn.Enable()
	g.statusLabel.SetText("Starting checker...")
	g.progressBar.Show()
}

func (g *GUI) stopChecking() {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	
	if !g.isRunning || g.checker == nil {
		return
	}
	
	g.checker.Stop()
	g.isRunning = false
	g.startBtn.Enable()
	g.stopBtn.Disable()
	g.statusLabel.SetText("Stopped")
	g.progressBar.Hide()
	g.logMessage("Checking stopped by user")
}

func (g *GUI) runChecker() {
	// Load combos
	g.logMessage("Loading combos...")
	if err := g.checker.LoadCombos(g.comboPath); err != nil {
		g.logMessage(fmt.Sprintf("Error loading combos: %v", err))
		g.stopChecking()
		return
	}
	g.logMessage(fmt.Sprintf("Loaded %d combos", len(g.checker.Combos)))
	
	// Load proxies
	if g.proxyPath != "" {
		g.logMessage("Loading proxies...")
		if err := g.checker.LoadProxies(g.proxyPath); err != nil {
			g.logMessage(fmt.Sprintf("Warning: Failed to load proxies: %v", err))
		} else {
			g.logMessage(fmt.Sprintf("Loaded %d proxies", len(g.checker.Proxies)))
		}
	} else {
		g.logMessage("Auto-scraping proxies...")
		if err := g.checker.LoadProxies(""); err != nil {
			g.logMessage(fmt.Sprintf("Warning: Failed to scrape proxies: %v", err))
		} else {
			g.logMessage(fmt.Sprintf("Scraped %d working proxies", len(g.checker.Proxies)))
		}
	}
	
	// Start checking
	g.logMessage("Starting checker...")
	if err := g.checker.Start(); err != nil {
		g.logMessage(fmt.Sprintf("Error starting checker: %v", err))
		g.stopChecking()
		return
	}
	
	// Update status periodically
	go g.updateStats()
}

func (g *GUI) updateStats() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if !g.isRunning {
				return
			}
			
			stats := g.checker.GetStats()
			
			// Update status
			g.statusLabel.SetText(fmt.Sprintf("Running - CPM: %.1f", stats.CurrentCPM))
			
			// Update progress
			totalTasks := stats.TotalCombos * len(g.checker.Configs)
			processed := stats.ValidCombos + stats.InvalidCombos + stats.ErrorCombos
			if totalTasks > 0 {
				progress := float64(processed) / float64(totalTasks)
				g.progressBar.SetValue(progress)
			}
			
			// Update stats
			statsText := fmt.Sprintf(`**Statistics**

⏱️ **Elapsed Time:** %s
📊 **Total Combos:** %d
✅ **Valid:** %d
❌ **Invalid:** %d
⚠️ **Errors:** %d
🚀 **Current CPM:** %.1f
👥 **Active Workers:** %d
🌐 **Working Proxies:** %d/%d
📈 **Progress:** %.1f%%`,
				g.formatDuration(stats.ElapsedTime),
				stats.TotalCombos,
				stats.ValidCombos,
				stats.InvalidCombos,
				stats.ErrorCombos,
				stats.CurrentCPM,
				stats.ActiveWorkers,
				stats.WorkingProxies,
				stats.TotalProxies,
				float64(processed)/float64(totalTasks)*100)
			
			g.statsLabel.ParseMarkdown(statsText)
		}
	}
}

func (g *GUI) getSelectedConfigs() []types.Config {
	var selected []types.Config
	for i, config := range g.configs {
		if g.selectedConfigs[i] {
			selected = append(selected, config)
		}
	}
	return selected
}

func (g *GUI) clearResults() {
	g.logArea.ParseMarkdown("")
	g.statsLabel.ParseMarkdown("")
	g.progressBar.SetValue(0)
	g.statusLabel.SetText("Ready")
}

func (g *GUI) logMessage(message string) {
	timestamp := time.Now().Format("15:04:05")
	logEntry := fmt.Sprintf("[%s] %s\n", timestamp, message)
	
	// Append to log area
	currentText := g.logArea.String()
	newText := currentText + logEntry
	g.logArea.ParseMarkdown(newText)
}

func (g *GUI) formatDuration(seconds int) string {
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

func (g *GUI) Run() {
	g.window.ShowAndRun()
}
