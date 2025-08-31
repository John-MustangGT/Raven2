// internal/database/store.go
package database

import (
    "context"
    "time"
)

// Store defines the interface for database operations
type Store interface {
    // Host operations
    GetHosts(ctx context.Context, filters HostFilters) ([]Host, error)
    GetHost(ctx context.Context, id string) (*Host, error)
    CreateHost(ctx context.Context, host *Host) error
    UpdateHost(ctx context.Context, host *Host) error
    DeleteHost(ctx context.Context, id string) error

    // Check operations
    GetChecks(ctx context.Context) ([]Check, error)
    GetCheck(ctx context.Context, id string) (*Check, error)
    CreateCheck(ctx context.Context, check *Check) error
    UpdateCheck(ctx context.Context, check *Check) error
    DeleteCheck(ctx context.Context, id string) error

    // Status operations
    GetStatus(ctx context.Context, filters StatusFilters) ([]Status, error)
    UpdateStatus(ctx context.Context, status *Status) error
    GetStatusHistory(ctx context.Context, hostID, checkID string, since time.Time) ([]Status, error)
    DeleteStatus(ctx context.Context, hostID, checkID string) error


    // Close the database connection
    Close() error
}
