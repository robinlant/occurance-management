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
	ErrOccurrenceNotFound   = errors.New("occurrence not found")
	ErrAlreadySignedUp      = errors.New("user already signed up for this occurrence")
	ErrNotSignedUp          = errors.New("user is not signed up for this occurrence")
	ErrUserOOO              = errors.New("user is out of office on the occurrence date")
	ErrAssignedDuringOOO    = errors.New("cannot set out-of-office: user is assigned to an occurrence on that date")
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
// Over-max is allowed — the caller receives isOverMax=true as a warning signal.
func (s *OccurrenceService) AssignParticipant(ctx context.Context, occurrenceID, userID int64) (isOverMax bool, err error) {
	occ, err := s.occurrences.FindByID(ctx, occurrenceID)
	if err != nil {
		return false, ErrOccurrenceNotFound
	}
	if err := s.checkOOO(ctx, userID, occ.Date); err != nil {
		return false, err
	}
	isOverMax, err = s.saveParticipation(ctx, occ, userID)
	return
}

// RemoveParticipant removes a user from an occurrence (organizer action).
func (s *OccurrenceService) RemoveParticipant(ctx context.Context, occurrenceID, userID int64) error {
	return s.participations.DeleteByOccurrenceAndUser(ctx, occurrenceID, userID)
}

// --- Participant ---

// SignUp signs a user up for an occurrence. Blocked if OOO. Over-max allowed with warning.
func (s *OccurrenceService) SignUp(ctx context.Context, occurrenceID, userID int64) (isOverMax bool, err error) {
	occ, err := s.occurrences.FindByID(ctx, occurrenceID)
	if err != nil {
		return false, ErrOccurrenceNotFound
	}
	if err := s.checkOOO(ctx, userID, occ.Date); err != nil {
		return false, err
	}
	isOverMax, err = s.saveParticipation(ctx, occ, userID)
	return
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

func (s *OccurrenceService) ListOpenOccurrences(ctx context.Context) ([]domain.Occurrence, error) {
	return s.occurrences.FindOpenSpots(ctx)
}

func (s *OccurrenceService) GetParticipants(ctx context.Context, occurrenceID int64) ([]domain.User, error) {
	participations, err := s.participations.FindByOccurrence(ctx, occurrenceID)
	if err != nil {
		return nil, err
	}
	users := make([]domain.User, 0, len(participations))
	for _, p := range participations {
		u, err := s.users.FindByID(ctx, p.UserID)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
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
		if u.Role == domain.RoleOrganizer || u.Role == domain.RoleAdmin {
			continue
		}
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

// GetLeaderboard returns all users with their participation count within the given date range,
// sorted descending. Pass zero-value times to get all-time counts.
func (s *OccurrenceService) GetLeaderboard(ctx context.Context, from, to time.Time) ([]LeaderboardEntry, error) {
	allUsers, err := s.users.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	entries := make([]LeaderboardEntry, 0, len(allUsers))
	for _, u := range allUsers {
		if u.Role == domain.RoleOrganizer || u.Role == domain.RoleAdmin {
			continue
		}
		var count int
		var err error
		if from.IsZero() && to.IsZero() {
			count, err = s.participations.CountByUser(ctx, u.ID)
		} else {
			count, err = s.participations.CountByUserInRange(ctx, u.ID, from, to)
		}
		if err != nil {
			return nil, err
		}
		entries = append(entries, LeaderboardEntry{User: u, Count: count})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Count > entries[j].Count
	})
	return entries, nil
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

func (s *OccurrenceService) saveParticipation(ctx context.Context, occ domain.Occurrence, userID int64) (isOverMax bool, err error) {
	existing, err := s.participations.FindByOccurrence(ctx, occ.ID)
	if err != nil {
		return false, err
	}
	for _, p := range existing {
		if p.UserID == userID {
			return false, ErrAlreadySignedUp
		}
	}
	isOverMax = len(existing) >= occ.MaxParticipants
	_, err = s.participations.Save(ctx, domain.Participation{
		UserID:       userID,
		OccurrenceID: occ.ID,
	})
	return isOverMax, err
}

func isOOOOnDate(ooos []domain.OutOfOffice, date time.Time) bool {
	d := date.Truncate(24 * time.Hour)
	for _, o := range ooos {
		from := o.From.Truncate(24 * time.Hour)
		to := o.To.Truncate(24 * time.Hour)
		if !d.Before(from) && !d.After(to) {
			return true
		}
	}
	return false
}
