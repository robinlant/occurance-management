package service

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"time"

	"github.com/robinlant/occurance-management/internal/domain"
	"github.com/robinlant/occurance-management/internal/repository"
)

type EmailService struct {
	settings       *SettingsService
	emailLog       repository.EmailLogRepository
	users          repository.UserRepository
	occurrences    repository.OccurrenceRepository
	participations repository.ParticipationRepository
	stopCh         chan struct{}
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
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Run once at startup after a short delay
		time.Sleep(10 * time.Second)
		s.runNotificationCycle()

		for {
			select {
			case <-ticker.C:
				s.runNotificationCycle()
			case <-s.stopCh:
				log.Println("[email] background job stopped")
				return
			}
		}
	}()
	log.Printf("[email] background job started (interval: %v)", interval)
}

// Stop stops the background job.
func (s *EmailService) Stop() {
	close(s.stopCh)
}

func (s *EmailService) runNotificationCycle() {
	ctx := context.Background()

	config, err := s.settings.GetEmailConfig(ctx)
	if err != nil {
		log.Printf("[email] failed to load config: %v", err)
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
		log.Printf("[email] failed to load users: %v", err)
		return
	}

	allOccs, err := s.occurrences.FindAll(ctx)
	if err != nil {
		log.Printf("[email] failed to load occurrences: %v", err)
		return
	}

	counts, err := s.participations.CountAllByOccurrence(ctx)
	if err != nil {
		log.Printf("[email] failed to load participation counts: %v", err)
		return
	}

	now := time.Now()
	reminderDeadline := now.AddDate(0, 0, config.UpcomingReminderDays)

	// Find new occurrences created in the last cycle (future occurrences with open spots)
	var newOccs []domain.Occurrence
	for _, o := range allOccs {
		if o.Date.After(now) {
			count := counts[o.ID]
			if count < o.MaxParticipants {
				newOccs = append(newOccs, o)
			}
		}
	}

	// Find upcoming occurrences that are under min_participants
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
			log.Printf("[email] failed to count emails for user %d: %v", user.ID, err)
			continue
		}
		if sentToday >= config.MaxEmailsPerDay {
			continue
		}

		var emailSent bool

		if user.Role == domain.RoleParticipant {
			// Send new occurrence notifications to participants
			if len(newOccs) > 0 {
				lastSent, _ := s.emailLog.LastSentAt(ctx, user.ID)
				// Only send if we haven't sent in the last hour
				if time.Since(lastSent) > time.Hour {
					if err := s.sendNewOccurrenceDigest(config, user, newOccs, counts); err != nil {
						log.Printf("[email] failed to send new occurrence digest to %s: %v", user.Email, err)
					} else {
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
						log.Printf("[email] failed to send unfilled notification to %s: %v", user.Email, err)
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
						log.Printf("[email] failed to send unfilled organizer notification to %s: %v", user.Email, err)
					} else {
						s.emailLog.LogSent(ctx, user.ID, "unfilled_organizer")
					}
				}
			}
		}
	}
}

func (s *EmailService) sendNewOccurrenceDigest(config EmailConfig, user domain.User, occs []domain.Occurrence, counts map[int64]int) error {
	subject := "New occurrences available - DutyRound"
	body := buildNewOccurrenceEmail(user.Name, occs, counts)
	return s.sendEmail(config, user.Email, subject, body)
}

func (s *EmailService) sendUnfilledParticipantNotification(config EmailConfig, user domain.User, occs []domain.Occurrence, counts map[int64]int) error {
	subject := "Upcoming occurrences need people - DutyRound"
	body := buildUnfilledParticipantEmail(user.Name, occs, counts)
	return s.sendEmail(config, user.Email, subject, body)
}

func (s *EmailService) sendUnfilledOrganizerNotification(config EmailConfig, user domain.User, occs []domain.Occurrence, counts map[int64]int) error {
	subject := "Upcoming occurrences still have free places - DutyRound"
	body := buildUnfilledOrganizerEmail(user.Name, occs, counts)
	return s.sendEmail(config, user.Email, subject, body)
}

func (s *EmailService) sendEmail(config EmailConfig, to, subject, htmlBody string) error {
	from := config.SenderEmail
	addr := fmt.Sprintf("%s:%d", config.SMTPHost, config.SMTPPort)

	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", config.SenderName, from)
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	var msg strings.Builder
	for k, v := range headers {
		fmt.Fprintf(&msg, "%s: %s\r\n", k, v)
	}
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)

	auth := smtp.PlainAuth("", config.SMTPUsername, config.SMTPPassword, config.SMTPHost)
	return smtp.SendMail(addr, auth, from, []string{to}, []byte(msg.String()))
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

	subject := "DutyRound - Test Email"
	body := buildTestEmail()
	return s.sendEmail(config, toEmail, subject, body)
}

// --- Email templates ---

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
      <span style="font-size:11px;color:#8b949e;">This is an automated notification from DutyRound. You can manage your notification preferences with your administrator.</span>
    </td>
  </tr>
