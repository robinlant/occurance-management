package sqlite

import (
	"context"
	"database/sql"

	"github.com/robinlant/dutyround/internal/domain"
)

type CommentRepository struct {
	db *sql.DB
}

func NewCommentRepository(db *sql.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) FindByOccurrence(ctx context.Context, occurrenceID int64) ([]domain.Comment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id, c.occurrence_id, c.user_id, u.name, c.body, c.created_at
		FROM comments c
		JOIN users u ON u.id = c.user_id
		WHERE c.occurrence_id = ?
		ORDER BY c.created_at ASC`, occurrenceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Comment
	for rows.Next() {
		var c domain.Comment
		if err := rows.Scan(&c.ID, &c.OccurrenceID, &c.UserID, &c.UserName, &c.Body, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (r *CommentRepository) FindByID(ctx context.Context, id int64) (domain.Comment, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, occurrence_id, user_id, body, created_at FROM comments WHERE id = ?`, id)
	var c domain.Comment
	err := row.Scan(&c.ID, &c.OccurrenceID, &c.UserID, &c.Body, &c.CreatedAt)
	return c, err
}

func (r *CommentRepository) Save(ctx context.Context, c domain.Comment) (domain.Comment, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO comments (occurrence_id, user_id, body) VALUES (?, ?, ?)`,
		c.OccurrenceID, c.UserID, c.Body,
	)
	if err != nil {
		return c, err
	}
	c.ID, err = res.LastInsertId()
	return c, err
}

func (r *CommentRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM comments WHERE id = ?`, id)
	return err
}
