package db

import (
	"io"
	"log"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	m.Run()
}

func TestSessionRepoStatsHandleLegacyAndNewRows(t *testing.T) {
	repo := newTestRepo(t)

	now := time.Now().Local()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)

	insertLegacySession(t, repo.db,
		yesterday.Add(23*time.Hour+50*time.Minute),
		today.Add(10*time.Minute),
		20*time.Minute,
		WorkSession,
	)
	insertLegacySession(t, repo.db,
		today.Add(1*time.Hour+30*time.Minute),
		today.Add(2*time.Hour+15*time.Minute),
		45*time.Minute,
		WorkSession,
	)
	insertNewSession(t, repo.db,
		today.Add(3*time.Hour),
		today.Add(3*time.Hour+30*time.Minute),
		30*time.Minute,
		WorkSession,
	)
	insertLegacySession(t, repo.db,
		today.Add(4*time.Hour),
		today.Add(4*time.Hour+15*time.Minute),
		15*time.Minute,
		BreakSession,
	)

	dailyStats, err := repo.GetLast24hDurationStats()
	require.NoError(t, err)
	assert.Equal(t, 95*time.Minute, dailyStats.WorkDuration)
	assert.Equal(t, 15*time.Minute, dailyStats.BreakDuration)

	weeklyStats, err := repo.GetWeeklyStats()
	require.NoError(t, err)

	statsByDate := make(map[string]time.Duration, len(weeklyStats))
	for _, stat := range weeklyStats {
		statsByDate[stat.Date] = stat.WorkDuration
	}

	assert.Equal(t, 20*time.Minute, statsByDate[yesterday.Format(DateFormat)])
	assert.Equal(t, 75*time.Minute, statsByDate[today.Format(DateFormat)])

	hourlyStats, err := repo.GetTodayHourlyStats()
	require.NoError(t, err)
	require.Len(t, hourlyStats, 24)

	assert.Equal(t, 30*time.Minute, hourlyStats[1].WorkDuration)
	assert.Equal(t, 15*time.Minute, hourlyStats[2].WorkDuration)
	assert.Equal(t, 30*time.Minute, hourlyStats[3].WorkDuration)
	assert.Zero(t, hourlyStats[0].WorkDuration)

	streakStats, err := repo.GetStreakStats()
	require.NoError(t, err)
	assert.Equal(t, 2, streakStats.Current)
	assert.Equal(t, 2, streakStats.Best)
}

func newTestRepo(t *testing.T) *SessionRepo {
	t.Helper()

	database, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = database.Close()
	})

	require.NoError(t, createSchema(database))

	return NewSessionRepo(database)
}

func insertLegacySession(t *testing.T, database *sqlx.DB, actualStart time.Time, recordedEnd time.Time, duration time.Duration, sessionType SessionType) {
	t.Helper()

	_, err := database.Exec(
		"INSERT INTO sessions (started_at, ended_at, duration, type) VALUES (?, ?, ?, ?)",
		recordedEnd.Format(time.RFC3339),
		"",
		duration,
		sessionType,
	)
	require.NoError(t, err)

	assert.Equal(t, recordedEnd.Sub(actualStart), duration)
}

func insertNewSession(t *testing.T, database *sqlx.DB, startedAt time.Time, endedAt time.Time, duration time.Duration, sessionType SessionType) {
	t.Helper()

	_, err := database.Exec(
		"INSERT INTO sessions (started_at, ended_at, duration, type) VALUES (?, ?, ?, ?)",
		startedAt.Format(time.RFC3339),
		endedAt.Format(time.RFC3339),
		duration,
		sessionType,
	)
	require.NoError(t, err)

	assert.Equal(t, endedAt.Sub(startedAt), duration)
}
