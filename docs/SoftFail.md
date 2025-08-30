# Soft Fail Feature Documentation

## Overview

The soft fail feature prevents false alerts by requiring multiple consecutive non-OK check results before changing a host's state from OK to a problem state (WARNING, CRITICAL, or UNKNOWN). This helps reduce alert fatigue caused by temporary network issues, brief service interruptions, or other transient problems.

## How It Works

### Basic Behavior

1. **Initial State**: When a host/check combination is first monitored, it starts in an UNKNOWN state
2. **OK to Problem State**: When transitioning from OK to any problem state, soft fail requires multiple consecutive non-OK results before changing the reported state
3. **Recovery**: When recovering from any problem state to OK, the state changes immediately (no soft fail delay)
4. **Problem to Problem**: When changing between different problem states, soft fail logic applies

### State Tracking

The system maintains two states for each host/check combination:
- **Current State**: The state reported to users and stored in the database
- **Pending State**: The actual state returned by the most recent check

During soft fail periods, these states may differ.

## Configuration

### Global Settings

Configure soft fail behavior in the main `config.yaml`:

```yaml
monitoring:
  soft_fail_enabled: true    # Enable soft fail globally
  default_threshold: 3       # Default consecutive failures needed
```

### Per-Check Settings

Override global settings for individual checks:

```yaml
checks:
  - id: "ping"
    name: "PING Check"
    type: "ping"
    threshold: 5              # Need 5 consecutive failures
    soft_fail_enabled: true   # Enable for this check
    
  - id: "critical_service"
    name: "Critical Service"
    type: "nagios"
    threshold: 2              # Need only 2 consecutive failures
    
  - id: "immediate_alert"
    name: "Immediate Alert"
    type: "nagios"
    threshold: 1              # Single failure triggers state change
    soft_fail_enabled: false  # Disable soft fail entirely
```

### Precedence Rules

1. Check-level `soft_fail_enabled` overrides global setting
2. Check-level `threshold` overrides global `default_threshold`
3. If `threshold` is 1, soft fail is effectively disabled
4. If `soft_fail_enabled` is false, soft fail is disabled regardless of threshold

## Examples

### Example 1: Basic Soft Fail (Threshold = 3)

```
Time    Check Result    Current State    Pending State    Count    Action
T1      OK             OK               OK               1        Normal operation
T2      CRITICAL       OK               CRITICAL         1        Soft fail begins
T3      CRITICAL       OK               CRITICAL         2        Still in soft fail
T4      CRITICAL       CRITICAL         CRITICAL         3        State change confirmed
T5      OK             OK               OK               1        Immediate recovery
```

### Example 2: Intermittent Failures

```
Time    Check Result    Current State    Pending State    Count    Action
T1      OK             OK               OK               1        Normal operation
T2      CRITICAL       OK               CRITICAL         1        Soft fail begins
T3      OK             OK               OK               1        Recovery, counter reset
T4      CRITICAL       OK               CRITICAL         1        Soft fail begins again
T5      CRITICAL       OK               CRITICAL         2        Still in soft fail
T6      OK             OK               OK               1        Recovery, counter reset
```

### Example 3: Disabled Soft Fail (Threshold = 1)

```
Time    Check Result    Current State    Pending State    Count    Action
T1      OK             OK               OK               1        Normal operation
T2      CRITICAL       CRITICAL         CRITICAL         1        Immediate state change
T3      OK             OK               OK               1        Immediate recovery
```

## Check Scheduling During Soft Fail

When a check is in soft fail mode (pending state differs from current state), the system schedules checks more frequently to quickly determine if the problem persists:

- **Normal interval**: Uses the interval configured for the current state
- **Soft fail interval**: Uses 1/3 of the normal interval (minimum 30 seconds)

This ensures faster detection of persistent problems while preventing excessive checking during normal operation.

## Output Modifications

During soft fail periods, check output is modified to indicate the soft fail status:

**Normal Output:**
```
PING OK - web01.example.com
```

**Soft Fail Output:**
```
SOFT FAIL (2/3) - PING CRITICAL - web01.example.com
```

**Long Output:**
```
Soft fail protection active. Consecutive non-OK results: 2/3 required.
Original output: PING CRITICAL - web01.example.com
Original long output: RTT: 150.2ms, Loss: 50%
```

## Metrics and Logging

### Logging

The system logs state changes with detailed information:

```
INFO[2025] State change confirmed after soft fail period
  key=web01:ping old_state=0 new_state=2 consecutive_count=3 threshold=3
```

During soft fail periods:

```
DEBUG[2025] Check completed
  host=web01 check=ping exit=2 reported=0 soft_fail=true consecutive=2 threshold=3
```

### Metrics

Metrics are recorded using the reported state (current state), not the actual check result, ensuring that dashboards and alerting systems only see confirmed state changes.

## Best Practices

### Threshold Selection

- **Network checks (ping)**: Use higher thresholds (3-5) for network stability
- **Critical services**: Use lower thresholds (2-3) for faster alerting
- **Transient services**: Use higher thresholds (4-5) for services with expected brief outages
- **Immediate alerting**: Use threshold 1 or disable soft fail

### Interval Configuration

Configure check intervals based on service criticality:

```yaml
interval:
  ok: 5m        # Normal monitoring frequency
  warning: 2m   # Increased frequency for warnings
  critical: 1m  # High frequency for critical issues
  unknown: 5m   # Normal frequency for unknown states
```

### Mixed Environments

Use include files to organize different soft fail policies:

```yaml
# config.d/critical-services.yaml
checks:
  - id: "database"
    threshold: 2
    soft_fail_enabled: true

# config.d/network-monitoring.yaml  
checks:
  - id: "network_ping"
    threshold: 5
    soft_fail_enabled: true
```

## Troubleshooting

### Common Issues

1. **State not changing**: Check if soft fail threshold is too high
2. **Immediate state changes**: Verify soft fail is enabled and threshold > 1
3. **Inconsistent behavior**: Ensure configuration is properly loaded and merged

### Debugging

Enable debug logging to see detailed soft fail behavior:

```yaml
logging:
  level: "debug"
```

Look for log entries containing `soft_fail=true` and `consecutive_count` fields.

### State Verification

The state tracker maintains internal state that can be examined during troubleshooting. Consider adding administrative endpoints to view current soft fail states if needed.
