package monitor

import (
	"context"
	"fmt"
	"time"

	rschema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/gatewaycontract"
	"github.com/tokenlive/tokenlive-admin/pkg/metrics"
)

type trendSeries struct {
	Label   string  `json:"label"`
	Success []int64 `json:"success"`
	Failure []int64 `json:"failure"`
	Total   []int64 `json:"total"`
}

type trafficResult struct {
	Series []trendSeries `json:"series"`
}

func emptyTrendSeries(numPoints int, label string) trendSeries {
	return trendSeries{
		Label:   label,
		Success: make([]int64, numPoints),
		Failure: make([]int64, numPoints),
		Total:   make([]int64, numPoints),
	}
}

func (a *Reader) readTraffic(ctx context.Context, groupBy, modelCode string, plan windowPlan) (*trafficResult, error) {
	if plan.minutes() == 0 {
		source(ctx, "catalog")
		return &trafficResult{}, nil
	}
	end := plan.End
	cacheKey := fmt.Sprintf("trends:%s:%s", groupBy, modelCode)
	if cached, ok := a.getCache(ctx, cacheKey); ok {
		return cached.(*trafficResult), nil
	}

	start := plan.Buckets[0].End
	promRange := fmt.Sprintf("%ds", int64(plan.Step/time.Second))

	// 初始化响应
	res := trafficResult{}

	// 尝试从 Prometheus 获取数据
	if a.isPrometheusAvailable(ctx) {
		source(ctx, "prometheus")
		successQuery, errorQuery := buildTrendQueries(groupBy, promRange, modelCode)

		if groupBy == "" {
			// 全局汇总
			successData, err1 := a.queryPrometheusRange(ctx, successQuery, start.Unix(), end.Unix(), int64(plan.Step/time.Second))
			errorData, err2 := a.queryPrometheusRange(ctx, errorQuery, start.Unix(), end.Unix(), int64(plan.Step/time.Second))

			if err1 == nil && err2 == nil {
				label := "global"
				if modelCode != "" {
					label = modelCode
				}
				series := trendSeries{
					Label:   label,
					Success: make([]int64, len(plan.Buckets)),
					Failure: make([]int64, len(plan.Buckets)),
					Total:   make([]int64, len(plan.Buckets)),
				}

				// 使用时间戳对齐填充
				alignPrometheusData(series.Success, successData, start.Unix(), int64(plan.Step/time.Second))
				alignPrometheusData(series.Failure, errorData, start.Unix(), int64(plan.Step/time.Second))

				for i := 0; i < len(plan.Buckets); i++ {
					series.Total[i] = series.Success[i] + series.Failure[i]
				}

				res.Series = []trendSeries{series}
				a.setCache(ctx, cacheKey, &res, 5*time.Second)
				return &res, nil
			}
		} else {
			// 按标签分组
			labelKey := groupBy
			successMap, err1 := a.queryPrometheusRangeMulti(ctx, successQuery, start.Unix(), end.Unix(), int64(plan.Step/time.Second), labelKey)
			errorMap, err2 := a.queryPrometheusRangeMulti(ctx, errorQuery, start.Unix(), end.Unix(), int64(plan.Step/time.Second), labelKey)

			if err1 == nil && err2 == nil {
				// 合并所有标签
				allLabels := make(map[string]bool)
				for label := range successMap {
					allLabels[label] = true
				}
				for label := range errorMap {
					allLabels[label] = true
				}

				res.Series = make([]trendSeries, 0, len(allLabels))
				for label := range allLabels {
					series := trendSeries{
						Label:   label,
						Success: make([]int64, len(plan.Buckets)),
						Failure: make([]int64, len(plan.Buckets)),
						Total:   make([]int64, len(plan.Buckets)),
					}

					if successValues, ok := successMap[label]; ok {
						alignPrometheusData(series.Success, successValues, start.Unix(), int64(plan.Step/time.Second))
					}

					if errorValues, ok := errorMap[label]; ok {
						alignPrometheusData(series.Failure, errorValues, start.Unix(), int64(plan.Step/time.Second))
					}

					for i := 0; i < len(plan.Buckets); i++ {
						series.Total[i] = series.Success[i] + series.Failure[i]
					}

					res.Series = append(res.Series, series)
				}

				a.setCache(ctx, cacheKey, &res, 5*time.Second)
				return &res, nil
			}
		}
	}

	if groupBy == "endpoint" {
		endpointIDs := a.getEndpointsForTrend(ctx, modelCode)
		if len(endpointIDs) == 0 {
			res.Series = []trendSeries{}
			a.setCache(ctx, cacheKey, &res, 5*time.Second)
			return &res, nil
		}

		if a.redis != nil {
			source(ctx, "redis")
			stepMinutes := int64(plan.Step/time.Second) / 60
			if stepMinutes <= 0 {
				stepMinutes = 1
			}
			windowMinutes := int64(plan.minutes())
			numMinutes := int(windowMinutes)
			if numMinutes > 120 {
				numMinutes = 120
			}
			currentMinute := end.Add(-time.Minute).Unix() / 60
			startMinute := plan.Start.Unix() / 60
			firstMinute := currentMinute - int64(numMinutes-1)

			res.Series = make([]trendSeries, 0, len(endpointIDs))
			for _, epID := range endpointIDs {
				keys := make([]string, numMinutes*2)
				for i := 0; i < numMinutes; i++ {
					minute := firstMinute + int64(i)
					keys[i*2] = gatewaycontract.Keys.Status.Endpoint(epID, minute, gatewaycontract.MetricSuccess)
					keys[i*2+1] = gatewaycontract.Keys.Status.Endpoint(epID, minute, gatewaycontract.MetricFailure)
				}
				vals, err := a.redisValues(ctx, keys...)
				series := trendSeries{
					Label:   epID,
					Success: make([]int64, len(plan.Buckets)),
					Failure: make([]int64, len(plan.Buckets)),
					Total:   make([]int64, len(plan.Buckets)),
				}
				if err == nil && len(vals) == numMinutes*2 {
					aggregateRedisTrendValues(&series, vals, currentMinute, startMinute, stepMinutes)
				}
				res.Series = append(res.Series, series)
			}
			a.setCache(ctx, cacheKey, &res, 5*time.Second)
			return &res, nil
		}

		source(ctx, "memory")
		res.Series = buildMemoryTrendSeries(a.memory(), "endpoint", endpointIDs, plan)
		a.setCache(ctx, cacheKey, &res, 5*time.Second)
		return &res, nil
	}

	if modelCode != "" {
		resolvedModelCode := modelCode
		if a.db != nil {
			var m rschema.Model
			if err := a.db.WithContext(ctx).Where("(id = ? OR model_code = ?) AND deleted = '0'", modelCode, modelCode).First(&m).Error; err == nil && m.ModelCode != "" {
				resolvedModelCode = m.ModelCode
			}
		}

		if a.redis != nil && groupBy == "" {
			source(ctx, "redis")
			stepMinutes := int64(plan.Step/time.Second) / 60
			if stepMinutes <= 0 {
				stepMinutes = 1
			}
			windowMinutes := int64(plan.minutes())
			numMinutes := int(windowMinutes)
			if numMinutes > 120 {
				numMinutes = 120
			}
			currentMinute := end.Add(-time.Minute).Unix() / 60
			startMinute := plan.Start.Unix() / 60
			firstMinute := currentMinute - int64(numMinutes-1)

			keys := make([]string, numMinutes*2)
			for i := 0; i < numMinutes; i++ {
				minute := firstMinute + int64(i)
				keys[i*2] = gatewaycontract.Keys.Status.Model(resolvedModelCode, minute, gatewaycontract.MetricSuccess)
				keys[i*2+1] = gatewaycontract.Keys.Status.Model(resolvedModelCode, minute, gatewaycontract.MetricFailure)
			}
			vals, err := a.redisValues(ctx, keys...)
			series := trendSeries{
				Label:   resolvedModelCode,
				Success: make([]int64, len(plan.Buckets)),
				Failure: make([]int64, len(plan.Buckets)),
				Total:   make([]int64, len(plan.Buckets)),
			}
			if err == nil && len(vals) == numMinutes*2 {
				aggregateRedisTrendValues(&series, vals, currentMinute, startMinute, stepMinutes)
			}
			res.Series = []trendSeries{series}
			a.setCache(ctx, cacheKey, &res, 5*time.Second)
			return &res, nil
		}

		if a.redis == nil && groupBy == "" {
			source(ctx, "memory")
			res.Series = buildMemoryTrendSeries(a.memory(), "model", []string{resolvedModelCode}, plan)
			a.setCache(ctx, cacheKey, &res, 5*time.Second)
			return &res, nil
		}
		if groupBy == "" {
			res.Series = []trendSeries{emptyTrendSeries(len(plan.Buckets), resolvedModelCode)}
		} else {
			res.Series = []trendSeries{}
		}
		a.setCache(ctx, cacheKey, &res, 5*time.Second)
		return &res, nil
	}

	// Redis 降级沿用两小时窗口；内存模式支持完整的七天保留窗口及分组。
	fallbackMinutes := min(plan.minutes(), 120)
	if a.redis == nil {
		fallbackMinutes = plan.minutes()
	}
	if fallbackMinutes > 0 {
		var vals []interface{}
		var err error
		if a.redis != nil {
			source(ctx, "redis")
			minute := end.Add(-time.Minute).Unix() / 60
			keys := make([]string, fallbackMinutes*2)
			for i := 0; i < fallbackMinutes; i++ {
				ts := minute - int64(fallbackMinutes-1-i)
				keys[i*2] = gatewaycontract.Keys.Status.Global(ts, gatewaycontract.MetricSuccess)
				keys[i*2+1] = gatewaycontract.Keys.Status.Global(ts, gatewaycontract.MetricFailure)
			}
			vals, err = a.redisValues(ctx, keys...)
		} else {
			var labels []string
			switch groupBy {
			case "model":
				labels = a.memory().GetModelCodes()
			case "provider":
				labels = a.memory().GetProviderNames()
			case "endpoint":
				labels = a.getEndpointsForTrend(ctx, "")
			default:
				labels = []string{"global"}
			}
			source(ctx, "memory")
			res.Series = buildMemoryTrendSeries(a.memory(), groupBy, labels, plan)
			a.setCache(ctx, cacheKey, &res, 5*time.Second)
			return &res, nil
		}

		if err == nil && len(vals) == fallbackMinutes*2 {
			series := trendSeries{
				Label:   "global",
				Success: make([]int64, len(plan.Buckets)),
				Failure: make([]int64, len(plan.Buckets)),
				Total:   make([]int64, len(plan.Buckets)),
			}

			aggregateRedisTrendValues(&series, vals, end.Add(-time.Minute).Unix()/60, plan.Start.Unix()/60, int64(plan.Step/time.Second)/60)

			res.Series = []trendSeries{series}
		} else {
			res.Series = []trendSeries{
				{
					Label:   "global",
					Success: make([]int64, len(plan.Buckets)),
					Failure: make([]int64, len(plan.Buckets)),
					Total:   make([]int64, len(plan.Buckets)),
				},
			}
		}
	} else {
		res.Series = []trendSeries{
			{
				Label:   "global",
				Success: make([]int64, len(plan.Buckets)),
				Failure: make([]int64, len(plan.Buckets)),
				Total:   make([]int64, len(plan.Buckets)),
			},
		}
	}

	a.setCache(ctx, cacheKey, &res, 5*time.Second)
	return &res, nil
}

