package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	lipgloss "github.com/charmbracelet/lipgloss"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFB86C")).PaddingBottom(1)
	peerStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).PaddingLeft(2)
	statusStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD")).Bold(true)
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Bold(true)
	footerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).PaddingTop(1)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF79C6"))
	boxStyle      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
)

type model struct {
	peers        []string
	files        []string
	selectedPeer int
	selectedFile int
	stage        string
	status       string
}

func initialModel() model {
	return model{
		peers:        []string{},
		files:        getFiles(),
		selectedPeer: 0,
		selectedFile: 0,
		stage:        "peers",
		status:       "🔍 Searching for peers...",
	}
}

func (m model) Init() tea.Cmd {
	return discoverPeers
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			return m, tea.Quit

		case tea.KeyDown:
			if m.stage == "peers" && len(m.peers) > 0 && m.selectedPeer < len(m.peers)-1 {
				m.selectedPeer++
			} else if m.stage == "files" && len(m.files) > 0 && m.selectedFile < len(m.files)-1 {
				m.selectedFile++
			}

		case tea.KeyUp:
			if m.stage == "peers" && m.selectedPeer > 0 {
				m.selectedPeer--
			} else if m.stage == "files" && m.selectedFile > 0 {
				m.selectedFile--
			}

		case tea.KeyEnter:
			if m.stage == "peers" && len(m.peers) > 0 {
				m.status = "📂 Select a file to send"
				m.stage = "files"
				m.selectedFile = 0
			} else if m.stage == "files" && len(m.files) > 0 {
				m.status = "📡 Sending file: " + m.files[m.selectedFile] + " to " + m.peers[m.selectedPeer]
				go sendFile(m.files[m.selectedFile], m.peers[m.selectedPeer])
			}
		}

	case []string:
		if len(msg) == 0 {
			m.status = errorStyle.Render("❌ No peers found.")
		} else {
			m.peers = msg
			m.status = statusStyle.Render("✅ Peers found! Select one.")
			m.selectedPeer = 0
		}
	}

	return m, nil
}

func (m model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("🔗 P2P File Sharing") + "\n\n")
	b.WriteString(boxStyle.Render(m.status) + "\n\n")

	if m.stage == "peers" {
		b.WriteString("🌍 Select a Peer:\n")
		for i, peer := range m.peers {
			if i == m.selectedPeer {
				b.WriteString(selectedStyle.Render("👉 "+peer) + "\n")
			} else {
				b.WriteString(peerStyle.Render("• "+peer) + "\n")
			}
		}
	} else if m.stage == "files" {
		b.WriteString("📂 Select a File:\n")
		for i, file := range m.files {
			if i == m.selectedFile {
				b.WriteString(selectedStyle.Render("👉 "+file) + "\n")
			} else {
				b.WriteString(peerStyle.Render("• "+file) + "\n")
			}
		}
	}

	b.WriteString(footerStyle.Render("\n↑↓ to navigate, Enter to select, 'q' to quit."))

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

	broadcastAddr := &net.UDPAddr{IP: net.IPv4bcast, Port: 9876}
	_, err = conn.WriteTo([]byte("DISCOVER_PEER"), broadcastAddr)
	if err != nil {
		return []string{"Error sending broadcast: " + err.Error()}
	}

	go func() {
		for {
			n, addr, err := conn.ReadFrom(buf)
			if err == nil {
				message := string(buf[:n])
				if message == "DISCOVER_PEER" {
					conn.WriteTo([]byte("PEER_RESPONSE"), addr)
				} else if message == "PEER_RESPONSE" {
					peers[addr.String()] = struct{}{}
				}
			}
		}
	}()

	time.Sleep(2 * time.Second)

	peerList := []string{}
	for peer := range peers {
		peerList = append(peerList, peer)
	}

	return peerList
}

func getFiles() []string {
	files := []string{}
	entries, err := os.ReadDir(".")
	if err != nil {
		return []string{"Error: Unable to read directory"}
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	return files
}

func startServer() {
	listener, err := net.Listen("tcp", ":9000")
	if err != nil {
		fmt.Println("Error starting TCP server:", err)
		return
	}
	defer listener.Close()

	fmt.Println("📡 Listening for incoming files...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Connection error:", err)
			continue
		}
		go receiveFile(conn)
	}
}

func sendFile(filename, peer string) {
	host, _, err := net.SplitHostPort(peer)
	if err != nil {
		fmt.Println(errorStyle.Render("❌ Invalid peer address:", peer))
		return
	}

	conn, err := net.Dial("tcp", host+":9000")
	if err != nil {
		fmt.Println(errorStyle.Render("❌ Error connecting to peer:", err.Error()))
		return
	}
	defer conn.Close()

	file, err := os.Open(filename)
	if err != nil {
		fmt.Println(errorStyle.Render("❌ Error opening file:", err.Error()))
		return
	}
	defer file.Close()

	_, err = io.Copy(conn, file)
	if err != nil {
		fmt.Println(errorStyle.Render("❌ Error sending file:", err.Error()))
		return
	}

	fmt.Println(statusStyle.Render("✅ File sent successfully!"))
}

func receiveFile(conn net.Conn) {
	defer conn.Close()
	file, err := os.Create("received_file")
	if err != nil {
		fmt.Println("❌ Error creating file:", err)
		return
	}
	defer file.Close()

	_, err = io.Copy(file, conn)
	if err != nil {
		fmt.Println("❌ Error receiving file:", err)
		return
	}

	fmt.Println("✅ File received successfully!")
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	go startServer()
}
