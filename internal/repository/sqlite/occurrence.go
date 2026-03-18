package sqlite

import (
	"context"
	"database/sql"

	"github.com/robinlant/occurance-management/internal/domain"
)

type OccurrenceRepository struct {
	db *sql.DB
}

func NewOccurrenceRepository(db *sql.DB) *OccurrenceRepository {
	return &OccurrenceRepository{db: db}
}

func (r *OccurrenceRepository) FindByID(ctx context.Context, id int64) (domain.Occurrence, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, group_id, title, description, date, min_participants, max_participants FROM occurrences WHERE id = ?`, id)
	return scanOccurrence(row)
}

func (r *OccurrenceRepository) FindAll(ctx context.Context) ([]domain.Occurrence, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, group_id, title, description, date, min_participants, max_participants FROM occurrences ORDER BY date`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOccurrences(rows)
}

func (r *OccurrenceRepository) FindByGroup(ctx context.Context, groupID int64) ([]domain.Occurrence, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, group_id, title, description, date, min_participants, max_participants FROM occurrences WHERE group_id = ? ORDER BY date`,
		groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOccurrences(rows)
}

func (r *OccurrenceRepository) FindOpenSpots(ctx context.Context) ([]domain.Occurrence, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT o.id, o.group_id, o.title, o.description, o.date, o.min_participants, o.max_participants
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

func (r *OccurrenceRepository) Save(ctx context.Context, o domain.Occurrence) (domain.Occurrence, error) {
	var groupID sql.NullInt64
	if o.GroupID != 0 {
		groupID = sql.NullInt64{Int64: o.GroupID, Valid: true}
	}
	if o.ID == 0 {
		res, err := r.db.ExecContext(ctx,
			`INSERT INTO occurrences (group_id, title, description, date, min_participants, max_participants) VALUES (?, ?, ?, ?, ?, ?)`,
			groupID, o.Title, o.Description, o.Date, o.MinParticipants, o.MaxParticipants,
		)
		if err != nil {
			return o, err
		}
		o.ID, err = res.LastInsertId()
		return o, err
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE occurrences SET group_id = ?, title = ?, description = ?, date = ?, min_participants = ?, max_participants = ? WHERE id = ?`,
		groupID, o.Title, o.Description, o.Date, o.MinParticipants, o.MaxParticipants, o.ID,
	)
	return o, err
}

func (r *OccurrenceRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM occurrences WHERE id = ?`, id)
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanOccurrence(row rowScanner) (domain.Occurrence, error) {
	var o domain.Occurrence
	var groupID sql.NullInt64
	err := row.Scan(&o.ID, &groupID, &o.Title, &o.Description, &o.Date, &o.MinParticipants, &o.MaxParticipants)
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
