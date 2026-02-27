package api

import (
	"encoding/json"
	"net/http"
	"service-monitor/internal/alerts"
	"service-monitor/internal/database"
	"service-monitor/internal/monitor"
	"strconv"

	"github.com/gorilla/mux"
)

// AdminAPI handles all admin endpoints
type AdminAPI struct {
	db        *database.DB
	scheduler *monitor.Scheduler
	alerter   *alerts.Alerter
}

// NewAdminAPI creates a new admin API instance
func NewAdminAPI(db *database.DB, scheduler *monitor.Scheduler, alerter *alerts.Alerter) *AdminAPI {
	return &AdminAPI{
		db:        db,
		scheduler: scheduler,
		alerter:   alerter,
	}
}

// SetupRoutes configures all API routes
func (api *AdminAPI) SetupRoutes(router *mux.Router) {
	// Health check
	router.HandleFunc("/health", api.healthHandler).Methods("GET")

	// Services
	router.HandleFunc("/api/services", api.getServicesHandler).Methods("GET")
	router.HandleFunc("/api/services", api.createServiceHandler).Methods("POST")
	router.HandleFunc("/api/services/{id:[0-9]+}", api.getServiceHandler).Methods("GET")
	router.HandleFunc("/api/services/{id:[0-9]+}", api.updateServiceHandler).Methods("PUT")
	router.HandleFunc("/api/services/{id:[0-9]+}", api.deleteServiceHandler).Methods("DELETE")
	router.HandleFunc("/api/services/{id:[0-9]+}/check", api.checkServiceHandler).Methods("POST")
	router.HandleFunc("/api/services/{id:[0-9]+}/history", api.getServiceHistoryHandler).Methods("GET")

	// Alert recipients
	router.HandleFunc("/api/alerts", api.getAlertRecipientsHandler).Methods("GET")
	router.HandleFunc("/api/alerts", api.createAlertRecipientHandler).Methods("POST")
	router.HandleFunc("/api/alerts/{id:[0-9]+}", api.updateAlertRecipientHandler).Methods("PUT")
	router.HandleFunc("/api/alerts/{id:[0-9]+}", api.deleteAlertRecipientHandler).Methods("DELETE")
	router.HandleFunc("/api/alerts/test", api.testEmailHandler).Methods("POST")
	router.HandleFunc("/api/alerts/logs", api.getAlertLogsHandler).Methods("GET")

	// Configuration
	router.HandleFunc("/api/config", api.getConfigHandler).Methods("GET")
	router.HandleFunc("/api/config", api.updateConfigHandler).Methods("PUT")
}

// Health check
func (api *AdminAPI) healthHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Services endpoints

func (api *AdminAPI) getServicesHandler(w http.ResponseWriter, r *http.Request) {
	services, err := api.db.GetAllServices()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get services")
		return
	}
	respondJSON(w, http.StatusOK, services)
}

func (api *AdminAPI) getServiceHandler(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid service ID")
		return
	}

	service, err := api.db.GetService(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Service not found")
		return
	}

	respondJSON(w, http.StatusOK, service)
}

func (api *AdminAPI) createServiceHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		IPAddress   string `json:"ip_address"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" || req.IPAddress == "" {
		respondError(w, http.StatusBadRequest, "Name and IP address are required")
		return
	}

	service, err := api.db.CreateService(req.Name, req.IPAddress, req.Description)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create service")
		return
	}

	respondJSON(w, http.StatusCreated, service)
}

func (api *AdminAPI) updateServiceHandler(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid service ID")
		return
	}

	var req struct {
		Name        string `json:"name"`
		IPAddress   string `json:"ip_address"`
		Description string `json:"description"`
		Enabled     bool   `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" || req.IPAddress == "" {
		respondError(w, http.StatusBadRequest, "Name and IP address are required")
		return
	}

	err = api.db.UpdateService(id, req.Name, req.IPAddress, req.Description, req.Enabled)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update service")
		return
	}

	service, _ := api.db.GetService(id)
	respondJSON(w, http.StatusOK, service)
}

func (api *AdminAPI) deleteServiceHandler(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid service ID")
		return
	}

	err = api.db.DeleteService(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete service")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Service deleted"})
}

func (api *AdminAPI) checkServiceHandler(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid service ID")
		return
	}

	status, err := api.scheduler.CheckServiceNow(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check service")
		return
	}

	respondJSON(w, http.StatusOK, status)
}

func (api *AdminAPI) getServiceHistoryHandler(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid service ID")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 100 // Default limit
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	history, err := api.db.GetServiceHistory(id, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get service history")
		return
	}

	respondJSON(w, http.StatusOK, history)
}

// Alert recipients endpoints

func (api *AdminAPI) getAlertRecipientsHandler(w http.ResponseWriter, r *http.Request) {
	recipients, err := api.db.GetAllAlertRecipients()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get alert recipients")
		return
	}
	respondJSON(w, http.StatusOK, recipients)
}

func (api *AdminAPI) createAlertRecipientHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" {
		respondError(w, http.StatusBadRequest, "Email is required")
		return
	}

	recipient, err := api.db.CreateAlertRecipient(req.Email)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create alert recipient")
		return
	}

	respondJSON(w, http.StatusCreated, recipient)
}

func (api *AdminAPI) updateAlertRecipientHandler(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid recipient ID")
		return
	}

	var req struct {
		Email   string `json:"email"`
		Enabled bool   `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" {
		respondError(w, http.StatusBadRequest, "Email is required")
		return
	}

	err = api.db.UpdateAlertRecipient(id, req.Email, req.Enabled)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update alert recipient")
		return
	}

	recipient, _ := api.db.GetAlertRecipient(id)
	respondJSON(w, http.StatusOK, recipient)
}

func (api *AdminAPI) deleteAlertRecipientHandler(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid recipient ID")
		return
	}

	err = api.db.DeleteAlertRecipient(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete alert recipient")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Alert recipient deleted"})
}

func (api *AdminAPI) testEmailHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" {
		respondError(w, http.StatusBadRequest, "Email is required")
		return
	}

	err := api.alerter.TestEmail(req.Email)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Test email sent successfully"})
}

func (api *AdminAPI) getAlertLogsHandler(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50 // Default limit
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	logs, err := api.db.GetRecentAlertLogs(limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get alert logs")
		return
	}

	respondJSON(w, http.StatusOK, logs)
}

// Configuration endpoints

func (api *AdminAPI) getConfigHandler(w http.ResponseWriter, r *http.Request) {
	config, err := api.db.GetAllConfig()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get configuration")
		return
	}
	respondJSON(w, http.StatusOK, config)
}

func (api *AdminAPI) updateConfigHandler(w http.ResponseWriter, r *http.Request) {
	var req map[string]string

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	for key, value := range req {
		if err := api.db.SetConfig(key, value); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to update configuration")
			return
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Configuration updated"})
}

// Helper functions

func getIDFromRequest(r *http.Request) (int, error) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	return strconv.Atoi(idStr)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
