package repository

import (
	"context"
	"time"

	"github.com/robinlant/dutyround/internal/domain"
)

type UserRepository interface {
	FindByID(ctx context.Context, id int64) (domain.User, error)
	FindByName(ctx context.Context, name string) (domain.User, error)
	FindByEmail(ctx context.Context, email string) (domain.User, error)
	FindAll(ctx context.Context) ([]domain.User, error)
	SearchByNameOrEmail(ctx context.Context, query string, limit int) ([]domain.User, error)
	Save(ctx context.Context, user domain.User) (domain.User, error)
	Delete(ctx context.Context, id int64) error
}

type GroupRepository interface {
	FindByID(ctx context.Context, id int64) (domain.Group, error)
	FindAll(ctx context.Context) ([]domain.Group, error)
	Save(ctx context.Context, group domain.Group) (domain.Group, error)
	Delete(ctx context.Context, id int64) error
}

type OccurrenceRepository interface {
	FindByID(ctx context.Context, id int64) (domain.Occurrence, error)
	FindAll(ctx context.Context) ([]domain.Occurrence, error)
	FindByGroup(ctx context.Context, groupID int64) ([]domain.Occurrence, error)
	FindByDate(ctx context.Context, date time.Time) ([]domain.Occurrence, error)
	FindOpenSpots(ctx context.Context) ([]domain.Occurrence, error)
	FindUpcomingByUser(ctx context.Context, userID int64, from time.Time) ([]domain.Occurrence, error)
	FindAllByUser(ctx context.Context, userID int64) ([]domain.Occurrence, error)
	FindByTitleLike(ctx context.Context, query string, limit int) ([]domain.Occurrence, error)
	FindInRange(ctx context.Context, from, to time.Time, groupID int64) ([]domain.Occurrence, error)
	FindByRecurrenceID(ctx context.Context, recurrenceID string) ([]domain.Occurrence, error)
	Save(ctx context.Context, occurrence domain.Occurrence) (domain.Occurrence, error)
	Delete(ctx context.Context, id int64) error
	DeleteByRecurrenceID(ctx context.Context, recurrenceID string) (int64, error)
	DeleteByRecurrenceIDFromDate(ctx context.Context, recurrenceID string, fromDate time.Time) (int64, error)
}

type ParticipationRepository interface {
	FindByID(ctx context.Context, id int64) (domain.Participation, error)
	FindByOccurrence(ctx context.Context, occurrenceID int64) ([]domain.Participation, error)
	FindByUser(ctx context.Context, userID int64) ([]domain.Participation, error)
	FindUsersByOccurrence(ctx context.Context, occurrenceID int64) ([]domain.User, error)
	CountByUser(ctx context.Context, userID int64) (int, error)
	CountByOccurrence(ctx context.Context, occurrenceID int64) (int, error)
	CountByUserInRange(ctx context.Context, userID int64, from, to time.Time) (int, error)
	CountAllByOccurrence(ctx context.Context) (map[int64]int, error)
	CountByUserGroupedByDate(ctx context.Context, userID int64, from, to time.Time) (map[string]int, error)
	ExistsForUserInDateRange(ctx context.Context, userID int64, from, to time.Time) (bool, error)
	LeaderboardAll(ctx context.Context, roles []domain.Role, groupID int64) ([]LeaderboardRow, error)
	LeaderboardInRange(ctx context.Context, from, to time.Time, roles []domain.Role, groupID int64) ([]LeaderboardRow, error)
	ExportInRange(ctx context.Context, from, to time.Time, roles []domain.Role, groupID int64) ([]ExportRow, error)
	CountAndInsert(ctx context.Context, occurrenceID, userID int64, maxParticipants int) (isOverMax bool, err error)
	Save(ctx context.Context, p domain.Participation) (domain.Participation, error)
	Delete(ctx context.Context, id int64) error
	DeleteByOccurrenceAndUser(ctx context.Context, occurrenceID, userID int64) error
}

// LeaderboardRow is a pre-joined row for leaderboard queries.
type LeaderboardRow struct {
	UserID int64
	Name   string
	Email  string
	Role   domain.Role
	Count  int
}

// ExportRow is a pre-joined row for detailed participation export.
type ExportRow struct {
	OccurrenceDate  time.Time
	OccurrenceTitle string
	GroupName       string
	UserName        string
	UserEmail       string
}

type OutOfOfficeRepository interface {
	FindByID(ctx context.Context, id int64) (domain.OutOfOffice, error)
	FindByUser(ctx context.Context, userID int64) ([]domain.OutOfOffice, error)
	Save(ctx context.Context, ooo domain.OutOfOffice) (domain.OutOfOffice, error)
	Delete(ctx context.Context, id int64) error
}

type UserSearchResult struct {
	ID    int64
	Name  string
	Email string
	Role  domain.Role
}

type SettingsRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	GetAll(ctx context.Context) (map[string]string, error)
	SetMultiple(ctx context.Context, settings map[string]string) error
}

type EmailLogRepository interface {
	LogSent(ctx context.Context, userID int64, emailType string) error
	LastSentAt(ctx context.Context, userID int64) (time.Time, error)
	LastSentAtByType(ctx context.Context, userID int64, emailType string) (time.Time, error)
	CountSentToday(ctx context.Context, userID int64) (int, error)
}

type CommentRepository interface {
	FindByOccurrence(ctx context.Context, occurrenceID int64) ([]domain.Comment, error)
	FindByID(ctx context.Context, id int64) (domain.Comment, error)
	Save(ctx context.Context, c domain.Comment) (domain.Comment, error)
	Delete(ctx context.Context, id int64) error
}
