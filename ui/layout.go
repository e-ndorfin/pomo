package ui

import (
	"fmt"
	"time"

	"github.com/Bahaaio/pomo/ui/ascii"
	"github.com/Bahaaio/pomo/ui/colors"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

const (
	maxWidth           = 80
	margin             = 4
	padding            = 2
	separator          = " — "
	pausedIndicator    = "(paused)"
	completedIndicator = "done!"
)

func (m *Model) buildConfirmDialogView() string {
	idle := time.Since(m.confirmStartTime).Truncate(time.Second)
	title := m.currentTaskType.Opposite().GetTask().Title

	// if we're prompting to start a long break
	if m.cyclePosition == m.longBreak.After {
		title = "long " + title
	}

	return m.confirmDialog.View("start "+title+"?", time.Duration(idle))
}

func (m *Model) buildMainContent() string {
	timeLeft := m.buildTimeLeft()

	if m.useTimerArt {
		return timeLeft + "\n\n" + m.currentTask.Title
	}

	content := m.currentTask.Title
	if !m.timer.Timedout() {
		content += separator + timeLeft
	}

	return content
}

func (m *Model) buildStatusIndicators() string {
	if m.timer.Timedout() {
		return separator + completedIndicator
	}

	indicators := ""

	if m.longBreak.Enabled {
		indicators += fmt.Sprintf(" · %d/%d", m.cyclePosition, m.longBreak.After)
	}

	if m.sessionState == Paused {
		indicators += " " + pausedIndicator
	}

	return indicators
}

func (m *Model) buildProgressBar() string {
	return "\n\n" + m.progressBar.View() + "\n"
}

// returns time left as a string in HH:MM:SS format
func (m *Model) buildTimeLeft() string {
	left := m.timer.Timeout
	hours := int(left.Hours())
	minutes := int(left.Minutes()) % 60
	seconds := int(left.Seconds()) % 60

	time := ""

	// only show hours if they are non-zero
	if hours > 0 {
		time += fmt.Sprintf("%02d:", hours)
	}
	time += fmt.Sprintf("%02d:%02d", minutes, seconds)

	if m.useTimerArt {
		time = ascii.RenderNumber(time, m.timerFont)

		// remove color on pause
		if m.sessionState == Paused {
			noColor := m.asciiTimerStyle.Foreground(colors.PauseFg)
			return noColor.Render(time)
		}

		return m.asciiTimerStyle.Render(time)
	}

	return time
}

func (m *Model) buildHelpView() string {
	if m.appMode {
		appKeyMap := keyMap
		appKeyMap.Quit = key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "back"),
		)
		return m.help.View(appKeyMap)
	}
	return m.help.View(keyMap)
}

func (m *Model) buildHomeView() string {
	tomato := buildTomatoArt()

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF6347"))

	menuItemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#AAAAAA"))

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true)

	title := titleStyle.Render("pomo")

	menu := lipgloss.JoinVertical(lipgloss.Left,
		keyStyle.Render("p/space")+"  "+menuItemStyle.Render("start session"),
		keyStyle.Render("b")+"       "+menuItemStyle.Render("start break"),
		keyStyle.Render("s")+"       "+menuItemStyle.Render("stats"),
		keyStyle.Render("q")+"       "+menuItemStyle.Render("quit"),
	)

	content := lipgloss.JoinVertical(lipgloss.Center,
		tomato,
		"",
		"",
		title,
		"",
		menu,
	)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

func buildTomatoArt() string {
	r := lipgloss.NewStyle().Foreground(lipgloss.Color("#E83020")) // red body
	d := lipgloss.NewStyle().Foreground(lipgloss.Color("#B82818")) // dark red shadow
	h := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF9080")) // highlight
	g := lipgloss.NewStyle().Foreground(lipgloss.Color("#3C8C2C")) // green stem
	l := lipgloss.NewStyle().Foreground(lipgloss.Color("#5BAD4B")) // light green leaf

	b := func(s lipgloss.Style, n int) string {
		out := ""
		for range n {
			out += "█"
		}
		return s.Render(out)
	}
	sp := func(n int) string {
		out := ""
		for range n {
			out += " "
		}
		return out
	}

	lines := []string{
		b(g, 2),                                       // stem
		b(l, 2) + sp(1) + b(g, 4) + sp(1) + b(l, 2), // leaves + stem base
		b(h, 2) + b(r, 14) + b(d, 2),                 // body top (18)
		b(h, 1) + b(r, 20) + b(d, 1),                 // widen (22)
		b(h, 1) + b(r, 21) + b(d, 2),                 // near max (24)
		b(r, 24),                                      // max width
		b(r, 24),                                      // max width
		b(r, 22) + b(d, 2),                            // taper + shadow (24)
		b(r, 20) + b(d, 2),                            // (22)
		b(r, 16) + b(d, 2),                            // (18)
		b(r, 12),                                      // bottom (12)
	}

	return lipgloss.JoinVertical(lipgloss.Center, lines...)
}

func (m Model) buildWaitingForCommandsView() string {
	help := m.help.View(KeyMap{Quit: keyMap.Quit})

	message := lipgloss.JoinVertical(
		lipgloss.Center,
		"Waiting for post commands to complete...",
		"\n",
		help,
	)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		message,
	)
}
