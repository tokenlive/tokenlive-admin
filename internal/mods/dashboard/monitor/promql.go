package monitor

import (
	"fmt"
	"strings"
)

func escapePromLabelValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return value
}

func promLabelMatcher(extra ...string) string {
	parts := make([]string, 0, len(extra))
	for _, part := range extra {
		if strings.TrimSpace(part) != "" {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ",")
}

func buildTrendQueries(groupBy, promRange, modelCode string) (successQuery, errorQuery string) {
	matchers := []string{}
	if modelCode != "" {
		matchers = append(matchers, fmt.Sprintf(`model="%s"`, escapePromLabelValue(modelCode)))
	}

	successMatchers := append([]string{`status="success"`}, matchers...)
	errorMatchers := append([]string{`status="error"`}, matchers...)
	successSelector := promLabelMatcher(successMatchers...)
	errorSelector := promLabelMatcher(errorMatchers...)

	switch groupBy {
	case "model":
		successQuery = fmt.Sprintf(`sum by (model) (increase(%s{%s}[%s]))`, mRequestTotal, successSelector, promRange)
		errorQuery = fmt.Sprintf(`sum by (model) (increase(%s{%s}[%s]))`, mRequestTotal, errorSelector, promRange)
	case "provider":
		successQuery = fmt.Sprintf(`sum by (provider) (increase(%s{%s}[%s]))`, mRequestTotal, successSelector, promRange)
		errorQuery = fmt.Sprintf(`sum by (provider) (increase(%s{%s}[%s]))`, mRequestTotal, errorSelector, promRange)
	case "tenant":
		successQuery = fmt.Sprintf(`sum by (tenant) (increase(%s{%s}[%s]))`, mRequestTotal, successSelector, promRange)
		errorQuery = fmt.Sprintf(`sum by (tenant) (increase(%s{%s}[%s]))`, mRequestTotal, errorSelector, promRange)
	case "endpoint":
		successQuery = fmt.Sprintf(`sum by (endpoint) (increase(%s{%s}[%s]))`, mRequestTotal, successSelector, promRange)
		errorQuery = fmt.Sprintf(`sum by (endpoint) (increase(%s{%s}[%s]))`, mRequestTotal, errorSelector, promRange)
	default:
		successQuery = fmt.Sprintf(`sum(increase(%s{%s}[%s]))`, mRequestTotal, successSelector, promRange)
		errorQuery = fmt.Sprintf(`sum(increase(%s{%s}[%s]))`, mRequestTotal, errorSelector, promRange)
	}
	return successQuery, errorQuery
}

func buildModelPerformanceQueries(modelCode, promRange string) (ttftQuery, otpsQuery string) {
	modelMatcher := fmt.Sprintf(`model="%s"`, escapePromLabelValue(modelCode))
	ttftQuery = fmt.Sprintf(
		`sum(increase(%s{%s}[%s])) / sum(increase(%s{%s}[%s])) * 1000`,
		mTtftSum,
		modelMatcher,
		promRange,
		mTtftCount,
		modelMatcher,
		promRange,
	)
	otpsQuery = fmt.Sprintf(
		`sum(increase(%s{%s,type="output"}[%s])) / sum(increase(%s{%s}[%s]))`,
		mTokensTotal,
		modelMatcher,
		promRange,
		mRequestDurationSum,
		modelMatcher,
		promRange,
	)
	return ttftQuery, otpsQuery
}
