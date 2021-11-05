package main

import (
	"errors"
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

func OpenInEditor(filename string) tea.Cmd {

	return func() tea.Msg {
		editorPath := os.Getenv("EDITOR")
		if editorPath == "" {
			return errors.New("$EDITOR not set")
		}
		editorCmd := exec.Command(editorPath, filePath+"/"+filename)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr

		err := editorCmd.Start()
		err = editorCmd.Wait()
		return err
	}

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
	return docStyle.Render(m.list.View())
}
