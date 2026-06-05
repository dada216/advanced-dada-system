package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

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
				selected = strings.ReplaceAll(selected, "\033[31m", "")
				selected = strings.ReplaceAll(selected, "\033[0m", "")
				cmd := exec.Command("tmux", "-L", "ads", "set-buffer", selected)
				_ = cmd.Run()
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
	titleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	sessionStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	rowStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	selectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("236"))
)

func cleanSnippet(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
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
			lineStyle := lipgloss.NewStyle()
			if m.cursor == i {
				cursor = cursorStyle.Render("> ")
				lineStyle = selectedStyle
			}

			cleanedSnippet := cleanSnippet(r.Snippet)
			line := fmt.Sprintf("%s%s %s %s",
				cursor,
				sessionStyle.Render(fmt.Sprintf("[%s]", r.SessionName)),
				rowStyle.Render(fmt.Sprintf("(row %d):", r.RowID)),
				cleanedSnippet)

			b.WriteString(lineStyle.Render(line) + "\n")
		}
	}

	b.WriteString("\n")
	return b.String()
}
