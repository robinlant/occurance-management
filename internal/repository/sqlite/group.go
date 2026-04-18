package sqlite

import (
	"context"
	"database/sql"

	"github.com/robinlant/dutyround/internal/domain"
)

type GroupRepository struct {
	db *sql.DB
}

func NewGroupRepository(db *sql.DB) *GroupRepository {
	return &GroupRepository{db: db}
}

func (r *GroupRepository) FindByID(ctx context.Context, id int64) (domain.Group, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, name, color FROM groups WHERE id = ?`, id)
	var g domain.Group
	err := row.Scan(&g.ID, &g.Name, &g.Color)
	return g, err
}

func (r *GroupRepository) FindAll(ctx context.Context) ([]domain.Group, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, color FROM groups`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []domain.Group
	for rows.Next() {
		var g domain.Group
		if err := rows.Scan(&g.ID, &g.Name, &g.Color); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func (r *GroupRepository) Save(ctx context.Context, g domain.Group) (domain.Group, error) {
	if g.ID == 0 {
		res, err := r.db.ExecContext(ctx, `INSERT INTO groups (name, color) VALUES (?, ?)`, g.Name, g.Color)
		if err != nil {
			return g, err
		}
		g.ID, err = res.LastInsertId()
		return g, err
	}
	_, err := r.db.ExecContext(ctx, `UPDATE groups SET name = ?, color = ? WHERE id = ?`, g.Name, g.Color, g.ID)
	return g, err
}

func (r *GroupRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM groups WHERE id = ?`, id)
	return err
}
