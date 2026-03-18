package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/robinlant/occurance-management/internal/domain"
)

type ParticipationRepository struct {
	db *sql.DB
}

func NewParticipationRepository(db *sql.DB) *ParticipationRepository {
	return &ParticipationRepository{db: db}
}

func (r *ParticipationRepository) FindByID(ctx context.Context, id int64) (domain.Participation, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, occurrence_id FROM participations WHERE id = ?`, id)
	var p domain.Participation
	err := row.Scan(&p.ID, &p.UserID, &p.OccurrenceID)
	return p, err
}

func (r *ParticipationRepository) FindByOccurrence(ctx context.Context, occurrenceID int64) ([]domain.Participation, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, occurrence_id FROM participations WHERE occurrence_id = ?`, occurrenceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanParticipations(rows)
}

func (r *ParticipationRepository) FindByUser(ctx context.Context, userID int64) ([]domain.Participation, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, occurrence_id FROM participations WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanParticipations(rows)
}

func (r *ParticipationRepository) CountByUser(ctx context.Context, userID int64) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM participations WHERE user_id = ?`, userID,
	).Scan(&count)
	return count, err
}

func (r *ParticipationRepository) CountByUserInRange(ctx context.Context, userID int64, from, to time.Time) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM participations p
		JOIN occurrences o ON o.id = p.occurrence_id
		WHERE p.user_id = ? AND o.date >= ? AND o.date <= ?`,
		userID, from, to,
	).Scan(&count)
	return count, err
}

func (r *ParticipationRepository) ExistsForUserInDateRange(ctx context.Context, userID int64, from, to time.Time) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM participations p
		JOIN occurrences o ON o.id = p.occurrence_id
		WHERE p.user_id = ? AND o.date >= ? AND o.date <= ?`,
		userID, from, to,
	).Scan(&count)
	return count > 0, err
}

func (r *ParticipationRepository) Save(ctx context.Context, p domain.Participation) (domain.Participation, error) {
	if p.ID == 0 {
		res, err := r.db.ExecContext(ctx,
			`INSERT INTO participations (user_id, occurrence_id) VALUES (?, ?)`,
			p.UserID, p.OccurrenceID,
		)
		if err != nil {
			return p, err
		}
		p.ID, err = res.LastInsertId()
		return p, err
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE participations SET user_id = ?, occurrence_id = ? WHERE id = ?`,
		p.UserID, p.OccurrenceID, p.ID,
	)
	return p, err
}

func (r *ParticipationRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM participations WHERE id = ?`, id)
	return err
}

func (r *ParticipationRepository) DeleteByOccurrenceAndUser(ctx context.Context, occurrenceID, userID int64) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM participations WHERE occurrence_id = ? AND user_id = ?`, occurrenceID, userID)
	return err
}

func scanParticipations(rows *sql.Rows) ([]domain.Participation, error) {
	var list []domain.Participation
	for rows.Next() {
		var p domain.Participation
		if err := rows.Scan(&p.ID, &p.UserID, &p.OccurrenceID); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}
