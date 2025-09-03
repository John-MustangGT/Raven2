# Pushover Integration Instructions

This integration adds comprehensive Pushover notification support to your Raven monitoring system with global settings and per-host/per-check overrides.

## Files to Add/Modify

### 1. New Files to Create:

```
internal/config/pushover.go          # Pushover configuration structures
internal/notifications/pushover.go   # Pushover API client implementation  
internal/web/notification_handlers.go # API endpoints for notification config
js/components/notifications-view.js  # Frontend notification settings UI
```

### 2. Files to Modify:

**internal/config/config.go** - Add Pushover field to Config struct:
```go
type Config struct {
    // ... existing fields ...
    Pushover   PushoverConfig   `yaml:"pushover"`     // ADD THIS LINE
    // ... rest of fields ...
}
```

**internal/monitoring/engine.go** - Add these changes:
- Import notifications package
- Add pushoverClient field to Engine struct  
- Initialize Pushover client in NewEngine()
- Add sendNotification(), TestPushoverConfig(), and GetNotificationStatus() methods
- Add notification cleanup routine in Start()

**internal/monitoring/scheduler.go** - Modify handleResult():
- Track previous status to detect changes
- Call handleNotification() when status changes
- Add shouldSendNotification() logic

**internal/web/server.go** - Add to setupRoutes():
```go
func (s *Server) setupRoutes() {
    // ... existing routes ...
    s.setupNotificationRoutes()  // ADD THIS LINE
}
```

**Main application (cmd/raven/main.go)** - No changes needed, configuration is loaded automatically.

### 3. Frontend Integration:

**Add to sidebar navigation** (js/components/sidebar.js):
```html
<li class="nav-item">
    <a class="nav-link" :class="{ active: currentView === 'notifications' }" 
       @click="$emit('set-view', 'notifications')">
        <i class="fas fa-bell"></i>
        <span>Notifications</span>
    </a>
</li>
```

**Add to main app** (js/app.js):
```javascript
// In components section:
'notifications-view': window.NotificationsView,

// In setView method:
else if (view === 'notifications') {
    // Load notification settings if needed
}
```

**Update index.html** - Add script tag:
```html
<script src="js/components/notifications-view.js"></script>
```

## Configuration Examples

### Basic Configuration:
```yaml
pushover:
  enabled: true
  user_key: "your-pushover-user-key"
  api_token: "your-app-token"
  priority: 0
  realert_interval: 1h
  max_realerts: 5
  send_recovery: true
```

### Advanced Configuration with Overrides:
```yaml
pushover:
  enabled: true
  user_key: "team-user-key"
  api_token: "app-token"
  priority: 0
  quiet_hours:
    enabled: true
    start_hour: 22
    end_hour: 7
    timezone: "America/New_York"
  realert_interval: 1h
  max_realerts: 5
  send_recovery: true
  title: "Production Monitor"
  url: "https://monitoring.company.com/hosts/{HOST_ID}"
  
  overrides:
    # Critical servers get emergency priority
    - name: "Critical Infrastructure"
      host_pattern: "(db|web)-prod-.*"
      priority: 2
      sound: "siren"
      realert_interval: 15m
      max_realerts: 10
      
    # Development servers - low priority
    - name: "Development"
      host_pattern: ".*-dev-.*"
      priority: -1
      max_realerts: 2
      
    # Database team notifications
    - name: "Database Alerts"
      check_pattern: "(mysql|postgres|redis).*"
      user_key: "dba-team-key"
      priority: 1
      sound: "tugboat"
```

### Per-Host/Check Overrides:
```yaml
hosts:
  - id: "web-01"
    name: "web-01"
    # ... other config ...
    notifications:
      pushover:
        priority: 2  # Emergency for main web server
        title: "🚨 MAIN WEBSITE"

checks:
  - id: "http-check"
    name: "Website Health"
    # ... other config ...
    notifications:
      pushover:
        priority: 2
        message_template: |
          🌐 WEBSITE ALERT: {SEVERITY}
          URL: http://{HOST_IP}/health
          Status: {OUTPUT}
          Duration: {DURATION}
```

## API Endpoints

- `GET /api/notifications/status` - Get notification system status
- `GET /api/notifications/pushover/config` - Get current Pushover config (sanitized)
- `PUT /api/notifications/pushover/config` - Update Pushover configuration
- `POST /api/notifications/pushover/test` - Test current Pushover settings
- `POST /api/notifications/pushover/test-config` - Test custom Pushover settings

## Template Variables

Available variables for custom message templates:

- `{HOST_NAME}` - Host name
- `{HOST_DISPLAY_NAME}` - Host display name  
- `{HOST_IP}` - Host IP address
- `{HOST_HOSTNAME}` - Host hostname
- `{HOST_GROUP}` - Host group
- `{CHECK_NAME}` - Check name
- `{CHECK_TYPE}` - Check type
- `{SEVERITY}` - Alert severity (OK, WARNING, CRITICAL, UNKNOWN)
- `{OUTPUT}` - Check output message
- `{TIMESTAMP}` - Alert timestamp
- `{DURATION}` - How long the issue has persisted
- `{ALERT_COUNT}` - Number of times alert has been sent

## Dependencies

Add to `go.mod`:
```go
// No additional dependencies needed beyond existing ones
```

## Build Instructions

1. Copy all new files to their respective locations
2. Update existing files with the specified changes
3. Rebuild the application:
```bash
go build -o bin/raven ./cmd/raven
```

## Setup Instructions

1. Create a Pushover application at https://pushover.net/apps/build
2. Get your User Key from https://pushover.net/
3. Configure Raven with your keys:
```bash
./bin/raven -config config.yaml
```
4. Access the web UI and go to Notifications →
