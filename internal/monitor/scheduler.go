package monitor

import (
	"log"
	"service-monitor/internal/database"
	"service-monitor/internal/models"
	"sync"
	"time"
)

// pingResult holds the result of a ping check before writing to DB
type pingResult struct {
	Service        models.Service
	Status         string
	PreviousStatus string
	ResponseTime   int
}

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
	results := make([]pingResult, 0, len(services))

	// Step 1: Run all pings concurrently (no DB writes yet)
	for _, service := range services {
		wg.Add(1)
		go func(svc models.Service) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Ping the service (no DB write)
			result := s.pingService(svc)

			// Add to results
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(service)
	}

	// Wait for all pings to complete
	wg.Wait()
	log.Printf("Completed pinging %d services", len(services))

	// Step 2: Write all results to database in a single transaction
	entries := make([]database.StatusEntry, len(results))
	for i, r := range results {
		entries[i] = database.StatusEntry{
			ServiceID:      r.Service.ID,
			Status:         r.Status,
			ResponseTimeMs: r.ResponseTime,
		}
		log.Printf("Service %s (%s): %s (response time: %dms)", r.Service.Name, r.Service.IPAddress, r.Status, r.ResponseTime)
	}

	err = s.db.AddStatusHistoryBatch(entries)
	if err != nil {
		log.Printf("Error batch inserting status history: %v", err)
		return
	}
	log.Println("All status history written to database")

	// Step 3: Fetch history for all services
	statuses := make([]models.ServiceStatus, len(results))
	for i, r := range results {
		history, err := s.db.GetServiceHistoryPoints(r.Service.ID, 100)
		if err != nil {
			log.Printf("Error getting history for service ID %d: %v", r.Service.ID, err)
			history = nil
		}
		statuses[i] = models.ServiceStatus{
			ID:      r.Service.ID,
			Name:    r.Service.Name,
			Status:  r.Status,
			History: history,
		}
	}
	log.Println("History fetched for all services")

	// Step 4: Trigger status change callbacks
	for _, r := range results {
		if r.Status != r.PreviousStatus && s.onStatusChange != nil {
			s.onStatusChange(r.Service, r.Status, r.PreviousStatus)
		}
	}

	// Step 5: Push to frontend
	if s.onCheckComplete != nil {
		s.onCheckComplete(statuses)
	}
}

// pingService pings a single service and returns the result (no DB write)
func (s *Scheduler) pingService(service models.Service) pingResult {
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

	return pingResult{
		Service:        service,
		Status:         newStatus,
		PreviousStatus: previousStatus,
		ResponseTime:   result.ResponseTime,
	}
}

// checkService checks a single service and stores the result (used by CheckServiceNow)
func (s *Scheduler) checkService(service models.Service) models.ServiceStatus {
	result := s.pingService(service)

	// Store in history
	err := s.db.AddStatusHistory(service.ID, result.Status, result.ResponseTime)
	if err != nil {
		log.Printf("Error storing status history for service %s: %v", service.Name, err)
	}

	log.Printf("Service %s (%s): %s (response time: %dms)", service.Name, service.IPAddress, result.Status, result.ResponseTime)

	// Trigger status change callback if status changed
	if result.Status != result.PreviousStatus && s.onStatusChange != nil {
		s.onStatusChange(service, result.Status, result.PreviousStatus)
	}

	return models.ServiceStatus{
		ID:     service.ID,
		Name:   service.Name,
		Status: result.Status,
	}
}

// CheckServiceNow immediately checks a specific service (useful for testing)
func (s *Scheduler) CheckServiceNow(serviceID int) (*models.ServiceStatus, error) {
	service, err := s.db.GetService(serviceID)
	if err != nil {
		return nil, err
	}

	status := s.checkService(*service)

	// Fetch history for this single service (100 points for frontend bar)
	history, err := s.db.GetServiceHistoryPoints(serviceID, 100)
	if err != nil {
		log.Printf("Error getting history for service %s: %v", service.Name, err)
		history = nil
	}
	status.History = history

	return &status, nil
}
