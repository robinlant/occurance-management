package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/robinlant/dutyround/internal/domain"
	"github.com/robinlant/dutyround/internal/repository"
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

func (r *ParticipationRepository) CountByOccurrence(ctx context.Context, occurrenceID int64) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM participations WHERE occurrence_id = ?`, occurrenceID,
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
		userID, from.Format("2006-01-02"), to.Format("2006-01-02"),
	).Scan(&count)
	return count, err
}

func (r *ParticipationRepository) CountAllByOccurrence(ctx context.Context) (map[int64]int, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT occurrence_id, COUNT(*) FROM participations GROUP BY occurrence_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[int64]int)
	for rows.Next() {
		var occID int64
		var cnt int
		if err := rows.Scan(&occID, &cnt); err != nil {
			return nil, err
		}
		m[occID] = cnt
	}
	return m, rows.Err()
}

func (r *ParticipationRepository) CountByUserGroupedByDate(ctx context.Context, userID int64, from, to time.Time) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DATE(o.date) AS d, COUNT(*)
		FROM participations p
		JOIN occurrences o ON o.id = p.occurrence_id
		WHERE p.user_id = ? AND o.date >= ? AND o.date <= ?
		GROUP BY d`, userID, from.Format("2006-01-02"), to.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]int)
	for rows.Next() {
		var d string
		var cnt int
		if err := rows.Scan(&d, &cnt); err != nil {
			return nil, err
		}
		m[d] = cnt
	}
	return m, rows.Err()
}

func (r *ParticipationRepository) ExistsForUserInDateRange(ctx context.Context, userID int64, from, to time.Time) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM participations p
		JOIN occurrences o ON o.id = p.occurrence_id
		WHERE p.user_id = ? AND o.date >= ? AND o.date <= ?`,
		userID, from.Format("2006-01-02"), to.Format("2006-01-02"),
	).Scan(&count)
	return count > 0, err
}

func (r *ParticipationRepository) FindUsersByOccurrence(ctx context.Context, occurrenceID int64) ([]domain.User, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.name, u.email, u.role, u.password_hash
		FROM participations p
		JOIN users u ON u.id = p.user_id
		WHERE p.occurrence_id = ?`, occurrenceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *ParticipationRepository) LeaderboardAll(ctx context.Context, roles []domain.Role, groupID int64) ([]repository.LeaderboardRow, error) {
	placeholders, roleArgs := roleFilterArgs(roles)
	joinSQL := "LEFT JOIN participations p ON p.user_id = u.id"
	var joinArgs []any
	if groupID > 0 {
		joinSQL += " AND p.occurrence_id IN (SELECT id FROM occurrences WHERE group_id = ?)"
		joinArgs = append(joinArgs, groupID)
	}
	args := append(joinArgs, roleArgs...)
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.name, u.email, u.role, COUNT(p.id) as cnt
		FROM users u
		`+joinSQL+`
		WHERE u.role IN (`+placeholders+`)
		GROUP BY u.id
		ORDER BY cnt DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLeaderboardRows(rows)
}

func (r *ParticipationRepository) LeaderboardInRange(ctx context.Context, from, to time.Time, roles []domain.Role, groupID int64) ([]repository.LeaderboardRow, error) {
	placeholders, roleArgs := roleFilterArgs(roles)
	joinSQL := "LEFT JOIN participations p ON p.user_id = u.id AND p.occurrence_id IN (SELECT id FROM occurrences WHERE date >= ? AND date <= ?"
	joinArgs := []any{from.Format("2006-01-02"), to.Format("2006-01-02")}
	if groupID > 0 {
		joinSQL += " AND group_id = ?"
		joinArgs = append(joinArgs, groupID)
	}
	joinSQL += ")"
	args := append(joinArgs, roleArgs...)
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.name, u.email, u.role, COUNT(p.id) as cnt
		FROM users u
		`+joinSQL+`
		WHERE u.role IN (`+placeholders+`)
		GROUP BY u.id
		ORDER BY cnt DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLeaderboardRows(rows)
}

func roleFilterArgs(roles []domain.Role) (string, []any) {
	if len(roles) == 0 {
		return "'participant'", nil
	}
	placeholders := make([]string, len(roles))
	args := make([]any, len(roles))
	for i, r := range roles {
		placeholders[i] = "?"
		args[i] = string(r)
	}
	return strings.Join(placeholders, ","), args
}

// CountAndInsert atomically checks the participation count and inserts if not already signed up.
// This fixes the TOCTOU race condition in signup.
func (r *ParticipationRepository) CountAndInsert(ctx context.Context, occurrenceID, userID int64, maxParticipants int) (isOverMax bool, err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM participations WHERE occurrence_id = ?`, occurrenceID).Scan(&count); err != nil {
		return false, err
	}

	isOverMax = count >= maxParticipants

	_, err = tx.ExecContext(ctx, `INSERT INTO participations (user_id, occurrence_id) VALUES (?, ?)`, userID, occurrenceID)
	if err != nil {
		return false, err
	}

	return isOverMax, tx.Commit()
}

func (r *ParticipationRepository) ExportInRange(ctx context.Context, from, to time.Time, roles []domain.Role, groupID int64) ([]repository.ExportRow, error) {
	placeholders, roleArgs := roleFilterArgs(roles)
	whereSQL := "WHERE o.date >= ? AND o.date <= ? AND u.role IN (" + placeholders + ")"
	args := []any{from.Format("2006-01-02"), to.Format("2006-01-02")}
	args = append(args, roleArgs...)
	if groupID > 0 {
		whereSQL += " AND o.group_id = ?"
		args = append(args, groupID)
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT o.date, o.title, COALESCE(g.name, ''), u.name, u.email
		FROM participations p
		JOIN occurrences o ON o.id = p.occurrence_id
		JOIN users u ON u.id = p.user_id
		LEFT JOIN groups g ON g.id = o.group_id
		`+whereSQL+`
		ORDER BY o.date, o.title, u.name`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []repository.ExportRow
	for rows.Next() {
		var row repository.ExportRow
		if err := rows.Scan(&row.OccurrenceDate, &row.OccurrenceTitle, &row.GroupName, &row.UserName, &row.UserEmail); err != nil {
			return nil, err
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

func scanLeaderboardRows(rows *sql.Rows) ([]repository.LeaderboardRow, error) {
	var list []repository.LeaderboardRow
	for rows.Next() {
		var r repository.LeaderboardRow
		if err := rows.Scan(&r.UserID, &r.Name, &r.Email, &r.Role, &r.Count); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
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
