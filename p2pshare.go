package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	lipgloss "github.com/charmbracelet/lipgloss"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFB86C")).PaddingBottom(1)
	peerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).PaddingLeft(2)
	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD")).Bold(true)
	footerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).PaddingTop(1)
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Bold(true)
	boxStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
)

type model struct {
	peers  []string
	status string
}

func initialModel() model {
	return model{
		peers:  []string{},
		status: "🔍 Searching for peers...",
	}
}

func (m model) Init() tea.Cmd {
	return discoverPeers
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" {
			return m, tea.Quit
		}
	case []string:
		if len(msg) == 0 {
			m.status = errorStyle.Render("❌ No peers found.")
		} else {
			m.peers = msg
			m.status = statusStyle.Render("✅ Peers found! Select a file to send.")
		}
	}
	return m, nil
}

func (m model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("🔗 P2P File Sharing") + "\n\n")

	b.WriteString(boxStyle.Render(m.status) + "\n\n")

	if len(m.peers) > 0 {
		b.WriteString("🌍 Available Peers:\n")
		for _, peer := range m.peers {
			b.WriteString(peerStyle.Render("• "+peer) + "\n")
		}
	} else {
		b.WriteString(peerStyle.Render("⏳ Waiting for peers...") + "\n")
	}

	b.WriteString(footerStyle.Render("\nPress 'q' to quit."))

	return b.String()
}

func discoverPeers() tea.Msg {
	conn, err := net.ListenPacket("udp4", ":9876")
	if err != nil {
		return []string{"Error: " + err.Error()}
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	peers := make(map[string]struct{})

	_, err = conn.WriteTo([]byte("DISCOVER_PEER"), &net.UDPAddr{
		IP:   net.IPv4bcast,
		Port: 9876,
	})
	if err != nil {
		return []string{"Error sending broadcast: " + err.Error()}
	}

	timeout := time.Now().Add(2 * time.Second)
	for time.Now().Before(timeout) {
		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		_, addr, err := conn.ReadFrom(buf)
		if err == nil {
			peers[addr.String()] = struct{}{}
		}
	}

	peerList := []string{}
	for peer := range peers {
		peerList = append(peerList, peer)
	}

	return peerList
}

func main() {
	p := tea.NewProgram(initialModel())
	if err := p.Start(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
