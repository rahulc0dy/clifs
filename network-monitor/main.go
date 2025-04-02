package main

import (
	"fmt"
	"net"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	psnet "github.com/shirou/gopsutil/net"
)

// Model holds application state.
type Model struct {
	interfaces   []net.Interface
	networkStats []psnet.IOCountersStat
	historySent  []float64 // history for bytes sent
	historyRecv  []float64 // history for bytes received
	latestSent   uint64    // new: current total bytes sent
	latestRecv   uint64    // new: current total bytes received
	err          error
	lastUpdate   time.Time
}

// TickMsg signals a tick update.
type TickMsg time.Time

// Init initializes the program.
func (m Model) Init() tea.Cmd {
	// Schedule initial fetches for interfaces and network stats.
	return tea.Batch(fetchInterfaces, fetchNetworkStats, tickCmd())
}

// fetchInterfaces returns a message with the current network interfaces.
func fetchInterfaces() tea.Msg {
	interfaces, err := net.Interfaces()
	if err != nil {
		return errMsg{err}
	}
	return interfacesMsg(interfaces)
}

// fetchNetworkStats returns a message with current network I/O counters.
func fetchNetworkStats() tea.Msg {
	stats, err := psnet.IOCounters(true)
	if err != nil {
		return errMsg{err}
	}
	return networkStatsMsg(stats)
}

// tickCmd sends a TickMsg after 5 seconds.
func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

type interfacesMsg []net.Interface
type networkStatsMsg []psnet.IOCountersStat
type errMsg struct {
	err error
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	case interfacesMsg:
		m.interfaces = []net.Interface(msg)
		m.lastUpdate = time.Now()
		return m, tickCmd()
	case TickMsg:
		// On tick, fetch both interfaces and network stats.
		return m, tea.Batch(fetchInterfaces, fetchNetworkStats, tickCmd())
	case networkStatsMsg:
		m.networkStats = []psnet.IOCountersStat(msg)
		// Compute total bytes sent/received across all interfaces.
		var totalSent, totalRecv uint64
		for _, stat := range m.networkStats {
			totalSent += stat.BytesSent
			totalRecv += stat.BytesRecv
		}
		m.latestSent = totalSent
		m.latestRecv = totalRecv
		// Update history slices.
		m.historySent = append(m.historySent, float64(totalSent))
		m.historyRecv = append(m.historyRecv, float64(totalRecv))
		// Limit history to latest 20 data points.
		if len(m.historySent) > 20 {
			m.historySent = m.historySent[1:]
			m.historyRecv = m.historyRecv[1:]
		}
		return m, nil
	case errMsg:
		m.err = msg.err
		return m, tea.Quit
	}
	return m, nil
}

const maxBarWidth = 50        // maximum bar width in characters
const scaleFactor = 1000000.0 // 1 unit per 1MB

// Add new network bar styles (similar to system monitor)
var (
	barBaseStyle    = lipgloss.NewStyle().Background(lipgloss.Color("#333333")).PaddingLeft(1).PaddingRight(1)
	netSentBarStyle = lipgloss.NewStyle().Background(lipgloss.Color("#FFB86C"))
	netRecvBarStyle = lipgloss.NewStyle().Background(lipgloss.Color("#8BE9FD"))
)

// Modify renderBar to accept maximum width and fill style.
func renderBar(value float64, maxWidth int, fillStyle lipgloss.Style) string {
	barWidth := int(value / scaleFactor)
	if barWidth > maxWidth {
		barWidth = maxWidth
	}
	filled := fillStyle.Width(barWidth).Render("")
	empty := lipgloss.NewStyle().Width(maxWidth - barWidth).Render("")
	return barBaseStyle.Render(filled + empty)
}

func (m Model) View() string {
	s := "Network Monitor\n\n"
	if m.err != nil {
		s += fmt.Sprintf("Error: %v\n", m.err)
		return s
	}
	s += fmt.Sprintf("Last Update: %s\n\n", m.lastUpdate.Format(time.RFC1123))
	s += "Interfaces:\n"
	for _, iface := range m.interfaces {
		s += fmt.Sprintf("- %s, Flags: %v\n", iface.Name, iface.Flags)
		addrs, err := iface.Addrs()
		if err == nil {
			for _, addr := range addrs {
				s += fmt.Sprintf("   %s\n", addr.String())
			}
		}
	}
	s += "\nNetwork Activity:\n"
	for _, stat := range m.networkStats {
		s += fmt.Sprintf("- %s: Sent: %d B, Received: %d B\n", stat.Name, stat.BytesSent, stat.BytesRecv)
	}
	s += "\nNetwork Bar Graphs:\n"
	// Use a fixed max width for the network bars (similar to system monitor)
	maxWidth := 50
	s += fmt.Sprintf("Sent: %s %d B\n", renderBar(float64(m.latestSent), maxWidth, netSentBarStyle), m.latestSent)
	s += fmt.Sprintf("Recv: %s %d B\n", renderBar(float64(m.latestRecv), maxWidth, netRecvBarStyle), m.latestRecv)
	s += "\nPress q to quit.\n"
	return s
}

func main() {
	p := tea.NewProgram(Model{})
	if err := p.Start(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
