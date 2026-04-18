package service

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"github.com/robinlant/dutyround/internal/domain"
	"github.com/robinlant/dutyround/internal/i18n"
	"github.com/robinlant/dutyround/internal/repository"
)

// emailLang is the language used for email notifications sent by the background job.
// Since there is no per-user language preference stored in the database, we default
// to German as this is a German-company internal application.
const emailLang = "de"

type EmailService struct {
	settings       *SettingsService
	emailLog       repository.EmailLogRepository
	users          repository.UserRepository
	occurrences    repository.OccurrenceRepository
	participations repository.ParticipationRepository
	stopCh         chan struct{}
	stopOnce       sync.Once
}

func NewEmailService(
	settings *SettingsService,
	emailLog repository.EmailLogRepository,
	users repository.UserRepository,
	occurrences repository.OccurrenceRepository,
	participations repository.ParticipationRepository,
) *EmailService {
	return &EmailService{
		settings:       settings,
		emailLog:       emailLog,
		users:          users,
		occurrences:    occurrences,
		participations: participations,
		stopCh:         make(chan struct{}),
	}
}

// StartBackgroundJob starts a ticker that runs the notification logic periodically.
func (s *EmailService) StartBackgroundJob(interval time.Duration) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("email: background job panicked", "panic", r)
			}
		}()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Run once at startup after a short delay
		time.Sleep(10 * time.Second)
		s.safeRunCycle()

		for {
			select {
			case <-ticker.C:
				s.safeRunCycle()
			case <-s.stopCh:
				slog.Info("email: background job stopped")
				return
			}
		}
	}()
	slog.Info("email: background job started", "interval", interval)
}

// safeRunCycle runs the notification cycle with panic recovery.
func (s *EmailService) safeRunCycle() {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("email: notification cycle panicked", "panic", r)
		}
	}()
	s.runNotificationCycle()
}

// Stop stops the background job. Safe to call multiple times.
func (s *EmailService) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
}

func (s *EmailService) runNotificationCycle() {
	ctx := context.Background()

	config, err := s.settings.GetEmailConfig(ctx)
	if err != nil {
		slog.Error("email: failed to load config", "error", err)
		return
	}
	if !config.Enabled {
		return
	}
	if config.SMTPHost == "" || config.SenderEmail == "" {
		return
	}

	allUsers, err := s.users.FindAll(ctx)
	if err != nil {
		slog.Error("email: failed to load users", "error", err)
		return
	}

	allOccs, err := s.occurrences.FindAll(ctx)
	if err != nil {
		slog.Error("email: failed to load occurrences", "error", err)
		return
	}

	counts, err := s.participations.CountAllByOccurrence(ctx)
	if err != nil {
		slog.Error("email: failed to load participation counts", "error", err)
		return
	}

	now := time.Now()
	reminderDeadline := now.AddDate(0, 0, config.UpcomingReminderDays)

	// Find upcoming occurrences that are under min_participants within the reminder window.
	var unfilledOccs []domain.Occurrence
	for _, o := range allOccs {
		if o.Date.After(now) && !o.Date.After(reminderDeadline) {
			count := counts[o.ID]
			if count < o.MinParticipants {
				unfilledOccs = append(unfilledOccs, o)
			}
		}
	}

	for _, user := range allUsers {
		// Admins do not receive these emails
		if user.Role == domain.RoleAdmin {
			continue
		}

		// Check daily limit
		sentToday, err := s.emailLog.CountSentToday(ctx, user.ID)
		if err != nil {
			slog.Error("email: failed to count sent emails", "user_id", user.ID, "error", err)
			continue
		}
		if sentToday >= config.MaxEmailsPerDay {
			continue
		}

		var emailSent bool

		if user.Role == domain.RoleParticipant {
			// Send notifications about occurrences that are genuinely new since the last
			// time this user was notified. Uses created_at to avoid re-notifying about
			// old open occurrences every cycle.
			lastNewOccSent, _ := s.emailLog.LastSentAtByType(ctx, user.ID, "new_occurrence")
			// For users who have never been notified, use 24h ago as the threshold to
			// avoid a flood of emails on first run after migration.
			if lastNewOccSent.IsZero() {
				lastNewOccSent = now.Add(-24 * time.Hour)
			}

			var newOccs []domain.Occurrence
			for _, o := range allOccs {
				if o.Date.After(now) && o.CreatedAt.After(lastNewOccSent) {
					count := counts[o.ID]
					if count < o.MaxParticipants {
						newOccs = append(newOccs, o)
					}
				}
			}

			if len(newOccs) > 0 {
				lastSent, _ := s.emailLog.LastSentAt(ctx, user.ID)
				if time.Since(lastSent) > time.Hour {
					if err := s.sendNewOccurrenceDigest(config, user, newOccs, counts); err != nil {
						slog.Error("email: send new_occurrence digest failed", "user_id", user.ID, "error", err)
					} else {
						slog.Info("email: sent new_occurrence digest", "user_id", user.ID)
						s.emailLog.LogSent(ctx, user.ID, "new_occurrence")
						emailSent = true
					}
				}
			}

			// Send upcoming unfilled spots to participants (only if we haven't already sent today)
			if !emailSent && len(unfilledOccs) > 0 {
				lastSent, _ := s.emailLog.LastSentAt(ctx, user.ID)
				if time.Since(lastSent) > time.Hour {
					if err := s.sendUnfilledParticipantNotification(config, user, unfilledOccs, counts); err != nil {
						slog.Error("email: send unfilled_participant notification failed", "user_id", user.ID, "error", err)
					} else {
						s.emailLog.LogSent(ctx, user.ID, "unfilled_participant")
					}
				}
			}
		}

		if user.Role == domain.RoleOrganizer {
			// Send upcoming unfilled spots to organizers
			if len(unfilledOccs) > 0 {
				lastSent, _ := s.emailLog.LastSentAt(ctx, user.ID)
				if time.Since(lastSent) > time.Hour {
					if err := s.sendUnfilledOrganizerNotification(config, user, unfilledOccs, counts); err != nil {
						slog.Error("email: send unfilled_organizer notification failed", "user_id", user.ID, "error", err)
					} else {
						s.emailLog.LogSent(ctx, user.ID, "unfilled_organizer")
					}
				}
			}
		}
	}
}

