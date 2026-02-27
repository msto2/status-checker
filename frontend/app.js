class StatusMonitor {
    constructor() {
        this.statusEndpoint = '/api/status';
        this.refreshInterval = 30000; // 30 seconds
        this.backendTimeout = 600000; // 10 minutes
        this.servicesContainer = document.getElementById('services-container');
        this.backendOfflineAlert = document.getElementById('backend-offline-alert');
        this.lastUpdateElement = document.getElementById('last-update');
    }

    async fetchStatus() {
        try {
            // Add cache-busting to ensure fresh data
            const url = `${this.statusEndpoint}?t=${Date.now()}`;
            const response = await fetch(url, {
                cache: 'no-store',
                headers: {
                    'Cache-Control': 'no-cache'
                }
            });
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

    renderServices(data) {
        // Check if backend is alive
        const backendAlive = data && this.isBackendAlive(data.last_update_time);

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
        const statusText = statusClass === 'up' ? 'Online' : statusClass === 'down' ? 'Offline' : 'Unknown';

        // Render history bar (newest on right, oldest on left)
        const historyHtml = this.renderHistoryBar(service.history);

        return `
            <div class="service-card status-${statusClass}">
                <div class="service-header">
                    <div class="service-name">${this.escapeHtml(service.name)}</div>
                    <div class="status-badge ${statusClass}">${statusText}</div>
                </div>
                <div class="history-section">
                    <div class="history-label">Recent History</div>
                    <div class="history-bar">
                        ${historyHtml}
                    </div>
                </div>
            </div>
        `;
    }

    renderHistoryBar(history) {
        if (!history || history.length === 0) {
            return '<div class="history-empty">No history yet</div>';
        }

        // Reverse so oldest is first (left) and newest is last (right)
        const reversed = [...history].reverse();

        return reversed.map((point, index) => {
            const isUp = point.status === 'up';
            const statusClass = isUp ? 'history-up' : 'history-down';
            const title = `${isUp ? 'Up' : 'Down'} - ${new Date(point.checked_at).toLocaleString()}`;
            return `<div class="history-segment ${statusClass}" title="${title}"></div>`;
        }).join('');
    }

    renderBackendOffline() {
        this.servicesContainer.innerHTML = `
            <div class="backend-offline-warning">
                <h2>Monitoring System Offline</h2>
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
