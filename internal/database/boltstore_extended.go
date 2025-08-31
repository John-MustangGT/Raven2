// Enhanced BoltDB implementation with purging capabilities
// internal/database/boltstore_extended.go
package database

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "strings"
    "time"

    "go.etcd.io/bbolt"
    "github.com/sirupsen/logrus"
)

// ExtendedBoltStore implements ExtendedStore interface
type ExtendedBoltStore struct {
    *BoltStore
}

// NewExtendedBoltStore creates a new extended BoltDB store
func NewExtendedBoltStore(path string) (ExtendedStore, error) {
    baseStore, err := NewBoltStore(path)
    if err != nil {
        return nil, err
    }
    
    return &ExtendedBoltStore{
        BoltStore: baseStore.(*BoltStore),
    }, nil
}

// DeleteStatus removes a specific status entry for a host-check combination
func (s *ExtendedBoltStore) DeleteStatus(ctx context.Context, hostID, checkID string) error {
    return s.db.Update(func(tx *bbolt.Tx) error {
        // Delete from current status bucket
        statusBucket := tx.Bucket(StatusBucket)
        if statusBucket != nil {
            key := fmt.Sprintf("%s:%s", hostID, checkID)
            if err := statusBucket.Delete([]byte(key)); err != nil {
                return fmt.Errorf("failed to delete current status: %w", err)
            }
        }
        
        return nil
    })
}

// DeleteStatusByHostCheck removes all status entries for a host-check combination
func (s *ExtendedBoltStore) DeleteStatusByHostCheck(ctx context.Context, hostID, checkID string) error {
    return s.db.Update(func(tx *bbolt.Tx) error {
        // Delete from current status
        statusBucket := tx.Bucket(StatusBucket)
        if statusBucket != nil {
            key := fmt.Sprintf("%s:%s", hostID, checkID)
            statusBucket.Delete([]byte(key))
        }
        
        // Delete from history
        historyBucket := tx.Bucket(StatusHistBucket)
        if historyBucket != nil {
            prefix := fmt.Sprintf("%s:%s:", hostID, checkID)
            
            // Collect keys to delete
            var keysToDelete [][]byte
            cursor := historyBucket.Cursor()
            
            for k, _ := cursor.Seek([]byte(prefix)); k != nil && strings.HasPrefix(string(k), prefix); k, _ = cursor.Next() {
                keysToDelete = append(keysToDelete, copyBytes(k))
            }
            
            // Delete collected keys
            for _, key := range keysToDelete {
                historyBucket.Delete(key)
            }
            
            logrus.WithFields(logrus.Fields{
                "host_id":     hostID,
                "check_id":    checkID,
                "history_deleted": len(keysToDelete),
            }).Debug("Deleted status history entries")
        }
        
        return nil
    })
}

// DeleteStatusHistoryBefore removes historical status entries older than cutoffTime
func (s *ExtendedBoltStore) DeleteStatusHistoryBefore(ctx context.Context, cutoffTime time.Time) (int, error) {
    deletedCount := 0
    
    err := s.db.Update(func(tx *bbolt.Tx) error {
        historyBucket := tx.Bucket(StatusHistBucket)
        if historyBucket == nil {
            return nil
        }
        
        cursor := historyBucket.Cursor()
        var keysToDelete [][]byte
        
        for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
            // Parse the key to extract timestamp
            keyStr := string(k)
            parts := strings.Split(keyStr, ":")
            if len(parts) < 3 {
                continue
            }
            
            // Get timestamp from status data
            var status Status
            if err := json.Unmarshal(v, &status); err != nil {
                continue
            }
            
            if status.Timestamp.Before(cutoffTime) {
                keysToDelete = append(keysToDelete, copyBytes(k))
            }
        }
        
        // Delete old entries
        for _, key := range keysToDelete {
            if err := historyBucket.Delete(key); err != nil {
                logrus.WithError(err).Error("Failed to delete history entry")
                continue
            }
            deletedCount++
        }
        
        return nil
    })
    
    if err != nil {
        return 0, fmt.Errorf("failed to delete old history: %w", err)
    }
    
    logrus.WithFields(logrus.Fields{
        "deleted_count": deletedCount,
        "cutoff_time":   cutoffTime,
    }).Info("Deleted old status history entries")
    
    return deletedCount, nil
}

// BulkDeleteStatuses efficiently deletes multiple host-check status combinations
func (s *ExtendedBoltStore) BulkDeleteStatuses(ctx context.Context, hostCheckPairs []HostCheckPair) (int, error) {
    deletedCount := 0
    
    err := s.db.Update(func(tx *bbolt.Tx) error {
        statusBucket := tx.Bucket(StatusBucket)
        historyBucket := tx.Bucket(StatusHistBucket)
        
        for _, pair := range hostCheckPairs {
            // Delete from current status
            if statusBucket != nil {
                key := fmt.Sprintf("%s:%s", pair.HostID, pair.CheckID)
                if statusBucket.Get([]byte(key)) != nil {
                    statusBucket.Delete([]byte(key))
                    deletedCount++
                }
            }
            
            // Delete from history
            if historyBucket != nil {
                prefix := fmt.Sprintf("%s:%s:", pair.HostID, pair.CheckID)
                cursor := historyBucket.Cursor()
                
                var historyKeysToDelete [][]byte
                for k, _ := cursor.Seek([]byte(prefix)); k != nil && strings.HasPrefix(string(k), prefix); k, _ = cursor.Next() {
                    historyKeysToDelete = append(historyKeysToDelete, copyBytes(k))
                }
                
                for _, key := range historyKeysToDelete {
                    historyBucket.Delete(key)
                }
            }
        }
        
        return nil
    })
    
    if err != nil {
        return 0, fmt.Errorf("bulk delete failed: %w", err)
    }
    
    logrus.WithFields(logrus.Fields{
        "deleted_count": deletedCount,
        "pairs_count":   len(hostCheckPairs),
    }).Info("Bulk deleted status entries")
    
    return deletedCount, nil
}

