package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

// Style definitions for password screen
var (
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}

	headerStyle = lipgloss.NewStyle().
			Foreground(highlight).
			Underline(true).
			MarginBottom(1)
)

// App screens
const (
	listScreen = iota
	passwordScreen
	spinnerScreen
)

type hostItem struct {
	host string
	desc string // user@ip, ip, or empty
}

func (i hostItem) Title() string       { return i.host }
func (i hostItem) Description() string { return i.desc }
func (i hostItem) FilterValue() string { return i.host }

type loginResultMsg struct {
	success bool
	err     error
}

// ListKeyMap defines the key bindings for the main list screen
type ListKeyMap struct {
	Enter  key.Binding
	Delete key.Binding
}

func (k ListKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Enter, k.Delete}
}

func (k ListKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Enter, k.Delete}}
}

// PasswordKeyMap defines the key bindings for the password screen
type PasswordKeyMap struct {
	Esc key.Binding
}

func (k PasswordKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Esc}
}

func (k PasswordKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Esc}}
}

type model struct {
	list         list.Model
	selectedHost string
	selectedDesc string
	screen       int
	password     string
	pwInput      textinput.Model
	errMsg       string
	spinner      spinner.Model
	loggingIn    bool
	shouldSSH    bool // NEW: set to true after successful login
	help         help.Model
	listKeys     ListKeyMap
	keys         PasswordKeyMap
	infoBox      string // Info box content for hovered host
}

func initialModel(items []list.Item) *model {
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "SSH Hosts"

	pw := textinput.New()
	pw.EchoMode = textinput.EchoPassword
	pw.EchoCharacter = '•'
	pw.Focus()

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	listKeys := ListKeyMap{
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "connect"),
		),
		Delete: key.NewBinding(
			key.WithKeys("delete", "x"),
			key.WithHelp("x", "remove host"),
		),
	}

	keys := PasswordKeyMap{
		Esc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "go back"),
		),
	}

	return &model{
		list:     l,
		screen:   listScreen,
		pwInput:  pw,
		spinner:  s,
		help:     help.New(),
		listKeys: listKeys,
		keys:     keys,
		infoBox:  "hello world",
	}
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case listScreen:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "enter":
				selected, ok := m.list.SelectedItem().(hostItem)
				if ok {
					m.selectedHost = selected.host
					m.selectedDesc = selected.desc
					m.pwInput.SetValue("")
					m.errMsg = ""
					m.screen = passwordScreen
					return m, nil
				}
			case "delete", "x":
				selected, ok := m.list.SelectedItem().(hostItem)
				if ok {
					// Delete the host from SSH config
					if err := deleteHostFromConfig(selected.host); err != nil {
						// Could show error message here if needed
						return m, nil
					}
					// Reload the list
					usr, _ := user.Current()
					sshConfigPath := filepath.Join(usr.HomeDir, ".ssh", "config")
					if hosts, err := parseSSHConfig(sshConfigPath); err == nil {
						items := make([]list.Item, len(hosts))
						for i, h := range hosts {
							items[i] = h
						}
						m.list.SetItems(items)
					}
					return m, nil
				}
			}
		case tea.WindowSizeMsg:
			h, v := docStyle.GetFrameSize()
			// Reserve space for info box (60 chars + 2 spaces)
			m.list.SetSize(msg.Width-h-62, msg.Height-v)
		}

		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)

		// Update info box content after list update
		if m.list.Index() < len(m.list.Items()) {
			if selected, ok := m.list.Items()[m.list.Index()].(hostItem); ok {
				hostInfo := getHostInfo(selected.host)
				m.infoBox = hostInfo
			}
		}

		return m, cmd
	case passwordScreen:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.screen = listScreen
				m.errMsg = ""
				return m, nil
			case "enter":
				m.password = m.pwInput.Value()
				m.errMsg = ""
				m.screen = spinnerScreen
				m.loggingIn = true
				return m, tea.Batch(m.spinner.Tick, tryLogin(m.selectedHost, m.password))
			}
		}
		var cmd tea.Cmd
		m.pwInput, cmd = m.pwInput.Update(msg)
		return m, cmd
	case spinnerScreen:
		switch msg := msg.(type) {
		case loginResultMsg:
			m.loggingIn = false
			if msg.success {
				// Success: set flag and quit TUI
				m.shouldSSH = true
				return m, tea.Quit
			} else {
				// Failure: go back to password input with error
				m.screen = passwordScreen
				m.errMsg = "Login failed: wrong password or SSH error."
				m.pwInput.SetValue("")
				return m, nil
			}
		default:
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func tryLogin(host, password string) tea.Cmd {
	return func() tea.Msg {
		// Try to SSH with sshpass and a quick command (exit)
		cmd := exec.Command("sshpass", "-p", password, "ssh", "-o", "StrictHostKeyChecking=no", "-o", "BatchMode=no", host, "exit")
		cmd.Stdin = nil
		cmd.Stdout = nil
		cmd.Stderr = nil
		err := cmd.Run()
		if err == nil {
			return loginResultMsg{success: true}
		}
		return loginResultMsg{success: false, err: err}
	}
}

func (m *model) passwordHelpBar() string {
	// Use the same style as the main list view's help text
	helpStyle := m.list.Styles.HelpStyle
	return helpStyle.Render("    esc    go back")
}

func (m *model) View() string {
	switch m.screen {
	case listScreen:
		// Create info box style
		infoBoxStyle := lipgloss.NewStyle().
			Width(60).
			Height(10).
			Align(lipgloss.Left, lipgloss.Top).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("69")).
			Padding(1, 1)

		// Create the info box content
		infoBox := infoBoxStyle.Render(m.infoBox)

		// Join list and info box horizontally
		content := lipgloss.JoinHorizontal(lipgloss.Top, m.list.View(), "  ", infoBox)

		var b strings.Builder
		b.WriteString(content)
		b.WriteString("\n")
		b.WriteString(m.help.View(m.listKeys))
		return docStyle.Render(b.String())
	case passwordScreen:
		var b strings.Builder

		// Styled header with host name
		header := headerStyle.Render(m.selectedHost)
		b.WriteString(header)
		b.WriteString("\n")

		// Error message if any
		if m.errMsg != "" {
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(m.errMsg))
			b.WriteString("\n\n")
		}

		// "Enter password:" text styled like help text
		helpStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
			Light: "#B2B2B2",
			Dark:  "#4A4A4A",
		})
		b.WriteString(helpStyle.Render("enter password:"))
		b.WriteString("\n")

		// Password input field
		b.WriteString(m.pwInput.View())
		b.WriteString("\n\n")

		// Help bar using the same system as the main list view
		b.WriteString(m.help.View(m.keys))
		return docStyle.Render(b.String())
	case spinnerScreen:
		var b strings.Builder
		b.WriteString("\n\n   ")
		b.WriteString(m.spinner.View())
		b.WriteString(" Logging in...")
		return docStyle.Render(b.String())
	}
	return ""
}

