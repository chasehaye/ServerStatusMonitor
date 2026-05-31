package main

import "github.com/charmbracelet/lipgloss"

var (
	colorUp       = lipgloss.Color("#00E676")
	colorRdr      = lipgloss.Color("#FFD700")
	colorWarn     = lipgloss.Color("#FF9100")
	colorDown     = lipgloss.Color("#FF5252")
	colorMuted   = lipgloss.Color("#626262")
	colorHeader  = lipgloss.Color("#B8860B")
	colorBorder  = lipgloss.Color("#3C3C3C")
	colorSelected = lipgloss.Color("#242424")

	styleHeader = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorHeader).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(colorBorder).
		Padding(0, 1)

		
	styleUp = lipgloss.NewStyle().
		Foreground(colorUp).
		Bold(true)

	styleDown = lipgloss.NewStyle().
		Foreground(colorDown).
		Bold(true)

	styleRdr = lipgloss.NewStyle().
        Foreground(colorRdr).
        Bold(true)

    styleWarn = lipgloss.NewStyle().
        Foreground(colorWarn).
        Bold(true)


	styleMuted = lipgloss.NewStyle().
		Foreground(colorMuted)

	styleSelected = lipgloss.NewStyle().
		Background(colorSelected)

	styleCell = lipgloss.NewStyle().
		Padding(0, 1)

	styleHelp = lipgloss.NewStyle().
		Foreground(colorMuted).
		MarginTop(1)
)
