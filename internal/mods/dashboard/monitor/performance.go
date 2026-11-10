package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tokenlive/tokenlive-admin/pkg/gatewaycontract"
)

func (a *Reader) getModelPerformanceTrends(ctx context.Context, modelCode, timeRange string) (*performanceResult, error) {
	return a.getModelPerformanceTrendsAt(ctx, modelCode, timeRange, a.now().Truncate(time.Minute))
}

func (a *Reader) getModelPerformanceTrendsAt(ctx context.Context, modelCode, timeRange string, end time.Time) (*performanceResult, error) {
	ctx, plan, planErr := a.queryPlan(ctx, timeRange, end)
	if planErr != nil {
		return nil, planErr
	}
	if plan.minutes() == 0 {
		source(ctx, "catalog")
		return &performanceResult{}, nil
	}
	end = plan.End
	cacheKey := fmt.Sprintf("model-performance-trends:%s:%s:%d", modelCode, timeRange, end.Unix())
	if cached, ok := a.getCache(ctx, cacheKey); ok {
		return cached.(*performanceResult), nil
	}

	rangeConfig := plan.trendConfig()
	end = rangeConfig.end
	start := plan.Buckets[0].End
	res := &performanceResult{
		AvgTTFTMs: make([]*float64, rangeConfig.numPoints),
		Otps:      make([]*float64, rangeConfig.numPoints),
	}

	if a.isPrometheusAvailable(ctx) {
		source(ctx, "prometheus")
		promRange := fmt.Sprintf("%ds", rangeConfig.stepSeconds)
		ttftQuery, otpsQuery := buildModelPerformanceQueries(modelCode, promRange)
		var ttftValues, otpsValues [][]interface{}
		var ttftErr, otpsErr error
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			ttftValues, ttftErr = a.queryPrometheusRange(ctx,
				ttftQuery,
				start.Unix(),
				end.Unix(),
				rangeConfig.stepSeconds,
			)
		}()
		go func() {
			defer wg.Done()
			otpsValues, otpsErr = a.queryPrometheusRange(ctx,
				otpsQuery,
				start.Unix(),
				end.Unix(),
				rangeConfig.stepSeconds,
			)
		}()
		wg.Wait()
		if ttftErr == nil {
			alignPrometheusFloatData(res.AvgTTFTMs, ttftValues, start.Unix(), rangeConfig.stepSeconds)
		}
		if otpsErr == nil {
			alignPrometheusFloatData(res.Otps, otpsValues, start.Unix(), rangeConfig.stepSeconds)
		}
	}

	if !hasFloatValues(res.AvgTTFTMs) || !hasFloatValues(res.Otps) {
		fallbackTTFT, fallbackOtps, err := a.getModelPerformanceFallback(ctx, modelCode, rangeConfig, start, end)
		if err == nil {
			if !hasFloatValues(res.AvgTTFTMs) {
				res.AvgTTFTMs = fallbackTTFT
			}
			if !hasFloatValues(res.Otps) {
				res.Otps = fallbackOtps
			}
		}
	}

	a.setCache(ctx, cacheKey, res, 5*time.Second)
	return res, nil
}

func hasFloatValues(values []*float64) bool {
	for _, value := range values {
		if value != nil {
			return true
		}
	}
	return false
}

