package service

import (
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/robinlant/occurance-management/internal/domain"
	"github.com/robinlant/occurance-management/internal/repository"
)

var (
	ErrUserNotFound          = errors.New("user not found")
	ErrEmailTaken            = errors.New("email already taken")
	ErrOOOConflict           = errors.New("user has participations assigned in the requested out-of-office period")
)

type UserService struct {
	users         repository.UserRepository
	ooo           repository.OutOfOfficeRepository
	participations repository.ParticipationRepository
}

func NewUserService(
	users repository.UserRepository,
	ooo repository.OutOfOfficeRepository,
	participations repository.ParticipationRepository,
) *UserService {
	return &UserService{users: users, ooo: ooo, participations: participations}
}

func (s *UserService) CreateUser(ctx context.Context, name, email, password string, role domain.Role) (domain.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, err
	}
	return s.users.Save(ctx, domain.User{
		Name:         name,
		Email:        email,
		Role:         role,
		PasswordHash: string(hash),
	})
}

func (s *UserService) ChangePassword(ctx context.Context, userID int64, newPassword string) error {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hash)
	_, err = s.users.Save(ctx, user)
	return err
}

// SetPassword is the admin variant — sets password for any user by ID.
func (s *UserService) SetPassword(ctx context.Context, userID int64, newPassword string) error {
	return s.ChangePassword(ctx, userID, newPassword)
}

func (s *UserService) AddOutOfOffice(ctx context.Context, userID int64, from, to time.Time) (domain.OutOfOffice, error) {
	exists, err := s.participations.ExistsForUserInDateRange(ctx, userID, from, to)
	if err != nil {
		return domain.OutOfOffice{}, err
	}
	if exists {
		return domain.OutOfOffice{}, ErrOOOConflict
	}
	return s.ooo.Save(ctx, domain.OutOfOffice{
		UserID: userID,
		From:   from,
		To:     to,
	})
}

func (s *UserService) RemoveOutOfOffice(ctx context.Context, id int64) error {
	return s.ooo.Delete(ctx, id)
}

func (s *UserService) GetOutOfOffice(ctx context.Context, userID int64) ([]domain.OutOfOffice, error) {
	return s.ooo.FindByUser(ctx, userID)
}

func (s *UserService) GetUser(ctx context.Context, userID int64) (domain.User, error) {
	return s.users.FindByID(ctx, userID)
}

func (s *UserService) ListUsers(ctx context.Context) ([]domain.User, error) {
	return s.users.FindAll(ctx)
}

func (s *UserService) DeleteUser(ctx context.Context, userID int64) error {
	return s.users.Delete(ctx, userID)
}
