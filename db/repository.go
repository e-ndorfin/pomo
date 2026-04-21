package db

import (
	"time"

	"github.com/jmoiron/sqlx"
)

const DateFormat = "2006-01-02"

type SessionRepo struct {
	db *sqlx.DB
}

func NewSessionRepo(db *sqlx.DB) *SessionRepo {
	return &SessionRepo{db: db}
}

// CreateSession inserts a new session record into the database.
func (r *SessionRepo) CreateSession(startedAt time.Time, endedAt time.Time, duration time.Duration, sessionType SessionType) error {
	startedAtStr := startedAt.Format(time.RFC3339)
	endedAtStr := endedAt.Format(time.RFC3339)

	if _, err := r.db.Exec(
		"insert into sessions (started_at, ended_at, duration, type) values (?, ?, ?, ?);",
		startedAtStr,
		endedAtStr,
		duration,
		sessionType,
	); err != nil {
		return err
	}

	return nil
}

// GetAllTimeStats retrieves aggregate statistics across all sessions.
func (r *SessionRepo) GetAllTimeStats() (AllTimeStats, error) {
	var totalStats AllTimeStats

	// sqlite treats (type = 'work') as 1 or 0
	if err := r.db.Get(
		&totalStats,
		`
		SELECT
			COUNT(*) AS total_sessions,
			COALESCE(SUM(duration * (type = 'work')), 0)  AS total_work_duration,
			COALESCE(SUM(duration * (type = 'break')), 0) AS total_break_duration
		FROM sessions;
		`,
	); err != nil {
		return AllTimeStats{}, err
	}

	return totalStats, nil
}

// GetLast24hDurationStats retrieves total work and break durations for the past 24 hours.
func (r *SessionRepo) GetLast24hDurationStats() (PeriodStats, error) {
	cutoff := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)

	var stats PeriodStats
	if err := r.db.Get(
		&stats,
		`
		SELECT
			COALESCE(SUM(duration * (type = 'work')), 0)  AS work_duration,
			COALESCE(SUM(duration * (type = 'break')), 0) AS break_duration
		FROM sessions
		WHERE started_at >= ?;
		`,
		cutoff,
	); err != nil {
		return PeriodStats{}, err
	}

	return stats, nil
}

// GetWeeklyDurationStats retrieves total work and break durations for the past 7 days.
func (r *SessionRepo) GetWeeklyDurationStats() (PeriodStats, error) {
	today := time.Now()
	firstDay := today.AddDate(0, 0, -6)
	return r.getPeriodStats(firstDay, today)
}

func (r *SessionRepo) getPeriodStats(from, to time.Time) (PeriodStats, error) {
	fromStr := from.Format(DateFormat)
	toStr := to.Format(DateFormat)

	var stats PeriodStats
	if err := r.db.Get(
		&stats,
		`
		SELECT
			COALESCE(SUM(duration * (type = 'work')), 0)  AS work_duration,
			COALESCE(SUM(duration * (type = 'break')), 0) AS break_duration
		FROM sessions
		WHERE date(started_at, 'localtime') BETWEEN ? AND ?;
		`,
		fromStr, toStr,
	); err != nil {
		return PeriodStats{}, err
	}

	return stats, nil
}

// GetWeeklyStats retrieves daily work duration statistics for the past 7 days.
func (r *SessionRepo) GetWeeklyStats() ([]DailyStat, error) {
	today := time.Now()
	firstDay := today.AddDate(0, 0, -6)

	return r.getDailyStats(firstDay, today)
}

// GetLastMonthsStats retrieves daily work duration statistics for the past specified number of months.
func (r *SessionRepo) GetLastMonthsStats(numberOfMonths int) ([]DailyStat, error) {
	today := time.Now()
	firstDay := today.AddDate(0, -numberOfMonths, -today.Day()+1)

	return r.getDailyStats(firstDay, today)
}

