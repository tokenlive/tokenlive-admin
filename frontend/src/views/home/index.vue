<template>
    <div class="dashboard">
        <!-- 核心报警：熔断隔离横幅 -->
        <a-alert
            v-if="circuitBreakers.length > 0"
            type="error"
            show-icon
            style="
                margin-bottom: 16px;
                border-radius: 8px;
                border: 1px solid var(--color-error);
                box-shadow: 0 2px 12px var(--color-error-bg);
            ">
            <template #message>
                <div style="display: flex; justify-content: space-between; align-items: center">
                    <span>
                        <alert-outlined style="margin-right: 8px; font-weight: bold; color: var(--color-error)" />
                        <strong style="color: var(--color-error)">{{ $t('pages.dashboard.alert.warning') }}:</strong>
                        {{ $t('pages.dashboard.alert.circuit_breakers', { count: circuitBreakers.length }) }}
                        <span style="margin-left: 8px">
                            <a-tooltip
                                v-for="cb in circuitBreakers"
                                :key="cb.id || cb">
                                <template #title>
                                    <div style="font-size: 12px; line-height: 1.6; padding: 4px">
                                        <div><strong>Type:</strong> {{ cb.type || 'endpoint' }}</div>
                                        <div><strong>ID:</strong> {{ cb.id || cb }}</div>
                                        <div
                                            v-if="cb.url"
                                            style="word-break: break-all">
                                            <strong>URL:</strong> {{ cb.url }}
                                        </div>
                                    </div>
                                </template>
                                <a-tag
                                    color="error"
                                    style="cursor: help; margin-right: 4px; border-radius: 4px; font-weight: 500">
                                    <alert-outlined style="margin-right: 4px" />{{ cb.name || cb.id || cb }}
                                </a-tag>
                            </a-tooltip>
                        </span>
                    </span>
                    <a-button
                        type="primary"
                        danger
                        size="small"
                        @click="goTo('/resource/model')">
                        {{ $t('pages.dashboard.alert.action') }}
                    </a-button>
                </div>
            </template>
        </a-alert>
        <a-alert
            v-else
            type="success"
            show-icon
            style="
                margin-bottom: 16px;
                border-radius: 8px;
                border: 1px solid var(--color-success);
                box-shadow: 0 2px 12px var(--color-success-bg);
            ">
            <template #message>
                <span style="color: var(--color-success); font-weight: 500">
                    {{ $t('pages.dashboard.alert.healthy') }}
                </span>
            </template>
        </a-alert>

        <!-- 第一行：遥测动态大盘卡片 -->
        <a-row
            :gutter="16"
            style="margin-bottom: 16px">
            <a-col
                :xs="24"
                :sm="12"
                :md="8"
                :lg="4">
                <a-card
                    class="telemetry-card telemetry-card--blue"
                    :bordered="false"
                    hoverable>
                    <div class="telemetry-title">{{ $t('pages.dashboard.metrics.daily_requests') }}</div>
                    <div class="telemetry-value">
                        {{ metrics.dailyRequests.toLocaleString() }}
                        <span class="telemetry-unit">{{ $t('pages.dashboard.units.requests') }}</span>
                    </div>
                    <div class="telemetry-footer">{{ $t('pages.dashboard.metrics.daily_requests.footer') }}</div>
                </a-card>
            </a-col>
            <a-col
                :xs="24"
                :sm="12"
                :md="8"
                :lg="4">
                <a-card
                    class="telemetry-card telemetry-card--green"
                    :bordered="false"
                    hoverable>
                    <div class="telemetry-title">
                        {{ $t('pages.dashboard.metrics.qps') }}
                        <span class="pulse-indicator"></span>
                    </div>
                    <div class="telemetry-value">
                        {{ metrics.qps.toFixed(2) }}
                        <span class="telemetry-unit">req/s</span>
                    </div>
                    <div class="telemetry-footer">{{ $t('pages.dashboard.metrics.qps.footer') }}</div>
                </a-card>
            </a-col>
            <a-col
                :xs="24"
                :sm="12"
                :md="8"
                :lg="4">
                <a-card
                    class="telemetry-card telemetry-card--purple"
                    :bordered="false"
                    hoverable>
                    <div class="telemetry-title">{{ $t('pages.dashboard.metrics.tokens') }}</div>
                    <div class="telemetry-value">
                        {{ (metrics.dailyPromptTokens + metrics.dailyCompletionTokens).toLocaleString() }}
                        <span class="telemetry-unit">Tks</span>
                    </div>
                    <div class="telemetry-footer">
                        Input: {{ metrics.dailyPromptTokens.toLocaleString() }} | Output:
                        {{ metrics.dailyCompletionTokens.toLocaleString() }}
                    </div>
                </a-card>
            </a-col>
            <a-col
                :xs="24"
                :sm="12"
                :md="8"
                :lg="4">
                <a-card
                    class="telemetry-card telemetry-card--orange"
                    :bordered="false"
                    hoverable>
                    <div class="telemetry-title">{{ $t('pages.dashboard.metrics.cost') }}</div>
                    <div class="telemetry-value">
                        {{ metrics.dailyCost.toFixed(4) }}
                        <span class="telemetry-unit">{{ $t('pages.dashboard.units.cost') }}</span>
                    </div>
                    <div class="telemetry-footer">
                        {{ $t('pages.dashboard.metrics.cost.footer') }}
                    </div>
                </a-card>
            </a-col>
            <!-- 平均响应延迟卡片 -->
            <a-col
                :xs="24"
                :sm="12"
                :md="8"
                :lg="4">
                <a-card
                    class="telemetry-card telemetry-card--cyan"
                    :bordered="false"
                    hoverable>
                    <div class="telemetry-title">{{ $t('pages.dashboard.metrics.avg_latency') }}</div>
                    <div class="telemetry-value">
                        {{ formatLatency(metrics.avgLatency) }}
                    </div>
                    <div class="telemetry-footer">{{ $t('pages.dashboard.metrics.avg_latency.footer') }}</div>
                </a-card>
            </a-col>
            <!-- 平均首包延迟卡片 -->
            <a-col
                :xs="24"
                :sm="12"
                :md="8"
                :lg="4">
                <a-card
                    class="telemetry-card telemetry-card--magenta"
                    :bordered="false"
                    hoverable>
                    <div class="telemetry-title">{{ $t('pages.dashboard.metrics.avg_ttft') }}</div>
                    <div class="telemetry-value">
                        {{ formatLatency(metrics.avgTTFT) }}
                    </div>
                    <div class="telemetry-footer">{{ $t('pages.dashboard.metrics.avg_ttft.footer') }}</div>
                </a-card>
            </a-col>
        </a-row>

        <!-- 第二行：遥测图表（走势 + Token 分布） -->
        <a-row :gutter="16">
            <a-col
                :xs="24"
                :lg="16"
                style="margin-bottom: 16px">
                <a-card
                    :title="$t('pages.dashboard.trends.title')"
                    :bordered="false"
                    hoverable>
                    <template #extra>
                        <a-space :size="8">
                            <a-select
                                v-model:value="trendsTimeRange"
                                style="width: 120px"
                                @change="handleTrendsGroupChange">
                                <a-select-option value="1h">{{
                                    $t('pages.dashboard.trends.range.1h')
                                }}</a-select-option>
                                <a-select-option value="6h">{{
                                    $t('pages.dashboard.trends.range.6h')
                                }}</a-select-option>
                                <a-select-option value="24h">{{
                                    $t('pages.dashboard.trends.range.24h')
                                }}</a-select-option>
                                <a-select-option value="7d">{{
                                    $t('pages.dashboard.trends.range.7d')
                                }}</a-select-option>
                                <a-select-option value="today">{{
                                    $t('pages.dashboard.trends.range.today')
                                }}</a-select-option>
                            </a-select>
                            <a-select
                                v-model:value="trendsGroupBy"
                                style="width: 120px"
                                @change="handleTrendsGroupChange">
                                <a-select-option value="">{{
                                    $t('pages.dashboard.trends.group.global')
                                }}</a-select-option>
                                <a-select-option value="model">{{
                                    $t('pages.dashboard.trends.group.model')
                                }}</a-select-option>
                                <a-select-option value="provider">{{
                                    $t('pages.dashboard.trends.group.provider')
                                }}</a-select-option>
                                <a-select-option value="tenant">{{
                                    $t('pages.dashboard.trends.group.tenant')
                                }}</a-select-option>
                                <a-select-option value="endpoint">{{
                                    $t('pages.dashboard.trends.group.endpoint')
                                }}</a-select-option>
                            </a-select>
                        </a-space>
                    </template>
                    <transition
                        name="fade-chart"
                        mode="out-in">
                        <div
                            v-if="firstLoading"
                            class="skeleton-container skeleton-line-container">
                            <div class="skeleton-header">
                                <span class="skeleton-legend-item"></span>
                                <span class="skeleton-legend-item"></span>
                                <span class="skeleton-legend-item"></span>
                            </div>
                            <div class="skeleton-body">
                                <svg
                                    class="skeleton-svg-line"
                                    viewBox="0 0 800 280"
                                    width="100%"
                                    height="100%">
                                    <line
                                        x1="40"
                                        y1="30"
                                        x2="780"
                                        y2="30"
                                        stroke="var(--skeleton-line-color)"
                                        stroke-dasharray="4,4" />
                                    <line
                                        x1="40"
                                        y1="90"
                                        x2="780"
                                        y2="90"
                                        stroke="var(--skeleton-line-color)"
                                        stroke-dasharray="4,4" />
                                    <line
                                        x1="40"
                                        y1="150"
                                        x2="780"
                                        y2="150"
                                        stroke="var(--skeleton-line-color)"
                                        stroke-dasharray="4,4" />
                                    <line
                                        x1="40"
                                        y1="210"
                                        x2="780"
                                        y2="210"
                                        stroke="var(--skeleton-line-color)"
                                        stroke-dasharray="4,4" />
                                    <line
                                        x1="40"
                                        y1="250"
                                        x2="780"
                                        y2="250"
                                        stroke="var(--skeleton-line-color)" />
                                    <rect
                                        x="10"
                                        y="25"
                                        width="20"
                                        height="10"
                                        rx="3"
                                        class="skeleton-block" />
                                    <rect
                                        x="10"
                                        y="85"
                                        width="20"
                                        height="10"
                                        rx="3"
                                        class="skeleton-block" />
                                    <rect
                                        x="10"
                                        y="145"
                                        width="20"
                                        height="10"
                                        rx="3"
                                        class="skeleton-block" />
                                    <rect
                                        x="10"
                                        y="205"
                                        width="20"
                                        height="10"
                                        rx="3"
                                        class="skeleton-block" />
                                    <rect
                                        x="10"
                                        y="245"
                                        width="20"
                                        height="10"
                                        rx="3"
                                        class="skeleton-block" />
                                    <rect
                                        x="70"
                                        y="260"
                                        width="40"
                                        height="10"
                                        rx="3"
                                        class="skeleton-block" />
                                    <rect
                                        x="210"
                                        y="260"
                                        width="40"
                                        height="10"
                                        rx="3"
                                        class="skeleton-block" />
                                    <rect
                                        x="350"
                                        y="260"
                                        width="40"
                                        height="10"
                                        rx="3"
                                        class="skeleton-block" />
                                    <rect
                                        x="490"
                                        y="260"
                                        width="40"
                                        height="10"
                                        rx="3"
                                        class="skeleton-block" />
                                    <rect
                                        x="630"
                                        y="260"
                                        width="40"
                                        height="10"
                                        rx="3"
                                        class="skeleton-block" />
                                    <rect
                                        x="730"
                                        y="260"
                                        width="40"
                                        height="10"
                                        rx="3"
                                        class="skeleton-block" />
                                    <path
                                        d="M 40 180 Q 150 120 260 200 T 480 80 T 700 160 T 780 110"
                                        fill="none"
                                        stroke="var(--skeleton-path-color-1)"
                                        stroke-width="3"
                                        stroke-linecap="round"
                                        class="skeleton-pulse" />
                                    <path
                                        d="M 40 240 Q 150 210 260 230 T 480 180 T 700 220 T 780 190"
                                        fill="none"
                                        stroke="var(--skeleton-path-color-2)"
                                        stroke-width="2"
                                        stroke-linecap="round"
                                        stroke-dasharray="3,3"
                                        class="skeleton-pulse" />
                                </svg>
                            </div>
                        </div>
                        <x-chart
                            v-else
                            :options="trendsChartOptions"
                            height="340" />
                    </transition>
                </a-card>
            </a-col>
            <a-col
                :xs="24"
                :lg="8"
                style="margin-bottom: 16px">
                <a-card
                    :title="$t('pages.dashboard.tokens.distribution')"
                    :bordered="false"
                    hoverable>
                    <transition
                        name="fade-chart"
                        mode="out-in">
                        <div
                            v-if="firstLoading"
                            class="skeleton-container skeleton-pie-container">
                            <div class="skeleton-body-pie">
                                <svg
                                    class="skeleton-svg-pie"
                                    viewBox="0 0 200 200"
                                    width="160"
                                    height="160">
                                    <circle
                                        cx="100"
                                        cy="100"
                                        r="70"
                                        fill="none"
                                        stroke="var(--skeleton-line-color)"
                                        stroke-width="18" />
                                    <circle
                                        cx="100"
                                        cy="100"
                                        r="70"
                                        fill="none"
                                        stroke="var(--skeleton-path-color-1)"
                                        stroke-width="18"
                                        stroke-dasharray="120 400"
                                        stroke-dashoffset="0"
                                        class="skeleton-pulse" />
                                    <circle
                                        cx="100"
                                        cy="100"
                                        r="70"
                                        fill="none"
                                        stroke="var(--skeleton-path-color-2)"
                                        stroke-width="18"
                                        stroke-dasharray="80 400"
                                        stroke-dashoffset="-130"
                                        class="skeleton-pulse" />
                                    <circle
                                        cx="100"
                                        cy="100"
                                        r="70"
                                        fill="none"
                                        stroke="var(--skeleton-path-color-3)"
                                        stroke-width="18"
                                        stroke-dasharray="50 400"
                                        stroke-dashoffset="-220"
                                        class="skeleton-pulse" />
                                    <rect
                                        x="75"
                                        y="85"
                                        width="50"
                                        height="15"
                                        rx="3"
                                        class="skeleton-block" />
                                    <rect
                                        x="85"
                                        y="108"
                                        width="30"
                                        height="10"
                                        rx="3"
                                        class="skeleton-block" />
                                </svg>
                                <div class="skeleton-pie-legends">
                                    <div class="skeleton-legend-row">
                                        <span class="skeleton-legend-dot color-1"></span
                                        ><span class="skeleton-legend-text"></span>
                                    </div>
                                    <div class="skeleton-legend-row">
                                        <span class="skeleton-legend-dot color-2"></span
                                        ><span class="skeleton-legend-text"></span>
                                    </div>
                                    <div class="skeleton-legend-row">
                                        <span class="skeleton-legend-dot color-3"></span
                                        ><span class="skeleton-legend-text"></span>
                                    </div>
                                    <div class="skeleton-legend-row">
                                        <span class="skeleton-legend-dot color-4"></span
                                        ><span class="skeleton-legend-text"></span>
                                    </div>
                                </div>
                            </div>
                        </div>
                        <x-chart
                            v-else
                            :options="tokenChartOptions"
                            height="340" />
                    </transition>
                </a-card>
            </a-col>
        </a-row>

        <!-- 第三行：模型使用排行 -->
        <a-row :gutter="16">
            <a-col
                :xs="24"
                style="margin-bottom: 16px">
                <a-card
                    :title="$t('pages.dashboard.modelRanking.title')"
                    :bordered="false"
                    hoverable>
                    <template #extra>
                        <a-space :size="8">
                            <a-select
                                v-model:value="rankingTimeRange"
                                style="width: 120px"
                                @change="handleRankingSortChange">
                                <a-select-option value="1h">{{
                                    $t('pages.dashboard.modelRanking.range.1h')
                                }}</a-select-option>
                                <a-select-option value="6h">{{
                                    $t('pages.dashboard.modelRanking.range.6h')
                                }}</a-select-option>
                                <a-select-option value="24h">{{
                                    $t('pages.dashboard.modelRanking.range.24h')
                                }}</a-select-option>
                                <a-select-option value="7d">{{
                                    $t('pages.dashboard.modelRanking.range.7d')
                                }}</a-select-option>
                                <a-select-option value="today">{{
                                    $t('pages.dashboard.modelRanking.range.today')
                                }}</a-select-option>
                            </a-select>
                            <a-select
                                v-model:value="rankingSortBy"
                                style="width: 120px"
                                @change="handleRankingSortChange">
                                <a-select-option value="request_count">{{
                                    $t('pages.dashboard.modelRanking.sort.request_count')
                                }}</a-select-option>
                                <a-select-option value="avg_latency">{{
                                    $t('pages.dashboard.modelRanking.sort.avg_latency')
                                }}</a-select-option>
                                <a-select-option value="avg_ttft">{{
                                    $t('pages.dashboard.modelRanking.sort.avg_ttft')
                                }}</a-select-option>
                                <a-select-option value="tokens">{{
                                    $t('pages.dashboard.modelRanking.sort.tokens')
                                }}</a-select-option>
                                <a-select-option value="cost">{{
                                    $t('pages.dashboard.modelRanking.sort.cost')
                                }}</a-select-option>
                                <a-select-option value="success_rate">{{
                                    $t('pages.dashboard.modelRanking.sort.success_rate')
                                }}</a-select-option>
                            </a-select>
                        </a-space>
                    </template>
                    <transition
                        name="fade-chart"
                        mode="out-in">
                        <div
                            v-if="firstLoading"
                            class="skeleton-container skeleton-table-container">
                            <div class="skeleton-table-header">
                                <div class="skeleton-table-col col-1">
                                    <span class="skeleton-block block-med"></span>
                                </div>
                                <div class="skeleton-table-col col-2">
                                    <span class="skeleton-block block-short"></span>
                                </div>
                                <div class="skeleton-table-col col-3">
                                    <span class="skeleton-block block-short"></span>
                                </div>
                                <div class="skeleton-table-col col-4">
                                    <span class="skeleton-block block-short"></span>
                                </div>
                                <div class="skeleton-table-col col-5">
                                    <span class="skeleton-block block-short"></span>
                                </div>
                            </div>
                            <div class="skeleton-table-body">
                                <div
                                    v-for="i in 4"
                                    :key="i"
                                    class="skeleton-table-row">
                                    <div class="skeleton-table-col col-1">
                                        <span class="skeleton-block block-long"></span>
                                    </div>
                                    <div class="skeleton-table-col col-2">
                                        <span class="skeleton-block block-med"></span>
                                    </div>
                                    <div class="skeleton-table-col col-3">
                                        <span class="skeleton-block block-short"></span>
                                    </div>
                                    <div class="skeleton-table-col col-4">
                                        <span class="skeleton-block block-short"></span>
                                    </div>
                                    <div class="skeleton-table-col col-5">
                                        <span class="skeleton-block block-short"></span>
                                    </div>
                                </div>
                            </div>
                        </div>
                        <div v-else-if="modelRanking.length > 0">
                            <a-table
                                :data-source="modelRanking"
                                :columns="columns"
                                :pagination="false"
                                size="middle"
                                row-key="model_code"
                                :scroll="{ x: 800 }"
                                style="border-radius: 8px; overflow: hidden">
                                <template #bodyCell="{ column, record }">
                                    <template v-if="column.key === 'model_name'">
                                        <div>
                                            <span
                                                style="font-weight: 600; color: var(--color-primary); cursor: pointer"
                                                @click="goToModel(record.model_id)">
                                                {{ record.model_name }}
                                            </span>
                                        </div>
                                        <div
                                            class="home-model-code"
                                            style="font-size: 12px; line-height: 1.4">
                                            {{ record.model_code }}
                                        </div>
                                    </template>
                                    <template v-else-if="column.key === 'request_count'">
                                        <span>{{ record.request_count?.toLocaleString() || '-' }}</span>
                                    </template>
                                    <template v-else-if="column.key === 'success_rate'">
                                        <a-tag
                                            :color="
                                                record.success_rate >= 98
                                                    ? 'success'
                                                    : record.success_rate >= 90
                                                      ? 'warning'
                                                      : 'error'
                                            ">
                                            {{ record.success_rate ? record.success_rate.toFixed(1) + '%' : 'N/A' }}
                                        </a-tag>
                                    </template>
                                    <template v-else-if="column.key === 'avg_latency'">
                                        <a-tooltip>
                                            <template #title>
                                                <div style="font-size: 12px">
                                                    <div>P50: {{ formatLatency(record.p50_latency_ms) }}</div>
                                                    <div>P95: {{ formatLatency(record.p95_latency_ms) }}</div>
                                                    <div>P99: {{ formatLatency(record.p99_latency_ms) }}</div>
                                                </div>
                                            </template>
                                            <span
                                                style="
                                                    cursor: help;
                                                    border-bottom: 1px dashed var(--color-text-tertiary);
                                                ">
                                                {{ formatLatency(record.avg_latency_ms) }}
                                            </span>
                                        </a-tooltip>
                                    </template>
                                    <template v-else-if="column.key === 'avg_ttft'">
                                        <a-tooltip>
                                            <template #title>
                                                <div style="font-size: 12px">
                                                    <div>P50: {{ formatLatency(record.p50_ttft_ms) }}</div>
                                                    <div>P95: {{ formatLatency(record.p95_ttft_ms) }}</div>
                                                    <div>P99: {{ formatLatency(record.p99_ttft_ms) }}</div>
                                                </div>
                                            </template>
                                            <span
                                                style="
                                                    cursor: help;
                                                    border-bottom: 1px dashed var(--color-text-tertiary);
                                                ">
                                                {{ formatLatency(record.avg_ttft_ms) }}
                                            </span>
                                        </a-tooltip>
                                    </template>
                                    <template v-else-if="column.key === 'total_tokens'">
                                        <span>{{ record.total_tokens?.toLocaleString() || '-' }}</span>
                                    </template>
                                    <template v-else-if="column.key === 'total_cost'">
                                        <span>{{ record.total_cost ? '¥' + record.total_cost.toFixed(4) : '-' }}</span>
                                    </template>
                                </template>
                            </a-table>
                        </div>
                        <a-empty
                            v-else
                            :description="$t('pages.dashboard.modelRanking.empty')" />
                    </transition>
                </a-card>
            </a-col>
        </a-row>

        <!-- 第四行：策略分布与资产汇总 -->
        <a-row
            :gutter="16"
            class="equal-height-row">
            <a-col
                :xs="24"
                :lg="8"
                style="margin-bottom: 16px; display: flex">
                <a-card
                    :title="$t('pages.dashboard.policyDistribution')"
                    :bordered="false"
                    hoverable
                    style="width: 100%">
                    <transition
                        name="fade-chart"
                        mode="out-in">
                        <div
                            v-if="firstLoading"
                            class="skeleton-container skeleton-pie-container"
                            style="height: 260px">
                            <div class="skeleton-body-pie">
                                <svg
                                    class="skeleton-svg-pie"
                                    viewBox="0 0 200 200"
                                    width="130"
                                    height="130">
                                    <circle
                                        cx="100"
                                        cy="100"
                                        r="70"
                                        fill="none"
                                        stroke="var(--skeleton-line-color)"
                                        stroke-width="18" />
                                    <circle
                                        cx="100"
                                        cy="100"
                                        r="70"
                                        fill="none"
                                        stroke="var(--skeleton-path-color-1)"
                                        stroke-width="18"
                                        stroke-dasharray="140 400"
                                        stroke-dashoffset="0"
                                        class="skeleton-pulse" />
                                    <circle
                                        cx="100"
                                        cy="100"
                                        r="70"
                                        fill="none"
                                        stroke="var(--skeleton-path-color-2)"
                                        stroke-width="18"
                                        stroke-dasharray="90 400"
                                        stroke-dashoffset="-150"
                                        class="skeleton-pulse" />
                                    <circle
                                        cx="100"
                                        cy="100"
                                        r="70"
                                        fill="none"
                                        stroke="var(--skeleton-path-color-3)"
                                        stroke-width="18"
                                        stroke-dasharray="40 400"
                                        stroke-dashoffset="-250"
                                        class="skeleton-pulse" />
                                </svg>
                                <div
                                    class="skeleton-pie-legends"
                                    style="gap: 8px">
                                    <div class="skeleton-legend-row">
                                        <span class="skeleton-legend-dot color-1"></span
                                        ><span
                                            class="skeleton-legend-text"
                                            style="width: 45px"></span>
                                    </div>
                                    <div class="skeleton-legend-row">
                                        <span class="skeleton-legend-dot color-2"></span
                                        ><span
                                            class="skeleton-legend-text"
                                            style="width: 45px"></span>
                                    </div>
                                    <div class="skeleton-legend-row">
                                        <span class="skeleton-legend-dot color-3"></span
                                        ><span
                                            class="skeleton-legend-text"
                                            style="width: 45px"></span>
                                    </div>
                                    <div class="skeleton-legend-row">
                                        <span class="skeleton-legend-dot color-4"></span
                                        ><span
                                            class="skeleton-legend-text"
                                            style="width: 45px"></span>
                                    </div>
                                </div>
                            </div>
                        </div>
                        <x-chart
                            v-else
                            :options="pieChartOptions"
                            height="260" />
                    </transition>
                </a-card>
            </a-col>
            <a-col
                :xs="24"
                :lg="16"
                style="margin-bottom: 16px; display: flex">
                <a-card
                    :title="$t('pages.dashboard.resourceOverview')"
                    :bordered="false"
                    hoverable
                    style="width: 100%">
                    <a-row
                        :gutter="16"
                        style="padding: 10px 0; text-align: center">
                        <a-col
                            :span="8"
                            @click="goTo('/space/list')"
                            style="cursor: pointer">
                            <a-statistic
                                :title="$t('pages.dashboard.spaces')"
                                :value="counts.spaces">
                                <template #prefix
                                    ><database-outlined style="color: var(--color-primary); margin-right: 8px"
                                /></template>
                            </a-statistic>
                        </a-col>
                        <a-col
                            :span="8"
                            @click="goTo('/resource/provider')"
                            style="cursor: pointer">
                            <a-statistic
                                :title="$t('pages.dashboard.providers')"
                                :value="counts.providers">
                                <template #prefix
                                    ><appstore-outlined style="color: var(--color-success); margin-right: 8px"
                                /></template>
                            </a-statistic>
                        </a-col>
                        <a-col
                            :span="8"
                            @click="goTo('/resource/model')"
                            style="cursor: pointer">
                            <a-statistic
                                :title="$t('pages.dashboard.models')"
                                :value="counts.models">
                                <template #prefix
                                    ><cloud-server-outlined style="color: var(--color-chart-5); margin-right: 8px"
                                /></template>
                            </a-statistic>
                        </a-col>
                    </a-row>

                    <!-- 快捷导航 -->
                    <div class="quick-links-section">
                        <h4 class="quick-links-title">{{ $t('pages.dashboard.quickLinks') }}:</h4>
                        <a-row :gutter="16">
                            <a-col
                                :xs="12"
                                :sm="6">
                                <a-card
                                    hoverable
                                    class="quick-link-card"
                                    @click="goTo('/space/list')"
                                    size="small">
                                    <database-outlined style="color: var(--color-primary); font-size: 20px" />
                                    <span class="quick-link-text">{{ $t('pages.dashboard.spaces') }}</span>
                                </a-card>
                            </a-col>
                            <a-col
                                :xs="12"
                                :sm="6">
                                <a-card
                                    hoverable
                                    class="quick-link-card"
                                    @click="goTo('/resource/provider')"
                                    size="small">
                                    <appstore-outlined style="color: var(--color-success); font-size: 20px" />
                                    <span class="quick-link-text">{{ $t('pages.dashboard.providers') }}</span>
                                </a-card>
                            </a-col>
                            <a-col
                                :xs="12"
                                :sm="6">
                                <a-card
                                    hoverable
                                    class="quick-link-card"
                                    @click="goTo('/resource/model')"
                                    size="small">
                                    <cloud-server-outlined style="color: var(--color-chart-5); font-size: 20px" />
                                    <span class="quick-link-text">{{ $t('pages.dashboard.models') }}</span>
                                </a-card>
                            </a-col>
                            <a-col
                                :xs="12"
                                :sm="6">
                                <a-card
                                    hoverable
                                    class="quick-link-card"
                                    @click="goTo('/policy/loadbalance')"
                                    size="small">
                                    <safety-outlined style="color: var(--color-warning); font-size: 20px" />
                                    <span class="quick-link-text">{{ $t('pages.dashboard.policies') }}</span>
                                </a-card>
                            </a-col>
                        </a-row>
                    </div>
                </a-card>
            </a-col>
        </a-row>
    </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/store'
