package main

import (
	"log"
	"os"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	"github.com/robinlant/occurance-management/internal/domain"
	"github.com/robinlant/occurance-management/internal/handler"
	"github.com/robinlant/occurance-management/internal/repository/sqlite"
	"github.com/robinlant/occurance-management/internal/service"
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

	// Repositories
	userRepo := sqlite.NewUserRepository(db)
	groupRepo := sqlite.NewGroupRepository(db)
	occRepo := sqlite.NewOccurrenceRepository(db)
	partRepo := sqlite.NewParticipationRepository(db)
	oooRepo := sqlite.NewOutOfOfficeRepository(db)

	// Services
	userSvc := service.NewUserService(userRepo, oooRepo, partRepo)
	groupSvc := service.NewGroupService(groupRepo)
	occSvc := service.NewOccurrenceService(occRepo, partRepo, userRepo, oooRepo)

	// Handlers
	authH := handler.NewAuthHandler(userRepo)
	dashH := handler.NewDashboardHandler(occSvc)
	occH := handler.NewOccurrenceHandler(occSvc, groupSvc)
	grpH := handler.NewGroupHandler(groupSvc)
	usrH := handler.NewUserAdminHandler(userSvc)
	profH := handler.NewProfileHandler(userSvc)
	lbH := handler.NewLeaderboardHandler(occSvc)
	calH := handler.NewCalendarHandler(occSvc, groupSvc)
	errH := handler.NewErrorHandler()

	// Router
	r := gin.Default()

	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = "dev-secret-change-in-production"
	}
	store := cookie.NewStore([]byte(sessionSecret))
	store.Options(sessions.Options{HttpOnly: true, MaxAge: 86400 * 7})
	r.Use(sessions.Sessions("dutyround", store))

	r.Static("/static", "./static")

	// Public
	r.GET("/login", authH.ShowLogin)
	r.POST("/login", authH.Login)
	r.NoRoute(errH.NotFound)

	// All authenticated
	auth := handler.AuthRequired(userRepo)
	protected := r.Group("/", auth)
	protected.POST("/logout", authH.Logout)
	protected.GET("/", dashH.Show)
	protected.GET("/occurrences", occH.List)
	protected.GET("/occurrences/:id", occH.Detail)
	protected.POST("/occurrences/:id/signup", occH.SignUp)
	protected.POST("/occurrences/:id/withdraw", occH.Withdraw)
	protected.GET("/leaderboard", lbH.Show)
	protected.GET("/profile", profH.Show)
	protected.POST("/profile/password", profH.ChangePassword)
	protected.POST("/profile/ooo", profH.AddOOO)
	protected.POST("/profile/ooo/:id/delete", profH.DeleteOOO)
	protected.GET("/calendar", calH.Show)
	protected.GET("/calendar/day", calH.DayOccurrences)

	// Organizer + Admin only
	staff := protected.Group("/", handler.RoleRequired(domain.RoleOrganizer, domain.RoleAdmin))
	staff.GET("/occurrences/new", occH.ShowCreate)
	staff.POST("/occurrences", occH.Create)
	staff.GET("/occurrences/:id/edit", occH.ShowEdit)
	staff.POST("/occurrences/:id", occH.Update)
	staff.POST("/occurrences/:id/delete", occH.Delete)
	staff.POST("/occurrences/:id/assign", occH.Assign)
	staff.POST("/occurrences/:id/remove/:uid", occH.RemoveParticipant)
	staff.GET("/occurrences/:id/available", occH.AvailableUsers)
	staff.GET("/groups", grpH.List)
	staff.POST("/groups", grpH.Create)
	staff.POST("/groups/:id/delete", grpH.Delete)

	// Admin only
	admin := protected.Group("/users", handler.RoleRequired(domain.RoleAdmin))
	admin.GET("", usrH.List)
	admin.POST("", usrH.Create)
	admin.POST("/:id/set-password", usrH.SetPassword)
	admin.POST("/:id/delete", usrH.Delete)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("DutyRound listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
