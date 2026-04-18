package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/robinlant/dutyround/internal/domain"
	"github.com/robinlant/dutyround/internal/repository"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrEmailTaken       = errors.New("email already taken")
	ErrOOOConflict      = errors.New("user has participations assigned in the requested out-of-office period")
	ErrOOONotOwner      = errors.New("out-of-office record does not belong to user")
	ErrOOOOverlap       = errors.New("overlapping out-of-office period exists")
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	ErrWrongPassword    = errors.New("current password is incorrect")
	ErrInvalidRole      = errors.New("invalid role")
	ErrNameEmpty        = errors.New("name must not be empty")
)

type UserService struct {
	users          repository.UserRepository
	ooo            repository.OutOfOfficeRepository
	participations repository.ParticipationRepository
}

func NewUserService(
	users repository.UserRepository,
	ooo repository.OutOfOfficeRepository,
	participations repository.ParticipationRepository,
) *UserService {
	return &UserService{users: users, ooo: ooo, participations: participations}
}

// ValidateRole checks that the role is one of the allowed values.
func ValidateRole(role domain.Role) error {
	switch role {
	case domain.RoleAdmin, domain.RoleOrganizer, domain.RoleParticipant:
		return nil
	default:
		return ErrInvalidRole
	}
}

func (s *UserService) CreateUser(ctx context.Context, name, email, password string, role domain.Role) (domain.User, error) {
	if strings.TrimSpace(name) == "" {
		return domain.User{}, ErrNameEmpty
	}
	if len(password) < 8 {
		return domain.User{}, ErrPasswordTooShort
	}
	if err := ValidateRole(role); err != nil {
		return domain.User{}, err
	}
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

func (s *UserService) ChangePassword(ctx context.Context, userID int64, currentPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrPasswordTooShort
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return ErrWrongPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hash)
	_, err = s.users.Save(ctx, user)
	return err
}

// SetPassword is the admin variant — sets password for any user by ID (no old password required).
func (s *UserService) SetPassword(ctx context.Context, userID int64, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrPasswordTooShort
	}
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

func (s *UserService) AddOutOfOffice(ctx context.Context, userID int64, from, to time.Time) (domain.OutOfOffice, error) {
	exists, err := s.participations.ExistsForUserInDateRange(ctx, userID, from, to)
	if err != nil {
		return domain.OutOfOffice{}, err
	}
	if exists {
		return domain.OutOfOffice{}, ErrOOOConflict
	}
	// Check for overlapping OOO periods.
	existing, err := s.ooo.FindByUser(ctx, userID)
	if err != nil {
		return domain.OutOfOffice{}, err
	}
	for _, o := range existing {
		// Two periods overlap if one starts before the other ends AND vice versa.
		// Adjacent periods (from == o.To or to == o.From) are allowed.
		if from.Before(o.To) && to.After(o.From) {
			return domain.OutOfOffice{}, ErrOOOOverlap
		}
	}
	return s.ooo.Save(ctx, domain.OutOfOffice{
		UserID: userID,
		From:   from,
		To:     to,
	})
}

// RemoveOutOfOffice deletes an OOO record, verifying ownership.
func (s *UserService) RemoveOutOfOffice(ctx context.Context, oooID, userID int64) error {
	ooo, err := s.ooo.FindByID(ctx, oooID)
	if err != nil {
		return err
	}
	if ooo.UserID != userID {
		return ErrOOONotOwner
	}
	return s.ooo.Delete(ctx, oooID)
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

func (s *UserService) SetEmail(ctx context.Context, userID int64, email string) error {
	existing, err := s.users.FindByEmail(ctx, email)
	if err == nil && existing.ID != userID {
		return ErrEmailTaken
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}
	user.Email = email
	_, err = s.users.Save(ctx, user)
	return err
}

func (s *UserService) DeleteUser(ctx context.Context, userID int64) error {
	if _, err := s.users.FindByID(ctx, userID); err != nil {
		return ErrUserNotFound
	}
	return s.users.Delete(ctx, userID)
}

// SearchUsers searches users by name or email using SQL LIKE.
func (s *UserService) SearchUsers(ctx context.Context, query string, limit int) ([]domain.User, error) {
	return s.users.SearchByNameOrEmail(ctx, query, limit)
}
