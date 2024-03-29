// Package main is the entry point for the Mentat application.
// Mentat is a note taking application that uses markdown files as the backend.
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

type item struct {
	title, desc string
}

type editorFinishedMsg struct{ err error }

type previewFileContentsMsg struct {
	title    string
	contents string
}

// TODO: find another way to get the term width?
// tea.WindowSizeMsg ?

type updatedListMsg struct{ items []list.Item }

// TODO: determine if this is needed
type doneWithEditorMsg struct{}

// type errMsg error

var docStyle = lipgloss.NewStyle()

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

// TODO: find a way to return just the title, so I can filter
// on title.desc
// func (i item) FilterValue() string { return i.title + i.desc }

type model struct {
	list          list.Model
	selectedTitle string
	filePreview   string
	err           error
}

// TODO: figure out why this is needed
// If I don't have this, it does't have a title to look at on start up
func initialModel() model {
	items := []list.Item{
		item{title: "", desc: ""},
	}
	return model{list: list.NewModel(items, list.NewDefaultDelegate(), 0, 0)}
}

func main() {
	if len(os.Getenv("MENTAT_DEBUG")) > 0 {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer f.Close()
	}

	viper.SetConfigName(".mentat") // name of config file (without extension)
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME")
	viper.SetDefault("filePath", "~/notes")

	err := viper.ReadInConfig()
	if err != nil { // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
	// Initialize
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func (m model) Init() tea.Cmd {
	return getUpdatedFiles
}

func getMarkdownNames() []list.Item {
	filePath := viper.GetString("filePath")
	files, err := os.ReadDir(filePath)
	sort.Slice(files, func(i, j int) bool {
		infoI, err := files[i].Info()
		if err != nil {
			log.Fatal(err)
		}
		infoJ, err := files[j].Info()
		if err != nil {
			log.Fatal(err)
		}
		return infoJ.ModTime().Before(infoI.ModTime())
	})
	if err != nil {
		log.Println(err)
	}
	items := []list.Item{}
	for _, file := range files {
		// not a hidden file
		if !strings.HasPrefix(file.Name(), ".") {
			if strings.HasSuffix(file.Name(), ".md") {
				newItem := item{title: file.Name()}
				items = append(items, newItem)
			}
		}
	}
	return items
}

func getUpdatedFiles() tea.Msg {
	var listMsg updatedListMsg
	viper.SetDefault("filePath", "~/notes")
	listMsg.items = getMarkdownNames()
	return updatedListMsg(listMsg)
}

func openInEditor(filename string) tea.Cmd {
	editorPath := os.Getenv("EDITOR")
	if editorPath == "" {
		editorPath = "vim"
	}
	filePath := viper.GetString("filePath")
	editorCmd := exec.Command(editorPath, filePath+"/"+filename)

	return tea.ExecProcess(editorCmd, func(err error) tea.Msg {
		return editorFinishedMsg{err}
	})

}

// getFilePreview() gets a preview of the file contents of the current file selected in the list
func getFilePreview(filename string) tea.Cmd {
	if filename == "" {
		return func() tea.Msg {
			return previewFileContentsMsg{title: "No file selected", contents: ""}
		}
	}
	lines := []string{}
	filePath := viper.GetString("filePath")
	fileName := filePath + filename
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println("fatal:", err)
	}
	defer file.Close()
	lineCounter := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// only want to preview 20 lines
		if lineCounter > 10 {
			break
		}
		line := scanner.Text()
		lines = append(lines, line)
		lineCounter++
	}
	contents := strings.Join(lines, "\n")

	if err != nil {
		return func() tea.Msg {
			return previewFileContentsMsg{title: fileName, contents: "Error reading file"}
		}
	}
	// TODO: only show first N lines, so it doesn't break on large files
	return func() tea.Msg {
		return previewFileContentsMsg{title: fileName, contents: contents}
	}
}

// TODO: handle pressing space when searching? doesn't seem to to work
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {

	case previewFileContentsMsg:
		m.filePreview = msg.contents
		m.selectedTitle = msg.title
		return m, nil

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
			return m, openInEditor(fileName)
		}
	case tea.WindowSizeMsg:
		w, h, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			panic(err)
		}
		docStyle.Width(w).Height(h)
		m.list.SetSize(int(float64(w/3)), h-4)
	}

	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)
	item := m.list.SelectedItem()
	if item != nil {
		fileName := item.FilterValue()
		cmd = getFilePreview(fileName)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.err != nil {
		log.Fatal(m.err)
		return m.err.Error()
	}
	displayString := m.list.View()
	// TODO: handle long filenames without breaking the view
	var listStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("63"))

	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		panic(err)
	}
	previewHeight := m.list.Height()
	previewWidth := float64(w * 2 / 3)

	// TODO: styling for preview
	var previewStyle = lipgloss.NewStyle().Width(int(previewWidth)).Height(previewHeight).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("43"))

		// TODO: styling for title
	previewString := previewStyle.Render(m.selectedTitle + "\n" + m.filePreview)
	listString := listStyle.Render(displayString)
	allString := lipgloss.JoinHorizontal(lipgloss.Bottom, listString, previewString)

	return docStyle.Render(allString)
}
