package service

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/robinlant/occurance-management/internal/domain"
	"github.com/robinlant/occurance-management/internal/repository"
)

var (
	ErrOccurrenceNotFound = errors.New("occurrence not found")
	ErrAlreadySignedUp    = errors.New("user already signed up for this occurrence")
	ErrNotSignedUp        = errors.New("user is not signed up for this occurrence")
	ErrUserOOO            = errors.New("user is out of office on the occurrence date")
	ErrAssignedDuringOOO  = errors.New("cannot set out-of-office: user is assigned to an occurrence on that date")
	ErrOccurrenceFull     = errors.New("occurrence is full and does not allow over-limit registrations")
)

type LeaderboardEntry struct {
	User  domain.User
	Count int
}

type AvailableUser struct {
	User  domain.User
	Count int // participations so far, for sorting
}

type OccurrenceService struct {
	occurrences    repository.OccurrenceRepository
	participations repository.ParticipationRepository
	users          repository.UserRepository
	ooo            repository.OutOfOfficeRepository
}

func NewOccurrenceService(
	occurrences repository.OccurrenceRepository,
	participations repository.ParticipationRepository,
	users repository.UserRepository,
	ooo repository.OutOfOfficeRepository,
) *OccurrenceService {
	return &OccurrenceService{
		occurrences:    occurrences,
		participations: participations,
		users:          users,
		ooo:            ooo,
	}
}

// --- Organizer ---

func (s *OccurrenceService) CreateOccurrence(ctx context.Context, o domain.Occurrence) (domain.Occurrence, error) {
	return s.occurrences.Save(ctx, o)
}

func (s *OccurrenceService) UpdateOccurrence(ctx context.Context, o domain.Occurrence) (domain.Occurrence, error) {
	return s.occurrences.Save(ctx, o)
}

func (s *OccurrenceService) DeleteOccurrence(ctx context.Context, id int64) error {
	return s.occurrences.Delete(ctx, id)
}

// AssignParticipant assigns a user to an occurrence. Blocked if the user is OOO on that date.
func (s *OccurrenceService) AssignParticipant(ctx context.Context, occurrenceID, userID int64) (isOverMax bool, err error) {
	occ, err := s.occurrences.FindByID(ctx, occurrenceID)
	if err != nil {
		return false, ErrOccurrenceNotFound
	}
	if err := s.checkOOO(ctx, userID, occ.Date); err != nil {
		return false, err
	}
	return s.participations.CountAndInsert(ctx, occ.ID, userID, occ.MaxParticipants)
}

// RemoveParticipant removes a user from an occurrence (organizer action).
func (s *OccurrenceService) RemoveParticipant(ctx context.Context, occurrenceID, userID int64) error {
	return s.participations.DeleteByOccurrenceAndUser(ctx, occurrenceID, userID)
}

// --- Participant ---

// SignUp signs a user up for an occurrence. Blocked if OOO or if full and over-limit not allowed.
func (s *OccurrenceService) SignUp(ctx context.Context, occurrenceID, userID int64) (isOverMax bool, err error) {
	occ, err := s.occurrences.FindByID(ctx, occurrenceID)
	if err != nil {
		return false, ErrOccurrenceNotFound
	}
	if err := s.checkOOO(ctx, userID, occ.Date); err != nil {
		return false, err
	}
	if !occ.AllowOverLimit {
		count, err := s.participations.CountByOccurrence(ctx, occ.ID)
		if err != nil {
			return false, err
		}
		if count >= occ.MaxParticipants {
			return false, ErrOccurrenceFull
		}
	}
	return s.participations.CountAndInsert(ctx, occ.ID, userID, occ.MaxParticipants)
}

// Withdraw removes a user's own participation. Always allowed.
func (s *OccurrenceService) Withdraw(ctx context.Context, occurrenceID, userID int64) error {
	return s.participations.DeleteByOccurrenceAndUser(ctx, occurrenceID, userID)
}

// --- Query ---

func (s *OccurrenceService) GetOccurrence(ctx context.Context, id int64) (domain.Occurrence, error) {
	return s.occurrences.FindByID(ctx, id)
}

func (s *OccurrenceService) ListOccurrences(ctx context.Context) ([]domain.Occurrence, error) {
	return s.occurrences.FindAll(ctx)
}

func (s *OccurrenceService) GetParticipantCountsByOccurrence(ctx context.Context) (map[int64]int, error) {
	return s.participations.CountAllByOccurrence(ctx)
}

func (s *OccurrenceService) ListOccurrencesByGroup(ctx context.Context, groupID int64) ([]domain.Occurrence, error) {
	return s.occurrences.FindByGroup(ctx, groupID)
}

