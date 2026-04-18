package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/robinlant/dutyround/internal/domain"
	"github.com/robinlant/dutyround/internal/repository/sqlite"
	"github.com/robinlant/dutyround/internal/service"
)

func main() {
	defaultDb := os.Getenv("DB_PATH")
	if defaultDb == "" {
		defaultDb = "dutyround.db"
	}

	dbPath := flag.String("db", defaultDb, "path to SQLite database")
	name := flag.String("name", "", "admin name (required)")
	email := flag.String("email", "", "admin email (required)")
	password := flag.String("password", "", "admin password (required)")
	flag.Parse()

	if *name == "" || *email == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "Usage: seed -name=Admin -email=admin@example.com -password=secret")
		os.Exit(1)
	}

	dbDir := filepath.Dir(*dbPath)
	if dbDir != "." {
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			log.Fatalf("could not create database directory: %v", err)
		}
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
		errMsg := err.Error()
		if strings.Contains(errMsg, "UNIQUE") || strings.Contains(errMsg, "already exists") {
			fmt.Printf("Notice: Admin with email %s already exists. No changes made.\n", *email)
			os.Exit(0)
		}

		log.Fatalf("create user error: %v", err)
	}

	fmt.Printf("Admin created: id=%d  name=%s  email=%s\n", user.ID, user.Name, user.Email)
}
