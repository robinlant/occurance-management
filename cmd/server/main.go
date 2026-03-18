package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/robinlant/occurance-management/internal/repository/sqlite"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "dutyround.db"
	}

	db, err := sqlite.Open(dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := runMigrations(db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	_ = sqlite.NewUserRepository(db)
	_ = sqlite.NewGroupRepository(db)
	_ = sqlite.NewOccurrenceRepository(db)
	_ = sqlite.NewParticipationRepository(db)
	_ = sqlite.NewOutOfOfficeRepository(db)

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "DutyRound is running"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("DutyRound listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
