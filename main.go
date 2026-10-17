package main

import (
	"fmt"
	"mcget/models"
	"os"

	tea "charm.land/bubbletea/v2"
)

func main() {
	state := &models.AppState{Language: "en"}

	p := tea.NewProgram(models.NewMenuModel(state))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
