package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"service-monitor/internal/alerts"
	"service-monitor/internal/api"
	"service-monitor/internal/database"
	"service-monitor/internal/models"
	"service-monitor/internal/monitor"
	"service-monitor/internal/pusher"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	log.Println("Starting Service Monitor...")

	// Load configuration
	config, err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.New(config.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	log.Println("Database initialized")

	// Initialize pusher
	pushClient := pusher.New(
		config.FrontendURL,
		config.APIKey,
		time.Duration(config.PushTimeoutSeconds)*time.Second,
		config.PushRetryAttempts,
	)

	// Initialize alerter
	alertClient := alerts.New(db, config.AlertCooldownMinutes)

	// Initialize scheduler
	scheduler := monitor.NewScheduler(
		db,
		time.Duration(config.CheckIntervalSeconds)*time.Second,
		config.PingTimeoutSeconds,
		config.PingCount,
		config.MaxConcurrentPings,
	)

	// Set up status change callback for alerts
	scheduler.OnStatusChange(func(service models.Service, newStatus string, oldStatus string) {
		log.Printf("Status change detected for %s: %s -> %s", service.Name, oldStatus, newStatus)
		alertClient.SendAlert(service, newStatus, oldStatus)
	})

	// Set up check complete callback for pushing to frontend
	scheduler.OnCheckComplete(func(statuses []models.ServiceStatus) {
		log.Println("Pushing status update to frontend...")
		err := pushClient.Push(statuses)
		if err != nil {
			log.Printf("Failed to push status update: %v", err)
		}
	})

	// Start scheduler in background
	go scheduler.Start()
	log.Println("Scheduler started")

	// Initialize admin API
	adminAPI := api.NewAdminAPI(db, scheduler, alertClient)

	// Setup HTTP router
	router := mux.NewRouter()

	// Apply middleware
	router.Use(api.Logging)
	router.Use(api.LocalNetworkOnly)
	router.Use(api.CORS)

	// Setup API routes
	adminAPI.SetupRoutes(router)

	// Serve admin interface
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web")))

	// Start HTTP server
	addr := fmt.Sprintf("%s:%d", config.AdminBind, config.AdminPort)
	log.Printf("Admin server starting on %s", addr)
	log.Printf("Admin interface: http://%s/admin.html", addr)

	// Setup graceful shutdown
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	log.Println("Service Monitor is running")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gracefully...")
	scheduler.Stop()
	log.Println("Service Monitor stopped")
}

// loadConfig loads the application configuration from a JSON file
func loadConfig(filename string) (*models.AppConfig, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var config models.AppConfig
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}
