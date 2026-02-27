package monitor

import (
	"log"
	"service-monitor/internal/database"
	"service-monitor/internal/models"
	"sync"
	"time"
)

// Scheduler manages periodic service checks
type Scheduler struct {
	db                 *database.DB
	checkInterval      time.Duration
	pingTimeout        int
	pingCount          int
	maxConcurrentPings int
	onStatusChange     func(service models.Service, newStatus string, oldStatus string)
	onCheckComplete    func(statuses []models.ServiceStatus)
	quit               chan bool
}

// NewScheduler creates a new scheduler instance
func NewScheduler(
	db *database.DB,
	checkInterval time.Duration,
	pingTimeout int,
	pingCount int,
	maxConcurrentPings int,
) *Scheduler {
	return &Scheduler{
		db:                 db,
		checkInterval:      checkInterval,
		pingTimeout:        pingTimeout,
		pingCount:          pingCount,
		maxConcurrentPings: maxConcurrentPings,
		quit:               make(chan bool),
	}
}

// OnStatusChange sets the callback for when a service status changes
func (s *Scheduler) OnStatusChange(callback func(service models.Service, newStatus string, oldStatus string)) {
	s.onStatusChange = callback
}

// OnCheckComplete sets the callback for when all services have been checked
func (s *Scheduler) OnCheckComplete(callback func(statuses []models.ServiceStatus)) {
	s.onCheckComplete = callback
}

// Start begins the periodic monitoring
func (s *Scheduler) Start() {
	log.Println("Scheduler started")

	// Run first check immediately
	s.checkAllServices()

	// Then run on interval
	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkAllServices()
		case <-s.quit:
			log.Println("Scheduler stopped")
			return
		}
	}
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.quit <- true
}

// checkAllServices checks all enabled services concurrently
func (s *Scheduler) checkAllServices() {
	log.Println("Starting service checks...")

	services, err := s.db.GetEnabledServices()
	if err != nil {
		log.Printf("Error getting enabled services: %v", err)
		return
	}

	if len(services) == 0 {
		log.Println("No enabled services to check")
		return
	}

	// Create semaphore to limit concurrent pings
	semaphore := make(chan struct{}, s.maxConcurrentPings)
	var wg sync.WaitGroup
	var mu sync.Mutex
	statuses := make([]models.ServiceStatus, 0, len(services))

	for _, service := range services {
		wg.Add(1)
		go func(svc models.Service) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Check the service
			status := s.checkService(svc)

			// Add to results
			mu.Lock()
			statuses = append(statuses, status)
			mu.Unlock()
		}(service)
	}

	// Wait for all checks to complete
	wg.Wait()

	log.Printf("Completed checking %d services", len(services))

	// Trigger callback with all statuses
	if s.onCheckComplete != nil {
		s.onCheckComplete(statuses)
	}
}

// checkService checks a single service and stores the result
func (s *Scheduler) checkService(service models.Service) models.ServiceStatus {
	// Get previous status
	lastStatus, err := s.db.GetLastStatus(service.ID)
	var previousStatus string
	if err == nil && lastStatus != nil {
		previousStatus = lastStatus.Status
	} else {
		previousStatus = "unknown"
	}

	// Perform ping
	result := Ping(service.IPAddress, s.pingTimeout, s.pingCount)

	// Determine new status
	var newStatus string
	if result.Success {
		newStatus = "up"
	} else {
		newStatus = "down"
	}

	// Store in history
	err = s.db.AddStatusHistory(service.ID, newStatus, result.ResponseTime)
	if err != nil {
		log.Printf("Error storing status history for service %s: %v", service.Name, err)
	}

	log.Printf("Service %s (%s): %s (response time: %dms)", service.Name, service.IPAddress, newStatus, result.ResponseTime)

	// Trigger status change callback if status changed
	if newStatus != previousStatus && s.onStatusChange != nil {
		s.onStatusChange(service, newStatus, previousStatus)
	}

	// Get recent history for frontend display (last 24 checks = ~2 hours of history at 5 min intervals)
	history, err := s.db.GetServiceHistoryPoints(service.ID, 24)
	if err != nil {
		log.Printf("Error getting history for service %s: %v", service.Name, err)
		history = nil
	}

	return models.ServiceStatus{
		ID:      service.ID,
		Name:    service.Name,
		Status:  newStatus,
		History: history,
	}
}

// CheckServiceNow immediately checks a specific service (useful for testing)
func (s *Scheduler) CheckServiceNow(serviceID int) (*models.ServiceStatus, error) {
	service, err := s.db.GetService(serviceID)
	if err != nil {
		return nil, err
	}

	status := s.checkService(*service)
	return &status, nil
}
