package sqlite

import (
	"context"
	"database/sql"

	"github.com/robinlant/dutyround/internal/domain"
)

type OutOfOfficeRepository struct {
	db *sql.DB
}

func NewOutOfOfficeRepository(db *sql.DB) *OutOfOfficeRepository {
	return &OutOfOfficeRepository{db: db}
}

func (r *OutOfOfficeRepository) FindByID(ctx context.Context, id int64) (domain.OutOfOffice, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, from_date, to_date FROM out_of_office WHERE id = ?`, id)
	var o domain.OutOfOffice
	err := row.Scan(&o.ID, &o.UserID, &o.From, &o.To)
	return o, err
}

func (r *OutOfOfficeRepository) FindByUser(ctx context.Context, userID int64) ([]domain.OutOfOffice, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, from_date, to_date FROM out_of_office WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.OutOfOffice
	for rows.Next() {
		var o domain.OutOfOffice
		if err := rows.Scan(&o.ID, &o.UserID, &o.From, &o.To); err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

func (r *OutOfOfficeRepository) Save(ctx context.Context, o domain.OutOfOffice) (domain.OutOfOffice, error) {
	if o.ID == 0 {
		res, err := r.db.ExecContext(ctx,
			`INSERT INTO out_of_office (user_id, from_date, to_date) VALUES (?, ?, ?)`,
			o.UserID, o.From, o.To,
		)
		if err != nil {
			return o, err
		}
		o.ID, err = res.LastInsertId()
		return o, err
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE out_of_office SET user_id = ?, from_date = ?, to_date = ? WHERE id = ?`,
		o.UserID, o.From, o.To, o.ID,
	)
	return o, err
}

func (r *OutOfOfficeRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM out_of_office WHERE id = ?`, id)
	return err
}
