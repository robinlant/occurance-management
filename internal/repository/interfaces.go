package repository

import (
	"context"

	"github.com/robinlant/occurance-management/internal/domain"
)

type UserRepository interface {
	FindByID(ctx context.Context, id int64) (domain.User, error)
	FindByName(ctx context.Context, name string) (domain.User, error)
	FindAll(ctx context.Context) ([]domain.User, error)
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
	FindOpenSpots(ctx context.Context) ([]domain.Occurrence, error)
	Save(ctx context.Context, occurrence domain.Occurrence) (domain.Occurrence, error)
	Delete(ctx context.Context, id int64) error
}

type ParticipationRepository interface {
	FindByID(ctx context.Context, id int64) (domain.Participation, error)
	FindByOccurrence(ctx context.Context, occurrenceID int64) ([]domain.Participation, error)
	FindByUser(ctx context.Context, userID int64) ([]domain.Participation, error)
	CountByUser(ctx context.Context, userID int64) (int, error)
	Save(ctx context.Context, p domain.Participation) (domain.Participation, error)
	Delete(ctx context.Context, id int64) error
	DeleteByOccurrenceAndUser(ctx context.Context, occurrenceID, userID int64) error
}

type OutOfOfficeRepository interface {
	FindByUser(ctx context.Context, userID int64) ([]domain.OutOfOffice, error)
	Save(ctx context.Context, ooo domain.OutOfOffice) (domain.OutOfOffice, error)
	Delete(ctx context.Context, id int64) error
}
