package models

import (
	"fmt"

	"charm.land/lipgloss/v2"

	tea "charm.land/bubbletea/v2"
)

var (
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))

	paperSelectStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#2761F5")).Bold(true)
	velocitySelectStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#27C8F5")).Bold(true)
	spigotSelectStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#f5cf27")).Bold(true)
	purpurSelectStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#a676ff")).Bold(true)
	forgeSelectStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#2c3172")).Bold(true)
	neoforgeSelectStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#fc8413")).Bold(true)
	fabricSelectStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#f5e7a9")).Bold(true)
)

type menuModel struct {
	choices []string
	cursor  int
	state   *AppState
}

func NewMenuModel(state *AppState) menuModel {
	return menuModel{
		state:   state,
		choices: []string{"download", "language", "quit"},
	}
}

func (m menuModel) Init() tea.Cmd {
	return nil
}

func (m menuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyPressMsg:

		switch msg.String() {

		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		case "enter", "space":
			switch m.choices[m.cursor] {
			case "download":
				newModel := NewBukkitModel(m.state)
				return newModel, newModel.Init()
			case "language":
				newModel := NewLanguageModel(m.state)
				return newModel, newModel.Init()
			case "quit":
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

func (m menuModel) View() tea.View {
	s := tr(m.state, "menu.title") + "\n\n"

	for i, choice := range m.choices {

		cursor := "○"
		if m.cursor == i {
			cursor = selectedStyle.Render("●")
		}

		label := tr(m.state, "menu."+choice)
		s += fmt.Sprintf("%s %s\n", cursor, label)
	}

	s += "\n" + tr(m.state, "menu.quit_hint") + "\n"

	return tea.NewView(s)
}
