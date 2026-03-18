package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/robinlant/occurance-management/internal/domain"
	"github.com/robinlant/occurance-management/internal/repository/sqlite"
	"github.com/robinlant/occurance-management/internal/service"
)

// Credentials printed at the end
const (
	adminEmail    = "admin@dutyround.dev"
	adminPassword = "admin123"

	organizerEmail    = "organizer@dutyround.dev"
	organizerPassword = "organizer123"
)

func main() {
	dbPath := "dutyround.db"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	db, err := sqlite.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	migration, err := os.ReadFile("migrations/001_init.sql")
	if err != nil {
		log.Fatalf("read migration: %v", err)
	}
	if _, err := db.Exec(string(migration)); err != nil {
		log.Fatalf("migration: %v", err)
	}

	userRepo := sqlite.NewUserRepository(db)
	groupRepo := sqlite.NewGroupRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)

	userSvc := service.NewUserService(userRepo, oooRepo, partRepo)
	groupSvc := service.NewGroupService(groupRepo)
	occSvc := service.NewOccurrenceService(occRepo, partRepo, userRepo, oooRepo)

	ctx := context.Background()
	rng := rand.New(rand.NewSource(42))

	fmt.Println("Creating users...")

	admin, err := userSvc.CreateUser(ctx, "Admin", adminEmail, adminPassword, domain.RoleAdmin)
	must(err, "create admin")

	organizer, err := userSvc.CreateUser(ctx, "Sophie Braun", organizerEmail, organizerPassword, domain.RoleOrganizer)
	must(err, "create organizer")
	_ = organizer

	participants := []struct{ name, email string }{
		{"Lukas Müller", "lukas@dutyround.dev"},
		{"Anna Schmidt", "anna@dutyround.dev"},
		{"Felix Wagner", "felix@dutyround.dev"},
		{"Laura Becker", "laura@dutyround.dev"},
		{"Jonas Fischer", "jonas@dutyround.dev"},
		{"Marie Hoffmann", "marie@dutyround.dev"},
		{"Tim Weber", "tim@dutyround.dev"},
		{"Lisa Schulz", "lisa@dutyround.dev"},
		{"Max Richter", "max@dutyround.dev"},
		{"Sarah Koch", "sarah@dutyround.dev"},
		{"Paul Bauer", "paul@dutyround.dev"},
		{"Jana Wolf", "jana@dutyround.dev"},
	}

	var participantUsers []domain.User
	for _, p := range participants {
		u, err := userSvc.CreateUser(ctx, p.name, p.email, "pass123", domain.RoleParticipant)
		must(err, "create participant "+p.name)
		participantUsers = append(participantUsers, u)
	}

	fmt.Println("Creating groups...")

	groupNames := []string{
		"Versandaktion",
		"Empfangvertretung",
		"Küchendienst",
		"Postverteilung",
		"IT-Support Bereitschaft",
	}
	var groups []domain.Group
	for _, name := range groupNames {
		g, err := groupSvc.Create(ctx, name)
		must(err, "create group "+name)
		groups = append(groups, g)
	}

	fmt.Println("Creating occurrences...")

	type occTemplate struct {
		title       string
		description string
		min, max    int
	}
	templates := []occTemplate{
		{"Versandaktion Januar", "Pakete sortieren und versenden für die Januar-Kampagne.", 2, 4},
		{"Versandaktion Februar", "Monatliche Versandaktion — Hilfe beim Packen und Etikettieren.", 2, 4},
		{"Versandaktion März", "Frühlingsversand — besonders viele Pakete erwartet.", 3, 5},
		{"Empfang Montag", "Vertretung am Empfang — Besucher empfangen und weiterleiten.", 1, 2},
		{"Empfang Dienstag", "Vertretung am Empfang.", 1, 2},
		{"Empfang Mittwoch", "Vertretung am Empfang — Telefondienst inklusive.", 1, 2},
		{"Empfang Donnerstag", "Vertretung am Empfang.", 1, 2},
		{"Empfang Freitag", "Vertretung am Empfang — früher Feierabend um 15 Uhr.", 1, 2},
		{"Küchendienst KW10", "Küche sauber halten, Geschirrspüler ein- und ausräumen.", 2, 3},
		{"Küchendienst KW11", "Wochendienst Küche.", 2, 3},
		{"Küchendienst KW12", "Wochendienst Küche — Kühlschrank aufräumen inklusive.", 2, 3},
		{"Postverteilung Montag", "Eingehende Post sortieren und verteilen.", 1, 1},
		{"Postverteilung Mittwoch", "Eingehende Post sortieren und verteilen.", 1, 1},
		{"Postverteilung Freitag", "Eingehende Post sortieren und verteilen.", 1, 1},
		{"IT-Support Bereitschaft KW9", "Erster Ansprechpartner für IT-Fragen der Kollegen.", 1, 2},
		{"IT-Support Bereitschaft KW10", "Erste-Hilfe IT — Passwörter, Drucker, VPN.", 1, 2},
		{"Großes Versandevent Q1", "Quartalsweiser Großversand — ganzer Tag eingeplant.", 4, 8},
		{"Büropflanzenpflege", "Wöchentliche Pflege der Büropflanzen.", 1, 2},
		{"Konferenzraum Vorbereitung", "Konferenzraum für den Vorstandsbesuch herrichten.", 2, 4},
		{"Parkplatzdienst", "Parkplätze für Gäste reservieren und einweisen.", 1, 3},
	}

	now := time.Now()
	var occurrences []domain.Occurrence

	for i, tmpl := range templates {
		// Spread occurrences over past 3 months and next 2 months
		dayOffset := rng.Intn(150) - 90
		hour := 8 + rng.Intn(8)
		date := now.AddDate(0, 0, dayOffset).Truncate(24*time.Hour).Add(time.Duration(hour) * time.Hour)

		var groupID int64
		if rng.Intn(3) != 0 { // 2/3 chance of having a group
			groupID = groups[i%len(groups)].ID
		}

		occ, err := occSvc.CreateOccurrence(ctx, domain.Occurrence{
			GroupID:         groupID,
			Title:           tmpl.title,
			Description:     tmpl.description,
			Date:            date,
			MinParticipants: tmpl.min,
			MaxParticipants: tmpl.max,
		})
		must(err, "create occurrence "+tmpl.title)
		occurrences = append(occurrences, occ)
	}

	fmt.Println("Creating participations...")

	for _, occ := range occurrences {
		// Past occurrences get filled up; future ones are partially filled
		isPast := occ.Date.Before(now)
		slots := occ.MaxParticipants
		if !isPast {
			slots = rng.Intn(occ.MaxParticipants + 1)
		}

		shuffled := rng.Perm(len(participantUsers))
		count := 0
		for _, idx := range shuffled {
			if count >= slots {
				break
			}
			u := participantUsers[idx]
			_, _ = occSvc.SignUp(ctx, occ.ID, u.ID) // ignore OOO / already-signed errors
			count++
		}
	}

	fmt.Println("Creating out-of-office periods...")

	// Give a few users OOO periods
	oooData := []struct {
		userIdx    int
		fromOffset int
		days       int
	}{
		{0, 10, 5},
		{1, -30, 7},
		{3, 20, 3},
		{5, 5, 14},
		{8, -10, 2},
	}
	for _, o := range oooData {
		u := participantUsers[o.userIdx]
		from := now.AddDate(0, 0, o.fromOffset).Truncate(24 * time.Hour)
		to := from.AddDate(0, 0, o.days)
		// Use repo directly since service blocks OOO if participations exist in range
		oooRepo.Save(ctx, domain.OutOfOffice{UserID: u.ID, From: from, To: to})
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Admin       %s / %s\n", adminEmail, adminPassword)
	fmt.Printf("  Organizer   %s / %s\n", organizerEmail, organizerPassword)
	fmt.Println("  Participants pass: pass123")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Users: %d  Groups: %d  Occurrences: %d\n",
		2+len(participantUsers), len(groups), len(occurrences))
	fmt.Printf("  Admin ID: %d\n", admin.ID)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func must(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %v", msg, err)
	}
}
