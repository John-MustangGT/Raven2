// internal/metrics/prometheus.go
package metrics

import (
    "context"
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/John-MustangGT/raven/internal/database"
)

// Prometheus metrics
var (
    CheckDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "raven_check_duration_seconds",
            Help:    "Time spent executing checks",
            Buckets: prometheus.DefBuckets,
        },
        []string{"host", "check_type", "status"},
    )

    CheckTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "raven_checks_total",
            Help: "Total number of checks executed",
        },
        []string{"host", "check_type", "status"},
    )

    HostStatus = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "raven_host_status",
            Help: "Current status of hosts (0=OK, 1=Warning, 2=Critical, 3=Unknown)",
        },
        []string{"host", "group", "check_type"},
    )

    ActiveHosts = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "raven_active_hosts_total",
            Help: "Number of active hosts being monitored",
        },
    )

    ActiveChecks = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "raven_active_checks_total",
            Help: "Number of active checks configured",
        },
    )

    DatabaseOperations = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "raven_database_operations_total",
            Help: "Total database operations performed",
        },
        []string{"operation", "status"},
    )

    WebSocketConnections = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "raven_websocket_connections_active",
            Help: "Number of active WebSocket connections",
        },
    )
)

type Collector struct {
    store database.Store
}

func NewCollector(store database.Store) *Collector {
    return &Collector{store: store}
}

func (c *Collector) RecordCheckResult(host, checkType string, exitCode int, duration time.Duration) {
    status := getStatusLabel(exitCode)
    CheckDuration.WithLabelValues(host, checkType, status).Observe(duration.Seconds())
    CheckTotal.WithLabelValues(host, checkType, status).Inc()
}

func (c *Collector) UpdateHostStatus(host, group, checkType string, exitCode int) {
    HostStatus.WithLabelValues(host, group, checkType).Set(float64(exitCode))
}

func (c *Collector) UpdateSystemMetrics(ctx context.Context) error {
    hosts, err := c.store.GetHosts(ctx, database.HostFilters{})
    if err != nil {
        DatabaseOperations.WithLabelValues("get_hosts", "error").Inc()
        return err
    }
    DatabaseOperations.WithLabelValues("get_hosts", "success").Inc()

    enabledHosts := 0
    for _, host := range hosts {
        if host.Enabled {
            enabledHosts++
        }
    }
    ActiveHosts.Set(float64(enabledHosts))

    checks, err := c.store.GetChecks(ctx)
    if err != nil {
        DatabaseOperations.WithLabelValues("get_checks", "error").Inc()
        return err
    }
    DatabaseOperations.WithLabelValues("get_checks", "success").Inc()

    enabledChecks := 0
    for _, check := range checks {
        if check.Enabled {
            enabledChecks++
        }
    }
    ActiveChecks.Set(float64(enabledChecks))

    return nil
}

func (c *Collector) RecordWebSocketConnection(delta int) {
    WebSocketConnections.Add(float64(delta))
}

func getStatusLabel(exitCode int) string {
    switch exitCode {
    case 0:
        return "ok"
    case 1:
        return "warning"
    case 2:
        return "critical"
    default:
        return "unknown"
    }
}
