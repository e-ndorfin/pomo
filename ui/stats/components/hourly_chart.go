package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/Bahaaio/pomo/db"
	"github.com/charmbracelet/lipgloss"
)

const (
	hourlyBarThickness = 1
	hourlySpacing      = 1
	hoursInDay         = 24
	hourlyLabelStep    = 3
)

var hourlySpacer = strings.Repeat(paddingChar, hourlySpacing)

type hourlyChartLayout struct {
	barHeight       int
	yAxisLabelWidth int
	yAxisWidth      int
	barAreaWidth    int
	totalWidth      int
}

type HourlyChart struct {
	hourlyChartLayout
}

func NewHourlyChart(height int) HourlyChart {
	return HourlyChart{
		hourlyChartLayout: hourlyChartLayout{
			barHeight: height - 1 - 1, // leave space for x-axis and labels
		},
	}
}

func (h *HourlyChart) calculateLayout(maxDuration, scale time.Duration) hourlyChartLayout {
	longestLabel := 0

	for duration := maxDuration; duration > 0; duration -= scale {
		label := formatDuration(duration)
		longestLabel = max(longestLabel, len(label))
	}

	yAxisLabelWidth := longestLabel
	yAxisWidth := yAxisLabelWidth + 1 + 1 // label + space + tick char

	barAreaWidth := hourlySpacing + (hourlyBarThickness+hourlySpacing)*hoursInDay

	return hourlyChartLayout{
		barHeight:       h.barHeight,
		yAxisLabelWidth: yAxisLabelWidth,
		yAxisWidth:      yAxisWidth,
		barAreaWidth:    barAreaWidth,
		totalWidth:      yAxisWidth + barAreaWidth,
	}
}

func (h *HourlyChart) View(stats []db.HourlyStat) string {
	if len(stats) == 0 {
		return ""
	}

	maxDuration := time.Hour
	scale := time.Minute * 12

	h.hourlyChartLayout = h.calculateLayout(maxDuration, scale)

	yAxis := h.buildYAxis(maxDuration, scale)
	bars := h.buildBars(stats, maxDuration)

	top := lipgloss.JoinHorizontal(lipgloss.Bottom, yAxis, hourlySpacer, bars)
	xAxis := h.buildXAxis()
	labels := h.buildLabels()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		top,
		xAxis,
		labels,
	)
}

func (h *HourlyChart) buildBars(stats []db.HourlyStat, maxDuration time.Duration) string {
	bars := make([]string, 0, len(stats)*2)

	for _, stat := range stats {
		if stat.WorkDuration == 0 {
			bars = append(bars, renderHourlyBar(0), hourlySpacer)
			continue
		}

		barHeight := int((float64(stat.WorkDuration) / float64(maxDuration)) * float64(h.barHeight))
		barHeight = min(barHeight, h.barHeight)
		bar := renderHourlyBar(barHeight)
		bars = append(bars, bar, hourlySpacer)
	}

	return lipgloss.JoinHorizontal(lipgloss.Bottom, bars...)
}

func (h *HourlyChart) buildYAxis(maxDuration, scale time.Duration) string {
	builder := strings.Builder{}

	epsilon := time.Millisecond * 500

	for duration := maxDuration; duration >= epsilon; duration -= scale {
		tick := fmt.Sprintf("%-*s %s\n", h.yAxisLabelWidth, formatDuration(duration), tickChar)
		builder.WriteString(tick)

		axis := strings.Repeat(paddingChar, h.yAxisLabelWidth) + paddingChar + axisChar
		builder.WriteString(axis)

		if duration-scale >= epsilon {
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

func (h *HourlyChart) buildXAxis() string {
	zeroLabel := fmt.Sprintf("%-*s", h.yAxisLabelWidth, "0")
	return zeroLabel + " " + cornerChar + strings.Repeat(lineChar, h.barAreaWidth)
}

func (h *HourlyChart) buildLabels() string {
	// Build label line: show a label every 3 hours, aligned under the bar
	// Each bar occupies (hourlyBarThickness + hourlySpacing) = 2 chars
	barSlot := hourlyBarThickness + hourlySpacing

	var labels strings.Builder
	for hour := 0; hour < hoursInDay; hour++ {
		if hour%hourlyLabelStep == 0 {
			label := fmt.Sprintf("%d", hour)
			labels.WriteString(label)
			// pad remaining slot chars
			remaining := barSlot - len(label)
			if remaining > 0 {
				labels.WriteString(strings.Repeat(paddingChar, remaining))
			}
		} else {
			labels.WriteString(strings.Repeat(paddingChar, barSlot))
		}
	}

	paddingLength := h.yAxisWidth + hourlySpacing
	padding := strings.Repeat(paddingChar, paddingLength)

	return padding + labels.String()
}

func renderHourlyBar(height int) string {
	if height == 0 {
		return strings.Repeat(paddingChar, hourlyBarThickness)
	}

	bar := strings.Repeat(barChar, hourlyBarThickness)

	return barStyle.Render(strings.Repeat(bar+"\n", height-1) + bar)
}
