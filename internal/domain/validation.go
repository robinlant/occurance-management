package domain

import (
	"errors"
	"net/mail"
)

var (
	ErrInvalidColor        = errors.New("invalid color")
	ErrInvalidEmail        = errors.New("invalid email format")
	ErrInvalidEditScope    = errors.New("invalid edit scope")
	ErrInvalidDeleteScope  = errors.New("invalid delete scope")
	ErrInvalidRepeatType   = errors.New("invalid repeat type")
	ErrInvalidStatusFilter = errors.New("invalid status filter")
	ErrInvalidRoleFilter   = errors.New("invalid role filter")
)

var allowedColors = map[string]bool{
	"red":    true,
	"orange": true,
	"yellow": true,
	"green":  true,
	"teal":   true,
	"blue":   true,
	"purple": true,
	"pink":   true,
}

func ValidateColor(color string) error {
	if !allowedColors[color] {
		return ErrInvalidColor
	}
	return nil
}

func ValidateEmail(email string) error {
	if email == "" {
		return ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return ErrInvalidEmail
	}
	// Reject display-name format like "John Doe" <john@example.com>.
	// Only bare addresses are valid for storage.
	if addr.Address != email {
		return ErrInvalidEmail
	}
	return nil
}

func ValidateEditScope(scope string) error {
	switch scope {
	case "single", "future", "all":
		return nil
	default:
		return ErrInvalidEditScope
	}
}

func ValidateDeleteScope(scope string) error {
	switch scope {
	case "single", "future", "all":
		return nil
	default:
		return ErrInvalidDeleteScope
	}
}

func ValidateRepeatType(repeat string) error {
	switch repeat {
	case "", "daily", "weekly", "biweekly", "monthly":
		return nil
	default:
		return ErrInvalidRepeatType
	}
}

func ValidateStatusFilter(status string) error {
	switch status {
	case "", "under", "good", "over":
		return nil
	default:
		return ErrInvalidStatusFilter
	}
}

func ValidateRoleFilter(role string) error {
	switch role {
	case "", "participants", "organizers", "all":
		return nil
	default:
		return ErrInvalidRoleFilter
	}
}
