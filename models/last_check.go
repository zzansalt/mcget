package models

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type lastCheckModel struct {
	state      *AppState
	destDir    string
	inputText  string
	inputMode  bool
	cursor     int
	choices    []string
	errMessage string
}

func NewLastCheckModel(state *AppState) lastCheckModel {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	return lastCheckModel{
		state:     state,
		destDir:   cwd,
		choices:   []string{"Download & Install", "Select Folder (Folder Dialog)", "Type Folder Path Manually", "Back"},
		inputMode: false,
	}
}

func (m lastCheckModel) Init() tea.Cmd {
	return nil
}

func openFolderPicker() (string, error) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmdStr := `
		Add-Type -AssemblyName System.Windows.Forms
		$d = New-Object System.Windows.Forms.FolderBrowserDialog
		$d.Description = "Select Installation Directory for Minecraft Server"
		$d.ShowNewFolderButton = $true
		if ($d.ShowDialog() -eq "OK") {
			Write-Output $d.SelectedPath
		}
	`
		powershell, err := exec.LookPath("powershell")
		if err != nil {
			return "", fmt.Errorf("PowerShell was not found")
		}
		cmd = exec.Command(powershell, "-NoProfile", "-Command", cmdStr)
	case "darwin":
		cmd = exec.Command("osascript", "-e", `POSIX path of (choose folder with prompt "Select Installation Directory for Minecraft Server")`)
	case "linux":
		if zenity, err := exec.LookPath("zenity"); err == nil {
			cmd = exec.Command(zenity, "--file-selection", "--directory", "--title=Select Installation Directory for Minecraft Server")
		} else if kdialog, err := exec.LookPath("kdialog"); err == nil {
			cmd = exec.Command(kdialog, "--getexistingdirectory", ".", "Select Installation Directory for Minecraft Server")
		} else {
			return "", fmt.Errorf("no supported folder dialog found (install zenity or kdialog, or enter the path manually)")
		}
	default:
		return "", fmt.Errorf("folder dialog is not supported on %s; enter the path manually", runtime.GOOS)
	}
	out, err := cmd.Output()
	if err != nil {
		// Dialog cancellation is not an error from the user's perspective.
		if _, ok := err.(*exec.ExitError); ok {
			return "", nil
		}
		return "", err
	}

	path := strings.TrimSpace(string(out))
	return path, nil
}

func (m lastCheckModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.inputMode {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			key := msg.String()
			switch key {
			case "enter":
				trimmed := strings.TrimSpace(m.inputText)
				if trimmed != "" {
					abs, err := filepath.Abs(trimmed)
					if err == nil {
						m.destDir = abs
					} else {
						m.destDir = trimmed
					}
				}
				m.inputMode = false
			case "esc":
				m.inputMode = false
			case "backspace":
				if len(m.inputText) > 0 {
					m.inputText = m.inputText[:len(m.inputText)-1]
				}
			case "space":
				m.inputText += " "
			default:
				if len(key) == 1 {
					m.inputText += key
				}
			}
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			newModel := NewBuildModel(m.state)
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
			selected := m.choices[m.cursor]
			switch selected {
			case "Download & Install":
				newModel := NewDownloadModel(m.state, m.destDir)
				return newModel, newModel.Init()

			case "Select Folder (Folder Dialog)":
				path, err := openFolderPicker()
				if err != nil {
					m.errMessage = fmt.Sprintf(tr(m.state, "confirm.folder_error"), err)
				} else if path != "" {
					m.destDir = path
					m.errMessage = ""
				}

			case "Type Folder Path Manually":
				m.inputText = m.destDir
				m.inputMode = true
				m.errMessage = ""

			case "Back":
				newModel := NewBuildModel(m.state)
				return newModel, newModel.Init()
			}
		}
	}

	return m, nil
}

func (m lastCheckModel) View() tea.View {
	s := tr(m.state, "confirm.title") + "\n\n"

	s += fmt.Sprintf(tr(m.state, "confirm.server"), selectedStyle.Render(m.state.Bukkit))
	s += fmt.Sprintf(tr(m.state, "confirm.version"), selectedStyle.Render(m.state.Version))
	s += fmt.Sprintf(tr(m.state, "confirm.build"), selectedStyle.Render(m.state.Build))
	s += fmt.Sprintf(tr(m.state, "confirm.directory"), paperSelectStyle.Render(m.destDir))

	if m.errMessage != "" {
		s += fmt.Sprintf(tr(m.state, "confirm.error"), m.errMessage)
	}

	if m.inputMode {
		s += tr(m.state, "confirm.input_title") + "\n"
		s += fmt.Sprintf("  > %s█\n\n", m.inputText)
		s += "  " + tr(m.state, "confirm.input_hint") + "\n"
	} else {
		for i, choice := range m.choices {
			cursor := "○"
			if m.cursor == i {
				cursor = selectedStyle.Render("●")
			}
			label := choice
			switch choice {
			case "Download & Install":
				label = tr(m.state, "confirm.download")
			case "Select Folder (Folder Dialog)":
				label = tr(m.state, "confirm.folder_dialog")
			case "Type Folder Path Manually":
				label = tr(m.state, "confirm.manual_path")
			case "Back":
				label = tr(m.state, "confirm.back")
			}
			s += fmt.Sprintf("%s %s\n", cursor, label)
		}
		s += "\n" + tr(m.state, "confirm.back_hint") + "\n"
	}

	return tea.NewView(s)
}