// GetStreakStats calculates the current and best streaks of consecutive work days.
// A streak is consecutive days with at least one 'work' session.
func (r *SessionRepo) GetStreakStats() (StreakStats, error) {
	var dates []string

	if err := r.db.Select(
		&dates,
		`
		SELECT DISTINCT date(started_at, 'localtime') AS day
		FROM sessions
		WHERE type = 'work'
		ORDER BY day DESC;
		`,
	); err != nil {
		return StreakStats{}, err
	}

	return calculateStreak(dates), nil
}

// GetTodayHourlyStats retrieves work duration broken down by hour for today.
// Session durations are split across hour boundaries so that each hour
// only counts the time actually spent within that hour.
func (r *SessionRepo) GetTodayHourlyStats() ([]HourlyStat, error) {
	today := time.Now().Format(DateFormat)

	var sessions []struct {
		StartedAt string        `db:"started_at"`
		Duration  time.Duration `db:"duration"`
	}

	if err := r.db.Select(
		&sessions,
		`
		SELECT started_at, duration
		FROM sessions
		WHERE date(started_at, 'localtime') = ? AND type = 'work'
		ORDER BY started_at;
		`,
		today,
	); err != nil {
		return nil, err
	}

	hourly := make(map[int]time.Duration)

	for _, s := range sessions {
		startedAt, err := time.Parse(time.RFC3339, s.StartedAt)
		if err != nil {
			continue
		}
		startedAt = startedAt.Local()
		remaining := s.Duration

		for remaining > 0 {
			hour := startedAt.Hour()
			nextHour := time.Date(
				startedAt.Year(), startedAt.Month(), startedAt.Day(),
				hour+1, 0, 0, 0, startedAt.Location(),
			)
			timeLeftInHour := nextHour.Sub(startedAt)

			if remaining <= timeLeftInHour {
				hourly[hour] += remaining
				remaining = 0
			} else {
				hourly[hour] += timeLeftInHour
				remaining -= timeLeftInHour
				startedAt = nextHour
			}
		}
	}

	stats := make([]HourlyStat, 0, len(hourly))
	for hour, dur := range hourly {
		stats = append(stats, HourlyStat{Hour: hour, WorkDuration: dur})
	}

	return normalizeHourlyStats(stats), nil
}

// normalizeHourlyStats ensures there is an entry for each hour 0-23.
func normalizeHourlyStats(stats []HourlyStat) []HourlyStat {
	m := make(map[int]HourlyStat)
	for _, s := range stats {
		m[s.Hour] = s
	}

	normalized := make([]HourlyStat, 24)
	for h := 0; h < 24; h++ {
		normalized[h] = HourlyStat{
			Hour:         h,
			WorkDuration: m[h].WorkDuration,
		}
	}

	return normalized
}

// retrieves daily work duration statistics between the specified dates.
// from and to are inclusive.
// The results are normalized to include all days in the range.
func (r *SessionRepo) getDailyStats(from, to time.Time) ([]DailyStat, error) {
	fromStr := from.Format(DateFormat)
	toStr := to.Format(DateFormat)

	var stats []DailyStat

	if err := r.db.Select(
		&stats,
		`
		SELECT
			date(started_at, 'localtime') AS day,
			COALESCE(SUM(duration * (type = 'work')), 0) AS work_duration
		FROM sessions
		WHERE date(started_at, 'localtime') BETWEEN ? AND ?
		GROUP BY day
		ORDER BY day;
		`,
		fromStr, toStr,
	); err != nil {
		return nil, err
	}

	return r.normalizeStats(from, to, stats), nil
}

// ensures that there is a DailyStat entry for each day
func (r *SessionRepo) normalizeStats(from, to time.Time, stats []DailyStat) []DailyStat {
	m := make(map[string]DailyStat)

	for _, stat := range stats {
		m[stat.Date] = stat
	}

	var normalized []DailyStat
	current := from
	for !current.After(to) {
		day := current.Format(DateFormat)

		normalized = append(normalized, DailyStat{
			Date:         day,
			WorkDuration: m[day].WorkDuration,
		})

		current = current.AddDate(0, 0, 1) // next day
	}

	return normalized
}
