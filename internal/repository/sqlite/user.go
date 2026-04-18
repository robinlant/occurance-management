package sqlite

import (
	"context"
	"database/sql"

	"github.com/robinlant/dutyround/internal/domain"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByID(ctx context.Context, id int64) (domain.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, email, role, password_hash FROM users WHERE id = ?`, id)
	return scanUser(row)
}

func (r *UserRepository) FindByName(ctx context.Context, name string) (domain.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, email, role, password_hash FROM users WHERE name = ?`, name)
	return scanUser(row)
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, email, role, password_hash FROM users WHERE email = ?`, email)
	return scanUser(row)
}

func (r *UserRepository) FindAll(ctx context.Context) ([]domain.User, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, email, role, password_hash FROM users`)
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

func (r *UserRepository) SearchByNameOrEmail(ctx context.Context, query string, limit int) ([]domain.User, error) {
	pattern := "%" + query + "%"
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, email, role, password_hash FROM users WHERE name LIKE ? OR email LIKE ? LIMIT ?`,
		pattern, pattern, limit)
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

func (r *UserRepository) Save(ctx context.Context, u domain.User) (domain.User, error) {
	if u.ID == 0 {
		res, err := r.db.ExecContext(ctx,
			`INSERT INTO users (name, email, role, password_hash) VALUES (?, ?, ?, ?)`,
			u.Name, u.Email, u.Role, u.PasswordHash,
		)
		if err != nil {
			return u, err
		}
		u.ID, err = res.LastInsertId()
		return u, err
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET name = ?, email = ?, role = ?, password_hash = ? WHERE id = ?`,
		u.Name, u.Email, u.Role, u.PasswordHash, u.ID,
	)
	return u, err
}

func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	return err
}

func scanUser(row rowScanner) (domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.PasswordHash)
	return u, err
}