import useUserStore from '@/store/modules/user'
import { config } from '@/config'

import * as echarts from 'echarts'
import {
    DatabaseOutlined,
    AppstoreOutlined,
    CloudServerOutlined,
    SafetyOutlined,
    AlertOutlined,
} from '@ant-design/icons-vue'
import apis from '@/apis'

defineOptions({
    name: 'home',
})

const router = useRouter()
const { t } = useI18n()
const appStore = useAppStore()

const userStore = useUserStore()

const timer = ref(null)
const firstLoading = ref(true)

// WebSocket states
let ws = null
let wsReconnectTimer = null
let wsReconnectDelay = 1000
let wsManualClose = false
const isUsingFallback = ref(false)

// 流量走势分组选择
const trendsGroupBy = ref('')
const trendsTimeRange = ref('1h')

// 模型排行榜排序选择
const rankingSortBy = ref('request_count')
const rankingTimeRange = ref('today')

const counts = reactive({
    spaces: 0,
    providers: 0,
    models: 0,
})

const metrics = reactive({
    qps: 0.0,
    dailyRequests: 0,
    dailyPromptTokens: 0,
    dailyCompletionTokens: 0,
    dailyCachedTokens: 0,
    dailyCacheCreationTokens: 0,
    dailyCost: 0.0,
    avgLatency: 0.0,
    avgTTFT: 0.0,
})