// GetDatabaseStats returns information about database size and health
func (s *ExtendedBoltStore) GetDatabaseStats(ctx context.Context) (*DatabaseStats, error) {
    stats := &DatabaseStats{}
    
    err := s.db.View(func(tx *bbolt.Tx) error {
        // Count hosts
        if hostsBucket := tx.Bucket(HostsBucket); hostsBucket != nil {
            stats.TotalHosts = hostsBucket.Stats().KeyN
        }
        
        // Count checks
        if checksBucket := tx.Bucket(ChecksBucket); checksBucket != nil {
            stats.TotalChecks = checksBucket.Stats().KeyN
        }
        
        // Count status entries and find date range
        if statusBucket := tx.Bucket(StatusBucket); statusBucket != nil {
            stats.TotalStatusEntries = statusBucket.Stats().KeyN
        }
        
        // Count history entries and analyze dates
        if historyBucket := tx.Bucket(StatusHistBucket); historyBucket != nil {
            stats.TotalHistorySize = historyBucket.Stats().KeyN
            
            // Find oldest and newest entries
            cursor := historyBucket.Cursor()
            
            // Get oldest (first entry)
            if k, v := cursor.First(); k != nil && v != nil {
                var status Status
                if err := json.Unmarshal(v, &status); err == nil {
                    stats.OldestEntry = status.Timestamp
                }
            }
            
            // Get newest (last entry)
            if k, v := cursor.Last(); k != nil && v != nil {
                var status Status
                if err := json.Unmarshal(v, &status); err == nil {
                    stats.NewestEntry = status.Timestamp
                }
            }
        }
        
        return nil
    })
    
    if err != nil {
        return nil, fmt.Errorf("failed to get database stats: %w", err)
    }
    
    // Get file size
    if fileInfo, err := os.Stat(s.path); err == nil {
        stats.DatabaseSize = fileInfo.Size()
    }
    
    return stats, nil
}

// CompactDatabase performs database maintenance and compaction
func (s *ExtendedBoltStore) CompactDatabase(ctx context.Context) error {
    logrus.Info("Starting database compaction")
    
    // BoltDB doesn't have built-in compaction, but we can:
    // 1. Create a new database
    // 2. Copy all data to it
    // 3. Replace the old database
    
    backupPath := s.path + ".compact.tmp"
    
    // Create new database
    newDB, err := bbolt.Open(backupPath, 0600, &bbolt.Options{
        Timeout: 1 * time.Second,
    })
    if err != nil {
        return fmt.Errorf("failed to create compact database: %w", err)
    }
    
    defer func() {
        newDB.Close()
        os.Remove(backupPath) // Clean up on error
    }()
    
    // Initialize buckets in new database
    err = newDB.Update(func(tx *bbolt.Tx) error {
        buckets := [][]byte{HostsBucket, ChecksBucket, StatusBucket, StatusHistBucket, MetaBucket}
        for _, bucket := range buckets {
            if _, err := tx.CreateBucket(bucket); err != nil {
                return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
            }
        }
        return nil
    })
    if err != nil {
        return fmt.Errorf("failed to initialize compact database: %w", err)
    }
    
    // Copy data from old to new database
    err = s.db.View(func(oldTx *bbolt.Tx) error {
        return newDB.Update(func(newTx *bbolt.Tx) error {
            buckets := [][]byte{HostsBucket, ChecksBucket, StatusBucket, StatusHistBucket, MetaBucket}
            
            for _, bucketName := range buckets {
                oldBucket := oldTx.Bucket(bucketName)
                newBucket := newTx.Bucket(bucketName)
                
                if oldBucket == nil {
                    continue
                }
                
                cursor := oldBucket.Cursor()
                for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
                    if err := newBucket.Put(copyBytes(k), copyBytes(v)); err != nil {
                        return fmt.Errorf("failed to copy data: %w", err)
                    }
                }
            }
            
            return nil
        })
    })
    
    if err != nil {
        return fmt.Errorf("failed to copy data to compact database: %w", err)
    }
    
    // Close databases
    oldPath := s.path
    newDB.Close()
    s.db.Close()
    
    // Replace old database with compacted version
    if err := os.Rename(backupPath, oldPath); err != nil {
        return fmt.Errorf("failed to replace database: %w", err)
    }
    
    // Reopen the compacted database
    s.db, err = bbolt.Open(oldPath, 0600, &bbolt.Options{
        Timeout: 1 * time.Second,
    })
    if err != nil {
        return fmt.Errorf("failed to reopen compacted database: %w", err)
    }
    
    logrus.Info("Database compaction completed successfully")
    return nil
}

// copyBytes creates a copy of a byte slice
func copyBytes(b []byte) []byte {
    if b == nil {
        return nil
    }
    copied := make([]byte, len(b))
    copy(copied, b)
    return copied
}
