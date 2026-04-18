package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/robinlant/dutyround/internal/domain"
)

type OccurrenceRepository struct {
	db *sql.DB
}

func NewOccurrenceRepository(db *sql.DB) *OccurrenceRepository {
	return &OccurrenceRepository{db: db}
}

const occurrenceCols = `id, group_id, title, description, date, min_participants, max_participants, allow_over_limit, recurrence_id, created_at`

func (r *OccurrenceRepository) FindByID(ctx context.Context, id int64) (domain.Occurrence, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+occurrenceCols+` FROM occurrences WHERE id = ?`, id)
	return scanOccurrence(row)
}

func (r *OccurrenceRepository) FindAll(ctx context.Context) ([]domain.Occurrence, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+occurrenceCols+` FROM occurrences ORDER BY date`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOccurrences(rows)
}

func (r *OccurrenceRepository) FindByGroup(ctx context.Context, groupID int64) ([]domain.Occurrence, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+occurrenceCols+` FROM occurrences WHERE group_id = ? ORDER BY date`,
		groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOccurrences(rows)
}

func (r *OccurrenceRepository) FindByDate(ctx context.Context, date time.Time) ([]domain.Occurrence, error) {
	dateStr := date.Format("2006-01-02")
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+occurrenceCols+` FROM occurrences WHERE DATE(date) = ? ORDER BY date`,
		dateStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOccurrences(rows)
}

func (r *OccurrenceRepository) FindOpenSpots(ctx context.Context) ([]domain.Occurrence, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT o.id, o.group_id, o.title, o.description, o.date, o.min_participants, o.max_participants, o.allow_over_limit, o.recurrence_id, o.created_at
		FROM occurrences o
		LEFT JOIN (
			SELECT occurrence_id, COUNT(*) as cnt
			FROM participations
			GROUP BY occurrence_id
		) p ON p.occurrence_id = o.id
		WHERE COALESCE(p.cnt, 0) < o.max_participants
		ORDER BY o.date
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOccurrences(rows)
}

func (r *OccurrenceRepository) FindUpcomingByUser(ctx context.Context, userID int64, from time.Time) ([]domain.Occurrence, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT o.id, o.group_id, o.title, o.description, o.date, o.min_participants, o.max_participants, o.allow_over_limit, o.recurrence_id, o.created_at
		FROM occurrences o
		JOIN participations p ON p.occurrence_id = o.id
		WHERE p.user_id = ? AND o.date >= ?
		ORDER BY o.date ASC`,
		userID, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOccurrences(rows)
}

func (r *OccurrenceRepository) FindAllByUser(ctx context.Context, userID int64) ([]domain.Occurrence, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT o.id, o.group_id, o.title, o.description, o.date, o.min_participants, o.max_participants, o.allow_over_limit, o.recurrence_id, o.created_at
		FROM occurrences o
		JOIN participations p ON p.occurrence_id = o.id
		WHERE p.user_id = ?
		ORDER BY o.date`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOccurrences(rows)
}

func (r *OccurrenceRepository) FindByTitleLike(ctx context.Context, query string, limit int) ([]domain.Occurrence, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+occurrenceCols+` FROM occurrences WHERE title LIKE ? ORDER BY date DESC LIMIT ?`,
		"%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOccurrences(rows)
}

func (r *OccurrenceRepository) FindInRange(ctx context.Context, from, to time.Time, groupID int64) ([]domain.Occurrence, error) {
	query := `SELECT ` + occurrenceCols + ` FROM occurrences WHERE date >= ? AND date <= ?`
	args := []any{from.Format("2006-01-02"), to.Format("2006-01-02")}
	if groupID > 0 {
		query += " AND group_id = ?"
		args = append(args, groupID)
	}
	query += " ORDER BY date, title"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOccurrences(rows)
}

func (r *OccurrenceRepository) Save(ctx context.Context, o domain.Occurrence) (domain.Occurrence, error) {
	var groupID sql.NullInt64
	if o.GroupID != 0 {
		groupID = sql.NullInt64{Int64: o.GroupID, Valid: true}
	}
	if o.ID == 0 {
		res, err := r.db.ExecContext(ctx,
			`INSERT INTO occurrences (group_id, title, description, date, min_participants, max_participants, allow_over_limit, recurrence_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			groupID, o.Title, o.Description, o.Date, o.MinParticipants, o.MaxParticipants, o.AllowOverLimit, o.RecurrenceID,
		)
		if err != nil {
			return o, err
		}
		o.ID, err = res.LastInsertId()
		return o, err
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE occurrences SET group_id = ?, title = ?, description = ?, date = ?, min_participants = ?, max_participants = ?, allow_over_limit = ?, recurrence_id = ? WHERE id = ?`,
		groupID, o.Title, o.Description, o.Date, o.MinParticipants, o.MaxParticipants, o.AllowOverLimit, o.RecurrenceID, o.ID,
	)
	return o, err
}

func (r *OccurrenceRepository) FindByRecurrenceID(ctx context.Context, recurrenceID string) ([]domain.Occurrence, error) {
	if recurrenceID == "" {
		return []domain.Occurrence{}, nil
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+occurrenceCols+` FROM occurrences WHERE recurrence_id = ? ORDER BY date`,
		recurrenceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOccurrences(rows)
}

func (r *OccurrenceRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM occurrences WHERE id = ?`, id)
	return err
}

func (r *OccurrenceRepository) DeleteByRecurrenceID(ctx context.Context, recurrenceID string) (int64, error) {
	if recurrenceID == "" {
		return 0, nil
	}
	res, err := r.db.ExecContext(ctx, `DELETE FROM occurrences WHERE recurrence_id = ?`, recurrenceID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *OccurrenceRepository) DeleteByRecurrenceIDFromDate(ctx context.Context, recurrenceID string, fromDate time.Time) (int64, error) {
	if recurrenceID == "" {
		return 0, nil
	}
	res, err := r.db.ExecContext(ctx, `DELETE FROM occurrences WHERE recurrence_id = ? AND date >= ?`, recurrenceID, fromDate)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanOccurrence(row rowScanner) (domain.Occurrence, error) {
	var o domain.Occurrence
	var groupID sql.NullInt64
	err := row.Scan(&o.ID, &groupID, &o.Title, &o.Description, &o.Date, &o.MinParticipants, &o.MaxParticipants, &o.AllowOverLimit, &o.RecurrenceID, &o.CreatedAt)
	if groupID.Valid {
		o.GroupID = groupID.Int64
	}
	return o, err
}

func scanOccurrences(rows *sql.Rows) ([]domain.Occurrence, error) {
	var list []domain.Occurrence
	for rows.Next() {
		o, err := scanOccurrence(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}
