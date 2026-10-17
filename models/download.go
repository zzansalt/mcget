package models

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	util "mcget/utils"

	tea "charm.land/bubbletea/v2"
)

type progressMsg int64
type downloadDoneMsg struct {
	err error
}
type progressClosedMsg struct{}

type detailsMsg struct {
	details *util.BuildDetails
	err     error
}

type downloadModel struct {
	state            *AppState
	destDir          string
	fileName         string
	totalBytes       int64
	downloadedBytes  int64
	loading          bool
	downloading      bool
	done             bool
	err              error
	progressChan     chan int64
	doneChan         chan downloadDoneMsg
	details          *util.BuildDetails
	confirmOverwrite bool
	choiceCursor     int
	cancel           context.CancelFunc
	cancelRequested  bool
	skipped          bool
	finalPath        string
	installActive    bool
}

func NewDownloadModel(state *AppState, destDir string) downloadModel {
	return downloadModel{
		state:   state,
		destDir: destDir,
		loading: true,
	}
}

func fetchDownloadDetailsCmd(bukkit string, version string, build string) tea.Cmd {
	return func() tea.Msg {
		details, err := util.GetServerBuildDetails(bukkit, version, build)
		return detailsMsg{details: details, err: err}
	}
}

func listenToProgress(ch chan int64) tea.Cmd {
	return func() tea.Msg {
		progress, ok := <-ch
		if !ok {
			return progressClosedMsg{}
		}
		return progressMsg(progress)
	}
}

func listenToDone(ch chan downloadDoneMsg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

func (m downloadModel) Init() tea.Cmd {
	return fetchDownloadDetailsCmd(m.state.Bukkit, m.state.Version, m.state.Build)
}

func (m downloadModel) startDownload(path string, overwrite bool) (tea.Model, tea.Cmd) {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	install := len(m.details.InstallArgs) > 0
	if install {
		m.finalPath = m.destDir
	} else {
		m.finalPath = path
	}
	m.progressChan = make(chan int64, 100)
	m.doneChan = make(chan downloadDoneMsg, 1)
	m.downloading = true
	m.installActive = install
	return m, tea.Batch(
		func() tea.Msg {
			var result downloadDoneMsg
			if install {
				result.err = runServerInstaller(ctx, m.destDir, m.details, m.state.Bukkit, m.progressChan)
			} else {
				result.err = util.DownloadFile(ctx, m.details.URL, path, m.details.SHA256, overwrite, m.progressChan)
			}
			close(m.progressChan)
			m.doneChan <- result
			return nil
		},
		listenToProgress(m.progressChan),
		listenToDone(m.doneChan),
	)
}

func runServerInstaller(ctx context.Context, destDir string, details *util.BuildDetails, server string, progress chan<- int64) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	workDir, err := os.MkdirTemp(destDir, ".mcget-installer-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workDir)
	installerPath := filepath.Join(workDir, details.Name)
	if err := util.DownloadFile(ctx, details.URL, installerPath, details.SHA256, false, progress); err != nil {
		return err
	}
	java, err := exec.LookPath("java")
	if err != nil {
		return fmt.Errorf("Java is required to install %s; install a supported Java runtime first", server)
	}
	args := append([]string{"-jar", installerPath}, details.InstallArgs...)
	if server == "fabric" {
		args = append(args, "-dir", destDir)
	}
	cmd := exec.CommandContext(ctx, java, args...)
	cmd.Dir = destDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		log := strings.TrimSpace(string(output))
		if len(log) > 1200 {
			log = log[len(log)-1200:]
		}
		return fmt.Errorf("server installer failed: %w\n%s", err, log)
	}
	return nil
}

