package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
)

// Vector reads preserve absent and non-finite samples instead of manufacturing
// zeros. Each read owns its failure context; sibling observations stay usable.
type metricVector struct {
	values        map[string]*float64
	state, reason string
}

type modelRequestVector struct {
	values        map[string]*RequestCounts
	state, reason string
}

func (a *Reader) instantVector(ctx context.Context, query string, p windowPlan) (prometheusQueryResponse, string) {
	c, _ := withPlan(ctx, p)
	api := config.C.Util.PrometheusServer.Address + "/api/v1/query?query=" + url.QueryEscape(query) + "&time=" + strconv.FormatInt(p.End.Unix(), 10)
	var response prometheusQueryResponse
	body, err := a.queryPrometheus(c, api, 3*time.Second)
	if err != nil {
		return response, "source_unavailable"
	}
	if err = json.Unmarshal(body, &response); err != nil || response.Status != "success" || response.Data.ResultType != "vector" {
		return response, "invalid_source_response"
	}
	return response, ""
}

func finiteSample(values []interface{}) *float64 {
	if len(values) < 2 {
		return nil
	}
	value, err := strconv.ParseFloat(fmt.Sprint(values[1]), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return nil
	}
	return &value
}

func (a *Reader) prometheusVector(ctx context.Context, query, label string, p windowPlan) metricVector {
	v := metricVector{values: map[string]*float64{}, state: "ok"}
	response, reason := a.instantVector(ctx, query, p)
	if reason != "" {
		v.state, v.reason = "error", reason
		return v
	}
	missingDimension := false
	for _, row := range response.Data.Result {
		key := row.Metric[label]
		if key == "" {
			missingDimension = true
			continue
		}
		v.values[key] = finiteSample(row.Value)
	}
	if missingDimension && len(v.values) == 0 {
		v.state, v.reason = "unsupported", label+"_dimension_unavailable"
	}
	return v
}

func (a *Reader) modelRequestVector(ctx context.Context, query string, p windowPlan) modelRequestVector {
	v := modelRequestVector{values: map[string]*RequestCounts{}, state: "ok"}
	response, reason := a.instantVector(ctx, query, p)
	if reason != "" {
		v.state, v.reason = "error", reason
		return v
	}
	invalid := map[string]bool{}
	counts := map[string]*RequestCounts{}
	missingModel, hasStatus := false, false
	for _, row := range response.Data.Result {
		model := row.Metric["model"]
		if model == "" {
			missingModel = true
			continue
		}
		if counts[model] == nil {
			counts[model] = &RequestCounts{}
		}
		status := row.Metric["status"]
		if status != "success" && status != "error" {
			invalid[model] = true
			continue
		}
		hasStatus = true
		number := finiteSample(row.Value)
		if number == nil || *number >= float64(math.MaxInt64) {
			invalid[model] = true
			continue
		}
		if status == "success" {
			counts[model].Success = int64(*number)
		} else {
			counts[model].Failure = int64(*number)
		}
	}
	for model, count := range counts {
		if invalid[model] || count.Success > math.MaxInt64-count.Failure {
			v.values[model] = nil
			continue
		}
		count.Total = count.Success + count.Failure
		v.values[model] = count
	}
	if missingModel && len(counts) == 0 {
		v.state, v.reason = "unsupported", "model_dimension_unavailable"
	} else if len(counts) > 0 && !hasStatus {
		v.state, v.reason = "unsupported", "request_status_dimension_unavailable"
	}
	return v
}