func (s *EmailService) sendNewOccurrenceDigest(config EmailConfig, user domain.User, occs []domain.Occurrence, counts map[int64]int) error {
	subject := i18n.T(emailLang, "email.subjectNew")
	body := buildNewOccurrenceEmail(emailLang, user.Name, occs, counts)
	return s.sendEmail(config, user.Email, subject, body)
}

func (s *EmailService) sendUnfilledParticipantNotification(config EmailConfig, user domain.User, occs []domain.Occurrence, counts map[int64]int) error {
	subject := i18n.T(emailLang, "email.subjectUnfilledPart")
	body := buildUnfilledParticipantEmail(emailLang, user.Name, occs, counts)
	return s.sendEmail(config, user.Email, subject, body)
}

func (s *EmailService) sendUnfilledOrganizerNotification(config EmailConfig, user domain.User, occs []domain.Occurrence, counts map[int64]int) error {
	subject := i18n.T(emailLang, "email.subjectUnfilledOrg")
	body := buildUnfilledOrganizerEmail(emailLang, user.Name, occs, counts)
	return s.sendEmail(config, user.Email, subject, body)
}

// sanitizeHeader removes CR/LF characters to prevent email header injection.
func sanitizeHeader(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	return s
}

func (s *EmailService) sendEmail(config EmailConfig, to, subject, htmlBody string) error {
	from := config.SenderEmail
	addr := net.JoinHostPort(config.SMTPHost, fmt.Sprintf("%d", config.SMTPPort))

	var msg strings.Builder
	fmt.Fprintf(&msg, "From: %s <%s>\r\n", sanitizeHeader(config.SenderName), sanitizeHeader(from))
	fmt.Fprintf(&msg, "To: %s\r\n", sanitizeHeader(to))
	fmt.Fprintf(&msg, "Subject: %s\r\n", sanitizeHeader(subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)

	// Use a dialer with timeout to prevent hanging on unreachable SMTP servers
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("SMTP connection failed: %w", err)
	}
	conn.SetDeadline(time.Now().Add(30 * time.Second))

	client, err := smtp.NewClient(conn, config.SMTPHost)
	if err != nil {
		conn.Close()
		return fmt.Errorf("SMTP handshake failed: %w", err)
	}
	defer client.Close()

	auth := smtp.PlainAuth("", config.SMTPUsername, config.SMTPPassword, config.SMTPHost)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT TO failed: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed: %w", err)
	}
	if _, err := w.Write([]byte(msg.String())); err != nil {
		return fmt.Errorf("SMTP write failed: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("SMTP send failed: %w", err)
	}
	return client.Quit()
}

// SendTestEmail sends a test email to verify the SMTP configuration.
func (s *EmailService) SendTestEmail(ctx context.Context, toEmail string) error {
	config, err := s.settings.GetEmailConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load email config: %w", err)
	}
	if config.SMTPHost == "" || config.SenderEmail == "" {
		return fmt.Errorf("SMTP not configured")
	}

	subject := i18n.T(emailLang, "email.testSubject")
	body := buildTestEmail(emailLang)
	return s.sendEmail(config, toEmail, subject, body)
}

