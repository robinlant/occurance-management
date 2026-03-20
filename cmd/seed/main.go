package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/robinlant/occurance-management/internal/domain"
	"github.com/robinlant/occurance-management/internal/repository/sqlite"
	"github.com/robinlant/occurance-management/internal/service"
)

func main() {
	dbPath := flag.String("db", "dutyround.db", "path to SQLite database")
	name := flag.String("name", "", "admin name (required)")
	email := flag.String("email", "", "admin email (required)")
	password := flag.String("password", "", "admin password (required)")
	flag.Parse()

	if *name == "" || *email == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "Usage: seed -name=Admin -email=admin@example.com -password=secret")
		os.Exit(1)
	}

	db, err := sqlite.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	userRepo := sqlite.NewUserRepository(db)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)
	svc := service.NewUserService(userRepo, oooRepo, partRepo)

	user, err := svc.CreateUser(context.Background(), *name, *email, *password, domain.RoleAdmin)
	if err != nil {
		log.Fatalf("create user: %v", err)
	}
	fmt.Printf("Admin created: id=%d  name=%s  email=%s\n", user.ID, user.Name, user.Email)
}
