package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/Bahaaio/pomo/db"
	"github.com/stretchr/testify/assert"
)

type stubSessionDeleter struct {
	session     *db.Session
	deleted     bool
	getCalls    int
	deleteCalls int
	err         error
}

func (s *stubSessionDeleter) GetMostRecentSession() (*db.Session, error) {
	s.getCalls++
	return s.session, s.err
}

func (s *stubSessionDeleter) DeleteMostRecentSession() (bool, error) {
	s.deleteCalls++
	return s.deleted, s.err
}

func TestRunUndoWithRepoCancelsOnN(t *testing.T) {
	repo := &stubSessionDeleter{session: undoTestSession(), deleted: true}
	var output bytes.Buffer

	runUndoWithRepo(repo, strings.NewReader("n\n"), &output)

	assert.Equal(t, 1, repo.getCalls)
	assert.Zero(t, repo.deleteCalls)
	assert.Contains(t, output.String(), "Undo cancelled.")
}

func TestRunUndoWithRepoRequiresConfirmation(t *testing.T) {
	repo := &stubSessionDeleter{session: undoTestSession(), deleted: true}
	var output bytes.Buffer

	runUndoWithRepo(repo, strings.NewReader("\n"), &output)

	assert.Equal(t, 1, repo.getCalls)
	assert.Zero(t, repo.deleteCalls)
	assert.Contains(t, output.String(), "Undo cancelled.")
}

func TestRunUndoWithRepoPrintsSessionDetailsBeforeConfirmation(t *testing.T) {
	repo := &stubSessionDeleter{session: undoTestSession(), deleted: true}
	var output bytes.Buffer

	runUndoWithRepo(repo, strings.NewReader("n\n"), &output)

	assert.Contains(t, output.String(), "Most recent session:")
	assert.Contains(t, output.String(), "Date: 2026-05-03")
	assert.Contains(t, output.String(), "Start time: 09:15")
	assert.Contains(t, output.String(), "End time: 09:40")
	assert.Contains(t, output.String(), "Duration: 25m")
	assert.Contains(t, output.String(), "Type: work")
	assert.Contains(t, output.String(), "Delete this session? Type y to confirm or n to cancel, then press enter:")
}

func TestRunUndoWithRepoDeletesOnY(t *testing.T) {
	repo := &stubSessionDeleter{session: undoTestSession(), deleted: true}
	var output bytes.Buffer

	runUndoWithRepo(repo, strings.NewReader("y\n"), &output)

	assert.Equal(t, 1, repo.getCalls)
	assert.Equal(t, 1, repo.deleteCalls)
	assert.Contains(t, output.String(), "Deleted the most recent session.")
}

func TestRunUndoWithRepoHandlesNoSessions(t *testing.T) {
	repo := &stubSessionDeleter{deleted: false}
	var output bytes.Buffer

	runUndoWithRepo(repo, strings.NewReader("y\n"), &output)

	assert.Equal(t, 1, repo.getCalls)
	assert.Zero(t, repo.deleteCalls)
	assert.Contains(t, output.String(), "No sessions to undo.")
}

func undoTestSession() *db.Session {
	startedAt := time.Date(2026, 5, 3, 9, 15, 0, 0, time.Local)
	endedAt := startedAt.Add(25 * time.Minute)

	return &db.Session{
		ID:        1,
		Type:      string(db.WorkSession),
		Duration:  25 * time.Minute,
		StartedAt: startedAt,
		EndedAt:   &endedAt,
	}
}