func (m downloadModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case detailsMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.fileName = msg.details.Name
		m.totalBytes = msg.details.Size
		m.details = msg.details
		destPath := filepath.Join(m.destDir, m.fileName)
		if len(msg.details.InstallArgs) == 0 {
			if _, err := os.Stat(destPath); err == nil {
				m.confirmOverwrite = true
				return m, nil
			}
		}
		if len(msg.details.InstallArgs) > 0 {
			return m.startDownload(destPath, false)
		}
		return m.startDownload(destPath, false)

	case progressMsg:
		m.downloadedBytes = int64(msg)
		return m, listenToProgress(m.progressChan)

	case progressClosedMsg:
		return m, nil

	case downloadDoneMsg:
		m.downloading = false
		m.done = true
		m.err = msg.err
		if m.cancel != nil {
			m.cancel()
		}
		return m, nil

	case tea.KeyPressMsg:
		if m.confirmOverwrite {
			switch msg.String() {
			case "up", "k":
				if m.choiceCursor > 0 {
					m.choiceCursor--
				}
			case "down", "j":
				if m.choiceCursor < 2 {
					m.choiceCursor++
				}
			case "enter", "space":
				m.confirmOverwrite = false
				path := filepath.Join(m.destDir, m.fileName)
				switch m.choiceCursor {
				case 0:
					return m.startDownload(path, true)
				case 1:
					m.done, m.skipped = true, true
					return m, nil
				case 2:
					ext := filepath.Ext(m.fileName)
					base := strings.TrimSuffix(m.fileName, ext)
					n := 2
					for {
						candidate := filepath.Join(m.destDir, fmt.Sprintf("%s (%d)%s", base, n, ext))
						if _, err := os.Stat(candidate); os.IsNotExist(err) {
							return m.startDownload(candidate, false)
						}
						n++
					}
				}
			}
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c", "q":
			if m.downloading {
				m.cancelRequested = true
				if m.cancel != nil {
					m.cancel()
				}
			} else {
				newModel := NewMenuModel(m.state)
				return newModel, newModel.Init()
			}
		case "enter":
			if m.done {
				newModel := NewMenuModel(m.state)
				return newModel, newModel.Init()
			}
		}
	}

	return m, nil
}

func (m downloadModel) View() tea.View {
	if m.loading {
		return tea.NewView(tr(m.state, "download.details_loading") + "\n")
	}
	if m.confirmOverwrite {
		choices := []string{
			tr(m.state, "download.overwrite"),
			tr(m.state, "download.skip"),
			tr(m.state, "download.numbered"),
		}
		s := fmt.Sprintf(tr(m.state, "download.overwrite_prompt"), filepath.Join(m.destDir, m.fileName))
		for i, choice := range choices {
			cursor := "○"
			if i == m.choiceCursor {
				cursor = selectedStyle.Render("●")
			}
			s += fmt.Sprintf("%s %s\n", cursor, choice)
		}
		return tea.NewView(s + "\n" + tr(m.state, "download.choice_hint") + "\n")
	}

	if m.err != nil {
		if m.cancelRequested {
			return tea.NewView(tr(m.state, "download.cancelled"))
		}
		return tea.NewView(fmt.Sprintf(tr(m.state, "download.error"), m.err))
	}

	s := tr(m.state, "download.title") + "\n\n"
	s += fmt.Sprintf(tr(m.state, "download.server"), selectedStyle.Render(m.state.Bukkit))
	s += fmt.Sprintf(tr(m.state, "download.version"), selectedStyle.Render(m.state.Version), m.state.Build)
	s += fmt.Sprintf(tr(m.state, "download.filename"), selectedStyle.Render(m.fileName))
	s += fmt.Sprintf(tr(m.state, "download.directory"), paperSelectStyle.Render(m.destDir))

	if m.downloading {
		var percent float64
		if m.totalBytes > 0 {
			percent = float64(m.downloadedBytes) / float64(m.totalBytes)
		}
		if percent > 1.0 {
			percent = 1.0
		}

		width := 30
		filled := int(percent * float64(width))
		bar := "["
		for i := 0; i < width; i++ {
			if i < filled {
				bar += "="
			} else if i == filled {
				bar += ">"
			} else {
				bar += " "
			}
		}
		bar += "]"

		s += fmt.Sprintf(tr(m.state, "download.progress"), bar, percent*100)
		s += fmt.Sprintf(tr(m.state, "download.bytes"), float64(m.downloadedBytes)/(1024*1024), float64(m.totalBytes)/(1024*1024))
		if m.installActive {
			s += tr(m.state, "download.install_wait")
		} else {
			s += tr(m.state, "download.wait")
		}
	} else if m.done && m.skipped {
		s += tr(m.state, "download.skipped")
	} else if m.done {
		if m.installActive {
			s += fmt.Sprintf(tr(m.state, "download.success_installed"), m.finalPath)
		} else if m.details.SHA256 == "" {
			s += fmt.Sprintf(tr(m.state, "download.success_no_hash"), m.finalPath)
		} else {
			s += fmt.Sprintf(tr(m.state, "download.success"), m.finalPath)
		}
	}

	return tea.NewView(s)
}
