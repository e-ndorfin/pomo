package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

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
	fmt.Fprint(output, "Delete the most recent session? Type y and press enter to confirm: ")

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
