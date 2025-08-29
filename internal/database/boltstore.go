// internal/database/boltstore.go - Complete BoltDB implementation
package database

import (
    "context"
    "encoding/json"
    "path/filepath"
    "fmt"
    "os"
    "strings"
    "time"

    "github.com/google/uuid"
    "go.etcd.io/bbolt"
)

var (
    HostsBucket      = []byte("hosts")
    ChecksBucket     = []byte("checks")
    StatusBucket     = []byte("status")
    StatusHistBucket = []byte("status_history")
    MetaBucket       = []byte("meta")
)

type BoltStore struct {
    db   *bbolt.DB
    path string
}

func NewBoltStore(path string) (Store, error) {
    // Create directory if it doesn't exist
    if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
        return nil, fmt.Errorf("failed to create data directory: %w", err)
    }

    db, err := bbolt.Open(path, 0600, &bbolt.Options{
        Timeout: 1 * time.Second,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to open BoltDB: %w", err)
    }

    store := &BoltStore{db: db, path: path}

    if err := store.initBuckets(); err != nil {
        db.Close()
        return nil, fmt.Errorf("failed to initialize buckets: %w", err)
    }

    return store, nil
}

func (s *BoltStore) initBuckets() error {
    return s.db.Update(func(tx *bbolt.Tx) error {
        buckets := [][]byte{HostsBucket, ChecksBucket, StatusBucket, StatusHistBucket, MetaBucket}
        for _, bucket := range buckets {
            if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
                return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
            }
        }
        return nil
    })
}

func (s *BoltStore) GetHosts(ctx context.Context, filters HostFilters) ([]Host, error) {
    var hosts []Host

    err := s.db.View(func(tx *bbolt.Tx) error {
        b := tx.Bucket(HostsBucket)
        return b.ForEach(func(k, v []byte) error {
            var host Host
            if err := json.Unmarshal(v, &host); err != nil {
                return fmt.Errorf("failed to unmarshal host %s: %w", k, err)
            }

            // Apply filters
            if filters.Group != "" && host.Group != filters.Group {
                return nil
            }
            if filters.Enabled != nil && host.Enabled != *filters.Enabled {
                return nil
            }

            hosts = append(hosts, host)
            return nil
        })
    })

    return hosts, err
}

func (s *BoltStore) GetHost(ctx context.Context, id string) (*Host, error) {
    var host Host

    err := s.db.View(func(tx *bbolt.Tx) error {
        b := tx.Bucket(HostsBucket)
        v := b.Get([]byte(id))
        if v == nil {
            return fmt.Errorf("host not found")
        }
        return json.Unmarshal(v, &host)
    })

    if err != nil {
        return nil, err
    }
    return &host, nil
}

func (s *BoltStore) CreateHost(ctx context.Context, host *Host) error {
    if host.ID == "" {
        host.ID = uuid.New().String()
    }
    host.CreatedAt = time.Now()
    host.UpdatedAt = time.Now()

    return s.db.Update(func(tx *bbolt.Tx) error {
        b := tx.Bucket(HostsBucket)
        
        data, err := json.Marshal(host)
        if err != nil {
            return fmt.Errorf("failed to marshal host: %w", err)
        }

        return b.Put([]byte(host.ID), data)
    })
}

func (s *BoltStore) UpdateHost(ctx context.Context, host *Host) error {
    host.UpdatedAt = time.Now()

    return s.db.Update(func(tx *bbolt.Tx) error {
        b := tx.Bucket(HostsBucket)
        
        data, err := json.Marshal(host)
        if err != nil {
            return fmt.Errorf("failed to marshal host: %w", err)
        }

        return b.Put([]byte(host.ID), data)
    })
}

func (s *BoltStore) DeleteHost(ctx context.Context, id string) error {
    return s.db.Update(func(tx *bbolt.Tx) error {
        b := tx.Bucket(HostsBucket)
        return b.Delete([]byte(id))
    })
}

func (s *BoltStore) GetChecks(ctx context.Context) ([]Check, error) {
    var checks []Check

    err := s.db.View(func(tx *bbolt.Tx) error {
        b := tx.Bucket(ChecksBucket)
        return b.ForEach(func(k, v []byte) error {
            var check Check
            if err := json.Unmarshal(v, &check); err != nil {
                return fmt.Errorf("failed to unmarshal check %s: %w", k, err)
            }
            checks = append(checks, check)
            return nil
        })
    })

    return checks, err
}

