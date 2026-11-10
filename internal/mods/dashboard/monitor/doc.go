// Package monitor reads legacy gateway monitoring snapshots. Reader.Read is the
// external query seam: it owns evaluation time, window geometry, source coverage
// and local reading failures. HTTP and WebSocket adapters serialize the same
// typed snapshot. ClickHouse request-log trends are a separate domain query.
package monitor