// parseSSHConfig parses the SSH config and returns hostItems with host and user@ip/ip as desc if available.
func parseSSHConfig(path string) ([]hostItem, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var items []hostItem
	var currentHosts []string
	var currentHostname string
	var currentUser string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(strings.ToLower(line), "host ") {
			// If we have a previous host group, add them
			if len(currentHosts) > 0 {
				for _, h := range currentHosts {
					if strings.ContainsAny(h, "*?[]!") {
						continue // skip wildcards
					}
					desc := ""
					if currentHostname != "" && currentUser != "" {
						desc = currentUser + "@" + currentHostname
					} else if currentHostname != "" {
						desc = currentHostname
					}
					items = append(items, hostItem{host: h, desc: desc})
				}
			}
			fields := strings.Fields(line)
			currentHosts = fields[1:]
			currentHostname = ""
			currentUser = ""
			continue
		}
		if len(currentHosts) > 0 {
			if strings.HasPrefix(strings.ToLower(line), "hostname ") {
				parts := strings.Fields(line)
				if len(parts) > 1 {
					currentHostname = parts[1]
				}
			}
			if strings.HasPrefix(strings.ToLower(line), "user ") {
				parts := strings.Fields(line)
				if len(parts) > 1 {
					currentUser = parts[1]
				}
			}
		}
	}
	// Add the last group
	if len(currentHosts) > 0 {
		for _, h := range currentHosts {
			if strings.ContainsAny(h, "*?[]!") {
				continue // skip wildcards
			}
			desc := ""
			if currentHostname != "" && currentUser != "" {
				desc = currentUser + "@" + currentHostname
			} else if currentHostname != "" {
				desc = currentHostname
			}
			items = append(items, hostItem{host: h, desc: desc})
		}
	}
	return items, scanner.Err()
}

