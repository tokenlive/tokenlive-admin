package monitor

import (
	"fmt"
	"math"
	"strconv"
)

func parseMetricInt(value interface{}) int64 {
	if value == nil {
		return 0
	}
	parsed, _ := strconv.ParseInt(fmt.Sprint(value), 10, 64)
	return parsed
}

func aggregateRedisTrendValues(series *trendSeries, values []interface{}, currentMinute, startMinute, stepMinutes int64) {
	numMinutes := len(values) / 2
	for i := 0; i < numMinutes; i++ {
		var succ, fail int64
		if values[i*2] != nil {
			succ, _ = strconv.ParseInt(fmt.Sprint(values[i*2]), 10, 64)
		}
		if values[i*2+1] != nil {
			fail, _ = strconv.ParseInt(fmt.Sprint(values[i*2+1]), 10, 64)
		}

		valueMinute := currentMinute - int64(numMinutes-1-i)
		diff := valueMinute - startMinute
		idx := int64(-1)
		if diff >= 0 {
			idx = diff / stepMinutes
		}
		if idx >= 0 && idx < int64(len(series.Total)) {
			series.Success[idx] += succ
			series.Failure[idx] += fail
			series.Total[idx] += succ + fail
		}
	}
}

func mustParseInt(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

// alignPrometheusData 将 Prometheus Range 查询返回的 values 按时间戳对齐填充到 dst 数组中
func alignPrometheusData(dst []int64, values [][]interface{}, startUnix int64, step int64) {
	numPoints := int64(len(dst))
	for _, v := range values {
		if len(v) < 2 {
			continue
		}

		// 解析时间戳
		var ts float64
		switch tVal := v[0].(type) {
		case float64:
			ts = tVal
		case string:
			ts, _ = strconv.ParseFloat(tVal, 64)
		}
		timestamp := int64(ts)

		// 解析数值
		var val int64
		if valStr, ok := v[1].(string); ok {
			if fVal, err := strconv.ParseFloat(valStr, 64); err == nil {
				val = int64(math.Round(fVal))
			}
		}

		// 计算该时间戳对应的目标数组索引 (四舍五入对齐)
		diff := timestamp - startUnix
		idx := (diff + step/2) / step

		// 边界处理
		if idx >= 0 && idx < numPoints {
			dst[idx] = val
		}
	}
}

func alignPrometheusFloatData(dst []*float64, values [][]interface{}, startUnix int64, step int64) {
	numPoints := int64(len(dst))
	for _, value := range values {
		if len(value) < 2 {
			continue
		}

		var timestampValue float64
		switch typed := value[0].(type) {
		case float64:
			timestampValue = typed
		case string:
			timestampValue, _ = strconv.ParseFloat(typed, 64)
		}

		var metricValue float64
		switch typed := value[1].(type) {
		case string:
			metricValue, _ = strconv.ParseFloat(typed, 64)
		case float64:
			metricValue = typed
		default:
			continue
		}
		if math.IsNaN(metricValue) || math.IsInf(metricValue, 0) {
			continue
		}

		diff := int64(timestampValue) - startUnix
		idx := (diff + step/2) / step
		if idx >= 0 && idx < numPoints {
			v := metricValue
			dst[idx] = &v
		}
	}
}
