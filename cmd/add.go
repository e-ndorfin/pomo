package cmd

import (
	"fmt"
	"time"

	"github.com/Bahaaio/pomo/db"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <start> <end>",
	Short: "Retroactively log a work session",
	Example: `  pomo add 9:00 10:30    # Log 1h30m work session
  pomo add 4:15 5:15     # Log 1h work session
  pomo add 16:15 17:15   # Log 1h work session (24h format)`,

	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		runAdd(args[0], args[1])
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}

func runAdd(startArg, endArg string) {
	start, err := parseClockTime(startArg)
	if err != nil {
		die(fmt.Errorf("invalid start time %q: %w", startArg, err))
	}

	end, err := parseClockTime(endArg)
	if err != nil {
		die(fmt.Errorf("invalid end time %q: %w", endArg, err))
	}

	duration := end.Sub(start)
	if duration <= 0 {
		die(fmt.Errorf("end time must be after start time"))
	}

	database, err := db.Connect()
	if err != nil {
		die(fmt.Errorf("failed to connect to database: %w", err))
	}
	defer database.Close()

	repo := db.NewSessionRepo(database)
	if err := repo.CreateSession(start, duration, db.WorkSession); err != nil {
		die(fmt.Errorf("failed to log session: %w", err))
	}

	fmt.Printf("Logged %s work session (%s - %s)\n",
		formatDuration(duration),
		start.Format("15:04"),
		end.Format("15:04"),
	)
}

// parseClockTime parses a clock time in H:MM or HH:MM format, pinned to today in local timezone.
func parseClockTime(input string) (time.Time, error) {
	t, err := time.Parse("15:04", input)
	if err != nil {
		return time.Time{}, fmt.Errorf("expected format H:MM or HH:MM (e.g. 9:00, 14:30)")
	}

	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, time.Local), nil
}

// formatDuration formats a duration as a compact string like "1h", "45m", or "1h30m".
func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60

	switch {
	case h > 0 && m > 0:
		return fmt.Sprintf("%dh%dm", h, m)
	case h > 0:
		return fmt.Sprintf("%dh", h)
	default:
		return fmt.Sprintf("%dm", m)
	}
}