const circuitBreakers = ref([])

const trends = reactive({
    times: [],
    series: [], // 新格式：[{ label, success, failure, total }]
})

const policyCounts = reactive({
    tagging: 0,
    limit: 0,
    invocation: 0,
    route: 0,
    loadbalance: 0,
    circuitBreak: 0,
})

const modelRanking = ref([])

function sendWsConfig() {
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(
            JSON.stringify({
                type: 'config',
                data: {
                    trends_time_range: trendsTimeRange.value,
                    trends_group_by: trendsGroupBy.value || '',
                    model_ranking_sort_by: rankingSortBy.value,
                    model_ranking_time_range: rankingTimeRange.value,
                },
            })
        )
    }
}

// 流量走势分组切换
async function handleTrendsGroupChange() {
    if (ws && ws.readyState === WebSocket.OPEN) {
        sendWsConfig()
    } else {
        try {
            const params = {}
            if (trendsGroupBy.value) params.group_by = trendsGroupBy.value
            if (trendsTimeRange.value) params.time_range = trendsTimeRange.value
            const trendsRes = await apis.dashboard.getTrends(params).catch(() => ({ data: { times: [], series: [] } }))
            if (trendsRes && trendsRes.data) {
                trends.times = trendsRes.data.times || []
                trends.series = trendsRes.data.series || []
            }
        } catch (e) {
            console.error('Failed to fetch trends', e)
        }
    }
}

