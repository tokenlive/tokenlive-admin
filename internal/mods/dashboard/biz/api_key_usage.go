package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/schema"
)

type Reader interface {
	Enabled() bool
	Read(context.Context, schema.Window) ([]schema.Candidate, error)
}

type Resolver interface {
	Resolve(context.Context, []schema.Candidate) schema.Resolution
	Describe(context.Context, []schema.Group) map[string]schema.Metadata
}

type Aggregation struct {
	Groups       []schema.Group
	All          schema.Totals
	Unattributed schema.Totals
	KeyCount     uint64
}

func refOrder(ref schema.KeyRef) string {
	value, _ := json.Marshal([]string{ref.WorkspaceID, ref.UserID, ref.TenantID, ref.KeyID, ref.Hash})
	return string(value)
}

func Aggregate(rows []schema.Candidate, resolution schema.Resolution, query schema.Query) Aggregation {
	result := Aggregation{Groups: []schema.Group{}}
	groups := map[string]*schema.Group{}
	for _, row := range rows {
		result.All.Add(row.Totals)
		key := resolution.Keys[row.Ref]
		if key == "" {
			result.Unattributed.Add(row.Totals)
			continue
		}
		group := groups[key]
		if group == nil {
			group = &schema.Group{Canonical: key, Ref: row.Ref, Display: row.Display}
			groups[key] = group
		} else {
			// Deterministic, prefer richer ID metadata to an otherwise equivalent
			// hash-only reference. Never change the credential grouping itself.
			if refOrder(row.Ref) > refOrder(group.Ref) {
				group.Ref = row.Ref
			}
			if group.Display == "" || (row.Display != "" && row.Display < group.Display) {
				group.Display = row.Display
			}
		}
		if strings.HasPrefix(key, "h:") {
			group.Ref.Hash = strings.TrimPrefix(key, "h:")
		}
		group.Totals.Add(row.Totals)
	}
	for _, group := range groups {
		result.Groups = append(result.Groups, *group)
	}
	result.KeyCount = uint64(len(result.Groups))
	sort.Slice(result.Groups, func(i, j int) bool {
		left, right := result.Groups[i], result.Groups[j]
		switch query.SortBy {
		case "cost":
			if cmp := left.Totals.Cost.Cmp(right.Totals.Cost); cmp != 0 {
				return cmp > 0
			}
		case "request_count":
			if left.Totals.Requests != right.Totals.Requests {
				return left.Totals.Requests > right.Totals.Requests
			}
		default:
			if left.Totals.Tokens() != right.Totals.Tokens() {
				return left.Totals.Tokens() > right.Totals.Tokens()
			}
		}
		return left.Canonical < right.Canonical
	})
	if query.Limit > 0 && len(result.Groups) > query.Limit {
		result.Groups = result.Groups[:query.Limit]
	}
	return result
}

func usageMetrics(totals schema.Totals, allTokens uint64) schema.Metrics {
	result := schema.Metrics{
		RequestCount: totals.Requests, SuccessCount: totals.Success,
		InputTokens: totals.Input, OutputTokens: totals.Output, CachedTokens: totals.Cached,
		CacheCreationTokens: totals.CacheCreation, TotalTokens: totals.Tokens(), TotalCost: totals.Cost.String(),
	}
	if totals.Requests > 0 {
		result.SuccessRate = float64(totals.Success) / float64(totals.Requests) * 100
	}
	if allTokens > 0 {
		share := float64(totals.Tokens()) / float64(allTokens) * 100
		result.TokenShare = &share
	}
	return result
}

func safeKeyDisplay(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "****" + value[len(value)-4:]
}

func (s *APIKeyUsageService) load(ctx context.Context, query schema.Query) (*schema.Response, error) {
	window := schema.ResolveWindow(query, s.now())
	rows, err := s.reader.Read(ctx, window)
	if err != nil {
		return nil, err
	}
	metaCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	resolution := s.resolver.Resolve(metaCtx, rows)
	aggregation := Aggregate(rows, resolution, query)
	metadata := s.resolver.Describe(metaCtx, aggregation.Groups)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	now := s.now()
	summary := schema.Summary{KeyCount: aggregation.KeyCount, Metrics: usageMetrics(aggregation.All, aggregation.All.Tokens())}
	unattributed := usageMetrics(aggregation.Unattributed, aggregation.All.Tokens())
	response := &schema.Response{
		State: "ready", DataSource: "clickhouse", Window: &window, GeneratedAt: &now,
		Summary: &summary, Items: make([]schema.Item, 0, len(aggregation.Groups)), Unattributed: &unattributed, Warnings: []string{},
	}
	warnings := map[string]bool{}
	for _, warning := range resolution.Warnings {
		warnings[warning] = true
	}
	if aggregation.Unattributed.Requests > 0 {
		warnings["unattributed_usage"] = true
	}
	for _, group := range aggregation.Groups {
		meta, ok := metadata[group.Canonical]
		if !ok {
			meta = unknownMetadata(group)
		}
		if meta.MetadataStatus != "ready" {
			warnings["metadata_unavailable"] = true
		}
		if meta.Display == "" {
			meta.Display = group.Display
		}
		digest := sha256.Sum256([]byte("api-key-usage-row-v1:" + group.Canonical))
		response.Items = append(response.Items, schema.Item{
			RowID: hex.EncodeToString(digest[:]), KeyName: meta.KeyName, KeyDisplay: safeKeyDisplay(meta.Display),
			Source: meta.Source, Owner: meta.Owner, KeyStatus: meta.KeyStatus, MetadataStatus: meta.MetadataStatus,
			Metrics: usageMetrics(group.Totals, aggregation.All.Tokens()),
		})
	}
	for warning := range warnings {
		response.Warnings = append(response.Warnings, warning)
	}
	sort.Strings(response.Warnings)
	return response, nil
}
