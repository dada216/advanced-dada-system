package main

import (
	"fmt"
	"os"
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
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc, tea.KeyEnter:
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case searchResultMsg:
		m.results = msg
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
	titleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	sessionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	rowStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

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
				b.WriteString(fmt.Sprintf("\n...and %d more hidden due to terminal height\n", len(m.results)-maxDisplay))
				break
			}
			b.WriteString(fmt.Sprintf("%s %s %s\n",
				sessionStyle.Render(fmt.Sprintf("[%s]", r.SessionName)),
				rowStyle.Render(fmt.Sprintf("(row %d):", r.RowID)),
				r.Snippet))
		}
	}

	b.WriteString("\n")
	return b.String()
}