// 模型排行榜排序/时间范围切换
async function handleRankingSortChange() {
    if (ws && ws.readyState === WebSocket.OPEN) {
        sendWsConfig()
    } else {
        try {
            const rankingRes = await apis.dashboard
                .getModelRanking({
                    sort_by: rankingSortBy.value,
                    time_range: rankingTimeRange.value,
                    limit: 10,
                })
                .catch(() => ({ data: [] }))
            modelRanking.value = rankingRes.data || []
        } catch (e) {
            console.error('Failed to fetch model ranking', e)
        }
    }
}

// 格式化延迟（毫秒转秒）
function formatLatency(ms) {
    if (!ms || ms === 0) return 'N/A'
    return (ms / 1000).toFixed(2) + 's'
}

const POLICY_META = [
    { key: 'tagging', fetch: apis.policy.getTaggingList },
    { key: 'limit', fetch: apis.policy.getLimitList },
    { key: 'invocation', fetch: apis.policy.getInvocationList },
    { key: 'route', fetch: apis.policy.getRouteList },
    { key: 'loadbalance', fetch: apis.policy.getLoadbalanceList },
    { key: 'circuitBreak', fetch: apis.policy.getCircuitBreakList },
]

const COLORS = ['#7c5cfc', '#ffc53d', '#36cfc9', '#52c41a', '#597ef7', '#ff7875']