func buildMemoryTrendSeries(store *metrics.MemoryStore, groupBy string, labels []string, plan windowPlan) []trendSeries {
	end := plan.End
	stepMinutes := int64(plan.Step/time.Second) / 60
	numMinutes := plan.minutes()
	currentMinute := end.Add(-time.Minute).Unix() / 60
	seriesList := make([]trendSeries, 0, len(labels))

	for _, label := range labels {
		values := make([]interface{}, numMinutes*2)
		for i := 0; i < numMinutes; i++ {
			minute := currentMinute - int64(numMinutes-1-i)
			var success, failure int64
			switch groupBy {
			case "model":
				success, failure = store.GetModelStatus(label, minute)
			case "provider":
				success, failure = store.GetProviderStatus(label, minute)
			case "endpoint":
				success, failure = store.GetEndpointStatus(label, minute)
			default:
				success, failure = store.GetGlobalStatus(minute)
			}
			if success > 0 {
				values[i*2] = success
			}
			if failure > 0 {
				values[i*2+1] = failure
			}
		}

		series := trendSeries{
			Label:   label,
			Success: make([]int64, len(plan.Buckets)),
			Failure: make([]int64, len(plan.Buckets)),
			Total:   make([]int64, len(plan.Buckets)),
		}
		aggregateRedisTrendValues(&series, values, currentMinute, plan.Start.Unix()/60, stepMinutes)
		seriesList = append(seriesList, series)
	}
	return seriesList
}

func (a *Reader) getEndpointsForTrend(ctx context.Context, modelCode string) []string {
	var endpointIDs []string
	if a.db != nil {
		if modelCode != "" {
			var model rschema.Model
			if err := a.db.WithContext(ctx).Where("(id = ? OR model_code = ?) AND deleted = '0'", modelCode, modelCode).First(&model).Error; err == nil {
				var eps []rschema.Endpoint
				if err := a.db.WithContext(ctx).Where("model_id = ? AND enabled = 1 AND deleted = '0'", model.ID).Find(&eps).Error; err == nil {
					for _, ep := range eps {
						endpointIDs = append(endpointIDs, ep.ID)
					}
				}
			}
		} else {
			var eps []rschema.Endpoint
			if err := a.db.WithContext(ctx).Where("enabled = 1 AND deleted = '0'").Find(&eps).Error; err == nil {
				for _, ep := range eps {
					endpointIDs = append(endpointIDs, ep.ID)
				}
			}
		}
	}
	if len(endpointIDs) == 0 {
		if modelCode != "" {
			endpointIDs = a.memory().GetModelEndpointIDs(modelCode)
		} else {
			endpointIDs = a.memory().GetEndpointIDs()
		}
	}
	return endpointIDs
}
