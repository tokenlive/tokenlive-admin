package monitor

import (
	"context"
	"time"

	rschema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
)

// Provider detail charts use the same upstream-attempt observations and identity
// selection as their health cards. Request counters grouped by final provider
// cannot reveal failed attempts that were recovered through another provider.
func (a *Reader) readProviderTraffic(ctx context.Context, filter string, p windowPlan) *Reading[Traffic] {
	r := &Reading[Traffic]{State: "ok", Window: p.Window, Buckets: append([]Bucket(nil), p.Buckets...)}
	if p.minutes() == 0 {
		r.State = "unavailable"
		r.Coverage.Reason = "no_completed_minutes"
		return r
	}
	if a.db == nil {
		r.State, r.Error = "error", "source_unavailable"
		return r
	}
	var provider rschema.Provider
	if err := a.db.WithContext(ctx).Where("id = ? OR code = ?", filter, filter).First(&provider).Error; err != nil {
		r.State, r.Error = "error", "source_unavailable"
		return r
	}
	items := []ProviderHealth{{ProviderID: provider.ID, ProviderCode: provider.Code, ProviderName: provider.Name}}
	observations := a.providerCallsSeries(ctx, items, [][]string{providerStoreKeys(&provider)}, p)
	observation := observations[0]
	m := observation.Metric
	r.State, r.Source, r.Coverage = m.State, m.Source, m.Coverage
	if m.Reason != "" {
		r.Coverage.Reason = m.Reason
	}
	if m.Value == nil {
		if m.State == "error" {
			r.Error = m.Reason
		} else {
			r.State = "unavailable"
			r.Coverage.Reason = m.Reason
		}
		return r
	}
	row := TrafficSeries{Label: provider.Code, Success: make([]*int64, len(p.Buckets)), Failure: make([]*int64, len(p.Buckets)), Total: make([]*int64, len(p.Buckets))}
	if row.Label == "" {
		row.Label = provider.Name
	}
	for i, bucket := range p.Buckets {
		if !bucketCovered(bucket, m.Coverage) {
			continue
		}
		value := ProviderCalls{}
		for minute := bucket.Start; minute.Before(bucket.End); minute = minute.Add(time.Minute) {
			counts := observation.Minutes[int(minute.Sub(p.Start)/time.Minute)]
			value.Success += counts.Success
			value.Failure += counts.Failure
			value.Total += counts.Total
		}
		row.Success[i], row.Failure[i], row.Total[i] = &value.Success, &value.Failure, &value.Total
	}
	r.Data = &Traffic{Series: []TrafficSeries{row}}
	return r
}
