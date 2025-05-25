package repository

import (
	"context"
	"database/sql"
	"fmt"
	models "github.com/chestorix/monmetrics/internal/metrics"
	_ "github.com/jackc/pgx/v5/stdlib"
	"sync"
	"time"
)

type PostgresStorage struct {
	db    *sql.DB
	mu    sync.RWMutex
	dbDSN string
}

func NewPostgresStorage(dsn string) (*PostgresStorage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}
	return &PostgresStorage{
		db:    db,
		dbDSN: dsn,
	}, nil
}

func createTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS gauges (
			name TEXT PRIMARY KEY,
			value DOUBLE PRECISION NOT NULL
		);
		
		CREATE TABLE IF NOT EXISTS counters (
			name TEXT PRIMARY KEY,
			value BIGINT NOT NULL
		);
	`)
	return err
}

func (p *PostgresStorage) UpdateGauge(name string, value float64) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	_, err := p.db.Exec(`
		INSERT INTO gauges (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value
	`, name, value)
	return err
}

func (p *PostgresStorage) UpdateCounter(name string, value int64) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	_, err := p.db.Exec(`
		INSERT INTO counters (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET value = counters.value + EXCLUDED.value
	`, name, value)
	return err
}
func (p *PostgresStorage) UpdateMetricsBatch(metrics []models.Metrics) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	gaugeStmt, err := tx.Prepare(`
        INSERT INTO gauges (name, value)
        VALUES ($1, $2)
        ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value
    `)
	if err != nil {
		return fmt.Errorf("failed to prepare gauge statement: %w", err)
	}
	defer gaugeStmt.Close()

	counterStmt, err := tx.Prepare(`
        INSERT INTO counters (name, value)
        VALUES ($1, $2)
        ON CONFLICT (name) DO UPDATE SET value = counters.value + EXCLUDED.value
    `)
	if err != nil {
		return fmt.Errorf("failed to prepare counter statement: %w", err)
	}
	defer counterStmt.Close()

	for _, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			if metric.Value == nil {
				return fmt.Errorf("gauge value is nil for metric %s", metric.ID)
			}
			if _, err := gaugeStmt.Exec(metric.ID, *metric.Value); err != nil {
				return fmt.Errorf("failed to update gauge: %w", err)
			}

		case models.Counter:
			if metric.Delta == nil {
				return fmt.Errorf("counter delta is nil for metric %s", metric.ID)
			}
			if _, err := counterStmt.Exec(metric.ID, *metric.Delta); err != nil {
				return fmt.Errorf("failed to update counter: %w", err)
			}
		}
	}

	return tx.Commit()
}
func (p *PostgresStorage) GetGauge(name string) (float64, bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var value float64
	err := p.db.QueryRow("SELECT value FROM gauges WHERE name = $1", name).Scan(&value)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return value, true, nil
}

func (p *PostgresStorage) GetCounter(name string) (int64, bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var value int64
	err := p.db.QueryRow("SELECT value FROM counters WHERE name = $1", name).Scan(&value)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return value, true, nil
}

func (p *PostgresStorage) GetAll() ([]models.Metric, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var metrics []models.Metric

	rows, err := p.db.Query("SELECT name, value FROM gauges")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value float64
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		metrics = append(metrics, models.Metric{
			Name:  name,
			Type:  models.Gauge,
			Value: value,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows, err = p.db.Query("SELECT name, value FROM counters")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value int64
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		metrics = append(metrics, models.Metric{
			Name:  name,
			Type:  models.Counter,
			Value: value,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return metrics, nil
}

func (p *PostgresStorage) Save() error {
	return nil
}

func (p *PostgresStorage) Load() error {
	return nil
}

func (p *PostgresStorage) Close() error {
	return p.db.Close()
}