async function fetchStaticCounts() {
    try {
        const params = { current: 1, pageSize: 1 }
        const [spaceRes, providerRes, modelRes] = await Promise.all([
            apis.space.getSpaceList(params).catch(() => ({ total: 0 })),
            apis.provider.getProviderList(params).catch(() => ({ total: 0 })),
            apis.model.getModelList(params).catch(() => ({ total: 0 })),
        ])

        counts.spaces = spaceRes.total || 0
        counts.providers = providerRes.total || 0
        counts.models = modelRes.total || 0

        const policyResults = await Promise.all(POLICY_META.map((p) => p.fetch(params).catch(() => ({ total: 0 }))))
        policyResults.forEach((res, i) => {
            policyCounts[POLICY_META[i].key] = res.total || 0
        })
    } catch (e) {
        console.error('Failed to fetch static counts', e)
    }
}

async function fetchTelemetryData() {
    try {
        // 1. 获取静态实体数量
        await fetchStaticCounts()

        // 2. 获取概览数据（合并 QPS + Metrics + CircuitBreakers）
        const overviewRes = await apis.dashboard.getOverview().catch(() => ({ data: {} }))
        if (overviewRes && overviewRes.data) {
            metrics.qps = overviewRes.data.qps || 0
            metrics.dailyRequests = overviewRes.data.daily_requests || 0
            metrics.dailyPromptTokens = overviewRes.data.daily_prompt_tokens || 0
            metrics.dailyCompletionTokens = overviewRes.data.daily_completion_tokens || 0
            metrics.dailyCachedTokens = overviewRes.data.daily_cached_tokens || 0
            metrics.dailyCacheCreationTokens = overviewRes.data.daily_cache_creation_tokens || 0
            metrics.dailyCost = overviewRes.data.daily_cost || 0
            metrics.avgLatency = overviewRes.data.avg_latency_ms || 0
            metrics.avgTTFT = overviewRes.data.avg_ttft_ms || 0
            circuitBreakers.value = overviewRes.data.active_circuit_breakers || []
        }

        // 3. 获取流量走势
        const trendsRes = await apis.dashboard
            .getTrends({
                group_by: trendsGroupBy.value || undefined,
                time_range: trendsTimeRange.value,
            })
            .catch(() => ({ data: { times: [], series: [] } }))
        if (trendsRes && trendsRes.data) {
            trends.times = trendsRes.data.times || []
            trends.series = trendsRes.data.series || []
        }

        // 4. 获取模型使用排行
        const rankingRes = await apis.dashboard
            .getModelRanking({
                sort_by: rankingSortBy.value,
                time_range: rankingTimeRange.value,
                limit: 10,
            })
            .catch(() => ({ data: [] }))
        modelRanking.value = rankingRes.data || []
    } catch (e) {
        console.error('Failed to query dashboard metrics', e)
    } finally {
        firstLoading.value = false
    }
}

function handleWsMessage(payload) {
    if (payload.overview) {
        const d = payload.overview
        metrics.qps = d.qps || 0
        metrics.dailyRequests = d.daily_requests || 0
        metrics.dailyPromptTokens = d.daily_prompt_tokens || 0
        metrics.dailyCompletionTokens = d.daily_completion_tokens || 0
        metrics.dailyCachedTokens = d.daily_cached_tokens || 0
        metrics.dailyCacheCreationTokens = d.daily_cache_creation_tokens || 0
        metrics.dailyCost = d.daily_cost || 0
        metrics.avgLatency = d.avg_latency_ms || 0
        metrics.avgTTFT = d.avg_ttft_ms || 0
        circuitBreakers.value = d.active_circuit_breakers || []
    }
    if (payload.trends) {
        trends.times = payload.trends.times || []
        trends.series = payload.trends.series || []
    }
    if (payload.model_ranking) {
        modelRanking.value = payload.model_ranking || []
    }
    firstLoading.value = false
}