func (s *OccurrenceService) ListOccurrencesByDate(ctx context.Context, date time.Time) ([]domain.Occurrence, error) {
	return s.occurrences.FindByDate(ctx, date)
}

func (s *OccurrenceService) ListOpenOccurrences(ctx context.Context) ([]domain.Occurrence, error) {
	return s.occurrences.FindOpenSpots(ctx)
}

func (s *OccurrenceService) GetUpcomingForUser(ctx context.Context, userID int64) ([]domain.Occurrence, error) {
	return s.occurrences.FindUpcomingByUser(ctx, userID, time.Now())
}

// GetParticipants returns users participating in an occurrence (single JOIN query, no N+1).
func (s *OccurrenceService) GetParticipants(ctx context.Context, occurrenceID int64) ([]domain.User, error) {
	return s.participations.FindUsersByOccurrence(ctx, occurrenceID)
}

// GetAvailableUsersForDate returns users who are not OOO on the given date,
// sorted by total participation count ascending (least done first).
func (s *OccurrenceService) GetAvailableUsersForDate(ctx context.Context, date time.Time) ([]AvailableUser, error) {
	allUsers, err := s.users.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	var available []AvailableUser
	for _, u := range allUsers {
		ooos, err := s.ooo.FindByUser(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		if isOOOOnDate(ooos, date) {
			continue
		}
		count, err := s.participations.CountByUser(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		available = append(available, AvailableUser{User: u, Count: count})
	}

	sort.Slice(available, func(i, j int) bool {
		return available[i].Count < available[j].Count
	})
	return available, nil
}

// GetLeaderboard returns all users with their participation count (single JOIN query, no N+1).
func (s *OccurrenceService) GetLeaderboard(ctx context.Context, from, to time.Time, roles []domain.Role, groupID int64) ([]LeaderboardEntry, error) {
	var rows []repository.LeaderboardRow
	var err error
	if from.IsZero() && to.IsZero() {
		rows, err = s.participations.LeaderboardAll(ctx, roles, groupID)
	} else {
		rows, err = s.participations.LeaderboardInRange(ctx, from, to, roles, groupID)
	}
	if err != nil {
		return nil, err
	}

	entries := make([]LeaderboardEntry, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, LeaderboardEntry{
			User: domain.User{
				ID:    r.UserID,
				Name:  r.Name,
				Email: r.Email,
				Role:  r.Role,
			},
			Count: r.Count,
		})
	}
	return entries, nil
}

// GetOccurrencesInRange returns all occurrences in a date range, optionally filtered by group.
func (s *OccurrenceService) GetOccurrencesInRange(ctx context.Context, from, to time.Time, groupID int64) ([]domain.Occurrence, error) {
	return s.occurrences.FindInRange(ctx, from, to, groupID)
}

// GetExportData returns detailed participation rows for CSV export.
func (s *OccurrenceService) GetExportData(ctx context.Context, from, to time.Time, roles []domain.Role, groupID int64) ([]repository.ExportRow, error) {
	return s.participations.ExportInRange(ctx, from, to, roles, groupID)
}

// GetParticipationCount returns the total participation count for a user.
func (s *OccurrenceService) GetParticipationCount(ctx context.Context, userID int64) (int, error) {
	return s.participations.CountByUser(ctx, userID)
}

// SearchOccurrences finds occurrences by title substring.
func (s *OccurrenceService) SearchOccurrences(ctx context.Context, query string, limit int) ([]domain.Occurrence, error) {
	return s.occurrences.FindByTitleLike(ctx, query, limit)
}

// GetActivityHeatmap returns participation counts grouped by date for a user.
func (s *OccurrenceService) GetActivityHeatmap(ctx context.Context, userID int64, from, to time.Time) (map[string]int, error) {
	return s.participations.CountByUserGroupedByDate(ctx, userID, from, to)
}

// ComputeOccStatus computes the status string for an occurrence given its participant count.
func ComputeOccStatus(o domain.Occurrence, count int) string {
	if count < o.MinParticipants {
		return "under"
	}
	if count > o.MaxParticipants {
		return "over"
	}
	return "good"
}

// --- Helpers ---

func (s *OccurrenceService) checkOOO(ctx context.Context, userID int64, date time.Time) error {
	ooos, err := s.ooo.FindByUser(ctx, userID)
	if err != nil {
		return err
	}
	if isOOOOnDate(ooos, date) {
		return ErrUserOOO
	}
	return nil
}

func isOOOOnDate(ooos []domain.OutOfOffice, date time.Time) bool {
	d := toDate(date)
	for _, o := range ooos {
		from := toDate(o.From)
		to := toDate(o.To)
		if !d.Before(from) && !d.After(to) {
			return true
		}
	}
	return false
}

func toDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
