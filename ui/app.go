// Package ui provides the terminal user interface for pomodoro sessions.
package ui

import (
	"github.com/Bahaaio/pomo/ui/confirm"
	"github.com/Bahaaio/pomo/ui/stats"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) Init() tea.Cmd {
	switch m.screen {
	case ScreenHome:
		return nil
	case ScreenStats:
		return m.statsModel.Init()
	default:
		return m.timer.Init()
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// handle window resize globally
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
		m.confirmDialog.HandleWindowResize(msg)
		m.progressBar.Width = min(m.width-2*padding-margin, maxWidth)

		if m.screen == ScreenStats {
			updatedStats, cmd := m.statsModel.Update(msg)
			m.statsModel = updatedStats.(stats.Model)
			return m, cmd
		}

		return m, nil
	}

	switch m.screen {
	case ScreenHome:
		return m, m.handleHomeKeys(msg)
	case ScreenStats:
		return m.updateStats(msg)
	default:
		return m.updateTimer(msg)
	}
}

func (m Model) updateTimer(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, m.handleKeys(msg)

	case timer.TickMsg:
		return m, m.handleTimerTick(msg)

	case confirmTickMsg:
		return m, m.handleConfirmTick()

	case timer.StartStopMsg:
		return m, m.handleTimerStartStop(msg)

	case progress.FrameMsg:
		return m, m.handleProgressBarFrame(msg)

	case confirm.ChoiceMsg:
		return m, m.handleConfirmChoice(msg)

	case commandsDoneMsg:
		return m, m.handleCommandsDone()

	default:
		return m, nil
	}
}

func (m Model) updateStats(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(msg, forceQuitKey) {
			return m, tea.Quit
		}
		if key.Matches(msg, keyMap.Quit) {
			if m.appMode {
				m.goHome()
				return m, nil
			}
			return m, tea.Quit
		}
	}

	updatedStats, cmd := m.statsModel.Update(msg)
	m.statsModel = updatedStats.(stats.Model)
	return m, cmd
}

func (m Model) View() string {
	switch m.screen {
	case ScreenHome:
		return m.buildHomeView()
	case ScreenStats:
		return m.statsModel.View()
	default:
		return m.buildTimerView()
	}
}

func (m Model) buildTimerView() string {
	if m.sessionState == Quitting {
		return ""
	}

	if m.sessionState == WaitingForCommands {
		return m.buildWaitingForCommandsView()
	}

	// show confirmation dialog
	if m.sessionState == ShowingConfirm {
		return m.buildConfirmDialogView()
	}

	content := m.buildMainContent()
	content += m.buildStatusIndicators()
	content += m.buildProgressBar()

	help := m.buildHelpView()

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, content, help),
	)
}