function connectWebSocket() {
    if (wsReconnectTimer) {
        clearTimeout(wsReconnectTimer)
        wsReconnectTimer = null
    }

    if (ws) {
        wsManualClose = true
        ws.onopen = null
        ws.onclose = null
        ws.onerror = null
        ws.onmessage = null
        ws.close()
        ws = null
        wsManualClose = false
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    const token = userStore.token
    let apiBasic = config('http.apiBasic') || ''
    if (apiBasic.endsWith('/')) {
        apiBasic = apiBasic.slice(0, -1)
    }
    const wsUrl = `${protocol}//${host}${apiBasic}/api/v1/dashboard/ws?accessToken=${token}`

    ws = new WebSocket(wsUrl)

    ws.onopen = () => {
        wsReconnectDelay = 1000
        isUsingFallback.value = false
        sendWsConfig()
    }

    ws.onmessage = (event) => {
        try {
            const payload = JSON.parse(event.data)
            handleWsMessage(payload)
            // 只有成功收到 WS 数据帧后，我们才停用 HTTP 轮询，确信 WS 可用且有数据
            if (timer.value) {
                clearInterval(timer.value)
                timer.value = null
            }
        } catch (e) {
            console.error('Failed to parse WS data', e)
        }
    }

    ws.onclose = () => {
        if (wsManualClose) return

        // 立即切换/维持 HTTP 轮询兜底
        isUsingFallback.value = true
        if (!timer.value) {
            fetchTelemetryData()
            timer.value = setInterval(fetchTelemetryData, 10000)
        }

        wsReconnectTimer = setTimeout(() => {
            wsReconnectDelay = Math.min(wsReconnectDelay * 2, 30000)
            connectWebSocket()
        }, wsReconnectDelay)
    }

    ws.onerror = () => {
        if (ws) {
            ws.close()
        }
    }
}

onMounted(async () => {
    // 1. 获取静态实体数量
    await fetchStaticCounts()
    // 2. 立即通过 HTTP 拉取初始指标，避免白屏
    await fetchTelemetryData()
    // 3. 启动默认 HTTP 轮询 定时器
    timer.value = setInterval(fetchTelemetryData, 10000)
    // 4. 尝试连接 WebSocket，若连接成功且有数据将被接管并自动关闭 HTTP 轮询定时器
    connectWebSocket()
})

onUnmounted(() => {
    wsManualClose = true
    if (wsReconnectTimer) {
        clearTimeout(wsReconnectTimer)
        wsReconnectTimer = null
    }
    if (ws) {
        ws.onopen = null
        ws.onclose = null
        ws.onerror = null
        ws.onmessage = null
        ws.close()
        ws = null
    }
    if (timer.value) {
        clearInterval(timer.value)
        timer.value = null
    }
})

// 计算双线渐变面积趋势图配置
const trendsChartOptions = computed(() => {
    const isDark = appStore.config.theme === 'dark'

    // 判断是否为分组模式
    const isGroupMode = trends.series.length > 1

    if (isGroupMode) {
        // 分组模式：每条 series 一条 total 折线
        const seriesColors = [
            '#7c5cfc',
            '#52c41a',
            '#ff7875',
            '#ffc53d',
            '#36cfc9',
            '#597ef7',
            '#f759ab',
            '#ffa940',
            '#9578ff',
            '#73d13d',
        ]

        return {
            tooltip: {
                trigger: 'axis',
                axisPointer: { type: 'cross' },
            },
            legend: {
                data: trends.series.map((s) => s.label),
                textStyle: { color: isDark ? 'rgba(255, 255, 255, 0.65)' : '#333' },
            },
            grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
            xAxis: {
                type: 'category',
                boundaryGap: false,
                data: trends.times,
                axisLabel: { color: isDark ? 'rgba(255, 255, 255, 0.45)' : '#666' },
            },
            yAxis: [
                {
                    type: 'value',
                    name: t('pages.dashboard.trends.requests'),
                    minInterval: 1,
                    axisLabel: { color: isDark ? 'rgba(255, 255, 255, 0.45)' : '#666' },
                    splitLine: { lineStyle: { color: isDark ? 'rgba(255, 255, 255, 0.06)' : 'rgba(0, 0, 0, 0.06)' } },
                },
            ],
            series: trends.series.map((s, index) => ({
                name: s.label,
                type: 'line',
                smooth: true,
                showSymbol: false,
                itemStyle: { color: seriesColors[index % seriesColors.length] },
                data: s.total || [],
            })),
        }
    } else {
        // 全局模式：保持原来的双折线（成功+失败）+ 成功率样式
        const successData = trends.series[0]?.success || []
        const failureData = trends.series[0]?.failure || []

        const successRates = []
        for (let i = 0; i < successData.length; i++) {
            const total = successData[i] + failureData[i]
            if (total === 0) {
                successRates.push(100.0)
            } else {
                successRates.push(parseFloat(((successData[i] / total) * 100).toFixed(1)))
            }
        }

        return {
            tooltip: {
                trigger: 'axis',
                axisPointer: { type: 'cross' },
            },
            legend: {
                data: [
                    t('pages.dashboard.trends.success_requests'),
                    t('pages.dashboard.trends.failed_requests'),
                    t('pages.dashboard.trends.success_rate'),
                ],
                textStyle: { color: isDark ? 'rgba(255, 255, 255, 0.65)' : '#333' },
            },
            grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
            xAxis: {
                type: 'category',
                boundaryGap: false,
                data: trends.times,
                axisLabel: { color: isDark ? 'rgba(255, 255, 255, 0.45)' : '#666' },
            },
            yAxis: [
                {
                    type: 'value',
                    name: t('pages.dashboard.trends.requests'),
                    minInterval: 1,
                    axisLabel: { color: isDark ? 'rgba(255, 255, 255, 0.45)' : '#666' },
                    splitLine: { lineStyle: { color: isDark ? 'rgba(255, 255, 255, 0.06)' : 'rgba(0, 0, 0, 0.06)' } },
                },
                {
                    type: 'value',
                    name: t('pages.dashboard.trends.success_rate'),
                    min: 0,
                    max: 100,
                    axisLabel: {
                        formatter: '{value} %',
                        color: isDark ? 'rgba(255, 255, 255, 0.45)' : '#666',
                    },
                    splitLine: { show: false },
                },
            ],
            series: [
                {
                    name: t('pages.dashboard.trends.success_requests'),
                    type: 'line',
                    smooth: true,
                    showSymbol: false,
                    areaStyle: {
                        color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                            { offset: 0, color: 'rgba(82, 196, 26, 0.4)' },
                            { offset: 1, color: 'rgba(82, 196, 26, 0.02)' },
                        ]),
                    },
                    itemStyle: { color: '#52c41a' },
                    data: successData,
                },
                {
                    name: t('pages.dashboard.trends.failed_requests'),
                    type: 'line',
                    smooth: true,
                    showSymbol: false,
                    areaStyle: {
                        color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                            { offset: 0, color: 'rgba(255, 77, 79, 0.4)' },
                            { offset: 1, color: 'rgba(255, 77, 79, 0.02)' },
                        ]),
                    },
                    itemStyle: { color: '#ff7875' },
                    data: failureData,
                },
                {
                    name: t('pages.dashboard.trends.success_rate'),
                    type: 'line',
                    yAxisIndex: 1,
                    smooth: true,
                    showSymbol: true,
                    symbolSize: 6,
                    itemStyle: { color: '#7c5cfc' },
                    lineStyle: { width: 3, type: 'dashed' },
                    data: successRates,
                },
            ],
        }
    }
})

