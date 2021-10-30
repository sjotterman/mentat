package main

import (
	"log"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type errMsg error

type item struct {
	title, desc string
}

var docStyle = lipgloss.NewStyle().Margin(1, 2)

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type model struct {
	quitting bool
	list     list.Model
	err      error
}

func initialModel() model {
	items := []list.Item{
		item{title: "foo.md", desc: "First file"},
		item{title: "bar.md", desc: "Second file"},
		item{title: "foobar.md", desc: "Third file"},
	}
	return model{list: list.NewModel(items, list.NewDefaultDelegate(), 0, 0)}
}

func main() {
	// Log to a file. To use, export BUBBLETEA_LOG
	logfilePath := os.Getenv("BUBBLETEA_LOG")
	if logfilePath != "" {
		if _, err := tea.LogToFile(logfilePath, "simple"); err != nil {
			log.Fatal(err)
		}
	}

	// Initialize
	p := tea.NewProgram(initialModel())
	p.EnterAltScreen()
	if err := p.Start(); err != nil {
		log.Fatal(err)
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, nil
		}
	case tea.WindowSizeMsg:
		top, right, bottom, left := docStyle.GetMargin()
		m.list.SetSize(msg.Width-left-right, msg.Height-top-bottom)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.err != nil {
		return m.err.Error()
	}
	return docStyle.Render(m.list.View())
}
