package main

import (
	"fmt"
	"strings"
	"time"
	"log"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Messages
type tickMsg time.Time
type resultMsg CheckResult
type allDoneMsg struct{}

// Model
type model struct {
	servers   []Server
	results   map[string]CheckResult
	checking  map[string]bool
	cursor    int
	interval  time.Duration
	spinner   spinner.Model
	lastCheck time.Time
	width     int
	height    int
}

func initialModel() model {
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Critical Error: Failed to initialize configuration framework: %v", err)
	}

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#8B6842"))

	return model{
		servers:  cfg.Servers,
		results:  make(map[string]CheckResult),
		checking: make(map[string]bool),
		interval: cfg.Interval,
		spinner:  sp,
	}
}


func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func checkCmd(s Server) tea.Cmd {
	return func() tea.Msg {
		return resultMsg(checkServer(s))
	}
}

func checkAllCmd(servers []Server) tea.Cmd {
	cmds := make([]tea.Cmd, len(servers))
	for i, s := range servers {
		cmds[i] = checkCmd(s)
	}
	return tea.Batch(cmds...)
}


func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		checkAllCmd(m.servers),
		tickCmd(m.interval),
	)
}


func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "left":
			if m.cursor >= 0 {
				m.cursor--
			}
		case "down", "right":
			if m.cursor < len(m.servers)-1 {
				m.cursor++
			}
		case "r":
			if m.cursor >= 0 {
                s := m.servers[m.cursor]
                m.checking[s.Name] = true
                return m, checkCmd(s)
            }
            return m, nil
		case "R":
			for _, s := range m.servers {
				m.checking[s.Name] = true
			}
			return m, checkAllCmd(m.servers)
		}

	case tickMsg:
		m.lastCheck = time.Time(msg)
		for _, s := range m.servers {
			m.checking[s.Name] = true
		}
		return m, tea.Batch(
			checkAllCmd(m.servers),
			tickCmd(m.interval),
		)

	case resultMsg:
		r := CheckResult(msg)
		m.results[r.Name] = r
		m.checking[r.Name] = false

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}


func (m model) View() string {
	var b strings.Builder

	colName    := 30
	colURL     := 60
	colStatus  := 20
	colCode    := 10
	colLatency := 10
	totalWidth := colName + colURL + colStatus + colCode + colLatency

	header := lipgloss.JoinHorizontal(lipgloss.Top,
		styleHeader.Width(colName).Render("Name"),
		styleHeader.Width(colURL).Render("URL"),
		styleHeader.Width(colStatus).Render("Status"),
		styleHeader.Width(colCode).Render("Code"),
		styleHeader.Width(colLatency).Render("Latency"),
	)
	b.WriteString(header + "\n")


	for i, s := range m.servers {
		r, hasResult := m.results[s.Name]
		isChecking   := m.checking[s.Name]
		isSelected   := i == m.cursor

		var nameStr string
        if isSelected {
            nameStr = "> " + truncate(s.Name, colName-4) 
        } else {
            nameStr = "" + truncate(s.Name, colName-4)
        }
		urlStr  := truncate(s.URL, colURL-2)
		
		cName   := styleCell.Copy()
		cURL    := styleCell.Copy()
		cStatus := styleCell.Copy()
		cCode   := styleCell.Copy()
		cLat    := styleCell.Copy()

		if isSelected {
			cName   = cName.Background(colorSelected)
			cURL    = cURL.Background(colorSelected)
			cStatus = cStatus.Background(colorSelected)
			cCode   = cCode.Background(colorSelected)
			cLat    = cLat.Background(colorSelected)
		}

		var statusStr string
		if isChecking && !hasResult {
			statusStr = m.spinner.View() + " checking…"
		} else if isChecking {
			statusStr = m.spinner.View() + " rechecking…"
		} else if hasResult {
			statusText := r.statusText()
            switch {
            case r.Up && !r.Warn:
                statusStr = styleUp.Copy().Inherit(cStatus).Render(statusText)
            case r.Up && r.Warn:
                statusStr = styleRdr.Copy().Inherit(cStatus).Render(statusText)
            case !r.Up && r.Warn:
                statusStr = styleWarn.Copy().Inherit(cStatus).Render(statusText)
            default:
                statusStr = styleDown.Copy().Inherit(cStatus).Render(statusText)
            }
		} else {
			statusStr = styleMuted.Render("pending…")
		}

		codeStr := "—"
		if hasResult {
			ct := r.codeText()
			switch {
			case r.Code >= 500:
				codeStr = styleDown.Copy().Inherit(cCode).Render(ct)
			case r.Code >= 400:
				codeStr = styleWarn.Copy().Inherit(cCode).Render(ct)
			case r.Code >= 300:
                codeStr = styleRdr.Copy().Inherit(cCode).Render(ct)
			case r.Code > 0:
				codeStr = styleUp.Copy().Inherit(cCode).Render(ct)
			default:
				codeStr = styleMuted.Copy().Inherit(cCode).Render(ct)
			}
		} else {
			codeStr = styleMuted.Copy().Inherit(cCode).Render(codeStr)
		}

		latStr := "—"
		if hasResult && r.Up {
			latStr = r.latencyText()
		} else {
			latStr = styleMuted.Copy().Inherit(cLat).Render(latStr)
		}

		row := lipgloss.JoinHorizontal(lipgloss.Top,
			cName.Width(colName).Render(nameStr),
			cURL.Width(colURL).Render(urlStr),
			cStatus.Width(colStatus).Render(statusStr),
			cCode.Width(colCode).Render(codeStr),
			cLat.Width(colLatency).Render(latStr),
		)
		b.WriteString(row + "\n")
		if isSelected && hasResult && !r.Up && r.Err != "" {
			errStyle := styleMuted.Copy().Background(colorSelected).Width(totalWidth)
			b.WriteString(errStyle.Render("  └ "+r.Err) + "\n")
		}
	}


	nextIn := ""
	if !m.lastCheck.IsZero() {
		elapsed := time.Since(m.lastCheck)
		remaining := m.interval - elapsed
		if remaining > 0 {
			nextIn = fmt.Sprintf(" · next check in %ds", int(remaining.Seconds()))
		}
	}
	help := styleHelp.Render(
		fmt.Sprintf("↑/↓ or ←/→ navigate · r recheck · R recheck all · q quit%s", nextIn),
	)
	b.WriteString("\n" + help)

	return b.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