func (a *Reader) getModelPerformanceFallback(
	ctx context.Context,
	modelCode string,
	rangeConfig trendRangeConfig,
	start time.Time,
	end time.Time,
) ([]*float64, []*float64, error) {
	ttft := make([]*float64, rangeConfig.numPoints)
	otps := make([]*float64, rangeConfig.numPoints)
	stepMinutes := rangeConfig.stepSeconds / 60
	windowMinutes := int64(rangeConfig.minutes)
	if stepMinutes <= 0 || windowMinutes > 7*24*60 {
		return ttft, otps, nil
	}

	currentMinute := end.Add(-time.Minute).Unix() / 60
	if a.redis != nil && windowMinutes > 120 {
		windowMinutes = 120
	}
	numMinutes := int(windowMinutes)
	const keysPerMinute = 6
	values := make([]interface{}, numMinutes*keysPerMinute)

	if a.redis != nil {
		source(ctx, "redis")
		keys := make([]string, len(values))
		firstMinute := currentMinute - int64(numMinutes-1)
		for i := 0; i < numMinutes; i++ {
			minute := firstMinute + int64(i)
			offset := i * keysPerMinute
			keys[offset] = gatewaycontract.Keys.Status.Model(modelCode, minute, gatewaycontract.MetricSuccess)
			keys[offset+1] = gatewaycontract.Keys.Status.Model(modelCode, minute, gatewaycontract.MetricFailure)
			keys[offset+2] = gatewaycontract.Keys.Status.Model(modelCode, minute, gatewaycontract.MetricTTFTSum)
			keys[offset+3] = gatewaycontract.Keys.Status.Model(modelCode, minute, gatewaycontract.MetricTTFTCount)
			keys[offset+4] = gatewaycontract.Keys.Status.Model(modelCode, minute, gatewaycontract.MetricOutputTokens)
			keys[offset+5] = gatewaycontract.Keys.Status.Model(modelCode, minute, gatewaycontract.MetricDurationMs)
		}

		values = values[:0]
		const batchSize = 500
		for i := 0; i < len(keys); i += batchSize {
			batchEnd := i + batchSize
			if batchEnd > len(keys) {
				batchEnd = len(keys)
			}
			batchValues, err := a.redisValues(ctx, keys[i:batchEnd]...)
			if err != nil {
				return ttft, otps, err
			}
			values = append(values, batchValues...)
		}
	} else {
		source(ctx, "memory")
		firstMinute := currentMinute - int64(numMinutes-1)
		for i := 0; i < numMinutes; i++ {
			minute := firstMinute + int64(i)
			perf := a.memory().GetModelMinutePerf(modelCode, minute)
			offset := i * keysPerMinute
			values[offset] = perf.Success
			values[offset+1] = perf.Fail
			values[offset+2] = perf.TTFTSum
			values[offset+3] = perf.TTFTCount
			values[offset+4] = perf.Output
			values[offset+5] = perf.DurationMs
		}
	}

	aggregateModelPerformanceValues(
		ttft,
		otps,
		values,
		currentMinute,
		rangeConfig.plan.Start.Unix()/60,
		stepMinutes,
	)
	return ttft, otps, nil
}

func aggregateModelPerformanceValues(
	ttft []*float64,
	otps []*float64,
	values []interface{},
	currentMinute int64,
	startMinute int64,
	stepMinutes int64,
) {
	const valuesPerMinute = 6
	numMinutes := len(values) / valuesPerMinute
	ttftSums := make([]int64, len(ttft))
	ttftCounts := make([]int64, len(ttft))
	outputSums := make([]int64, len(otps))
	durationSums := make([]int64, len(otps))

	for i := 0; i < numMinutes; i++ {
		valueMinute := currentMinute - int64(numMinutes-1-i)
		diff := valueMinute - startMinute
		index := int64(0)
		if diff >= 0 {
			index = diff / stepMinutes
		}
		if index < 0 || index >= int64(len(ttft)) {
			continue
		}

		offset := i * valuesPerMinute
		ttftSums[index] += parseMetricInt(values[offset+2])
		ttftCounts[index] += parseMetricInt(values[offset+3])
		outputSums[index] += parseMetricInt(values[offset+4])
		durationSums[index] += parseMetricInt(values[offset+5])
	}

	for i := range ttft {
		if ttftCounts[i] > 0 {
			value := float64(ttftSums[i]) / float64(ttftCounts[i])
			ttft[i] = &value
		}
		if outputSums[i] > 0 && durationSums[i] > 0 {
			value := float64(outputSums[i]) / (float64(durationSums[i]) / 1000)
			otps[i] = &value
		}
	}
}