// 计算 Token 占比饼图配置
const tokenChartOptions = computed(() => {
    const isDark = appStore.config.theme === 'dark'
    const input = metrics.dailyPromptTokens
    const output = metrics.dailyCompletionTokens
    const cached = metrics.dailyCachedTokens
    const cacheCreation = metrics.dailyCacheCreationTokens
    const inputNonCache = Math.max(0, input - cached - cacheCreation)
    const total = inputNonCache + output + cached + cacheCreation

    return {
        title: {
            text: total.toLocaleString(),
            subtext: t('pages.dashboard.metrics.tokens'),
            left: 'center',
            top: '38%',
            textStyle: {
                fontSize: 20,
                fontWeight: 'bold',
                color: isDark ? 'rgba(255, 255, 255, 0.85)' : '#000',
            },
            subtextStyle: {
                fontSize: 12,
                color: isDark ? 'rgba(255, 255, 255, 0.45)' : 'rgba(0,0,0,0.45)',
            },
        },
        tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
        legend: {
            orient: 'vertical',
            left: '6%',
            bottom: '25',
            textStyle: { color: isDark ? 'var(--color-text-secondary)' : '#333' },
        },
        series: [
            {
                type: 'pie',
                radius: ['52%', '72%'],
                center: ['50%', '46%'],
                avoidLabelOverlap: false,
                itemStyle: {
                    borderRadius: 6,
                    borderColor: isDark ? '#141722' : '#fff',
                    borderWidth: 2,
                },
                label: {
                    show: true,
                    formatter: '{b}: {c} ({d}%)',
                    fontSize: 11,
                    color: isDark ? 'rgba(255, 255, 255, 0.85)' : '#333',
                },
                data: [
                    { value: inputNonCache, name: t('pages.dashboard.tokens.input'), itemStyle: { color: '#7c5cfc' } },
                    {
                        value: output,
                        name: t('pages.dashboard.tokens.output'),
                        itemStyle: { color: '#b37feb' },
                    },
                    {
                        value: cached,
                        name: t('pages.dashboard.tokens.cached'),
                        itemStyle: { color: '#73d13d' },
                    },
                    {
                        value: cacheCreation,
                        name: t('pages.dashboard.tokens.cache_creation'),
                        itemStyle: { color: '#5cdbd3' },
                    },
                ],
            },
        ],
    }
})

// 策略分布饼图
const pieChartOptions = computed(() => {
    const isDark = appStore.config.theme === 'dark'
    return {
        tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
        legend: {
            orient: 'vertical',
            left: '6%',
            top: 'middle',
            textStyle: { color: isDark ? 'rgba(255, 255, 255, 0.65)' : '#333' },
        },
        series: [
            {
                type: 'pie',
                radius: ['45%', '75%'],
                center: ['62%', '50%'],
                avoidLabelOverlap: false,
                itemStyle: {
                    borderRadius: 6,
                    borderColor: isDark ? '#141722' : '#fff',
                    borderWidth: 2,
                },
                label: { show: false },
                emphasis: { label: { show: true, fontSize: 13, fontWeight: 'bold' } },
                data: POLICY_META.map((p, i) => ({
                    name: t(`pages.dashboard.policies.${p.key}`),
                    value: policyCounts[p.key],
                    itemStyle: { color: COLORS[i] },
                })),
            },
        ],
    }
})

// 模型性能排行榜表格列配置
const columns = computed(() => [
    {
        title: t('pages.dashboard.modelRanking.columns.model'),
        dataIndex: 'model_name',
        key: 'model_name',
        fixed: 'left',
        width: 200,
    },
    {
        title: t('pages.dashboard.modelRanking.columns.requests'),
        dataIndex: 'request_count',
        key: 'request_count',
        sorter: (a, b) => (a.request_count || 0) - (b.request_count || 0),
    },
    {
        title: t('pages.dashboard.modelRanking.columns.successRate'),
        dataIndex: 'success_rate',
        key: 'success_rate',
        sorter: (a, b) => (a.success_rate || 0) - (b.success_rate || 0),
    },
    {
        title: t('pages.dashboard.modelRanking.columns.avgLatency'),
        dataIndex: 'avg_latency_ms',
        key: 'avg_latency',
        sorter: (a, b) => (a.avg_latency_ms || 0) - (b.avg_latency_ms || 0),
    },
    {
        title: t('pages.dashboard.modelRanking.columns.avgTTFT'),
        dataIndex: 'avg_ttft_ms',
        key: 'avg_ttft',
        sorter: (a, b) => (a.avg_ttft_ms || 0) - (b.avg_ttft_ms || 0),
    },
    {
        title: t('pages.dashboard.modelRanking.columns.tokens'),
        dataIndex: 'total_tokens',
        key: 'total_tokens',
        sorter: (a, b) => (a.total_tokens || 0) - (b.total_tokens || 0),
    },
    {
        title: t('pages.dashboard.modelRanking.columns.estimatedCost'),
        dataIndex: 'total_cost',
        key: 'total_cost',
        sorter: (a, b) => (a.total_cost || 0) - (b.total_cost || 0),
    },
])

function goTo(path) {
    router.push(path)
}

function goToModel(modelId) {
    router.push({ name: 'modelDetail', params: { id: modelId } })
}
</script>

<style lang="less" scoped>
.dashboard {
    padding: 0;
    position: relative;

    // 签名元素：微妙的紫色流动光效
    &::before {
        content: '';
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        height: 200px;
        background: radial-gradient(ellipse at 30% 50%, rgba(124, 92, 252, 0.06) 0%, transparent 70%);
        pointer-events: none;
        z-index: 0;
        animation: breathe 8s ease-in-out infinite;
    }

    // 让内容在光效之上
    > * {
        position: relative;
        z-index: 1;
    }
}

@keyframes breathe {
    0%,
    100% {
        opacity: 0.5;
        transform: translateX(0);
    }
    50% {
        opacity: 1;
        transform: translateX(10px);
    }
}

.equal-height-row {
    display: flex;
    flex-wrap: wrap;
}

// 遥测卡片现代发光渐变样式
.telemetry-card {
    border-radius: 12px;
    padding: 2px;
    color: #fff;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
    transition: all 0.4s cubic-bezier(0.25, 0.8, 0.25, 1);
    position: relative;
    overflow: hidden;

    &::after {
        content: '';
        position: absolute;
        top: 0;
        left: -100%;
        width: 100%;
        height: 100%;
        background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.15), transparent);
        transition: left 0.6s ease;
    }

    &:hover {
        transform: translateY(-6px);
        box-shadow: 0 16px 32px rgba(124, 92, 252, 0.2);
    }

    &:hover::after {
        left: 100%;
    }
}

