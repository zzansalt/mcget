package models

import tea "charm.land/bubbletea/v2"

type languageModel struct {
	state  *AppState
	cursor int
}

func NewLanguageModel(state *AppState) languageModel {
	return languageModel{state: state}
}

func (m languageModel) Init() tea.Cmd { return nil }

func (m languageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "q", "esc", "ctrl+c":
			menu := NewMenuModel(m.state)
			return menu, menu.Init()
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < 1 {
				m.cursor++
			}
		case "enter", "space":
			if m.cursor == 0 {
				m.state.Language = "en"
			} else {
				m.state.Language = "ko"
			}
			menu := NewMenuModel(m.state)
			return menu, menu.Init()
		}
	}
	return m, nil
}

func (m languageModel) View() tea.View {
	choices := []string{tr(m.state, "language.english"), tr(m.state, "language.korean")}
	s := tr(m.state, "language.title") + "\n\n"
	for i, choice := range choices {
		cursor := "○"
		if i == m.cursor {
			cursor = selectedStyle.Render("●")
		}
		s += cursor + " " + choice + "\n"
	}
	return tea.NewView(s + "\n" + tr(m.state, "language.back_hint") + "\n")
}
