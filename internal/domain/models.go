package domain

import "time"

type Role string

const (
	RoleAdmin       Role = "admin"
	RoleOrganizer   Role = "organizer"
	RoleParticipant Role = "participant"
)

type User struct {
	ID           int64
	Name         string
	Email        string
	Role         Role
	PasswordHash string
}

type Group struct {
	ID    int64
	Name  string
	Color string // theme color name: red, orange, yellow, green, teal, blue, purple, pink
}

type Occurrence struct {
	ID              int64
	GroupID         int64
	Title           string
	Description     string
	Date            time.Time
	MinParticipants int
	MaxParticipants int
	AllowOverLimit  bool
	RecurrenceID    string
	CreatedAt       time.Time
}

type Participation struct {
	ID           int64
	UserID       int64
	OccurrenceID int64
}

type OutOfOffice struct {
	ID     int64
	UserID int64
	From   time.Time
	To     time.Time
}

type Comment struct {
	ID           int64
	OccurrenceID int64
	UserID       int64
	UserName     string
	Body         string
	CreatedAt    time.Time
}
