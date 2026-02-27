package pusher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"service-monitor/internal/models"
	"time"
)

// Pusher sends status updates to the frontend
type Pusher struct {
	frontendURL    string
	apiKey         string
	timeout        time.Duration
	retryAttempts  int
	httpClient     *http.Client
}

// New creates a new pusher instance
func New(frontendURL, apiKey string, timeout time.Duration, retryAttempts int) *Pusher {
	return &Pusher{
		frontendURL:   frontendURL,
		apiKey:        apiKey,
		timeout:       timeout,
		retryAttempts: retryAttempts,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Push sends the current status to the frontend
func (p *Pusher) Push(statuses []models.ServiceStatus) error {
	// Create payload
	payload := models.FrontendPayload{
		Services:       statuses,
		LastUpdateTime: time.Now(),
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Try multiple times with exponential backoff
	var lastErr error
	for attempt := 1; attempt <= p.retryAttempts; attempt++ {
		err := p.sendRequest(jsonData)
		if err == nil {
			log.Printf("Successfully pushed status update to frontend (attempt %d/%d)", attempt, p.retryAttempts)
			return nil
		}

		lastErr = err
		log.Printf("Failed to push status update (attempt %d/%d): %v", attempt, p.retryAttempts, err)

		// Wait before retrying (exponential backoff)
		if attempt < p.retryAttempts {
			backoff := time.Duration(attempt*attempt) * time.Second
			time.Sleep(backoff)
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", p.retryAttempts, lastErr)
}

// sendRequest sends a single HTTP POST request
func (p *Pusher) sendRequest(jsonData []byte) error {
	req, err := http.NewRequest("POST", p.frontendURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
