package metrics

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

// ContainerMetric represents a point-in-time usage record for a container.
type ContainerMetric struct {
	ID            int64
	WorkerID      string
	ContainerID   string
	ClientKey     string
	CPUUsage      float64
	MemUsageMb    int64
	NetTxBytes    int64
	NetRxBytes    int64
	Timestamp     int64 // Nanoseconds
}

// MetricRepository handles SQLite persistence for container metrics.
type MetricRepository struct {
	db *sql.DB
}

// NewMetricRepository initializes a new SQLite repository and creates tables if missing.
func NewMetricRepository(dbPath string) (*MetricRepository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	// Create table for metrics
	query := `
	CREATE TABLE IF NOT EXISTS container_metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		worker_id TEXT,
		container_id TEXT,
		client_key TEXT,
		cpu_usage REAL,
		mem_usage_mb INTEGER,
		net_tx_bytes INTEGER,
		net_rx_bytes INTEGER,
		timestamp INTEGER,
		sent INTEGER DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_sent ON container_metrics(sent);
	`
	if _, err := db.Exec(query); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return &MetricRepository{db: db}, nil
}

// Save records a new metric entry.
func (r *MetricRepository) Save(ctx context.Context, m ContainerMetric) error {
	const query = `
	INSERT INTO container_metrics (worker_id, container_id, client_key, cpu_usage, mem_usage_mb, net_tx_bytes, net_rx_bytes, timestamp)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query, m.WorkerID, m.ContainerID, m.ClientKey, m.CPUUsage, m.MemUsageMb, m.NetTxBytes, m.NetRxBytes, m.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to save metric: %w", err)
	}
	return nil
}

// GetPending retrieves unsent metrics for shipping.
func (r *MetricRepository) GetPending(ctx context.Context, limit int) ([]ContainerMetric, error) {
	const query = `
	SELECT id, worker_id, container_id, client_key, cpu_usage, mem_usage_mb, net_tx_bytes, net_rx_bytes, timestamp 
	FROM container_metrics 
	WHERE sent = 0 
	ORDER BY timestamp ASC 
	LIMIT ?
	`
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending metrics: %w", err)
	}
	defer rows.Close()

	var results []ContainerMetric
	for rows.Next() {
		var m ContainerMetric
		if err := rows.Scan(&m.ID, &m.WorkerID, &m.ContainerID, &m.ClientKey, &m.CPUUsage, &m.MemUsageMb, &m.NetTxBytes, &m.NetRxBytes, &m.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan metric row: %w", err)
		}
		results = append(results, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return results, nil
}

// MarkSent removes specified metric IDs from the database to save local space.
func (r *MetricRepository) MarkSent(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	// Use a single query with IN clause for performance
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf("DELETE FROM container_metrics WHERE id IN (%s)", strings.Join(placeholders, ","))
	
	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete sent metrics: %w", err)
	}

	return nil
}

// Close closes the database connection.
func (r *MetricRepository) Close() error {
	return r.db.Close()
}
