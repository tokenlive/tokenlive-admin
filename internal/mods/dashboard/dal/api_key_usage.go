package dal

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/shopspring/decimal"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/schema"
)

const candidateSQL = `
SELECT
    api_key_hash, api_key_id, workspace_id, user_id, tenant_id,
    argMax(api_key, time),
    count(),
    countIf(status_code >= 200 AND status_code < 400),
    sum(input_tokens), sum(output_tokens),
    sum(cached_tokens), sum(cache_creation_tokens),
    toString(sum(cost))
FROM access_logs FINAL
WHERE time >= ? AND time < ?
GROUP BY api_key_hash, api_key_id, workspace_id, user_id, tenant_id`

type usageRows interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close() error
}

type ClickHouseReader struct {
	enabled      bool
	initErr      error
	queryTimeout time.Duration
	query        func(context.Context, string, ...any) (usageRows, error)
}

// NewClickHouseReader never contacts the server at construction time. An optional
// unavailable usage source must not prevent the management console from starting.
func NewClickHouseReader(cfg config.ClickHouseConfig) (*ClickHouseReader, func()) {
	r := &ClickHouseReader{enabled: cfg.Enabled}
	if !cfg.Enabled {
		return r, func() {}
	}
	if len(cfg.Addr) == 0 || cfg.Database == "" || cfg.DialTimeoutSeconds < 0 || cfg.QueryTimeoutSeconds < 0 {
		r.initErr = errors.New("invalid ClickHouse usage configuration")
		return r, func() {}
	}
	if cfg.DialTimeoutSeconds == 0 {
		cfg.DialTimeoutSeconds = 3
	}
	if cfg.QueryTimeoutSeconds == 0 {
		cfg.QueryTimeoutSeconds = 5
	}
	r.queryTimeout = time.Duration(cfg.QueryTimeoutSeconds) * time.Second
	opts := &clickhouse.Options{
		Addr: cfg.Addr, DialTimeout: time.Duration(cfg.DialTimeoutSeconds) * time.Second,
		Auth: clickhouse.Auth{Database: cfg.Database, Username: cfg.Username, Password: cfg.Password},
	}
	if cfg.TLS {
		opts.TLS = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	conn, err := clickhouse.Open(opts)
	if err != nil {
		r.initErr = errors.New("unable to initialize ClickHouse usage client")
		return r, func() {}
	}
	r.query = func(ctx context.Context, query string, args ...any) (usageRows, error) {
		return conn.Query(ctx, query, args...)
	}
	var once sync.Once
	return r, func() { once.Do(func() { _ = conn.Close() }) }
}

func (r *ClickHouseReader) Enabled() bool { return r != nil && r.enabled }

func (r *ClickHouseReader) Read(ctx context.Context, window schema.Window) ([]schema.Candidate, error) {
	if !r.Enabled() {
		return nil, nil
	}
	if r.initErr != nil {
		return nil, r.initErr
	}
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows, err := r.query(ctx, candidateSQL, window.Start, window.End)
	if err != nil {
		return nil, fmt.Errorf("read usage candidates: %w", err)
	}
	defer rows.Close()
	result := make([]schema.Candidate, 0)
	for rows.Next() {
		var row schema.Candidate
		var cost string
		if err := rows.Scan(
			&row.Ref.Hash, &row.Ref.KeyID, &row.Ref.WorkspaceID, &row.Ref.UserID, &row.Ref.TenantID,
			&row.Display, &row.Totals.Requests, &row.Totals.Success,
			&row.Totals.Input, &row.Totals.Output, &row.Totals.Cached, &row.Totals.CacheCreation, &cost,
		); err != nil {
			return nil, errors.New("invalid usage candidate row")
		}
		row.Totals.Cost, err = decimal.NewFromString(cost)
		if err != nil {
			return nil, errors.New("invalid usage cost")
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate usage candidates: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
