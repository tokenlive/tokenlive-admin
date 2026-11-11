package biz

import (
	"context"
	"strings"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/compositerouting"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"gorm.io/gorm"
)

type CompositePreviewForm struct {
	Routing *schema.SmartRouting `json:"routing"`
	Tags    map[string]string    `json:"tags"`
	Headers map[string]string    `json:"headers"`
}

type CompositePreviewResult struct {
	compositerouting.Plan
	Version          int64    `json:"version"`
	Source           string   `json:"source"`
	TargetStatus     string   `json:"target_status"`
	RuntimeAvailable bool     `json:"runtime_available"`
	Limitations      []string `json:"limitations"`
}

// PreviewCompositeRouting selects against a draft or saved configuration, with
// the administrator's existing model permissions. It never impersonates an API
// caller, probes an upstream, mutates quotas or writes model configuration.
func (m *Model) PreviewCompositeRouting(ctx context.Context, id string, form *CompositePreviewForm) (*CompositePreviewResult, error) {
	model, err := m.ModelDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if model == nil {
		return nil, errors.NotFound("", "Model not found")
	}
	if err := m.requireModelMutationPermission(ctx, model, 1); err != nil {
		return nil, err
	}
	if model.ModelType != schema.ModelTypeSmart {
		return nil, errors.BadRequest("", "仅复合模型支持选模预演")
	}
	routing, source := model.SmartRouting, "saved"
	if form.Routing != nil {
		routing, source = form.Routing, "draft"
	}
	routing = routing.Clone()
	if err := routing.Validate(); err != nil {
		return nil, err
	}
	if routing.RoutingMode() == compositerouting.Difficulty {
		return nil, errors.BadRequest("", "难度裁决需要实际模型推理，不支持无调用预演")
	}
	if len(form.Tags) > 50 || len(form.Headers) > 10 {
		return nil, errors.BadRequest("", "预演标签或 Header 数量过多")
	}
	for key, value := range form.Headers {
		if !compositerouting.AllowedHeader(key) || len(value) > 4096 {
			return nil, errors.BadRequest("", "预演只允许受支持的业务 Header")
		}
	}
	for key, value := range form.Tags {
		if strings.TrimSpace(key) == "" || len(key) > 128 || len(value) > 4096 {
			return nil, errors.BadRequest("", "预演标签名称或内容过长")
		}
	}
	plan, err := compositerouting.Select(routing.PlanConfig(), compositerouting.Input{Tags: form.Tags, Headers: form.Headers})
	if err != nil {
		return nil, errors.BadRequest("", "%s", err.Error())
	}
	result := &CompositePreviewResult{Plan: plan, Version: routing.Version, Source: source, TargetStatus: "configured", RuntimeAvailable: config.C.Gateway.CompositeRoutingV2,
		Limitations: []string{"仅预演首次选模，不调用模型、不校验真实调用身份或配额", "备用链以实际调用身份的调用策略为准，本次未解析", "端点健康、请求能力、延迟和费用须在实际执行时检查"}}
	target, err := m.ModelDAL.Get(ctx, plan.Target)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if target == nil {
		result.Target, result.TargetStatus = "", "unavailable_or_not_authorized"
		return result, nil
	}
	if err := m.requireModelMutationPermission(ctx, target, 1); err != nil {
		// Do not distinguish hidden targets from missing ones.
		result.Target, result.TargetStatus = "", "unavailable_or_not_authorized"
		return result, nil
	}
	if target.ID == model.ID || target.SpaceCode != model.SpaceCode || target.ModelType == schema.ModelTypeSmart || !supportsChat(target.RequestTypes) {
		result.TargetStatus = "incompatible"
	} else if target.Enabled != 1 {
		result.TargetStatus = "disabled"
	}
	return result, nil
}
