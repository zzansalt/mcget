package models

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

type AppState struct {
	Language string
	Bukkit   string
	Version  string
	Build    string
}

type bukkitModel struct {
	choices []Item
	cursor  int
	state   *AppState
}

type Item struct {
	Label string
	Value string
}

func NewBukkitModel(state *AppState) bukkitModel {
	return bukkitModel{
		state: state,
		choices: []Item{
			{Label: "Paper", Value: "paper"},
			{Label: "Velocity", Value: "velocity"},
			{Label: "Spigot", Value: "spigot"},
			{Label: "Purpur", Value: "purpur"},
			{Label: "Forge", Value: "forge"},
			{Label: "NeoForge", Value: "neoforge"},
			{Label: "Fabric", Value: "fabric"},
			{Label: "", Value: "back"},
		},
	}
}

func (m bukkitModel) Init() tea.Cmd {
	return nil
}

func (m bukkitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	selected := m.choices[m.cursor]

	switch msg := msg.(type) {

	case tea.KeyPressMsg:

		switch msg.String() {

		case "ctrl+c", "q":
			newModel := NewMenuModel(m.state)
			return newModel, newModel.Init()

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		case "enter", "space":
			if selected.Value == "back" {
				newModel := NewMenuModel(m.state)
				return newModel, newModel.Init()
			}

			m.state.Bukkit = selected.Value

			newModel := NewVersionModel(m.state)
			return newModel, newModel.Init()
		}
	}

	return m, nil
}

func (m bukkitModel) View() tea.View {
	s := tr(m.state, "server.title") + "\n\n"

	for i, item := range m.choices {

		cursor := "○"
		if m.cursor == i {
			cursor = selectedStyle.Render("●")
		}

		label := item.Label
		if item.Value == "back" {
			label = tr(m.state, "server.back")
		} else {
			label = tr(m.state, "server."+item.Value)
		}

		// 스타일은 여기서만 적용
		switch item.Value {
		case "paper":
			label = paperSelectStyle.Render(label)
		case "velocity":
			label = velocitySelectStyle.Render(label)
		case "spigot":
			label = spigotSelectStyle.Render(label)
		case "purpur":
			label = purpurSelectStyle.Render(label)
		case "forge":
			label = forgeSelectStyle.Render(label)
		case "neoforge":
			label = neoforgeSelectStyle.Render(label)
		case "fabric":
			label = fabricSelectStyle.Render(label)
		}

		s += fmt.Sprintf("%s %s\n", cursor, label)
	}

	s += "\n" + tr(m.state, "server.previous_hint") + "\n"

	return tea.NewView(s)
}