// --- Email templates (all user content is HTML-escaped) ---

func emailWrapper(content string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="margin:0;padding:0;background:#0d1117;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;">
<table width="100%%" cellpadding="0" cellspacing="0" style="background:#0d1117;padding:32px 16px;">
<tr><td align="center">
<table width="600" cellpadding="0" cellspacing="0" style="background:#161b22;border:1px solid #30363d;border-radius:10px;overflow:hidden;">
  <tr>
    <td style="background:#dc2626;padding:20px 24px;">
      <span style="font-size:18px;font-weight:700;color:#fff;letter-spacing:-0.01em;">&#9889; DutyRound</span>
    </td>
  </tr>
  <tr>
    <td style="padding:24px;">
      %s
    </td>
  </tr>
  <tr>
    <td style="padding:16px 24px;border-top:1px solid #30363d;">
      <span style="font-size:11px;color:#8b949e;">%s</span>
    </td>
  </tr>
</table>
</td></tr>
</table>
</body>
</html>`, content, i18n.T(emailLang, "email.footer"))
}

func buildNewOccurrenceEmail(lang, userName string, occs []domain.Occurrence, counts map[int64]int) string {
	var rows strings.Builder
	for _, o := range occs {
		count := counts[o.ID]
		spotsLeft := max(o.MaxParticipants-count, 0)
		fmt.Fprintf(&rows, `
      <tr>
        <td style="padding:10px 12px;border-bottom:1px solid #30363d;color:#e6edf3;font-size:13px;">%s</td>
        <td style="padding:10px 12px;border-bottom:1px solid #30363d;color:#8b949e;font-size:13px;">%s</td>
        <td style="padding:10px 12px;border-bottom:1px solid #30363d;color:#8b949e;font-size:13px;text-align:center;">%d/%d</td>
        <td style="padding:10px 12px;border-bottom:1px solid #30363d;font-size:13px;text-align:center;">
          <span style="color:%s;font-weight:600;">%d %s</span>
        </td>
      </tr>`,
			html.EscapeString(o.Title),
			o.Date.Format("02.01.2006 15:04"),
			count, o.MaxParticipants,
			spotColor(spotsLeft, o.MinParticipants-count),
			spotsLeft,
			i18n.T(lang, "email.left"),
		)
	}

	content := fmt.Sprintf(`
      <p style="color:#e6edf3;font-size:15px;margin:0 0 8px;">%s %s,</p>
      <p style="color:#8b949e;font-size:13px;margin:0 0 20px;">%s</p>
      <table width="100%%" cellpadding="0" cellspacing="0" style="border:1px solid #30363d;border-radius:6px;overflow:hidden;">
        <tr style="background:#21262d;">
          <th style="padding:8px 12px;text-align:left;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">%s</th>
          <th style="padding:8px 12px;text-align:left;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">%s</th>
          <th style="padding:8px 12px;text-align:center;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">%s</th>
          <th style="padding:8px 12px;text-align:center;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">%s</th>
        </tr>
        %s
      </table>`,
		i18n.T(lang, "email.hi"),
		html.EscapeString(userName),
		i18n.T(lang, "email.newOccAvailable"),
		i18n.T(lang, "email.headerOccurrence"),
		i18n.T(lang, "email.headerDate"),
		i18n.T(lang, "email.headerSignedUp"),
		i18n.T(lang, "email.headerSpots"),
		rows.String(),
	)

	return emailWrapper(content)
}

func buildUnfilledParticipantEmail(lang, userName string, occs []domain.Occurrence, counts map[int64]int) string {
	var rows strings.Builder
	for _, o := range occs {
		count := counts[o.ID]
		needed := max(o.MinParticipants-count, 0)
		daysUntil := int(time.Until(o.Date).Hours() / 24)
		fmt.Fprintf(&rows, `
      <tr>
        <td style="padding:10px 12px;border-bottom:1px solid #30363d;color:#e6edf3;font-size:13px;">%s</td>
        <td style="padding:10px 12px;border-bottom:1px solid #30363d;color:#8b949e;font-size:13px;">%s</td>
        <td style="padding:10px 12px;border-bottom:1px solid #30363d;font-size:13px;text-align:center;">
          <span style="color:#f85149;font-weight:600;">%d %s</span>
        </td>
        <td style="padding:10px 12px;border-bottom:1px solid #30363d;color:#d29922;font-size:13px;text-align:center;font-weight:600;">%d %s</td>
      </tr>`,
			html.EscapeString(o.Title),
			o.Date.Format("02.01.2006 15:04"),
			needed,
			i18n.T(lang, "email.needed"),
			daysUntil,
			i18n.T(lang, "email.days"),
		)
	}

	content := fmt.Sprintf(`
      <p style="color:#e6edf3;font-size:15px;margin:0 0 8px;">%s %s,</p>
      <p style="color:#8b949e;font-size:13px;margin:0 0 20px;">%s</p>
      <table width="100%%" cellpadding="0" cellspacing="0" style="border:1px solid #30363d;border-radius:6px;overflow:hidden;">
        <tr style="background:#21262d;">
          <th style="padding:8px 12px;text-align:left;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">%s</th>
          <th style="padding:8px 12px;text-align:left;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">%s</th>
          <th style="padding:8px 12px;text-align:center;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">%s</th>
          <th style="padding:8px 12px;text-align:center;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">%s</th>
        </tr>
        %s
      </table>`,
		i18n.T(lang, "email.hi"),
		html.EscapeString(userName),
		i18n.T(lang, "email.unfilledParticipantMsg"),
		i18n.T(lang, "email.headerOccurrence"),
		i18n.T(lang, "email.headerDate"),
		i18n.T(lang, "email.headerPeopleNeeded"),
		i18n.T(lang, "email.headerDaysUntil"),
		rows.String(),
	)

	return emailWrapper(content)
}

func buildUnfilledOrganizerEmail(lang, userName string, occs []domain.Occurrence, counts map[int64]int) string {
	var rows strings.Builder
	for _, o := range occs {
		count := counts[o.ID]
		needed := max(o.MinParticipants-count, 0)
		daysUntil := int(time.Until(o.Date).Hours() / 24)
		fmt.Fprintf(&rows, `
      <tr>
        <td style="padding:10px 12px;border-bottom:1px solid #30363d;color:#e6edf3;font-size:13px;">%s</td>
        <td style="padding:10px 12px;border-bottom:1px solid #30363d;color:#8b949e;font-size:13px;">%s</td>
        <td style="padding:10px 12px;border-bottom:1px solid #30363d;color:#8b949e;font-size:13px;text-align:center;">%d/%d</td>
        <td style="padding:10px 12px;border-bottom:1px solid #30363d;font-size:13px;text-align:center;">
          <span style="color:#f85149;font-weight:600;">%d %s</span>
        </td>
        <td style="padding:10px 12px;border-bottom:1px solid #30363d;color:#d29922;font-size:13px;text-align:center;font-weight:600;">%d %s</td>
      </tr>`,
			html.EscapeString(o.Title),
			o.Date.Format("02.01.2006 15:04"),
			count, o.MinParticipants,
			needed,
			i18n.T(lang, "email.needed"),
			daysUntil,
			i18n.T(lang, "email.days"),
		)
	}

	content := fmt.Sprintf(`
      <p style="color:#e6edf3;font-size:15px;margin:0 0 8px;">%s %s,</p>
      <p style="color:#8b949e;font-size:13px;margin:0 0 20px;">%s</p>
      <table width="100%%" cellpadding="0" cellspacing="0" style="border:1px solid #30363d;border-radius:6px;overflow:hidden;">
        <tr style="background:#21262d;">
          <th style="padding:8px 12px;text-align:left;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">%s</th>
          <th style="padding:8px 12px;text-align:left;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">%s</th>
          <th style="padding:8px 12px;text-align:center;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">%s</th>
          <th style="padding:8px 12px;text-align:center;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">%s</th>
          <th style="padding:8px 12px;text-align:center;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">%s</th>
        </tr>
        %s
      </table>`,
		i18n.T(lang, "email.hi"),
		html.EscapeString(userName),
		i18n.T(lang, "email.unfilledOrganizerMsg"),
		i18n.T(lang, "email.headerOccurrence"),
		i18n.T(lang, "email.headerDate"),
		i18n.T(lang, "email.headerSignedUp"),
		i18n.T(lang, "email.headerStillNeeded"),
		i18n.T(lang, "email.headerDaysUntil"),
		rows.String(),
	)

	return emailWrapper(content)
}

func buildTestEmail(lang string) string {
	content := fmt.Sprintf(`
      <p style="color:#e6edf3;font-size:15px;margin:0 0 8px;">%s</p>
      <p style="color:#8b949e;font-size:13px;margin:0 0 20px;">%s</p>
      <div style="background:#21262d;border:1px solid #30363d;border-radius:6px;padding:16px;text-align:center;">
        <span style="color:#3fb950;font-size:14px;font-weight:600;">&#10003; %s</span>
      </div>`,
		i18n.T(lang, "email.testTitle"),
		i18n.T(lang, "email.testBody"),
		i18n.T(lang, "email.testVerified"),
	)

	return emailWrapper(content)
}

func spotColor(spotsLeft, needed int) string {
	if needed > 0 {
		return "#f85149" // danger
	}
	if spotsLeft <= 2 {
		return "#d29922" // warning
	}
	return "#3fb950" // success
}