.telemetry-card--blue {
    background: linear-gradient(135deg, #7c5cfc 0%, #9578ff 100%);
}

.telemetry-card--green {
    background: linear-gradient(135deg, #36cfc9 0%, #5cdbd3 100%);
}

.telemetry-card--purple {
    background: linear-gradient(135deg, #6347e0 0%, #9578ff 100%);
}

.telemetry-card--orange {
    background: linear-gradient(135deg, #ffa940 0%, #ffc53d 100%);
}

.telemetry-card--cyan {
    background: linear-gradient(135deg, #13c2c2 0%, #36cfc9 100%);
}

.telemetry-card--magenta {
    background: linear-gradient(135deg, #f759ab 0%, #ff85c0 100%);
}

.telemetry-title {
    font-size: 13px;
    opacity: 0.9;
    font-weight: 500;
    margin-bottom: 6px;
    display: flex;
    align-items: center;
}

.telemetry-value {
    font-size: 28px;
    font-weight: 700;
    font-feature-settings: 'tnum'; /* 等宽数字，对齐更整齐 */
    letter-spacing: -0.02em; /* 标题收紧，更精致 */
    line-height: 1.1;
    margin-bottom: 6px;
}

.telemetry-unit {
    font-size: 14px;
    font-weight: normal;
    margin-left: 4px;
}

.telemetry-footer {
    font-size: 11px;
    opacity: 0.75;
    border-top: 1px solid rgba(255, 255, 255, 0.2);
    padding-top: 6px;
    margin-top: 4px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

// 脉冲呼吸灯微动画
.pulse-indicator {
    display: inline-block;
    width: 8px;
    height: 8px;
    background-color: #fff;
    border-radius: 50%;
    margin-left: 8px;
    box-shadow: 0 0 0 0 rgba(255, 255, 255, 0.7);
    animation: pulse 1.6s infinite;
}

@keyframes pulse {
    0% {
        transform: scale(0.95);
        box-shadow: 0 0 0 0 rgba(255, 255, 255, 0.7);
    }
    70% {
        transform: scale(1);
        box-shadow: 0 0 0 8px rgba(255, 255, 255, 0);
    }
    100% {
        transform: scale(0.95);
        box-shadow: 0 0 0 0 rgba(255, 255, 255, 0);
    }
}

// 快捷导航区域与标题
.quick-links-section {
    border-top: 1px solid var(--color-border-secondary);
    padding-top: 16px;
    margin-top: 12px;
}

.quick-links-title {
    margin-bottom: 12px;
    font-weight: 500;
    font-size: 13px;
    color: var(--color-text-tertiary);
}

// 快捷连接卡片
.quick-link-card {
    text-align: center;
    cursor: pointer;
    transition: all 0.3s;
    margin-top: 8px;
    border: 1px dashed var(--color-border);
    border-radius: 8px;
    background: var(--color-bg-container);

    &:hover {
        box-shadow: 0 4px 12px var(--color-primary-bg);
        border-color: var(--color-primary);
        transform: translateY(-2px);
    }
}

.quick-link-text {
    display: block;
    margin-top: 4px;
    font-size: 12px;
    color: var(--color-text-secondary);
}

// 模型链接
.model-link {
    cursor: pointer;
    text-decoration: none;
    transition: opacity 0.2s;

    &:hover {
        opacity: 0.8;
    }
}

// 骨架屏基础样式与动画
:root {
    --skeleton-bg: #f9f9f9;
    --skeleton-shimmer-start: #f2f2f2;
    --skeleton-shimmer-end: #e6e6e6;
    --skeleton-line-color: #ededed;
    --skeleton-path-color-1: rgba(124, 92, 252, 0.08);
    --skeleton-path-color-2: rgba(82, 196, 26, 0.08);
    --skeleton-path-color-3: rgba(255, 77, 79, 0.08);
}

[data-theme='dark'] {
    --skeleton-bg: #1f1f1f;
    --skeleton-shimmer-start: #2a2a2a;
    --skeleton-shimmer-end: #333333;
    --skeleton-line-color: #2e2e2e;
    --skeleton-path-color-1: rgba(124, 92, 252, 0.15);
    --skeleton-path-color-2: rgba(82, 196, 26, 0.15);
    --skeleton-path-color-3: rgba(255, 77, 79, 0.15);
}

@keyframes skeleton-shimmer {
    0% {
        background-position: -200% 0;
    }
    100% {
        background-position: 200% 0;
    }
}

.skeleton-container {
    width: 100%;
    background: transparent;
    display: flex;
    flex-direction: column;
    box-sizing: border-box;
    padding: 10px 0;
}

.skeleton-line-container {
    height: 340px;
    justify-content: space-between;
}

.skeleton-pie-container {
    height: 340px;
    justify-content: center;
    align-items: center;
}

.skeleton-header {
    display: flex;
    justify-content: center;
    gap: 20px;
    margin-bottom: 15px;
}

.skeleton-legend-item {
    width: 80px;
    height: 12px;
    border-radius: 4px;
    background: linear-gradient(
        90deg,
        var(--skeleton-shimmer-start) 25%,
        var(--skeleton-shimmer-end) 37%,
        var(--skeleton-shimmer-start) 63%
    );
    background-size: 400% 100%;
    animation: skeleton-shimmer 1.4s ease infinite;
}

.skeleton-body {
    flex: 1;
    position: relative;
    overflow: hidden;
}

.skeleton-body-pie {
    display: flex;
    align-items: center;
    justify-content: space-around;
    width: 100%;
    padding: 0 10px;
}

.skeleton-svg-line {
    width: 100%;
    height: 100%;
}

.skeleton-block {
    background: linear-gradient(
        90deg,
        var(--skeleton-shimmer-start) 25%,
        var(--skeleton-shimmer-end) 37%,
        var(--skeleton-shimmer-start) 63%
    );
    background-size: 400% 100%;
    animation: skeleton-shimmer 1.4s ease infinite;
}

.block-long {
    width: 140px;
    height: 16px;
    border-radius: 4px;
    display: inline-block;
}
.block-med {
    width: 80px;
    height: 16px;
    border-radius: 4px;
    display: inline-block;
}
.block-short {
    width: 40px;
    height: 16px;
    border-radius: 4px;
    display: inline-block;
}

.skeleton-pulse {
    animation: skeleton-pulse 2s ease-in-out infinite;
}

@keyframes skeleton-pulse {
    0%,
    100% {
        opacity: 0.6;
    }
    50% {
        opacity: 1;
    }
}

.skeleton-pie-legends {
    display: flex;
    flex-direction: column;
    gap: 12px;
}

.skeleton-legend-row {
    display: flex;
    align-items: center;
    gap: 8px;
}

.skeleton-legend-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--skeleton-line-color);
}

.skeleton-legend-dot.color-1 {
    background: #7c5cfc;
}
.skeleton-legend-dot.color-2 {
    background: #b37feb;
}
.skeleton-legend-dot.color-3 {
    background: #73d13d;
}
.skeleton-legend-dot.color-4 {
    background: #5cdbd3;
}

.skeleton-legend-text {
    width: 60px;
    height: 12px;
    border-radius: 3px;
    background: linear-gradient(
        90deg,
        var(--skeleton-shimmer-start) 25%,
        var(--skeleton-shimmer-end) 37%,
        var(--skeleton-shimmer-start) 63%
    );
    background-size: 400% 100%;
    animation: skeleton-shimmer 1.4s ease infinite;
}

// 表格骨架屏样式
.skeleton-table-container {
    padding: 8px;
}

.skeleton-table-header {
    display: flex;
    padding: 12px 16px;
    background: var(--skeleton-shimmer-start);
    border-radius: 6px 6px 0 0;
    font-weight: bold;
}

.skeleton-table-row {
    display: flex;
    padding: 16px;
    border-bottom: 1px solid var(--skeleton-line-color);
    align-items: center;
}

.skeleton-table-col {
    flex: 1;
    display: flex;
    align-items: center;
}

.skeleton-table-col.col-1 {
    flex: 2;
} // 模型名字列更宽

// 淡入淡出过渡动画
.fade-chart-enter-active,
.fade-chart-leave-active {
    transition: opacity 0.35s ease;
}

.fade-chart-enter-from,
.fade-chart-leave-to {
    opacity: 0;
}

.home-model-code {
    color: rgba(0, 0, 0, 0.45);
}

[data-theme='dark'] .home-model-code {
    color: rgba(255, 255, 255, 0.45);
}
</style>
