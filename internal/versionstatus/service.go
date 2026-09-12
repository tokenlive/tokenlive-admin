// Package versionstatus combines local identity, active Gateway reports and
// shared update-check state without caching comparisons against Gateway nodes.
package versionstatus

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/updatecheck"
	"github.com/tokenlive/tokenlive-admin/pkg/logging"
	"github.com/tokenlive/tokenlive-admin/pkg/productversion"
	"github.com/tokenlive/tokenlive-admin/pkg/versionregistry"
	"go.uber.org/zap"
)

type Service struct {
	identity productversion.Identity
	store    versionregistry.Store
	checker  *updatecheck.Checker
	scope    string
}

// New borrows store and owns only the supplied checker's lifecycle.
// Construction never starts background work or contacts a source.
func New(identity productversion.Identity, store versionregistry.Store, checker *updatecheck.Checker, scope string) *Service {
	return &Service{identity: identity, store: store, checker: checker, scope: scope}
}

func (s *Service) Summary(ctx context.Context, canManage bool) (Summary, error) {
	gateway, err := s.gateway(ctx)
	return Summary{Identity: s.identity, Gateway: gateway, CanManageUpdates: canManage}, err
}

// Updates is read-only with respect to external sources. A fresh registry read
// ensures expired or newly reported Gateway builds immediately affect notices.
func (s *Service) Updates(ctx context.Context) (Updates, error) {
	return s.updates(ctx, s.checker.Snapshot())
}

// Check uses the checker's instance-wide manual cooldown and in-flight request.
// Per-source failures remain in the returned component states.
func (s *Service) Check(ctx context.Context) (Updates, error) {
	state, checkErr := s.checker.Check(ctx, true)
	result, err := s.updates(ctx, state)
	if checkErr != nil {
		return result, checkErr
	}
	return result, err
}

func (s *Service) Report(ctx context.Context, node versionregistry.Node) error {
	return s.store.Upsert(ctx, node)
}

func (s *Service) Start() { s.checker.Start() }

// Close is idempotent and never closes the store's borrowed Redis client.
func (s *Service) Close() { s.checker.Close() }

func (s *Service) updates(ctx context.Context, state updatecheck.CheckResult) (Updates, error) {
	gateway, err := s.gateway(ctx)
	result := Compose(s.identity, gateway.Groups, state)
	if gateway.Status == "unavailable" && result.Enabled {
		for i := range result.Components {
			if result.Components[i].Component == "gateway" {
				result.Components[i].State = "unavailable"
			}
		}
	}
	return result, err
}

func (s *Service) gateway(ctx context.Context) (GatewayView, error) {
	view := GatewayView{Status: "unknown", Scope: s.scope, Groups: []versionregistry.Group{}}
	if err := ctx.Err(); err != nil {
		view.Status = "unavailable"
		return view, err
	}
	if s.identity.Edition == "standalone" {
		view.Status = "not_applicable"
		return view, nil
	}
	nodes, err := s.store.List(ctx)
	if err != nil {
		view.Status = "unavailable"
		if contextErr := ctx.Err(); contextErr != nil {
			return view, contextErr
		}
		// Diagnostics can contain private Redis keys, instance IDs or addresses.
		// Keep them in internal logs, never in public DTOs or returned errors.
		logging.Context(ctx).Warn("failed to read Gateway version registry", zap.Error(err))
		return view, nil
	}
	view.Groups = versionregistry.GroupNodes(nodes)
	if len(view.Groups) > 0 {
		view.Status = "observed"
	}
	return view, nil
}
