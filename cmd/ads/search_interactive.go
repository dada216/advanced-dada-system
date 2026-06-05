package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/advanced-dada-system/ads/internal/ansi"
	"github.com/advanced-dada-system/ads/internal/search"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var searchInteractiveCmd = &cobra.Command{
	Use:   "search-interactive",
	Short: "Interactive TUI for federated search",
	Run: func(cmd *cobra.Command, args []string) {
		p := tea.NewProgram(initialModel(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Alas, there's been an error: %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(searchInteractiveCmd)
}

type model struct {
	textInput textinput.Model
	results   []search.Result
	err       error
	width     int
	height    int
	cursor    int
}

type searchResultMsg []search.Result
type errMsg error

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Type a keyword..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50

	return model{
		textInput: ti,
		results:   nil,
		err:       nil,
		cursor:    0,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func doSearch(query string) tea.Cmd {
	return func() tea.Msg {
		if query == "" {
			return searchResultMsg(nil)
		}
		res, err := search.Query(query)
		if err != nil {
			return errMsg(err)
		}
		return searchResultMsg(res)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if len(m.results) > 0 && m.cursor < len(m.results) {
				selected := cleanSnippet(m.results[m.cursor].Snippet)
				selected = ansi.Strip([]byte(selected))
				cmd := exec.Command("tmux", "-L", "ads", "set-buffer", selected)
				_ = cmd.Run()

				displayStr := selected
				if len(displayStr) > 40 {
					displayStr = displayStr[:37] + "..."
				}
				msgCmd := exec.Command("tmux", "-L", "ads", "display-message", "-d", "2000", fmt.Sprintf("ADS Copied: %s", displayStr))
				_ = msgCmd.Run()
			}
			return m, tea.Quit
		case "up", "ctrl+p", "ctrl+k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "ctrl+n", "ctrl+j":
			if m.cursor < len(m.results)-1 {
				m.cursor++
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case searchResultMsg:
		m.results = msg
		m.cursor = 0 // Reset cursor on new search
		return m, nil

	case errMsg:
		m.err = msg
		return m, nil
	}

	oldVal := m.textInput.Value()
	m.textInput, cmd = m.textInput.Update(msg)

	if m.textInput.Value() != oldVal {
		return m, tea.Batch(cmd, doSearch(m.textInput.Value()))
	}

	return m, cmd
}

var (
	titleStyle   = lipgloss.NewStyle().Background(lipgloss.Color("2")).Foreground(lipgloss.Color("0")).Bold(true)
	sessionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	rowStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	cursorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
)

func cleanSnippet(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\t", "    ")
	s = strings.ReplaceAll(s, "\b", "")
	return s
}

func (m model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(" ADS Interactive Search "))
	b.WriteString("\n\n")
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(fmt.Sprintf("Error: %v\n", m.err))
	} else if len(m.results) == 0 && m.textInput.Value() != "" {
		b.WriteString("No matches found.\n")
	} else {
		maxDisplay := m.height - 6
		if maxDisplay < 0 {
			maxDisplay = 0
		}
		for i, r := range m.results {
			if i >= maxDisplay {
				b.WriteString(fmt.Sprintf("\n...and %d more hidden\n", len(m.results)-maxDisplay))
				break
			}

			cursor := "  "
			if m.cursor == i {
				cursor = cursorStyle.Render("> ")
			}

			cleanedSnippet := cleanSnippet(r.Snippet)

			rightContent := rowStyle.Render(fmt.Sprintf("(row %d)", r.RowID))
			availableLeft := m.width - lipgloss.Width(rightContent) - 1
			if availableLeft < 10 {
				availableLeft = 10
			}

			leftContent := fmt.Sprintf("%s%s %s",
				cursor,
				sessionStyle.Render(fmt.Sprintf("[%s]", r.SessionName)),
				cleanedSnippet)

			leftStyle := lipgloss.NewStyle().MaxWidth(availableLeft)
			truncatedLeft := leftStyle.Render(leftContent)

			padding := m.width - lipgloss.Width(truncatedLeft) - lipgloss.Width(rightContent)
			if padding < 0 {
				padding = 0
			}

			line := truncatedLeft + strings.Repeat(" ", padding) + rightContent

			if m.cursor == i {
				line = strings.ReplaceAll(line, "\033[0m", "\033[0m\033[48;5;22m")
				line = "\033[48;5;22m" + line + "\033[0m"
			}

			b.WriteString(line + "\n")
		}
	}

	b.WriteString("\n")
	return b.String()
}
