# Pushover Notifications for Raven Network Monitoring

This guide explains how to set up and configure Pushover push notifications for the Raven monitoring system. With Pushover integration, you'll receive instant mobile notifications when your monitored services experience issues or recover from problems.

## Table of Contents

1. [What is Pushover?](#what-is-pushover)
2. [Prerequisites](#prerequisites)
3. [Getting Pushover Credentials](#getting-pushover-credentials)
4. [Configuration](#configuration)
5. [Web Interface Setup](#web-interface-setup)
6. [Message Templates](#message-templates)
7. [Throttling and Rate Limiting](#throttling-and-rate-limiting)
8. [Testing Notifications](#testing-notifications)
9. [Troubleshooting](#troubleshooting)
10. [Best Practices](#best-practices)

## What is Pushover?

[Pushover](https://pushover.net/) is a simple push notification service that makes it easy to get real-time notifications on your phone, tablet, desktop, and smartwatch. It's perfect for monitoring systems because it's reliable, has a simple API, and supports various notification priorities and sounds.

### Why Pushover for Raven?

- **Instant delivery**: Get notified within seconds of an issue
- **Cross-platform**: Works on iOS, Android, and desktop
- **Priority levels**: From silent notifications to emergency alerts
- **Rich formatting**: Support for emojis and basic formatting
- **Delivery confirmation**: Emergency priority messages require acknowledgment
- **Offline delivery**: Messages are delivered when your device comes online

## Prerequisites

Before setting up Pushover notifications in Raven:

1. **Pushover Account**: Sign up at [https://pushover.net/](https://pushover.net/)
2. **Pushover App**: Install the mobile app on your device(s)
3. **Raven v2.0+**: Ensure you're running Raven with notification support
4. **API Access**: You'll need to create an application in Pushover to get API credentials

## Getting Pushover Credentials

### Step 1: Create a Pushover Application

1. Log in to your Pushover account at [https://pushover.net/](https://pushover.net/)
2. Click "Create an Application/API Token" or go to [https://pushover.net/apps/build](https://pushover.net/apps/build)
3. Fill out the application form:
   - **Name**: "Raven Network Monitor" (or your preferred name)
   - **Type**: Application
   - **Description**: "Network monitoring alerts from Raven"
   - **URL**: Your Raven web interface URL (optional)
   - **Icon**: Upload a custom icon if desired (optional)
4. Click "Create Application"

### Step 2: Get Your Credentials

After creating the application, you'll see two important pieces of information:

- **API Token/Key**: A 30-character string (e.g., `azGDORePK8gMaC0QOYAMyEEuzJnyUi`)
- **User Key**: Found on your main Pushover dashboard, also 30 characters

**Important**: Keep these credentials secure and never share them publicly.

## Configuration

### Method 1: Configuration File (config.yaml)

Add the notifications section to your `config.yaml`:

```yaml
notifications:
  enabled: true
  
  pushover:
    enabled: true
    api_token: "YOUR_API_TOKEN_HERE"
    user_key: "YOUR_USER_KEY_HERE"
    priority: 0
    sound: "pushover"
    
    # Message templates
    title: "🐦 Raven Alert: {{.Host}}"
    template: "{{.StatusEmoji}} {{.Check}} on {{.Host}} is {{.Status}}\n\nOutput: {{.Output}}\nTime: {{.Timestamp}}"
    
    # Notification conditions
    only_on_state: ["critical", "warning", "recovery"]
    
    # Rate limiting
    throttle:
      enabled: true
      window: 15m
      max_per_host: 5
      max_total: 20
```

### Method 2: Environment Variables

You can also set credentials via environment variables for security:

```bash
export RAVEN_PUSHOVER_API_TOKEN="your_api_token_here"
export RAVEN_PUSHOVER_USER_KEY="your_user_key_here"
```

Then in your config:

```yaml
notifications:
  enabled: true
  pushover:
    enabled: true
    api_token: "${RAVEN_PUSHOVER_API_TOKEN}"
    user_key: "${RAVEN_PUSHOVER_USER_KEY}"
```

## Web Interface Setup

Raven provides a comprehensive web interface for managing notification settings:

### Accessing Notification Settings

1. Open the Raven web interface
2. Click "Notifications" in the sidebar
3. Enable notifications with the master switch
4. Configure Pushover settings in the dedicated section

### Configuration Options

#### **Basic Settings**
- **Enable Notifications**: Master switch for all notification services
- **Enable Pushover**: Specific toggle for Pushover integration
- **API Token**: Your application's API token from Pushover
- **User Key**: Your user key from Pushover

#### **Priority Levels**
- **Silent (-2)**: No notification generated
- **Quiet (-1)**: Quiet notification (no sound/vibration)
- **Normal (0)**: Standard notification with sound
- **High (1)**: High-priority notification (bypasses quiet hours)
- **Emergency (2)**: Requires user acknowledgment (see Emergency Priority section)

#### **Emergency Priority Settings**
When using emergency priority (level 2):
- **Retry Interval**: How often to retry (minimum 30 seconds)
- **Expire After**: Stop retrying after this time (maximum 3 hours)

Example: Retry every 60 seconds for up to 1 hour until acknowledged.

#### **Notification Sounds**
Choose from 20+ available sounds including:
- pushover (default)
- bike, bugle, cashregister
- classical, cosmic, falling
- siren, spacealarm, tugboat
- silent (no sound)

#### **Device Targeting**
- Leave empty to send to all your devices
- Enter a specific device name to target one device

## Message Templates

Raven uses Go templates for customizable notification messages. Templates support variables that are automatically replaced with real monitoring data.

### Available Variables

#### Host Variables
- `{{.Host}}` - Host name
- `{{.HostDisplay}}` - Host display name
- `{{.Group}}` - Host group
- `{{.IPv4}}` - Host IP address
- `{{.Hostname}}` - Host hostname

#### Check Variables
- `{{.Check}}` - Check name
- `{{.Status}}` - Current status (ok, warning, critical, unknown)
- `{{.Output}}` - Check output message
- `{{.LongOutput}}` - Extended check output

#### Status Variables
- `{{.StatusEmoji}}` - Status emoji (✅ ⚠️ 🚨 ❓)
- `{{.IsRecovery}}` - True if recovering to OK status
- `{{.PreviousStatus}}` - Previous status before change
- `{{.Duration}}` - Check execution duration
- `{{.Timestamp}}` - Check timestamp

### Template Examples

#### Simple Alert
```yaml
title: "Raven Alert: {{.Host}}"
template: "{{.Check}} on {{.Host}} is {{.Status}}: {{.Output}}"
```
**Result**: 
> **Title**: Raven Alert: web-server-01
> **Message**: HTTP Check on web-server-01 is critical: Connection refused

#### Detailed Alert with Emojis
```yaml
title: "{{.StatusEmoji}} {{.Host}} - {{.Status}}"
template: |
  {{.StatusEmoji}} {{.Check}} on {{.Host}} ({{.Group}}) is {{.Status}}
  
  Message: {{.Output}}
  Duration: {{.Duration}}
  Time: {{.Timestamp}}
```

#### Recovery-Specific Alert
```yaml
title: "✅ Recovery: {{.Host}}"
template: |
  {{if .IsRecovery}}🎉 RECOVERED! {{end}}{{.Check}} on {{.Host}} is now {{.Status}}
  
  Previous: {{.PreviousStatus}}
  Message: {{.Output}}
```

### Template Testing

Use the web interface template editor to:
- Insert variables with quick-click buttons
- Preview how templates will look
- Use pre-built examples as starting points
- Validate template syntax before saving

## Throttling and Rate Limiting

To prevent notification spam, Raven includes configurable throttling:

### Throttling Settings
- **Window**: Time period for counting notifications (e.g., 15 minutes)
- **Max Per Host**: Maximum notifications per host in the time window
- **Max Total**: Maximum total notifications in the time window

### Example Scenarios

**Configuration**:
```yaml
throttle:
  enabled: true
  window: 15m
  max_per_host: 5
  max_total: 20
```

**Behavior**:
- If `web-server-01` has 5 issues in 15 minutes, further notifications for that host are suppressed
- If any combination of hosts generates 20 total notifications in 15 minutes, all further notifications are suppressed until the window resets

### Throttling Bypass

Certain notifications can bypass throttling:
- **Recovery notifications**: Always sent when services return to OK
- **Critical escalations**: When a warning becomes critical
- **Emergency priority**: When manually set to priority 2

## Testing Notifications

### Web Interface Test

1. Go to Notifications → Test Notification section
2. Enter a custom test message
3. Click "Send Test Notification"
4. Check your device(s) for the notification

### Command Line Test

You can test notifications via the API:

```bash
curl -X POST http://localhost:8000/api/notifications/test \
  -H "Content-Type: application/json" \
  -d '{"message": "Test from command line"}'
```

### What to Expect

A successful test notification will:
- Appear on your device within 5-10 seconds
- Use your configured sound and priority
- Show "🧪 Test notification from Raven monitoring system!"
- Confirm your configuration is working correctly

## Troubleshooting

### Common Issues

#### 1. No Notifications Received

**Check Configuration**:
```bash
# Verify configuration
curl http://localhost:8000/api/notifications/settings
```

**Verify Credentials**:
- Ensure API token is exactly 30 characters
- Ensure user key is exactly 30 characters
- Check for typos or extra spaces

**Check Pushover App**:
- Ensure the app is installed and logged in
- Check device notification settings
- Verify you're using the correct user account

#### 2. API Errors

**Invalid Token Error**:
```
pushover API error: ["application token is invalid"]
```
- Double-check your API token from the Pushover apps page
- Ensure you're using the application token, not your user key

**Invalid User Error**:
```
pushover API error: ["user identifier is invalid"]
```
- Verify your user key from the main Pushover dashboard
- Ensure the user key belongs to the same account as the application

#### 3. Template Errors

**Template Parse Error**:
```
invalid message template: template: message:1: unexpected "}"
```
- Check for mismatched curly braces in templates
- Validate Go template syntax
- Use the web interface template validator

#### 4. Emergency Priority Issues

**Retry Too Low Error**:
```
retry must be at least 30 seconds for emergency priority
```
- Emergency priority requires retry ≥ 30 seconds
- Emergency priority requires expire between 60-10800 seconds

### Debugging Steps

1. **Check Raven logs**:
   ```bash
   tail -f raven.log | grep -i notification
   ```

2. **Verify API connectivity**:
   ```bash
   curl -X POST https://api.pushover.net/1/messages.json \
     -F "token=YOUR_API_TOKEN" \
     -F "user=YOUR_USER_KEY" \
     -F "message=Test from command line"
   ```

3. **Check notification stats**:
   ```bash
   curl http://localhost:8000/api/notifications/stats
   ```

### Log Analysis

Successful notifications show:
```
INFO[2024-01-15T14:30:22Z] Pushover notification sent successfully host="web-server-01" priority=0 sound="pushover"
```

Failed notifications show:
```
ERROR[2024-01-15T14:30:22Z] Failed to send Pushover notification error="pushover API error: [\"application token is invalid\"]"
```

## Best Practices

### Security
- **Never commit credentials**: Use environment variables or secure configuration management
- **Rotate tokens periodically**: Create new API tokens every 6-12 months
- **Limit access**: Only give API tokens to necessary team members

### Configuration
- **Start with normal priority**: Use priority 0 for most alerts
- **Reserve emergency for true emergencies**: Use priority 2 sparingly
- **Enable throttling**: Prevent notification spam with sensible limits
- **Test thoroughly**: Verify all scenarios work as expected

### Message Design
- **Use clear, actionable messages**: Include enough context to understand the issue
- **Include relevant details**: Host, service, error message, and timestamp
- **Use emojis judiciously**: They help quick recognition but don't overdo it
- **Keep titles concise**: Focus on the most important information

### Monitoring Strategy
- **Critical alerts first**: Ensure critical service failures always notify
- **Include recoveries**: Know when problems are resolved
- **Group related alerts**: Use host groups to organize notifications
- **Monitor the monitor**: Set up alerts for Raven itself

### Team Coordination
- **Shared applications**: Create separate Pushover applications for different teams
- **Escalation paths**: Use priority levels to implement escalation procedures
- **On-call integration**: Integrate with on-call schedules and handoffs
- **Documentation**: Keep notification procedures documented and up-to-date

## Advanced Configuration

### Multiple Recipients

For team notifications, you can use group delivery:

1. Create a Pushover delivery group at [https://pushover.net/groups](https://pushover.net/groups)
2. Use the group key instead of your user key in the configuration
3. Manage team members through the Pushover web interface

### Priority-Based Routing

Configure different priorities for different types of alerts:

```yaml
# In your check configurations
checks:
  - id: "critical-service-check"
    # ... other settings
    options:
      pushover_priority: 2  # Emergency for critical services
  
  - id: "non-critical-check" 
    # ... other settings
    options:
      pushover_priority: -1  # Quiet for non-critical
```

### Custom Device Targeting

Route different alert types to different devices:

```yaml
# Database alerts to work phone
database_notifications:
  device: "work-phone"
  priority: 1

# Infrastructure alerts to personal device
infrastructure_notifications:
  device: "personal-device"  
  priority: 0
```

## Integration with Other Tools

### Prometheus AlertManager

If you're also using Prometheus, you can coordinate alerts:

```yaml
# Raven handles infrastructure monitoring
# Prometheus handles application metrics
# Configure different notification channels to avoid duplication
```

### PagerDuty Integration

For enterprise environments, consider using Raven notifications as a first layer:

1. Raven sends immediate Pushover notifications
2. If not acknowledged within X minutes, escalate to PagerDuty
3. Use priority levels to determine escalation timing

### Incident Response

Integrate notifications with your incident response process:

1. **Detection**: Raven sends immediate notification
2. **Assessment**: Responder evaluates via mobile notification
3. **Response**: Responder can acknowledge and begin resolution
4. **Recovery**: Raven sends recovery notification when resolved

This completes the comprehensive Pushover integration for Raven Network Monitoring. The system provides reliable, immediate notifications that help ensure you're always aware of your network's status, wherever you are.
