package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type item struct {
	title, desc string
}

type editorFinishedMsg struct{ err error }

// TODO: remove hardcoded path
const filePath = "/Users/samuel.otterman/Dropbox/notes"

// TODO: find another way to get the term width?
// tea.WindowSizeMsg ?

type updatedListMsg struct{ items []list.Item }

// TODO: determine if this is needed
type doneWithEditorMsg struct{}

type errMsg error

var docStyle = lipgloss.NewStyle().Margin(1, 0, 0, 0)

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

// TODO: find a way to return just the title, so I can filter
// on title.desc
// func (i item) FilterValue() string { return i.title + i.desc }

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
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if err := p.Start(); err != nil {
		log.Fatal(err)
	}
}

func (m model) Init() tea.Cmd {
	log.Println("INIT")
	return GetUpdatedFiles
}

func getMarkdownNames(filepath string) []list.Item {
	files, err := ioutil.ReadDir(filePath)
	sort.Slice(files, func(i, j int) bool {
		return files[j].ModTime().Before(files[i].ModTime())
	})
	if err != nil {
		log.Println(err)
	}
	items := []list.Item{}
	for _, file := range files {
		// not a hidden file
		if !strings.HasPrefix(file.Name(), ".") {
			if strings.HasSuffix(file.Name(), ".md") {
				var newItem item = item{title: file.Name()}
				items = append(items, newItem)
			}
		}
	}
	return items
}

func GetUpdatedFiles() tea.Msg {
	var listMsg updatedListMsg
	listMsg.items = getMarkdownNames(filePath)
	return updatedListMsg(listMsg)
}

func openEditor() tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	c := exec.Command(editor)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err}
	})
}

func OpenInEditor(filename string) tea.Cmd {
	editorPath := os.Getenv("EDITOR")
	if editorPath == "" {
		editorPath = "vim"
	}
	editorCmd := exec.Command(editorPath, filePath+"/"+filename)

	return tea.ExecProcess(editorCmd, func(err error) tea.Msg {
		return editorFinishedMsg{err}
	})

}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case doneWithEditorMsg:
		log.Println("doneWithEditorMsg")
		return m, nil

	case updatedListMsg:
		m.list.SetItems(msg.items)
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, nil
		}
		// TODO: periodically update list of files
		if msg.String() == "enter" {
			count := len(m.list.VisibleItems())
			if count == 0 {
				log.Print("Empty filter, should create item")
				return m, nil
			}
			log.Print("Nonzero items")
			item := m.list.SelectedItem()
			fileName := item.FilterValue()
			statusMessage := "Selected: " + string(item.FilterValue())

			m.list.NewStatusMessage(statusMessage)
			return m, OpenInEditor(fileName)
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
	displayString := filePath + "\n"
	displayString += m.list.View()
	return docStyle.Render(displayString)
}