</table>
</td></tr>
</table>
</body>
</html>`, content)
}

func buildNewOccurrenceEmail(userName string, occs []domain.Occurrence, counts map[int64]int) string {
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
          <span style="color:%s;font-weight:600;">%d left</span>
        </td>
      </tr>`,
			o.Title,
			o.Date.Format("02.01.2006 15:04"),
			count, o.MaxParticipants,
			spotColor(spotsLeft, o.MinParticipants-count),
			spotsLeft,
		)
	}

	content := fmt.Sprintf(`
      <p style="color:#e6edf3;font-size:15px;margin:0 0 8px;">Hi %s,</p>
      <p style="color:#8b949e;font-size:13px;margin:0 0 20px;">New occurrences are available. Sign up if you have time!</p>
      <table width="100%%" cellpadding="0" cellspacing="0" style="border:1px solid #30363d;border-radius:6px;overflow:hidden;">
        <tr style="background:#21262d;">
          <th style="padding:8px 12px;text-align:left;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">Occurrence</th>
          <th style="padding:8px 12px;text-align:left;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">Date</th>
          <th style="padding:8px 12px;text-align:center;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">Signed Up</th>
          <th style="padding:8px 12px;text-align:center;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">Spots</th>
        </tr>
        %s
      </table>`, userName, rows.String())

	return emailWrapper(content)
}

func buildUnfilledParticipantEmail(userName string, occs []domain.Occurrence, counts map[int64]int) string {
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
          <span style="color:#f85149;font-weight:600;">%d needed</span>
        </td>
        <td style="padding:10px 12px;border-bottom:1px solid #30363d;color:#d29922;font-size:13px;text-align:center;font-weight:600;">%d days</td>
      </tr>`,
			o.Title,
			o.Date.Format("02.01.2006 15:04"),
			needed,
			daysUntil,
		)
	}

	content := fmt.Sprintf(`
      <p style="color:#e6edf3;font-size:15px;margin:0 0 8px;">Hi %s,</p>
      <p style="color:#8b949e;font-size:13px;margin:0 0 20px;">The following upcoming occurrences still need people. Take one if you have time!</p>
      <table width="100%%" cellpadding="0" cellspacing="0" style="border:1px solid #30363d;border-radius:6px;overflow:hidden;">
        <tr style="background:#21262d;">
          <th style="padding:8px 12px;text-align:left;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">Occurrence</th>
          <th style="padding:8px 12px;text-align:left;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">Date</th>
          <th style="padding:8px 12px;text-align:center;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">People Needed</th>
          <th style="padding:8px 12px;text-align:center;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">Days Until</th>
        </tr>
        %s
      </table>`, userName, rows.String())

	return emailWrapper(content)
}

func buildUnfilledOrganizerEmail(userName string, occs []domain.Occurrence, counts map[int64]int) string {
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
          <span style="color:#f85149;font-weight:600;">%d needed</span>
        </td>
        <td style="padding:10px 12px;border-bottom:1px solid #30363d;color:#d29922;font-size:13px;text-align:center;font-weight:600;">%d days</td>
      </tr>`,
			o.Title,
			o.Date.Format("02.01.2006 15:04"),
			count, o.MinParticipants,
			needed,
			daysUntil,
		)
	}

	content := fmt.Sprintf(`
      <p style="color:#e6edf3;font-size:15px;margin:0 0 8px;">Hi %s,</p>
      <p style="color:#8b949e;font-size:13px;margin:0 0 20px;">The following upcoming occurrences still have free places that need to be filled.</p>
      <table width="100%%" cellpadding="0" cellspacing="0" style="border:1px solid #30363d;border-radius:6px;overflow:hidden;">
        <tr style="background:#21262d;">
          <th style="padding:8px 12px;text-align:left;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">Occurrence</th>
          <th style="padding:8px 12px;text-align:left;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">Date</th>
          <th style="padding:8px 12px;text-align:center;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">Signed Up</th>
          <th style="padding:8px 12px;text-align:center;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">Still Needed</th>
          <th style="padding:8px 12px;text-align:center;font-size:11px;font-weight:700;color:#8b949e;text-transform:uppercase;letter-spacing:0.06em;">Days Until</th>
        </tr>
        %s
      </table>`, userName, rows.String())

	return emailWrapper(content)
}

func buildTestEmail() string {
	content := `
      <p style="color:#e6edf3;font-size:15px;margin:0 0 8px;">Test Email</p>
      <p style="color:#8b949e;font-size:13px;margin:0 0 20px;">If you are reading this, your DutyRound email configuration is working correctly.</p>
      <div style="background:#21262d;border:1px solid #30363d;border-radius:6px;padding:16px;text-align:center;">
        <span style="color:#3fb950;font-size:14px;font-weight:600;">&#10003; SMTP configuration verified</span>
      </div>`

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
