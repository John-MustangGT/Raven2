# Alert Purging System Documentation

## Overview

The Raven v2.0 alert purging system automatically cleans up outdated alerts and database entries when hosts or checks are removed or renamed in the YAML configuration. This prevents the accumulation of stale data and ensures the monitoring system stays clean and efficient.

## Features

- **Automatic Purging**: Runs on startup and periodically to clean up stale data
- **Configuration-Driven**: Only keeps alerts for hosts/checks that exist in the current YAML config
- **Multiple Purge Types**: Handles orphaned hosts, checks, and status entries
- **Database Maintenance**: Includes compaction and optimization features
- **API Control**: RESTful endpoints for manual purge operations
- **Comprehensive Logging**: Detailed logging of all purge operations

## Configuration

### Database Configuration

```yaml
database:
  type: "boltdb"
  path: "./data/raven.db"
  backup_interval: "24h"
  cleanup_interval: "6h"        # How often to run automatic purging
  history_retention: "720h"     # 30 days of history retention
  compact_interval: "168h"      # Weekly database compaction
```

### Monitoring Configuration

```yaml
monitoring:
  default_interval: "5m"
  max_retries: 3
  timeout: "30s"
  batch_size: 100
  # Alert purging settings
  purge_on_startup: true        # Purge stale alerts on startup
  purge_orphaned_hosts: true    # Remove hosts not in config
  purge_orphaned_checks: true   # Remove checks not in config
```

## How It Works

### 1. Startup Purging

When Raven starts, it:

1. Loads the current YAML configuration
2. Syncs configuration to the database
3. Identifies valid host:check combinations
4. Purges any alerts for combinations that no longer exist
5. Removes orphaned hosts and checks from the database

### 2. Periodic Purging

The system runs automatic purging based on the `cleanup_interval`:

- **Every 6 hours** (default): Purge stale alerts
- **Daily at 2 AM**: Full database maintenance including compaction
- **Weekly**: Database compaction (configurable)

### 3. Configuration Changes

When configuration is refreshed (via API or restart):

1. New hosts and checks are added to the database
2. Modified hosts and checks are updated
3. Stale alerts for removed/renamed items are purged
4. Database is optimized

## What Gets Purged

### Stale Alerts
- Status entries for host:check combinations that no longer exist in config
- Historical data for removed hosts or checks
- Alerts for disabled hosts or checks

### Orphaned Database Entries
- Hosts in database but not in current YAML config
- Checks in database but not in current YAML config
- Status history older than retention period

### Example Scenarios

#### Scenario 1: Host Renamed
```yaml
# Old config
hosts:
  - id: "server-01"
    name: "Old Server"

# New config  
hosts:
  - id: "web-server-01"  # ID changed
    name: "Web Server"
```

**Result**: Alerts for `server-01` are purged, new alerts track `web-server-01`.

#### Scenario 2: Check Removed
```yaml
# Old config
checks:
  - id: "legacy-check"
    name: "Legacy Check"
    hosts: ["server-01"]

# New config (check removed)
```

**Result**: All alerts and history for `legacy-check` are purged.

#### Scenario 3: Host Disabled
```yaml
hosts:
  - id: "server-01"
    enabled: false  # Changed from true
```

**Result**: Active monitoring stops, but existing alerts may be preserved until next purge cycle.

## API Endpoints

### Manual Purge Operations

```bash
# Purge stale alerts only
curl -X DELETE http://localhost:8000/api/alerts/purge

# Purge orphaned hosts
curl -X DELETE http://localhost:8000/api/alerts/purge/hosts

# Purge orphaned checks  
curl -X DELETE http://localhost:8000/api/alerts/purge/checks

# Purge all stale data
curl -X DELETE http://localhost:8000/api/alerts/purge/all

# Get alert statistics
curl http://localhost:8000/api/alerts/stats
```

### Database Management

```bash
# Get database statistics
curl http://localhost:8000/api/database/stats

# Manual database compaction
curl -X POST http://localhost:8000/api/database/compact

# Full database maintenance
curl -X POST http://localhost:8000/api/database/maintenance

# Purge history older than 30 days
curl -X DELETE http://localhost:8000/api/database/history/30
```

### Configuration Management

```bash
# Refresh config with purge
curl -X POST http://localhost:8000/api/config/refresh

# Validate configuration
curl http://localhost:8000/api/config/validate
```

## Command Line Operations

### Maintenance Mode
```bash
# Run maintenance and exit
./raven -config config.yaml -maintenance

# Purge alerts and exit
./raven -config config.yaml -purge-alerts

# Validate configuration
./raven -config config.yaml -validate
```

## Monitoring and Logging

### Log Messages

```bash
# Startup purging
INFO[2025] Starting alert purge process
INFO[2025] Built valid host:check combinations map  valid_combinations=15
INFO[2025] Alert purge completed                    purged_count=3

# Orphaned data removal
INFO[2025] Purging orphaned host from database      host_id=old-server host_name=Old Server
INFO[2025] Orphaned host purge completed           purged_hosts=2

# Database maintenance  
INFO[2025] Starting database maintenance
INFO[2025] Database stats before maintenance        hosts=10 checks=5 db_size_mb=25
INFO[2025] Database maintenance completed successfully saved_mb=5 reduction_pct=20
```

### Metrics

The system exposes Prometheus metrics for monitoring:

```
# Database size and health
raven_database_size_bytes
raven_database_entries_total{type="hosts|checks|status|history"}

# Purge operations
raven_purge_operations_total{type="alerts|hosts|checks"}
raven_purged_entries_total{type="alerts|hosts|checks"}

# Maintenance operations
raven_maintenance_duration_seconds
raven_compaction_operations_total
```

## Best Practices

### 1. Configuration Management
- Use version control for your YAML configuration
- Test configuration changes in a development environment first
- Use the validation endpoint before deploying changes

### 2. Backup Strategy
- Regular database backups before major configuration changes
- Monitor database size and growth trends
- Set appropriate history retention periods

### 3. Monitoring
- Monitor purge operation logs for unexpected deletions
- Set up alerts for maintenance failures
- Track database size metrics

### 4. Maintenance Windows
- Schedule manual maintenance during low-traffic periods
- Allow sufficient time for database compaction operations
- Monitor system resources during maintenance

## Troubleshooting

### Common Issues

#### Too Much Data Purged
```bash
# Check what would be purged before running
curl http://localhost:8000/api/config/validate

# Review configuration for missing hosts/checks
grep -r "old-host-name" config/
```

#### Purge Not Working
```bash
# Check logs for errors
journalctl -u raven -f | grep -i purge

# Manually trigger purge
curl -X DELETE http://localhost:8000/api/alerts/purge/all

# Verify database stats
curl http://localhost:8000/api/database/stats
```

#### Database Growing Too Large
```bash
# Check current size
curl http://localhost:8000/api/database/stats

# Run compaction
curl -X POST http://localhost:8000/api/database/compact

# Reduce history retention
# Edit config.yaml: history_retention: "168h"  # 7 days
```

### Debug Mode

Enable debug logging to see detailed purge operations:

```yaml
logging:
  level: "debug"
  format: "json"
```

## Migration from v1.0

If migrating from Raven v1.0:

1. **Backup existing data**
2. **Run initial purge** to clean up legacy data
3. **Monitor logs** for unexpected purges
4. **Adjust retention settings** based on your needs

The system will automatically handle the transition and clean up any inconsistencies.