// deleteHostFromConfig removes a host entry from the SSH config file
func deleteHostFromConfig(hostToDelete string) error {
	usr, err := user.Current()
	if err != nil {
		return err
	}

	configPath := filepath.Join(usr.HomeDir, ".ssh", "config")

	// Read the entire config file
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	var inHostBlock bool
	var currentHosts []string
	var skipBlock bool

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if strings.HasPrefix(strings.ToLower(trimmedLine), "host ") {
			// Check if this host block contains our target
			fields := strings.Fields(trimmedLine)
			currentHosts = fields[1:]

			// If this block contains our target, mark it for skipping
			if contains(currentHosts, hostToDelete) {
				skipBlock = true
				continue
			} else {
				skipBlock = false
				inHostBlock = true
				newLines = append(newLines, line)
			}
			continue
		}

		// If we're skipping this block, don't add any lines
		if skipBlock {
			// If this line is not indented, we're out of the host block
			if len(line) > 0 && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
				skipBlock = false
				inHostBlock = false
				newLines = append(newLines, line)
			}
			continue
		}

		// If this line is not indented, we're out of the host block
		if inHostBlock && len(line) > 0 && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			inHostBlock = false
		}

		newLines = append(newLines, line)
	}

	// Write the modified content back to the file
	newContent := strings.Join(newLines, "\n")
	return os.WriteFile(configPath, []byte(newContent), 0644)
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// getHostInfo extracts all SSH config information for a specific host
func getHostInfo(hostName string) string {
	usr, err := user.Current()
	if err != nil {
		return "Error: Could not get user info"
	}

	configPath := filepath.Join(usr.HomeDir, ".ssh", "config")
	content, err := os.ReadFile(configPath)
	if err != nil {
		return "Error: Could not read SSH config"
	}

	lines := strings.Split(string(content), "\n")

	// Find the selected host and its ProxyJump
	selectedHostInfo := getHostBlock(lines, hostName)
	if selectedHostInfo == nil {
		return fmt.Sprintf("Host: %s\n\nNo config found", hostName)
	}

	// Find jump host if this host uses ProxyJump
	var jumpHostInfo *hostBlock
	if jumpHost := getProxyJumpHost(selectedHostInfo.lines); jumpHost != "" {
		jumpHostInfo = getHostBlock(lines, jumpHost)
	}

	// Find hosts that jump through this host
	var jumpingHosts []*hostBlock
	for _, block := range getAllHostBlocks(lines) {
		if getProxyJumpHost(block.lines) == hostName {
			jumpingHosts = append(jumpingHosts, block)
		}
	}

	// Format the information
	var result strings.Builder

	// Show jump host (if this host uses ProxyJump)
	if jumpHostInfo != nil {
		result.WriteString(fmt.Sprintf("Jump Host: %s\n", jumpHostInfo.hostName))
		result.WriteString(strings.Repeat("─", 20) + "\n")
		for _, line := range jumpHostInfo.lines {
			if strings.TrimSpace(line) != "" && !strings.HasPrefix(strings.TrimSpace(line), "Host ") {
				result.WriteString(line + "\n")
			}
		}
		result.WriteString("\n")
	}

	// Show selected host
	result.WriteString(fmt.Sprintf("Host: %s\n", hostName))
	result.WriteString(strings.Repeat("─", 20) + "\n")
	for _, line := range selectedHostInfo.lines {
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(strings.TrimSpace(line), "Host ") {
			result.WriteString(line + "\n")
		}
	}

	// Show hosts that jump through this host
	if len(jumpingHosts) > 0 {
		result.WriteString("\n")
		result.WriteString("Jumped by:\n")
		result.WriteString(strings.Repeat("─", 20) + "\n")
		for _, block := range jumpingHosts {
			result.WriteString(fmt.Sprintf("Host: %s\n", block.hostName))
			for _, line := range block.lines {
				if strings.TrimSpace(line) != "" && !strings.HasPrefix(strings.TrimSpace(line), "Host ") {
					result.WriteString(line + "\n")
				}
			}
			result.WriteString("\n")
		}
	}

	return result.String()
}

