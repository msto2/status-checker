class StatusMonitor {
    constructor() {
        this.statusEndpoint = '/api/status';
        this.refreshInterval = 10000; // 10 seconds
        this.backendTimeout = 600000; // 10 minutes
        this.servicesContainer = document.getElementById('services-container');
        this.backendOfflineAlert = document.getElementById('backend-offline-alert');
        this.lastUpdateElement = document.getElementById('last-update');
    }

    async fetchStatus() {
        try {
            const response = await fetch(this.statusEndpoint);
            if (!response.ok) {
                throw new Error('Failed to fetch status');
            }
            const data = await response.json();
            return data;
        } catch (error) {
            console.error('Failed to fetch status:', error);
            return null;
        }
    }

    isBackendAlive(lastUpdateTime) {
        if (!lastUpdateTime) return false;
        const lastUpdate = new Date(lastUpdateTime);
        const now = new Date();
        return (now - lastUpdate) < this.backendTimeout;
    }

    updateStats(services, backendAlive) {
        const totalElement = document.getElementById('total-services');
        const upElement = document.getElementById('services-up');
        const downElement = document.getElementById('services-down');

        if (!backendAlive || !services) {
            totalElement.textContent = '-';
            upElement.textContent = '-';
            downElement.textContent = '-';
            return;
        }

        const total = services.length;
        const up = services.filter(s => s.status === 'up').length;
        const down = services.filter(s => s.status === 'down').length;

        totalElement.textContent = total;
        upElement.textContent = up;
        downElement.textContent = down;
    }

    renderServices(data) {
        // Check if backend is alive
        const backendAlive = data && this.isBackendAlive(data.last_update_time);

        // Update stats
        this.updateStats(data?.services, backendAlive);

        // Show/hide backend offline alert
        if (!backendAlive) {
            this.backendOfflineAlert.style.display = 'block';
        } else {
            this.backendOfflineAlert.style.display = 'none';
        }

        // Update last update time
        if (data && data.last_update_time) {
            const lastUpdate = new Date(data.last_update_time);
            this.lastUpdateElement.textContent = lastUpdate.toLocaleString();
        }

        // Render services
        if (!data || !backendAlive) {
            this.renderBackendOffline();
            return;
        }

        if (!data.services || data.services.length === 0) {
            this.servicesContainer.innerHTML = '<div class="loading">No services configured yet.</div>';
            return;
        }

        const html = `
            <div class="services-grid">
                ${data.services.map(service => this.renderServiceCard(service)).join('')}
            </div>
        `;
        this.servicesContainer.innerHTML = html;
    }

    renderServiceCard(service) {
        const statusClass = service.status || 'unknown';
        const responseTime = service.response_time_ms > 0 ? `${service.response_time_ms}ms` : 'N/A';

        return `
            <div class="service-card status-${statusClass}">
                <div class="service-header">
                    <div class="service-name">${this.escapeHtml(service.name)}</div>
                    <div class="status-badge ${statusClass}">${statusClass.toUpperCase()}</div>
                </div>
                <div class="service-details">
                    <div class="service-detail">
                        <span class="detail-label">IP Address</span>
                        <span class="detail-value">${this.escapeHtml(service.ip_address)}</span>
                    </div>
                    <div class="service-detail">
                        <span class="detail-label">Response Time</span>
                        <span class="detail-value">${responseTime}</span>
                    </div>
                </div>
            </div>
        `;
    }

    renderBackendOffline() {
        this.servicesContainer.innerHTML = `
            <div class="backend-offline-warning">
                <h2>⚠️ Monitoring System Offline</h2>
                <p>The backend monitoring system has not reported any updates in over 10 minutes.</p>
                <p>All services should be considered in an unknown state until the monitoring system comes back online.</p>
            </div>
        `;
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    start() {
        // Initial fetch
        this.update();

        // Set up periodic refresh
        setInterval(() => this.update(), this.refreshInterval);
    }

    async update() {
        const data = await this.fetchStatus();
        this.renderServices(data);
    }
}

// Initialize monitor when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    const monitor = new StatusMonitor();
    monitor.start();
});
