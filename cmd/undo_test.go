package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type stubSessionDeleter struct {
	deleted bool
	calls   int
	err     error
}

func (s *stubSessionDeleter) DeleteMostRecentSession() (bool, error) {
	s.calls++
	return s.deleted, s.err
}

func TestRunUndoWithRepoRequiresConfirmation(t *testing.T) {
	repo := &stubSessionDeleter{deleted: true}
	var output bytes.Buffer

	runUndoWithRepo(repo, strings.NewReader("n\n"), &output)

	assert.Zero(t, repo.calls)
	assert.Contains(t, output.String(), "Undo cancelled.")
}

func TestRunUndoWithRepoDeletesOnY(t *testing.T) {
	repo := &stubSessionDeleter{deleted: true}
	var output bytes.Buffer

	runUndoWithRepo(repo, strings.NewReader("y\n"), &output)

	assert.Equal(t, 1, repo.calls)
	assert.Contains(t, output.String(), "Deleted the most recent session.")
}

func TestRunUndoWithRepoHandlesNoSessions(t *testing.T) {
	repo := &stubSessionDeleter{deleted: false}
	var output bytes.Buffer

	runUndoWithRepo(repo, strings.NewReader("y\n"), &output)

	assert.Equal(t, 1, repo.calls)
	assert.Contains(t, output.String(), "No sessions to undo.")
}
