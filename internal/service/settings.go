package service

import (
	"context"
	"strconv"

	"github.com/robinlant/dutyround/internal/repository"
)

// EmailConfig holds the parsed email settings for convenience.
type EmailConfig struct {
	Enabled              bool
	SMTPHost             string
	SMTPPort             int
	SMTPUsername         string
	SMTPPassword         string
	SenderEmail          string
	SenderName           string
	MaxEmailsPerDay      int
	UpcomingReminderDays int
}

type SettingsService struct {
	settings repository.SettingsRepository
}

func NewSettingsService(settings repository.SettingsRepository) *SettingsService {
	return &SettingsService{settings: settings}
}

func (s *SettingsService) GetAll(ctx context.Context) (map[string]string, error) {
	return s.settings.GetAll(ctx)
}

func (s *SettingsService) SaveAll(ctx context.Context, settings map[string]string) error {
	return s.settings.SetMultiple(ctx, settings)
}

func (s *SettingsService) Get(ctx context.Context, key string) (string, error) {
	return s.settings.Get(ctx, key)
}

func (s *SettingsService) GetEmailConfig(ctx context.Context) (EmailConfig, error) {
	all, err := s.settings.GetAll(ctx)
	if err != nil {
		return EmailConfig{}, err
	}

	port, _ := strconv.Atoi(all["smtp_port"])
	if port == 0 {
		port = 587
	}
	maxEmails, _ := strconv.Atoi(all["max_emails_per_day"])
	if maxEmails == 0 {
		maxEmails = 1
	}
	reminderDays, _ := strconv.Atoi(all["upcoming_reminder_days"])
	if reminderDays == 0 {
		reminderDays = 3
	}

	return EmailConfig{
		Enabled:              all["email_enabled"] == "true",
		SMTPHost:             all["smtp_host"],
		SMTPPort:             port,
		SMTPUsername:         all["smtp_username"],
		SMTPPassword:         all["smtp_password"],
		SenderEmail:          all["sender_email"],
		SenderName:           all["sender_name"],
		MaxEmailsPerDay:      maxEmails,
		UpcomingReminderDays: reminderDays,
	}, nil
}
