package main

import (
	"log"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type item struct {
	title, desc string
}

const filePath = "/Users/samuel.otterman/Dropbox/notes"

type updatedListMsg struct{ items []list.Item }

type errMsg error

var docStyle = lipgloss.NewStyle().Margin(1, 0, 0, 0)

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
		item{title: "baz.md", desc: "Fourth file"},
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
	return GetUpdatedFiles
}

func GetUpdatedFiles() tea.Msg {
	files, err := os.ReadDir(filePath)
	if err != nil {
		log.Println(err)
	}
	items := []list.Item{}
	for _, file := range files {
		// not a hidden file
		if !strings.HasPrefix(file.Name(), ".") {
			var newItem item = item{title: file.Name()}
			items = append(items, newItem)
		}
	}
	var listMsg updatedListMsg
	listMsg.items = items
	return updatedListMsg(listMsg)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case updatedListMsg:
		m.list.SetItems(msg.items)
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, nil
		}
		if msg.String() == "enter" {
			log.Print("Enter pressed!")
			count := len(m.list.VisibleItems())
			if count == 0 {
				log.Print("Empty filter, should create item")
				return m, nil
			}
			log.Print("Nonzero items")
			item := m.list.SelectedItem()
			statusMessage := "Selected: " + string(item.FilterValue())

			m.list.NewStatusMessage(statusMessage)
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
		log.Fatal(m.err)
		return m.err.Error()
	}
	return docStyle.Render(m.list.View())
}
