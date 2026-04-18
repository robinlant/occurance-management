package sqlite

import (
	"context"
	"database/sql"
	"time"
)

type EmailLogRepository struct {
	db *sql.DB
}

func NewEmailLogRepository(db *sql.DB) *EmailLogRepository {
	return &EmailLogRepository{db: db}
}

func (r *EmailLogRepository) LogSent(ctx context.Context, userID int64, emailType string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO email_notifications_log (user_id, email_type, sent_at) VALUES (?, ?, ?)`,
		userID, emailType, time.Now().UTC())
	return err
}

func (r *EmailLogRepository) LastSentAt(ctx context.Context, userID int64) (time.Time, error) {
	var sentAt time.Time
	err := r.db.QueryRowContext(ctx,
		`SELECT sent_at FROM email_notifications_log WHERE user_id = ? ORDER BY sent_at DESC LIMIT 1`,
		userID).Scan(&sentAt)
	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}
	return sentAt, err
}

func (r *EmailLogRepository) LastSentAtByType(ctx context.Context, userID int64, emailType string) (time.Time, error) {
	var sentAt time.Time
	err := r.db.QueryRowContext(ctx,
		`SELECT sent_at FROM email_notifications_log WHERE user_id = ? AND email_type = ? ORDER BY sent_at DESC LIMIT 1`,
		userID, emailType).Scan(&sentAt)
	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}
	return sentAt, err
}

func (r *EmailLogRepository) CountSentToday(ctx context.Context, userID int64) (int, error) {
	todayStart := time.Now().UTC().Truncate(24 * time.Hour)
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM email_notifications_log WHERE user_id = ? AND sent_at >= ?`,
		userID, todayStart).Scan(&count)
	return count, err
}
