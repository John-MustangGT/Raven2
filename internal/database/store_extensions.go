// internal/database/store_extensions.go - Extended store interface for alert purging
package database

import (
    "context"
    "time"
)

// ExtendedStore extends the basic Store interface with purging operations
type ExtendedStore interface {
    Store
    
    // Alert and status purging operations
    DeleteStatus(ctx context.Context, hostID, checkID string) error
    DeleteStatusHistoryBefore(ctx context.Context, cutoffTime time.Time) (int, error)
    DeleteStatusByHostCheck(ctx context.Context, hostID, checkID string) error
    
    // Bulk operations for efficiency
    BulkDeleteStatuses(ctx context.Context, hostCheckPairs []HostCheckPair) (int, error)
    
    // Data cleanup operations
    CompactDatabase(ctx context.Context) error
    GetDatabaseStats(ctx context.Context) (*DatabaseStats, error)
}

// HostCheckPair represents a host-check combination for bulk operations
type HostCheckPair struct {
    HostID  string
    CheckID string
}

// DatabaseStats provides information about database size and health
type DatabaseStats struct {
    TotalHosts         int           `json:"total_hosts"`
    TotalChecks        int           `json:"total_checks"`
    TotalStatusEntries int           `json:"total_status_entries"`
    TotalHistorySize   int           `json:"total_history_size"`
    DatabaseSize       int64         `json:"database_size_bytes"`
    OldestEntry        time.Time     `json:"oldest_entry"`
    NewestEntry        time.Time     `json:"newest_entry"`
}
