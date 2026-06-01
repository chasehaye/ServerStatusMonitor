package main

import (
	"database/sql" 
	"fmt"
	"os/exec"
	"strings"
	"time"
	"log"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Messages
type tickMsg struct{ serverName string }
type resultMsg CheckResult

// Model
type model struct {
	db        *sql.DB
	uptime    map[string]float64 
	servers   []Server
	results   map[string]CheckResult
	checking  map[string]bool
	cursor    int
	interval  time.Duration
	spinner   spinner.Model
	nextCheck map[string]time.Time
	muted        bool
	alarmRunning bool
	stopAlarm    chan struct{}
}

func (m *model) syncAlarm() {
    anyDown := false
    for _, s := range m.servers {
        if res, ok := m.results[s.Name]; ok && !res.Up {
            anyDown = true
            break
        }
    }
    shouldPlay := anyDown && !m.muted
    if shouldPlay && !m.alarmRunning {
        m.alarmRunning = true
        stop := make(chan struct{})
        m.stopAlarm = stop
        go func() {
            cmd := exec.Command("ffplay", "-nodisp", "-loop", "0", "./alarm.wav")
            cmd.Start()
            <-stop
            cmd.Process.Kill()
        }()
    }
    if !shouldPlay && m.alarmRunning {
        m.alarmRunning = false
        close(m.stopAlarm)
        m.stopAlarm = nil
    }
}

func initialModel(db *sql.DB) model {
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Critical Error: Failed to initialize configuration framework: %v", err)
	}

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#8B6842"))

	return model{
		db:        db,
		uptime:    make(map[string]float64),
		servers:   cfg.Servers,
		results:   make(map[string]CheckResult),
		checking:  make(map[string]bool),
		interval:  cfg.Interval,
		spinner:   sp,
		nextCheck: make(map[string]time.Time),
	}
}


func tickCmd(s Server, globalInterval time.Duration) tea.Cmd {
	d := s.interval(globalInterval)
	return tea.Tick(d, func(time.Time) tea.Msg {
		return tickMsg{serverName: s.Name}
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
	now := time.Now()
	for _, s := range m.servers {
		m.nextCheck[s.Name] = now.Add(s.interval(m.interval))
	}
	cmds := []tea.Cmd{
		m.spinner.Tick,
		checkAllCmd(m.servers),
	}
	for _, s := range m.servers {
		cmds = append(cmds, tickCmd(s, m.interval))
	}
	return tea.Batch(cmds...)
}



func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {


	case tea.KeyMsg:
		switch msg.String() {
			case "q", "ctrl+c":
				if m.alarmRunning {
					close(m.stopAlarm)
					m.stopAlarm = nil
				}
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
			case "a":
				m.muted = !m.muted
				m.syncAlarm()
				return m, nil
		}

	case tickMsg:
		for _, s := range m.servers {
			if s.Name == msg.serverName {
				m.checking[s.Name] = true
				m.nextCheck[s.Name] = time.Now().Add(s.interval(m.interval))
				return m, tea.Batch(
					checkCmd(s),
					tickCmd(s, m.interval),
				)
			}
		}

	case resultMsg:
		r := CheckResult(msg)
		m.results[r.Name] = r
		m.checking[r.Name] = false
		if m.db != nil {
			go logEvent(m.db, r.Name, r.URL, r.Up)
			if pct := calcUptime(m.db, r.Name, 24*time.Hour); pct >= 0 {
				m.uptime[r.Name] = pct
			}
		}
		m.syncAlarm()
		return m, nil

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
	colUptime  := 10
	totalWidth := colName + colURL + colStatus + colCode + colLatency + colUptime

	header := lipgloss.JoinHorizontal(lipgloss.Top,
		styleHeader.Width(colName).Render("Name"),
		styleHeader.Width(colURL).Render("URL"),
		styleHeader.Width(colStatus).Render("Status"),
		styleHeader.Width(colCode).Render("Code"),
		styleHeader.Width(colLatency).Render("Latency"),
		styleHeader.Width(colUptime).Render("Uptime"),
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
		cUptime := styleCell.Copy() 

		if isSelected {
			cName   = cName.Background(colorSelected)
			cURL    = cURL.Background(colorSelected)
			cStatus = cStatus.Background(colorSelected)
			cCode   = cCode.Background(colorSelected)
			cLat    = cLat.Background(colorSelected)
			cUptime = cUptime.Background(colorSelected)
		}

		var statusStr string
		if isChecking {
			statusStr = m.spinner.View() + cStatus.Copy().Render(" checking…")
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

		uptimeStr := "—"
		if pct, ok := m.uptime[s.Name]; ok {
			uptimeStr = fmt.Sprintf("%.2f%%", pct)
		}

		row := lipgloss.JoinHorizontal(lipgloss.Top,
			cName.Width(colName).Render(nameStr),
			cURL.Width(colURL).Render(urlStr),
			cStatus.Width(colStatus).Render(statusStr),
			cCode.Width(colCode).Render(codeStr),
			cLat.Width(colLatency).Render(latStr),
			cUptime.Width(colUptime).Render(uptimeStr),
		)
		b.WriteString(row + "\n")
		if isSelected && hasResult && !r.Up && r.Err != "" {
			errStyle := styleMuted.Copy().Background(colorSelected).Width(totalWidth)
			b.WriteString(errStyle.Render("  └ "+r.Err) + "\n")
		}
	}


	var footerRight string
	if m.cursor >= 0 && m.cursor < len(m.servers) {
		s := m.servers[m.cursor]
		if t, ok := m.nextCheck[s.Name]; ok {
			if remaining := time.Until(t); remaining > 0 {
				footerRight = fmt.Sprintf(" · %s next check in %ds", s.Name, int(remaining.Seconds()))
			}
		}
	}
	if footerRight == "" {
		var soonest time.Time
		for _, t := range m.nextCheck {
			if soonest.IsZero() || t.Before(soonest) {
				soonest = t
			}
		}
		if !soonest.IsZero() {
			if remaining := time.Until(soonest); remaining > 0 {
				footerRight = fmt.Sprintf(" · next check in %ds", int(remaining.Seconds()))
			}
		}
	}
	help := styleHelp.Render(
    	fmt.Sprintf("↑/↓ or ←/→ navigate · r recheck · R recheck all · a mute · q quit%s", footerRight),
	)
	muteStr := ""
	if m.muted {
		muteStr = styleDown.Render("[muted]")
		b.WriteString("\n" + muteStr)
	}
	b.WriteString("\n" + help)


	return b.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