func (s *BoltStore) GetCheck(ctx context.Context, id string) (*Check, error) {
    var check Check

    err := s.db.View(func(tx *bbolt.Tx) error {
        b := tx.Bucket(ChecksBucket)
        v := b.Get([]byte(id))
        if v == nil {
            return fmt.Errorf("check not found")
        }
        return json.Unmarshal(v, &check)
    })

    if err != nil {
        return nil, err
    }
    return &check, nil
}

func (s *BoltStore) CreateCheck(ctx context.Context, check *Check) error {
    if check.ID == "" {
        check.ID = uuid.New().String()
    }
    check.CreatedAt = time.Now()
    check.UpdatedAt = time.Now()

    return s.db.Update(func(tx *bbolt.Tx) error {
        b := tx.Bucket(ChecksBucket)
        
        data, err := json.Marshal(check)
        if err != nil {
            return fmt.Errorf("failed to marshal check: %w", err)
        }

        return b.Put([]byte(check.ID), data)
    })
}

func (s *BoltStore) GetStatus(ctx context.Context, filters StatusFilters) ([]Status, error) {
    var statuses []Status

    err := s.db.View(func(tx *bbolt.Tx) error {
        b := tx.Bucket(StatusBucket)
        return b.ForEach(func(k, v []byte) error {
            var status Status
            if err := json.Unmarshal(v, &status); err != nil {
                return nil // Skip malformed entries
            }

            // Apply filters
            if filters.HostID != "" && status.HostID != filters.HostID {
                return nil
            }
            if filters.CheckID != "" && status.CheckID != filters.CheckID {
                return nil
            }
            if filters.ExitCode != nil && status.ExitCode != *filters.ExitCode {
                return nil
            }

            statuses = append(statuses, status)
            
            if filters.Limit > 0 && len(statuses) >= filters.Limit {
                return fmt.Errorf("limit_reached")
            }

            return nil
        })
    })

    if err != nil && err.Error() == "limit_reached" {
        err = nil
    }

    return statuses, err
}

func (s *BoltStore) UpdateStatus(ctx context.Context, status *Status) error {
    if status.ID == "" {
        status.ID = uuid.New().String()
    }

    return s.db.Update(func(tx *bbolt.Tx) error {
        b := tx.Bucket(StatusBucket)
        
        // Store current status
        key := fmt.Sprintf("%s:%s", status.HostID, status.CheckID)
        data, err := json.Marshal(status)
        if err != nil {
            return fmt.Errorf("failed to marshal status: %w", err)
        }

        if err := b.Put([]byte(key), data); err != nil {
            return err
        }

        // Also store in history
        hb := tx.Bucket(StatusHistBucket)
        histKey := fmt.Sprintf("%s:%s:%d", status.HostID, status.CheckID, status.Timestamp.Unix())
        return hb.Put([]byte(histKey), data)
    })
}

func (s *BoltStore) GetStatusHistory(ctx context.Context, hostID, checkID string, since time.Time) ([]Status, error) {
    var statuses []Status

    err := s.db.View(func(tx *bbolt.Tx) error {
        b := tx.Bucket(StatusHistBucket)
        c := b.Cursor()

        prefix := fmt.Sprintf("%s:%s:", hostID, checkID)
        
        for k, v := c.Seek([]byte(prefix)); k != nil && strings.HasPrefix(string(k), prefix); k, v = c.Next() {
            var status Status
            if err := json.Unmarshal(v, &status); err != nil {
                continue
            }

            if status.Timestamp.After(since) {
                statuses = append(statuses, status)
            }
        }

        return nil
    })

    return statuses, err
}

func (s *BoltStore) UpdateCheck(ctx context.Context, check *Check) error {
    check.UpdatedAt = time.Now()

    return s.db.Update(func(tx *bbolt.Tx) error {
        b := tx.Bucket(ChecksBucket)
        
        data, err := json.Marshal(check)
        if err != nil {
            return fmt.Errorf("failed to marshal check: %w", err)
        }

        return b.Put([]byte(check.ID), data)
    })
}

func (s *BoltStore) DeleteCheck(ctx context.Context, id string) error {
    return s.db.Update(func(tx *bbolt.Tx) error {
        b := tx.Bucket(ChecksBucket)
        return b.Delete([]byte(id))
    })
}

func (s *BoltStore) Close() error {
    return s.db.Close()
}