// hostBlock represents a host configuration block
type hostBlock struct {
	hostName string
	lines    []string
}

// getHostBlock extracts a host block from the config lines
func getHostBlock(lines []string, hostName string) *hostBlock {
	var hostLines []string
	var inHostBlock bool
	var currentHosts []string
	var foundHostName string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if strings.HasPrefix(strings.ToLower(trimmedLine), "host ") {
			// Check if this host block contains our target
			fields := strings.Fields(trimmedLine)
			currentHosts = fields[1:]

			if contains(currentHosts, hostName) {
				inHostBlock = true
				foundHostName = hostName
				hostLines = append(hostLines, line)
				continue
			} else {
				inHostBlock = false
				continue
			}
		}

		// If we're in the target host block, collect all lines
		if inHostBlock {
			// If this line is not indented, we're out of the host block
			if len(line) > 0 && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
				inHostBlock = false
				break
			}
			hostLines = append(hostLines, line)
		}
	}

	if len(hostLines) == 0 {
		return nil
	}

	return &hostBlock{
		hostName: foundHostName,
		lines:    hostLines,
	}
}

// getAllHostBlocks extracts all host blocks from the config
func getAllHostBlocks(lines []string) []*hostBlock {
	var blocks []*hostBlock
	var currentBlock *hostBlock
	var inHostBlock bool

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if strings.HasPrefix(strings.ToLower(trimmedLine), "host ") {
			// Save previous block if exists
			if currentBlock != nil {
				blocks = append(blocks, currentBlock)
			}

			// Start new block
			fields := strings.Fields(trimmedLine)
			currentHosts := fields[1:]
			if len(currentHosts) > 0 {
				currentBlock = &hostBlock{
					hostName: currentHosts[0], // Use first host name
					lines:    []string{line},
				}
				inHostBlock = true
			}
			continue
		}

		// Add line to current block
		if inHostBlock && currentBlock != nil {
			// If this line is not indented, we're out of the host block
			if len(line) > 0 && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
				inHostBlock = false
				blocks = append(blocks, currentBlock)
				currentBlock = nil
			} else {
				currentBlock.lines = append(currentBlock.lines, line)
			}
		}
	}

	// Add the last block
	if currentBlock != nil {
		blocks = append(blocks, currentBlock)
	}

	return blocks
}

// getProxyJumpHost extracts the ProxyJump host from host lines
func getProxyJumpHost(lines []string) string {
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(trimmedLine), "proxyjump ") {
			fields := strings.Fields(trimmedLine)
			if len(fields) > 1 {
				return fields[1]
			}
		}
	}
	return ""
}

func checkSshpass() {
	_, err := exec.LookPath("sshpass")
	if err == nil {
		return
	}
	fmt.Println("Error: sshpass is not installed.")
	fmt.Println()
	fmt.Println("This app requires sshpass to provide passwords to ssh non-interactively.")
	fmt.Println()
	if runtime.GOOS == "darwin" {
		fmt.Println("Install it with:")
		fmt.Println("  brew install hudochenkov/sshpass/sshpass")
	} else if runtime.GOOS == "linux" {
		fmt.Println("Install it with:")
		fmt.Println("  sudo apt install sshpass")
	} else {
		fmt.Println("Please install sshpass for your platform.")
	}
	os.Exit(1)
}

func main() {
	checkSshpass()
	usr, err := user.Current()
	if err != nil {
		fmt.Println("Could not get current user:", err)
		os.Exit(1)
	}
	sshConfigPath := filepath.Join(usr.HomeDir, ".ssh", "config")
	parsed, err := parseSSHConfig(sshConfigPath)
	if err != nil {
		fmt.Println("Could not parse ~/.ssh/config:", err)
		os.Exit(1)
	}
	if len(parsed) == 0 {
		fmt.Println("No hosts found in ~/.ssh/config")
		os.Exit(0)
	}

	items := make([]list.Item, len(parsed))
	for i, it := range parsed {
		items[i] = it
	}

	m := initialModel(items)
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	// After TUI exits, if login was successful, run SSH
	if m.shouldSSH && m.selectedHost != "" && m.password != "" {
		cmd := exec.Command("sshpass", "-p", m.password, "ssh", "-t", m.selectedHost, "env TERM=xterm-256color bash --login")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}
}
