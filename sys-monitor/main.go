package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk" // added for disk monitoring
	"github.com/shirou/gopsutil/v3/mem"
)

// Model represents the application state
type Model struct {
	cpuUsage    float64
	memoryUsage float64
	memoryTotal uint64
	diskUsage   float64 // added for disk usage percentage
	diskTotal   uint64  // added for disk total bytes
	width       int
	height      int
}

// Define some styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			PaddingLeft(2).
			PaddingRight(2)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Italic(true)

	barBaseStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#333333")).
			PaddingLeft(1).
			PaddingRight(1)

	cpuBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#FF4757"))

	memBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#2ED573"))

	diskBarStyle = lipgloss.NewStyle(). // new style for disk usage bar
			Background(lipgloss.Color("#1E90FF"))
)

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tick()
}

// Update updates the model based on messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case tickMsg:
		// Get CPU usage
		cpuPercentages, err := cpu.Percent(0, false)
		if err == nil && len(cpuPercentages) > 0 {
			m.cpuUsage = cpuPercentages[0]
		}

		// Get memory usage
		memInfo, err := mem.VirtualMemory()
		if err == nil {
			m.memoryUsage = memInfo.UsedPercent
			m.memoryTotal = memInfo.Total
		}

		// Disk usage update (using "C:" drive)
		diskInfo, err := disk.Usage("C:")
		if err == nil {
			m.diskUsage = diskInfo.UsedPercent
			m.diskTotal = diskInfo.Total
		}

		return m, tick()
	}

	return m, nil
}

// View renders the UI
func (m Model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	// Calculate bar width (max 50 chars or screen width - 20)
	maxBarWidth := m.width - 20
	if maxBarWidth > 50 {
		maxBarWidth = 50
	}
	if maxBarWidth < 10 {
		maxBarWidth = 10
	}

	// Render CPU usage bar
	cpuBarWidth := int((m.cpuUsage / 100) * float64(maxBarWidth))
	cpuBar := barBaseStyle.Render(
		cpuBarStyle.Width(cpuBarWidth).Render("") +
			lipgloss.NewStyle().Width(maxBarWidth-cpuBarWidth).Render(""),
	)

	// Render Memory usage bar
	memBarWidth := int((m.memoryUsage / 100) * float64(maxBarWidth))
	memBar := barBaseStyle.Render(
		memBarStyle.Width(memBarWidth).Render("") +
			lipgloss.NewStyle().Width(maxBarWidth-memBarWidth).Render(""),
	)

	// Render Disk usage bar
	diskBarWidth := int((m.diskUsage / 100) * float64(maxBarWidth))
	diskBar := barBaseStyle.Render(
		diskBarStyle.Width(diskBarWidth).Render("") +
			lipgloss.NewStyle().Width(maxBarWidth-diskBarWidth).Render(""),
	)

	// Calculate memory usage in GB
	memUsedGB := float64(m.memoryTotal) * m.memoryUsage / 100 / 1024 / 1024 / 1024
	memTotalGB := float64(m.memoryTotal) / 1024 / 1024 / 1024

	// Calculate disk usage in GB
	diskUsedGB := float64(m.diskTotal) * m.diskUsage / 100 / 1024 / 1024 / 1024
	diskTotalGB := float64(m.diskTotal) / 1024 / 1024 / 1024

	return fmt.Sprintf(
		"\n %s\n\n CPU Usage:       %s %.1f%%\n\n Memory Usage:    %s %.1f%% (%.1f/%.1f GB)\n\n Disk Usage (C:): %s %.1f%% (%.1f/%.1f GB)\n\n %s\n\n",
		titleStyle.Render(" SYSTEM MONITOR "),
		cpuBar,
		m.cpuUsage,
		memBar,
		m.memoryUsage,
		memUsedGB,
		memTotalGB,
		diskBar,
		m.diskUsage,
		diskUsedGB,
		diskTotalGB,
		infoStyle.Render("Press q to quit"),
	)
}

// Define a message type for our timer tick
type tickMsg time.Time

// tick creates a command that will send a tick message after a short delay
func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func main() {
	p := tea.NewProgram(
		Model{},
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
	}
}
