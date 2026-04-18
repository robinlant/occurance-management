package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	"github.com/robinlant/dutyround/internal/domain"
	"github.com/robinlant/dutyround/internal/handler"
	"github.com/robinlant/dutyround/internal/logger"
	"github.com/robinlant/dutyround/internal/repository/sqlite"
	"github.com/robinlant/dutyround/internal/service"
)

func main() {
	production := os.Getenv("GIN_MODE") == "release"
	logger.Init(production)

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
	settingsRepo := sqlite.NewSettingsRepository(db)
	emailLogRepo := sqlite.NewEmailLogRepository(db)
	commentRepo := sqlite.NewCommentRepository(db)

	// Services
	userSvc := service.NewUserService(userRepo, oooRepo, partRepo)
	groupSvc := service.NewGroupService(groupRepo)
	occSvc := service.NewOccurrenceService(occRepo, partRepo, userRepo, oooRepo)
	settingsSvc := service.NewSettingsService(settingsRepo)
	emailSvc := service.NewEmailService(settingsSvc, emailLogRepo, userRepo, occRepo, partRepo)

	// Start background email notification job (runs every hour)
	emailSvc.StartBackgroundJob(1 * time.Hour)
	defer emailSvc.Stop()

	// Handlers
	authH := handler.NewAuthHandler(userRepo)
	dashH := handler.NewDashboardHandler(occSvc)
	occH := handler.NewOccurrenceHandler(occSvc, groupSvc, userSvc, commentRepo)
	grpH := handler.NewGroupHandler(groupSvc)
	usrH := handler.NewUserAdminHandler(userSvc)
	profH := handler.NewProfileHandler(userSvc, occSvc, groupSvc)
	lbH := handler.NewLeaderboardHandler(occSvc, groupSvc)
	calH := handler.NewCalendarHandler(occSvc, groupSvc)
	searchH := handler.NewSearchHandler(occSvc, userSvc)
	settingsH := handler.NewSettingsHandler(settingsSvc, emailSvc)
	errH := handler.NewErrorHandler()

	// Router
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(handler.RequestLogger())

	// Session secret
	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		if production {
			log.Fatal("SESSION_SECRET must be set in production (GIN_MODE=release)")
		}
		slog.Warn("using default session secret — set SESSION_SECRET for production")
		sessionSecret = "dev-secret-change-in-production"
	}
	store := cookie.NewStore([]byte(sessionSecret))
	store.Options(sessions.Options{
		HttpOnly: true,
		MaxAge:   86400 * 7,
		SameSite: http.SameSiteLaxMode,
		Secure:   production,
	})
	r.Use(sessions.Sessions("dutyround", store))

	// Security headers middleware
	r.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	})

	// CSRF middleware for POST requests
	r.Use(handler.CSRFMiddleware())

	r.Static("/static", "./static")
	r.StaticFile("/favicon.ico", "./static/favicon.ico")
	r.StaticFile("/favicon.svg", "./static/favicon.svg")
	r.StaticFile("/apple-touch-icon.png", "./static/apple-touch-icon.png")

	// Public
	r.GET("/login", authH.ShowLogin)
	r.POST("/login", authH.Login)
	r.GET("/lang", handler.SwitchLang)
	r.NoRoute(errH.NotFound)

	// All authenticated
	auth := handler.AuthRequired(userRepo)
	protected := r.Group("/", auth)
	protected.POST("/logout", authH.Logout)
	protected.GET("/", dashH.Show)
	protected.GET("/duties", occH.List)
	protected.GET("/duties/:id", occH.Detail)
	protected.POST("/duties/:id/signup", occH.SignUp)
	protected.POST("/duties/:id/withdraw", occH.Withdraw)
	protected.POST("/duties/:id/comments", occH.AddComment)
	protected.POST("/duties/:id/comments/:cid/delete", occH.DeleteComment)
	protected.GET("/occurrences", redirectOccurrencesToDuties)
	protected.GET("/occurrences/:id", redirectOccurrencesToDuties)
	protected.POST("/occurrences/:id/signup", occH.SignUp)
	protected.POST("/occurrences/:id/withdraw", occH.Withdraw)
	protected.POST("/occurrences/:id/comments", occH.AddComment)
	protected.POST("/occurrences/:id/comments/:cid/delete", occH.DeleteComment)
	protected.GET("/leaderboard", lbH.Show)
	protected.GET("/leaderboard/export", lbH.Export)
	protected.GET("/profile", profH.Show)
	protected.POST("/profile/password", profH.ChangePassword)
	protected.POST("/profile/ooo", profH.AddOOO)
	protected.POST("/profile/ooo/:id/delete", profH.DeleteOOO)
	protected.GET("/calendar", calH.Show)
	protected.GET("/calendar/day", calH.DayOccurrences)
	protected.GET("/search", searchH.Search)
	protected.GET("/profile/:id", profH.ShowPublic)

	// Organizer + Admin only
	staff := protected.Group("/", handler.RoleRequired(domain.RoleOrganizer, domain.RoleAdmin))
	staff.GET("/duties/new", occH.ShowCreate)
	staff.POST("/duties", occH.Create)
	staff.GET("/duties/:id/edit", occH.ShowEdit)
	staff.POST("/duties/:id", occH.Update)
	staff.POST("/duties/:id/delete", occH.Delete)
	staff.POST("/duties/:id/assign", occH.Assign)
	staff.POST("/duties/:id/remove/:uid", occH.RemoveParticipant)
	staff.GET("/duties/:id/available", occH.AvailableUsers)
	staff.GET("/occurrences/new", redirectOccurrencesToDuties)
	staff.POST("/occurrences", occH.Create)
	staff.GET("/occurrences/:id/edit", redirectOccurrencesToDuties)
	staff.POST("/occurrences/:id", occH.Update)
	staff.POST("/occurrences/:id/delete", occH.Delete)
	staff.POST("/occurrences/:id/assign", occH.Assign)
	staff.POST("/occurrences/:id/remove/:uid", occH.RemoveParticipant)
	staff.GET("/occurrences/:id/available", redirectOccurrencesToDuties)
	staff.GET("/groups", grpH.List)
	staff.POST("/groups", grpH.Create)
	staff.POST("/groups/:id/color", grpH.UpdateColor)
	staff.POST("/groups/:id/delete", grpH.Delete)

	// Admin only
	adminUsers := protected.Group("/users", handler.RoleRequired(domain.RoleAdmin))
	adminUsers.GET("", usrH.List)
	adminUsers.POST("", usrH.Create)
	adminUsers.POST("/:id/set-password", usrH.SetPassword)
	adminUsers.POST("/:id/set-email", usrH.SetEmail)
	adminUsers.POST("/:id/delete", usrH.Delete)

	adminSettings := protected.Group("/settings", handler.RoleRequired(domain.RoleAdmin))
	adminSettings.GET("", settingsH.Show)
	adminSettings.POST("", settingsH.Save)
	adminSettings.POST("/test-email", settingsH.SendTestEmail)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Graceful shutdown
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		slog.Info("DutyRound started", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	slog.Info("server exited")
}

func redirectOccurrencesToDuties(c *gin.Context) {
	target := "/duties" + strings.TrimPrefix(c.Request.URL.Path, "/occurrences")
	if c.Request.URL.RawQuery != "" {
		target += "?" + c.Request.URL.RawQuery
	}
	c.Redirect(http.StatusMovedPermanently, target)
}
