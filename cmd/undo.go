package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Bahaaio/pomo/db"
	"github.com/spf13/cobra"
)

var undoCmd = &cobra.Command{
	Use:   "undo",
	Args:  cobra.NoArgs,
	Short: "Undo the most recent session",
	Run: func(cmd *cobra.Command, args []string) {
		runUndo(os.Stdin, os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(undoCmd)
}

type sessionDeleter interface {
	GetMostRecentSession() (*db.Session, error)
	DeleteMostRecentSession() (bool, error)
}

func runUndo(input io.Reader, output io.Writer) {
	database, err := db.Connect()
	if err != nil {
		die(fmt.Errorf("failed to connect to database: %w", err))
	}
	defer database.Close()

	runUndoWithRepo(db.NewSessionRepo(database), input, output)
}

func runUndoWithRepo(repo sessionDeleter, input io.Reader, output io.Writer) {
	session, err := repo.GetMostRecentSession()
	if err != nil {
		die(fmt.Errorf("failed to find most recent session: %w", err))
	}

	if session == nil {
		fmt.Fprintln(output, "No sessions to undo.")
		return
	}

	printSessionUndoConfirmation(output, session)

	reader := bufio.NewReader(input)
	answer, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		die(fmt.Errorf("failed to read confirmation: %w", err))
	}

	if strings.TrimSpace(answer) != "y" {
		fmt.Fprintln(output, "Undo cancelled.")
		return
	}

	deleted, err := repo.DeleteMostRecentSession()
	if err != nil {
		die(fmt.Errorf("failed to undo most recent session: %w", err))
	}

	if !deleted {
		fmt.Fprintln(output, "No sessions to undo.")
		return
	}

	fmt.Fprintln(output, "Deleted the most recent session.")
}

func printSessionUndoConfirmation(output io.Writer, session *db.Session) {
	endedAt := session.StartedAt.Add(session.Duration)
	if session.EndedAt != nil {
		endedAt = *session.EndedAt
	}

	startedAt := session.StartedAt.Local()
	endedAt = endedAt.Local()

	fmt.Fprintln(output, "Most recent session:")
	fmt.Fprintf(output, "  Date: %s\n", startedAt.Format(time.DateOnly))
	fmt.Fprintf(output, "  Start time: %s\n", startedAt.Format("15:04"))
	fmt.Fprintf(output, "  End time: %s\n", endedAt.Format("15:04"))
	fmt.Fprintf(output, "  Duration: %s\n", formatDuration(session.Duration))
	fmt.Fprintf(output, "  Type: %s\n", session.Type)
	fmt.Fprint(output, "Delete this session? Type y to confirm or n to cancel, then press enter: ")
}
