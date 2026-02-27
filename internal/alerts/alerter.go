package alerts

import (
	"fmt"
	"log"
	"net/smtp"
	"service-monitor/internal/database"
	"service-monitor/internal/models"
	"strings"
	"time"
)

// Alerter handles email alerts for service status changes
type Alerter struct {
	db               *database.DB
	cooldownDuration time.Duration
}

// New creates a new alerter instance
func New(db *database.DB, cooldownMinutes int) *Alerter {
	return &Alerter{
		db:               db,
		cooldownDuration: time.Duration(cooldownMinutes) * time.Minute,
	}
}

// SendAlert sends an email alert for a service status change
func (a *Alerter) SendAlert(service models.Service, newStatus string, oldStatus string) {
	// Only send alerts when service goes down
	if newStatus != "down" {
		log.Printf("Service %s is now %s (was %s), no alert needed", service.Name, newStatus, oldStatus)
		return
	}

	// Check cooldown period
	if a.isInCooldown(service.ID) {
		log.Printf("Alert for service %s is in cooldown period, skipping", service.Name)
		return
	}

	// Get SMTP configuration
	smtpConfig, err := a.getSMTPConfig()
	if err != nil {
		log.Printf("Failed to get SMTP configuration: %v", err)
		return
	}

	// Check if SMTP is configured
	if smtpConfig.Host == "" || smtpConfig.From == "" {
		log.Println("SMTP not configured, skipping email alerts")
		return
	}

	// Get recipients
	recipients, err := a.db.GetEnabledAlertRecipients()
	if err != nil {
		log.Printf("Failed to get alert recipients: %v", err)
		return
	}

	if len(recipients) == 0 {
		log.Println("No alert recipients configured")
		return
	}

	// Send to all recipients
	for _, recipient := range recipients {
		err := a.sendEmail(smtpConfig, recipient.Email, service, newStatus, oldStatus)
		if err != nil {
			log.Printf("Failed to send alert to %s: %v", recipient.Email, err)
			a.db.LogAlert(service.ID, recipient.Email, "failed", err.Error())
		} else {
			log.Printf("Alert sent to %s for service %s", recipient.Email, service.Name)
			a.db.LogAlert(service.ID, recipient.Email, "sent", "")
		}
	}
}

// isInCooldown checks if an alert was recently sent for this service
func (a *Alerter) isInCooldown(serviceID int) bool {
	lastAlert, err := a.db.GetLastAlertForService(serviceID)
	if err != nil || lastAlert == nil {
		return false
	}

	timeSinceLastAlert := time.Since(lastAlert.SentAt)
	return timeSinceLastAlert < a.cooldownDuration
}

// SMTPConfig represents SMTP configuration
type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

// getSMTPConfig retrieves SMTP configuration from database
func (a *Alerter) getSMTPConfig() (*SMTPConfig, error) {
	config, err := a.db.GetAllConfig()
	if err != nil {
		return nil, err
	}

	return &SMTPConfig{
		Host:     config["smtp_host"],
		Port:     config["smtp_port"],
		Username: config["smtp_username"],
		Password: config["smtp_password"],
		From:     config["smtp_from"],
	}, nil
}

// sendEmail sends an email using SMTP
func (a *Alerter) sendEmail(config *SMTPConfig, to string, service models.Service, newStatus string, oldStatus string) error {
	// Build email message
	subject := fmt.Sprintf("Alert: Service %s is DOWN", service.Name)
	body := a.buildEmailBody(service, newStatus, oldStatus)

	message := []byte(fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=UTF-8\r\n"+
			"\r\n"+
			"%s\r\n",
		config.From, to, subject, body,
	))

	// Setup authentication
	var auth smtp.Auth
	if config.Username != "" && config.Password != "" {
		auth = smtp.PlainAuth("", config.Username, config.Password, config.Host)
	}

	// Send email
	addr := fmt.Sprintf("%s:%s", config.Host, config.Port)
	err := smtp.SendMail(addr, auth, config.From, []string{to}, message)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// buildEmailBody creates an HTML email body
func (a *Alerter) buildEmailBody(service models.Service, newStatus string, oldStatus string) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<style>
		body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
		.container { max-width: 600px; margin: 0 auto; padding: 20px; }
		.header { background-color: #d32f2f; color: white; padding: 20px; border-radius: 5px 5px 0 0; }
		.content { background-color: #f5f5f5; padding: 20px; border-radius: 0 0 5px 5px; }
		.info { background-color: white; padding: 15px; margin: 10px 0; border-left: 4px solid #d32f2f; }
		.label { font-weight: bold; color: #666; }
		.footer { margin-top: 20px; font-size: 12px; color: #999; }
	</style>
</head>
<body>
	<div class="container">
		<div class="header">
			<h2>⚠️ Service Alert</h2>
		</div>
		<div class="content">
			<p>A service status change has been detected:</p>
			<div class="info">
				<p><span class="label">Service Name:</span> %s</p>
				<p><span class="label">IP Address:</span> %s</p>
				<p><span class="label">Previous Status:</span> %s</p>
				<p><span class="label">Current Status:</span> <strong style="color: #d32f2f;">%s</strong></p>
				<p><span class="label">Timestamp:</span> %s</p>
				%s
			</div>
			<p>Please investigate the issue as soon as possible.</p>
			<div class="footer">
				<p>This is an automated alert from Service Monitor.</p>
			</div>
		</div>
	</div>
</body>
</html>
	`, service.Name, service.IPAddress, strings.ToUpper(oldStatus), strings.ToUpper(newStatus), timestamp, formatDescription(service.Description))
}

// formatDescription formats the service description if present
func formatDescription(description string) string {
	if description != "" {
		return fmt.Sprintf(`<p><span class="label">Description:</span> %s</p>`, description)
	}
	return ""
}

// TestEmail sends a test email to verify SMTP configuration
func (a *Alerter) TestEmail(recipientEmail string) error {
	config, err := a.getSMTPConfig()
	if err != nil {
		return fmt.Errorf("failed to get SMTP configuration: %w", err)
	}

	if config.Host == "" || config.From == "" {
		return fmt.Errorf("SMTP not configured")
	}

	// Create a test service
	testService := models.Service{
		Name:        "Test Service",
		IPAddress:   "127.0.0.1",
		Description: "This is a test email to verify SMTP configuration",
	}

	return a.sendEmail(config, recipientEmail, testService, "down", "up")
}
