package database

import (
	"database/sql"
	"fmt"
	"service-monitor/internal/models"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

// New creates a new database connection and initializes the schema
func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// initSchema creates all necessary tables
func (db *DB) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS services (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		ip_address TEXT NOT NULL,
		description TEXT,
		enabled BOOLEAN DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS status_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		service_id INTEGER NOT NULL,
		status TEXT NOT NULL,
		response_time_ms INTEGER,
		checked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_status_history_service_time
		ON status_history(service_id, checked_at DESC);

	CREATE TABLE IF NOT EXISTS alert_recipients (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL UNIQUE,
		enabled BOOLEAN DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS alert_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		service_id INTEGER NOT NULL,
		recipient_email TEXT NOT NULL,
		status TEXT NOT NULL,
		error_message TEXT,
		sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (service_id) REFERENCES services(id)
	);

	CREATE TABLE IF NOT EXISTS config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Insert default config values if they don't exist
	INSERT OR IGNORE INTO config (key, value) VALUES
		('smtp_host', ''),
		('smtp_port', '587'),
		('smtp_username', ''),
		('smtp_password', ''),
		('smtp_from', '');
	`

	_, err := db.conn.Exec(schema)
	return err
}

// Services

func (db *DB) CreateService(name, ipAddress, description string) (*models.Service, error) {
	result, err := db.conn.Exec(
		"INSERT INTO services (name, ip_address, description) VALUES (?, ?, ?)",
		name, ipAddress, description,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return db.GetService(int(id))
}

func (db *DB) GetService(id int) (*models.Service, error) {
	var s models.Service
	err := db.conn.QueryRow(
		"SELECT id, name, ip_address, description, enabled, created_at, updated_at FROM services WHERE id = ?",
		id,
	).Scan(&s.ID, &s.Name, &s.IPAddress, &s.Description, &s.Enabled, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (db *DB) GetAllServices() ([]models.Service, error) {
	rows, err := db.conn.Query(
		"SELECT id, name, ip_address, description, enabled, created_at, updated_at FROM services ORDER BY name",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []models.Service
	for rows.Next() {
		var s models.Service
		if err := rows.Scan(&s.ID, &s.Name, &s.IPAddress, &s.Description, &s.Enabled, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		services = append(services, s)
	}
	return services, rows.Err()
}

func (db *DB) GetEnabledServices() ([]models.Service, error) {
	rows, err := db.conn.Query(
		"SELECT id, name, ip_address, description, enabled, created_at, updated_at FROM services WHERE enabled = 1 ORDER BY name",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []models.Service
	for rows.Next() {
		var s models.Service
		if err := rows.Scan(&s.ID, &s.Name, &s.IPAddress, &s.Description, &s.Enabled, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		services = append(services, s)
	}
	return services, rows.Err()
}

func (db *DB) UpdateService(id int, name, ipAddress, description string, enabled bool) error {
	_, err := db.conn.Exec(
		"UPDATE services SET name = ?, ip_address = ?, description = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		name, ipAddress, description, enabled, id,
	)
	return err
}

func (db *DB) DeleteService(id int) error {
	_, err := db.conn.Exec("DELETE FROM services WHERE id = ?", id)
	return err
}

// Status History

func (db *DB) AddStatusHistory(serviceID int, status string, responseTimeMs int) error {
	_, err := db.conn.Exec(
		"INSERT INTO status_history (service_id, status, response_time_ms) VALUES (?, ?, ?)",
		serviceID, status, responseTimeMs,
	)
	return err
}

func (db *DB) GetServiceHistory(serviceID int, limit int) ([]models.StatusHistory, error) {
	rows, err := db.conn.Query(
		"SELECT id, service_id, status, response_time_ms, checked_at FROM status_history WHERE service_id = ? ORDER BY checked_at DESC LIMIT ?",
		serviceID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []models.StatusHistory
	for rows.Next() {
		var h models.StatusHistory
		if err := rows.Scan(&h.ID, &h.ServiceID, &h.Status, &h.ResponseTimeMs, &h.CheckedAt); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, rows.Err()
}

func (db *DB) GetLastStatus(serviceID int) (*models.StatusHistory, error) {
	var h models.StatusHistory
	err := db.conn.QueryRow(
		"SELECT id, service_id, status, response_time_ms, checked_at FROM status_history WHERE service_id = ? ORDER BY checked_at DESC LIMIT 1",
		serviceID,
	).Scan(&h.ID, &h.ServiceID, &h.Status, &h.ResponseTimeMs, &h.CheckedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// Alert Recipients

func (db *DB) CreateAlertRecipient(email string) (*models.AlertRecipient, error) {
	result, err := db.conn.Exec(
		"INSERT INTO alert_recipients (email) VALUES (?)",
		email,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return db.GetAlertRecipient(int(id))
}

func (db *DB) GetAlertRecipient(id int) (*models.AlertRecipient, error) {
	var ar models.AlertRecipient
	err := db.conn.QueryRow(
		"SELECT id, email, enabled, created_at FROM alert_recipients WHERE id = ?",
		id,
	).Scan(&ar.ID, &ar.Email, &ar.Enabled, &ar.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &ar, nil
}

func (db *DB) GetAllAlertRecipients() ([]models.AlertRecipient, error) {
	rows, err := db.conn.Query(
		"SELECT id, email, enabled, created_at FROM alert_recipients ORDER BY email",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recipients []models.AlertRecipient
	for rows.Next() {
		var ar models.AlertRecipient
		if err := rows.Scan(&ar.ID, &ar.Email, &ar.Enabled, &ar.CreatedAt); err != nil {
			return nil, err
		}
		recipients = append(recipients, ar)
	}
	return recipients, rows.Err()
}

func (db *DB) GetEnabledAlertRecipients() ([]models.AlertRecipient, error) {
	rows, err := db.conn.Query(
		"SELECT id, email, enabled, created_at FROM alert_recipients WHERE enabled = 1",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recipients []models.AlertRecipient
	for rows.Next() {
		var ar models.AlertRecipient
		if err := rows.Scan(&ar.ID, &ar.Email, &ar.Enabled, &ar.CreatedAt); err != nil {
			return nil, err
		}
		recipients = append(recipients, ar)
	}
	return recipients, rows.Err()
}

func (db *DB) UpdateAlertRecipient(id int, email string, enabled bool) error {
	_, err := db.conn.Exec(
		"UPDATE alert_recipients SET email = ?, enabled = ? WHERE id = ?",
		email, enabled, id,
	)
	return err
}

func (db *DB) DeleteAlertRecipient(id int) error {
	_, err := db.conn.Exec("DELETE FROM alert_recipients WHERE id = ?", id)
	return err
}

// Alert Log

func (db *DB) LogAlert(serviceID int, recipientEmail, status, errorMessage string) error {
	_, err := db.conn.Exec(
		"INSERT INTO alert_log (service_id, recipient_email, status, error_message) VALUES (?, ?, ?, ?)",
		serviceID, recipientEmail, status, errorMessage,
	)
	return err
}

func (db *DB) GetRecentAlertLogs(limit int) ([]models.AlertLog, error) {
	rows, err := db.conn.Query(
		"SELECT id, service_id, recipient_email, status, error_message, sent_at FROM alert_log ORDER BY sent_at DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.AlertLog
	for rows.Next() {
		var log models.AlertLog
		var errorMsg sql.NullString
		if err := rows.Scan(&log.ID, &log.ServiceID, &log.RecipientEmail, &log.Status, &errorMsg, &log.SentAt); err != nil {
			return nil, err
		}
		if errorMsg.Valid {
			log.ErrorMessage = errorMsg.String
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func (db *DB) GetLastAlertForService(serviceID int) (*models.AlertLog, error) {
	var log models.AlertLog
	var errorMsg sql.NullString
	err := db.conn.QueryRow(
		"SELECT id, service_id, recipient_email, status, error_message, sent_at FROM alert_log WHERE service_id = ? ORDER BY sent_at DESC LIMIT 1",
		serviceID,
	).Scan(&log.ID, &log.ServiceID, &log.RecipientEmail, &log.Status, &errorMsg, &log.SentAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if errorMsg.Valid {
		log.ErrorMessage = errorMsg.String
	}
	return &log, nil
}

// Config

func (db *DB) GetConfig(key string) (string, error) {
	var value string
	err := db.conn.QueryRow("SELECT value FROM config WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (db *DB) SetConfig(key, value string) error {
	_, err := db.conn.Exec(
		"INSERT OR REPLACE INTO config (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)",
		key, value,
	)
	return err
}

func (db *DB) GetAllConfig() (map[string]string, error) {
	rows, err := db.conn.Query("SELECT key, value FROM config")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	config := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		config[key] = value
	}
	return config, rows.Err()
}

// Cleanup old history (optional utility function)
func (db *DB) CleanupOldHistory(daysToKeep int) error {
	cutoffDate := time.Now().AddDate(0, 0, -daysToKeep)
	_, err := db.conn.Exec(
		"DELETE FROM status_history WHERE checked_at < ?",
		cutoffDate,
	)
	return err
}

// GetServiceHistoryPoints returns recent history as HistoryPoint slices for frontend display
func (db *DB) GetServiceHistoryPoints(serviceID int, limit int) ([]models.HistoryPoint, error) {
	rows, err := db.conn.Query(
		"SELECT status, checked_at FROM status_history WHERE service_id = ? ORDER BY checked_at DESC LIMIT ?",
		serviceID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []models.HistoryPoint
	for rows.Next() {
		var status string
		var checkedAt time.Time
		if err := rows.Scan(&status, &checkedAt); err != nil {
			return nil, err
		}
		points = append(points, models.HistoryPoint{
			Status:    status,
			CheckedAt: checkedAt.Format(time.RFC3339),
		})
	}
	return points, rows.Err()
}
