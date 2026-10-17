package models

import (
	"fmt"
	"strings"

	util "mcget/utils"

	tea "charm.land/bubbletea/v2"
)

type buildsMsg []string

type errMsg struct {
	err error
}

type buildModel struct {
	choices    []string
	cursor     int
	loading    bool
	state      *AppState
	err        error
	filterText string
	filterMode bool
}

func NewBuildModel(state *AppState) buildModel {
	return buildModel{
		state:   state,
		loading: true,
	}
}

func fetchPaperBuildsCmd(version string) tea.Cmd {
	return func() tea.Msg {
		v, err := util.GetPaperBuilds(version)
		if err != nil {
			return errMsg{err}
		}
		return buildsMsg(v)
	}
}

func fetchServerBuildsCmd(server, version string) tea.Cmd {
	return func() tea.Msg {
		v, err := util.GetServerBuilds(server, version)
		if err != nil {
			return errMsg{err}
		}
		return buildsMsg(v)
	}
}

func fetchVelocityBuildsCmd(version string) tea.Cmd {
	return func() tea.Msg {
		v, err := util.GetVelocityBuilds(version)
		if err != nil {
			return errMsg{err}
		}
		return buildsMsg(v)
	}
}

func (m buildModel) Init() tea.Cmd {
	return fetchServerBuildsCmd(m.state.Bukkit, m.state.Version)
}

func (m buildModel) getFilteredChoices() []string {
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

func (m *buildModel) clampCursor(filteredLen int) {
	if m.cursor < 0 {
		m.cursor = 0
	}
	if filteredLen > 0 && m.cursor >= filteredLen {
		m.cursor = filteredLen - 1
	}
}

func (m buildModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case buildsMsg:
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
				newModel := NewVersionModel(m.state)
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
			newModel := NewVersionModel(m.state)
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
				newModel := NewVersionModel(m.state)
				return newModel, newModel.Init()
			}

			m.state.Build = selected

			newModel := NewLastCheckModel(m.state)
			return newModel, newModel.Init()
		}
	}

	return m, nil
}

func (m buildModel) View() tea.View {
	if m.loading {
		return tea.NewView(tr(m.state, "builds.loading") + "\n")
	}

	if m.err != nil {
		return tea.NewView(fmt.Sprintf(tr(m.state, "builds.error"), m.err))
	}

	s := fmt.Sprintf(tr(m.state, "builds.title"), m.state.Version) + "\n"
	if m.filterText != "" {
		s += fmt.Sprintf(tr(m.state, "builds.filtering"), m.filterText) + "\n\n"
	} else {
		s += tr(m.state, "builds.search_hint") + "\n\n"
	}

	filtered := m.getFilteredChoices()

	if len(filtered) == 1 && filtered[0] == "<" {
		s += "  " + tr(m.state, "builds.no_match") + "\n\n"
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
		s += fmt.Sprintf(tr(m.state, "builds.search"), m.filterText) + "\n"
		s += "  " + tr(m.state, "builds.filter_hint") + "\n"
	} else {
		s += tr(m.state, "builds.back_hint") + "\n"
	}
	return tea.NewView(s)
}
