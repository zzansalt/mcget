package models

import (
	"fmt"
	"strings"

	util "mcget/utils"

	tea "charm.land/bubbletea/v2"
)

type versionsMsg []string

type versionModel struct {
	choices    []string
	cursor     int
	loading    bool
	state      *AppState
	err        error
	filterText string
	filterMode bool
}

func NewVersionModel(state *AppState) versionModel {
	return versionModel{
		state:   state,
		loading: true,
	}
}

func fetchPaperVersionsCmd() tea.Cmd {
	return func() tea.Msg {
		v, err := util.GetPaperVersions()
		if err != nil {
			return errMsg{err}
		}
		return versionsMsg(v)
	}
}

func fetchServerVersionsCmd(server string) tea.Cmd {
	return func() tea.Msg {
		v, err := util.GetServerVersions(server)
		if err != nil {
			return errMsg{err}
		}
		return versionsMsg(v)
	}
}

func fetchVelocityVersionsCmd() tea.Cmd {
	return func() tea.Msg {
		v, err := util.GetVelocityVersions()
		if err != nil {
			return errMsg{err}
		}
		return versionsMsg(v)
	}
}

func (m versionModel) Init() tea.Cmd {
	return fetchServerVersionsCmd(m.state.Bukkit)
}

func (m versionModel) getFilteredChoices() []string {
	if m.filterText == "" {
		return m.choices
	}
	var filtered []string
	lowerFilter := strings.ToLower(m.filterText)
	for _, choice := range m.choices {
		if choice == "<" {
			continue
		}
		if strings.Contains(strings.ToLower(choice), lowerFilter) {
			filtered = append(filtered, choice)
		}
	}
	filtered = append(filtered, "<")
	return filtered
}

func (m *versionModel) clampCursor(filteredLen int) {
	if m.cursor < 0 {
		m.cursor = 0
	}
	if filteredLen > 0 && m.cursor >= filteredLen {
		m.cursor = filteredLen - 1
	}
}

func (m versionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case versionsMsg:
		m.choices = msg
		m.choices = append(m.choices, "<")
		m.loading = false
		m.cursor = 0
		return m, nil

	case errMsg:
		m.loading = false
		m.err = msg.err
		m.choices = []string{"<"}
		m.cursor = 0
		return m, nil

	case tea.KeyPressMsg:
		if m.err != nil {
			switch msg.String() {
			case "enter", "space", "q", "ctrl+c":
				newModel := NewBukkitModel(m.state)
				return newModel, newModel.Init()
			}
			return m, nil
		}

		if m.loading || len(m.choices) == 0 {
			return m, nil
		}

		filtered := m.getFilteredChoices()

		if m.filterMode {
			key := msg.String()
			switch key {
			case "enter":
				m.filterMode = false
			case "esc":
				m.filterMode = false
				m.filterText = ""
				m.cursor = 0
			case "backspace":
				if len(m.filterText) > 0 {
					m.filterText = m.filterText[:len(m.filterText)-1]
					m.cursor = 0
				}
			case "space":
				m.filterText += " "
				m.cursor = 0
			default:
				if len(key) == 1 {
					m.filterText += key
					m.cursor = 0
				}
			}
			m.clampCursor(len(m.getFilteredChoices()))
			return m, nil
		}

		switch msg.String() {

		case "/":
			m.filterMode = true
			m.cursor = 0
			return m, nil

		case "q", "ctrl+c":
			newModel := NewBukkitModel(m.state)
			return newModel, newModel.Init()

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(filtered)-1 {
				m.cursor++
			}

		case "enter":
			if len(filtered) == 0 {
				return m, nil
			}
			selected := filtered[m.cursor]
			if selected == "<" {
				newModel := NewBukkitModel(m.state)
				return newModel, newModel.Init()
			}

			m.state.Version = selected

			newModel := NewBuildModel(m.state)
			return newModel, newModel.Init()
		}
	}

	return m, nil
}

func (m versionModel) View() tea.View {
	if m.loading {
		return tea.NewView(tr(m.state, "versions.loading") + "\n")
	}

	if m.err != nil {
		return tea.NewView(fmt.Sprintf(tr(m.state, "versions.error"), m.err))
	}

	titleKey := "versions.paper_title"
	if m.state.Bukkit == "velocity" {
		titleKey = "versions.velocity_title"
	}
	s := tr(m.state, titleKey) + "\n"
	if m.filterText != "" {
		s += fmt.Sprintf(tr(m.state, "versions.filtering"), m.filterText) + "\n\n"
	} else {
		s += tr(m.state, "versions.search_hint") + "\n\n"
	}

	filtered := m.getFilteredChoices()

	if len(filtered) == 1 && filtered[0] == "<" {
		s += "  " + tr(m.state, "versions.no_match") + "\n\n"
	}

	maxVisible := 10
	start := 0
	if len(filtered) > maxVisible {
		start = m.cursor - maxVisible/2
		if start < 0 {
			start = 0
		}
		if start+maxVisible > len(filtered) {
			start = len(filtered) - maxVisible
		}
	}
	end := start + maxVisible
	if end > len(filtered) {
		end = len(filtered)
	}

	if start > 0 {
		s += "  ▲ ...\n"
	}

	for i := start; i < end; i++ {
		choice := filtered[i]
		cursor := "○"
		if m.cursor == i {
			cursor = selectedStyle.Render("●")
		}
		s += fmt.Sprintf("%s %s\n", cursor, choice)
	}

	if end < len(filtered) {
		s += "  ▼ ...\n"
	}

	s += "\n"
	if m.filterMode {
		s += fmt.Sprintf(tr(m.state, "versions.search"), m.filterText) + "\n"
		s += "  " + tr(m.state, "versions.filter_hint") + "\n"
	} else {
		s += tr(m.state, "versions.back_hint") + "\n"
	}
	return tea.NewView(s)
}
