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
	ID   int64
	Name string
}

type Occurrence struct {
	ID              int64
	GroupID         int64
	Title           string
	Description     string
	Date            time.Time
	MinParticipants int
	MaxParticipants int
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
