<template>
    <div class="dashboard">
        <!-- 核心报警：熔断隔离横幅 -->
        <a-alert
            v-if="circuitBreakers.length > 0"
            class="dashboard-health-alert dashboard-health-alert--error"
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
                        @click="goTo('/space/model')">
                        {{ $t('pages.dashboard.alert.action') }}
                    </a-button>
                </div>
            </template>
        </a-alert>
        <a-alert
            v-else
            class="dashboard-health-alert dashboard-health-alert--success"
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
                        {{ formatTokens(metrics.dailyPromptTokens + metrics.dailyCompletionTokens) }}
                        <span class="telemetry-unit">{{ $t('pages.dashboard.units.tokens') }}</span>
                    </div>
                    <div class="telemetry-footer">
                        Input: {{ formatTokens(metrics.dailyPromptTokens) }} | Output:
                        {{ formatTokens(metrics.dailyCompletionTokens) }}
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
                    class="dashboard-panel"
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
                    class="dashboard-panel"
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
                    class="dashboard-panel dashboard-ranking-panel"
                    :title="$t('pages.dashboard.modelRanking.title')"
                    :bordered="false"
                    hoverable>
                    <template #extra>
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
                                class="dashboard-ranking-table"
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
                                        <span>{{ formatTokens(record.total_tokens) }}</span>
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
                    class="dashboard-panel"
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
                    class="dashboard-panel dashboard-resource-panel"
                    :title="$t('pages.dashboard.resourceOverview')"
                    :bordered="false"
                    hoverable
                    style="width: 100%">
                    <a-row
                        :gutter="16"
                        style="padding: 10px 0; text-align: center">
                        <a-col
                            class="resource-stat-item"
                            :span="6"
                            @click="goTo('/system/tenant')"
                            style="cursor: pointer">
                            <a-statistic
                                :title="$t('pages.dashboard.tenants')"
                                :value="counts.tenants">
                                <template #prefix
                                    ><team-outlined style="color: var(--color-primary); margin-right: 8px"
                                /></template>
                            </a-statistic>
                        </a-col>
                        <a-col
                            class="resource-stat-item"
                            :span="6"
                            @click="goTo('/space/provider')"
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
                            class="resource-stat-item"
                            :span="6"
                            @click="goTo('/space/model')"
                            style="cursor: pointer">
                            <a-statistic
                                :title="$t('pages.dashboard.models')"
                                :value="counts.models">
                                <template #prefix
                                    ><cloud-server-outlined style="color: var(--color-chart-5); margin-right: 8px"
                                /></template>
                            </a-statistic>
                        </a-col>
                        <a-col
                            class="resource-stat-item"
                            :span="6">
                            <a-statistic
                                :title="$t('pages.dashboard.endpoints')"
                                :value="counts.endpoints">
                                <template #prefix
                                    ><api-outlined style="color: var(--color-warning); margin-right: 8px"
                                /></template>
                            </a-statistic>
                        </a-col>
                    </a-row>

                    <!-- 快捷导航 -->
                    <div class="quick-links-section">
                        <h4 class="quick-links-title">{{ $t('pages.dashboard.quickLinks') }}:</h4>
                        <a-row :gutter="16">
                            <a-col
                                v-for="link in policyQuickLinks"
                                :key="link.path"
                                :xs="12"
                                :sm="8"
                                :lg="4">
                                <a-card
                                    hoverable
                                    class="quick-link-card"
                                    @click="goTo(link.path)"
                                    size="small">
                                    <component
                                        :is="link.icon"
                                        :style="{ color: link.color, fontSize: '20px' }" />
                                    <span class="quick-link-text">{{ $t(link.label) }}</span>
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
    TeamOutlined,
    AppstoreOutlined,
    CloudServerOutlined,
    ApiOutlined,
    HighlightOutlined,
    DashboardOutlined,
    SafetyCertificateOutlined,
    BranchesOutlined,
    SlidersOutlined,
    ThunderboltOutlined,
    AlertOutlined,
} from '@ant-design/icons-vue'
import apis from '@/apis'
import { formatTokens } from '@/utils/util'

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

// 模型排行榜：固定按请求数排序，表格列可本地二次排序
const rankingSortBy = 'request_count'
const rankingTimeRange = ref('today')

const counts = reactive({
    tenants: 0,
    providers: 0,
    models: 0,
    endpoints: 0,
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
                    model_ranking_sort_by: rankingSortBy,
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
                    sort_by: rankingSortBy,
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

const COLORS = ['#2f8cff', '#23c7b7', '#ffb020', '#8b5cf6', '#ff4d5e', '#6ee7d8']

const policyQuickLinks = [
    {
        path: '/policy/tagging',
        label: 'pages.dashboard.policies.tagging',
        icon: HighlightOutlined,
        color: '#2f8cff',
    },
    {
        path: '/policy/limit',
        label: 'pages.dashboard.policies.limit',
        icon: DashboardOutlined,
        color: '#ffb020',
    },
    {
        path: '/policy/invocation',
        label: 'pages.dashboard.policies.invocation',
        icon: SafetyCertificateOutlined,
        color: '#23c7b7',
    },
    {
        path: '/policy/route',
        label: 'pages.dashboard.policies.route',
        icon: BranchesOutlined,
        color: '#8b5cf6',
    },
    {
        path: '/policy/loadbalance',
        label: 'pages.dashboard.policies.loadbalance',
        icon: SlidersOutlined,
        color: '#6ee7d8',
    },
    {
        path: '/policy/circuit-break',
        label: 'pages.dashboard.policies.circuitBreak',
        icon: ThunderboltOutlined,
        color: '#ff4d5e',
    },
]

async function fetchStaticCounts() {
    try {
        const params = { current: 1, pageSize: 1 }
        const [tenantRes, providerRes, modelRes, endpointRes] = await Promise.all([
            apis.tenant.getList(params).catch(() => ({ total: 0 })),
            apis.provider.getProviderList(params).catch(() => ({ total: 0 })),
            apis.model.getModelList(params).catch(() => ({ total: 0 })),
            apis.endpoint.getEndpointList(params).catch(() => ({ total: 0 })),
        ])

        counts.tenants = tenantRes.total || 0
        counts.providers = providerRes.total || 0
        counts.models = modelRes.total || 0
        counts.endpoints = endpointRes.total || 0

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
    const chartTextColor = isDark ? 'rgba(231, 236, 246, 0.68)' : 'rgba(31, 41, 55, 0.68)'
    const chartTooltip = {
        backgroundColor: isDark ? 'rgba(18, 24, 38, 0.96)' : 'rgba(255, 255, 255, 0.98)',
        borderColor: isDark ? 'rgba(126, 145, 178, 0.18)' : 'rgba(15, 23, 42, 0.1)',
        textStyle: { color: isDark ? '#e7ecf6' : '#1f2937' },
        extraCssText: 'box-shadow: 0 12px 32px rgba(15, 23, 42, 0.18); border-radius: 8px;',
    }
    const splitLineStyle = {
        lineStyle: { color: isDark ? 'rgba(126, 145, 178, 0.11)' : 'rgba(15, 23, 42, 0.08)' },
    }

    // 判断是否为分组模式
    const isGroupMode = trends.series.length > 1

    if (isGroupMode) {
        // 分组模式：每条 series 一条 total 折线
        const seriesColors = [
            '#2f8cff',
            '#23c7b7',
            '#ff4d5e',
            '#ffb020',
            '#8b5cf6',
            '#6ee7d8',
            '#f472b6',
            '#7bb8ff',
            '#a78bfa',
            '#58d68d',
        ]

        return {
            tooltip: {
                ...chartTooltip,
                trigger: 'axis',
                axisPointer: { type: 'cross' },
            },
            legend: {
                data: trends.series.map((s) => s.label),
                textStyle: { color: chartTextColor },
            },
            grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
            xAxis: {
                type: 'category',
                boundaryGap: false,
                data: trends.times,
                axisLabel: { color: chartTextColor },
                axisLine: { lineStyle: { color: isDark ? 'rgba(126, 145, 178, 0.18)' : 'rgba(15, 23, 42, 0.12)' } },
            },
            yAxis: [
                {
                    type: 'value',
                    name: t('pages.dashboard.trends.requests'),
                    minInterval: 1,
                    axisLabel: { color: chartTextColor },
                    splitLine: splitLineStyle,
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
                ...chartTooltip,
                trigger: 'axis',
                axisPointer: { type: 'cross' },
            },
            legend: {
                data: [
                    t('pages.dashboard.trends.success_requests'),
                    t('pages.dashboard.trends.failed_requests'),
                    t('pages.dashboard.trends.success_rate'),
                ],
                textStyle: { color: chartTextColor },
            },
            grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
            xAxis: {
                type: 'category',
                boundaryGap: false,
                data: trends.times,
                axisLabel: { color: chartTextColor },
                axisLine: { lineStyle: { color: isDark ? 'rgba(126, 145, 178, 0.18)' : 'rgba(15, 23, 42, 0.12)' } },
            },
            yAxis: [
                {
                    type: 'value',
                    name: t('pages.dashboard.trends.requests'),
                    minInterval: 1,
                    axisLabel: { color: chartTextColor },
                    splitLine: splitLineStyle,
                },
                {
                    type: 'value',
                    name: t('pages.dashboard.trends.success_rate'),
                    min: 0,
                    max: 100,
                    axisLabel: {
                        formatter: '{value} %',
                        color: chartTextColor,
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
                            { offset: 0, color: 'rgba(35, 199, 183, 0.32)' },
                            { offset: 1, color: 'rgba(35, 199, 183, 0.02)' },
                        ]),
                    },
                    itemStyle: { color: '#23c7b7' },
                    data: successData,
                },
                {
                    name: t('pages.dashboard.trends.failed_requests'),
                    type: 'line',
                    smooth: true,
                    showSymbol: false,
                    areaStyle: {
                        color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                            { offset: 0, color: 'rgba(255, 77, 94, 0.3)' },
                            { offset: 1, color: 'rgba(255, 77, 94, 0.02)' },
                        ]),
                    },
                    itemStyle: { color: '#ff4d5e' },
                    data: failureData,
                },
                {
                    name: t('pages.dashboard.trends.success_rate'),
                    type: 'line',
                    yAxisIndex: 1,
                    smooth: true,
                    showSymbol: true,
                    symbolSize: 6,
                    itemStyle: { color: '#8b5cf6' },
                    lineStyle: { width: 2, type: 'dashed' },
                    data: successRates,
                },
            ],
        }
    }
})

// 计算 Token 占比饼图配置
const tokenChartOptions = computed(() => {
    const isDark = appStore.config.theme === 'dark'
    const chartTextColor = isDark ? 'rgba(231, 236, 246, 0.72)' : 'rgba(31, 41, 55, 0.72)'
    const chartTooltip = {
        backgroundColor: isDark ? 'rgba(18, 24, 38, 0.96)' : 'rgba(255, 255, 255, 0.98)',
        borderColor: isDark ? 'rgba(126, 145, 178, 0.18)' : 'rgba(15, 23, 42, 0.1)',
        textStyle: { color: isDark ? '#e7ecf6' : '#1f2937' },
        extraCssText: 'box-shadow: 0 12px 32px rgba(15, 23, 42, 0.18); border-radius: 8px;',
    }
    const input = metrics.dailyPromptTokens
    const output = metrics.dailyCompletionTokens
    const cached = metrics.dailyCachedTokens
    const cacheCreation = metrics.dailyCacheCreationTokens
    const inputNonCache = Math.max(0, input - cached - cacheCreation)
    const total = inputNonCache + output + cached + cacheCreation

    return {
        title: {
            text: formatTokens(total),
            subtext: `${t('pages.dashboard.metrics.tokens')} (k)`,
            left: 'center',
            top: '38%',
            textStyle: {
                fontSize: 20,
                fontWeight: 'bold',
                color: isDark ? '#e7ecf6' : '#111827',
            },
            subtextStyle: {
                fontSize: 12,
                color: chartTextColor,
            },
        },
        tooltip: {
            ...chartTooltip,
            trigger: 'item',
            formatter: (params) => {
                const marker = params.marker || ''
                const val = formatTokens(params.value)
                const percent = params.percent !== undefined ? params.percent : 0
                return `${marker} ${params.name}: <b>${val} k</b> (${percent}%)`
            },
        },
        legend: {
            orient: 'vertical',
            left: '6%',
            bottom: '25',
            textStyle: { color: chartTextColor },
            data: [
                t('pages.dashboard.tokens.input_non_cache'),
                t('pages.dashboard.tokens.cached'),
                t('pages.dashboard.tokens.cache_creation'),
                t('pages.dashboard.tokens.output'),
            ],
        },
        series: [
            {
                name: 'Token Category',
                type: 'pie',
                radius: ['38%', '52%'],
                center: ['50%', '46%'],
                avoidLabelOverlap: false,
                itemStyle: {
                    borderRadius: 4,
                    borderColor: isDark ? '#111827' : '#fff',
                    borderWidth: 2,
                },
                label: {
                    show: false,
                },
                labelLine: {
                    show: false,
                },
                data: [
                    {
                        value: inputNonCache + cached + cacheCreation,
                        name: t('pages.dashboard.tokens.input'),
                        itemStyle: { color: '#2f8cff' },
                    },
                    {
                        value: output,
                        name: t('pages.dashboard.tokens.output'),
                        itemStyle: { color: '#8b5cf6' },
                    },
                ],
            },
            {
                name: 'Token Detail',
                type: 'pie',
                radius: ['58%', '72%'],
                center: ['50%', '46%'],
                avoidLabelOverlap: true,
                itemStyle: {
                    borderRadius: 6,
                    borderColor: isDark ? '#111827' : '#fff',
                    borderWidth: 2,
                },
                label: {
                    show: true,
                    formatter: (params) => {
                        return `${params.name}: ${formatTokens(params.value)} k (${params.percent}%)`
                    },
                    fontSize: 11,
                    color: chartTextColor,
                },
                data: [
                    {
                        value: inputNonCache,
                        name: t('pages.dashboard.tokens.input_non_cache'),
                        itemStyle: { color: '#60a5fa' },
                    },
                    {
                        value: cached,
                        name: t('pages.dashboard.tokens.cached'),
                        itemStyle: { color: '#34d399' },
                    },
                    {
                        value: cacheCreation,
                        name: t('pages.dashboard.tokens.cache_creation'),
                        itemStyle: { color: '#2dd4bf' },
                    },
                    {
                        value: output,
                        name: t('pages.dashboard.tokens.output'),
                        itemStyle: { color: '#a78bfa' },
                    },
                ],
            },
        ],
    }
})

// 策略分布饼图
const pieChartOptions = computed(() => {
    const isDark = appStore.config.theme === 'dark'
    const chartTextColor = isDark ? 'rgba(231, 236, 246, 0.72)' : 'rgba(31, 41, 55, 0.72)'
    return {
        tooltip: {
            trigger: 'item',
            formatter: '{b}: {c} ({d}%)',
            backgroundColor: isDark ? 'rgba(18, 24, 38, 0.96)' : 'rgba(255, 255, 255, 0.98)',
            borderColor: isDark ? 'rgba(126, 145, 178, 0.18)' : 'rgba(15, 23, 42, 0.1)',
            textStyle: { color: isDark ? '#e7ecf6' : '#1f2937' },
            extraCssText: 'box-shadow: 0 12px 32px rgba(15, 23, 42, 0.18); border-radius: 8px;',
        },
        legend: {
            orient: 'vertical',
            left: '6%',
            top: 'middle',
            textStyle: { color: chartTextColor },
        },
        series: [
            {
                type: 'pie',
                radius: ['45%', '75%'],
                center: ['62%', '50%'],
                avoidLabelOverlap: false,
                itemStyle: {
                    borderRadius: 6,
                    borderColor: isDark ? '#111827' : '#fff',
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
    --dashboard-panel: #ffffff;
    --dashboard-panel-soft: #f8fafc;
    --dashboard-panel-elevated: #ffffff;
    --dashboard-border: rgba(15, 23, 42, 0.08);
    --dashboard-border-strong: rgba(15, 23, 42, 0.14);
    --dashboard-text: #111827;
    --dashboard-text-secondary: rgba(31, 41, 55, 0.68);
    --dashboard-text-tertiary: rgba(31, 41, 55, 0.48);
    --dashboard-shadow: 0 12px 32px rgba(15, 23, 42, 0.07);
    --dashboard-shadow-hover: 0 18px 42px rgba(15, 23, 42, 0.1);
    --dashboard-control-bg: #f8fafc;
    --dashboard-row-hover: rgba(47, 140, 255, 0.06);
    --dashboard-accent: #2f8cff;
    padding: 0;
    position: relative;

    &::before {
        content: '';
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        height: 200px;
        background:
            radial-gradient(ellipse at 24% 18%, rgba(47, 140, 255, 0.08) 0%, transparent 58%),
            radial-gradient(ellipse at 84% 10%, rgba(35, 199, 183, 0.06) 0%, transparent 54%);
        pointer-events: none;
        z-index: 0;
        opacity: 0.9;
    }

    > * {
        position: relative;
        z-index: 1;
    }
}

[data-theme='dark'] .dashboard {
    --dashboard-panel: #141926;
    --dashboard-panel-soft: #101520;
    --dashboard-panel-elevated: #171c2a;
    --dashboard-border: rgba(126, 145, 178, 0.12);
    --dashboard-border-strong: rgba(126, 145, 178, 0.2);
    --dashboard-text: #e7ecf6;
    --dashboard-text-secondary: rgba(231, 236, 246, 0.68);
    --dashboard-text-tertiary: rgba(231, 236, 246, 0.46);
    --dashboard-shadow: 0 14px 42px rgba(0, 0, 0, 0.18);
    --dashboard-shadow-hover: 0 20px 48px rgba(0, 0, 0, 0.24);
    --dashboard-control-bg: rgba(255, 255, 255, 0.035);
    --dashboard-row-hover: rgba(47, 140, 255, 0.1);
}

.equal-height-row {
    display: flex;
    flex-wrap: wrap;
}

.dashboard-health-alert {
    :deep(.ant-alert-message) {
        width: 100%;
    }
}

.dashboard-health-alert--success {
    background: linear-gradient(90deg, rgba(82, 196, 26, 0.08), rgba(35, 199, 183, 0.04));
}

.dashboard-health-alert--error {
    background: linear-gradient(90deg, rgba(255, 77, 94, 0.1), rgba(255, 176, 32, 0.04));
}

.dashboard-panel {
    border: 1px solid var(--dashboard-border);
    border-radius: 8px;
    background: linear-gradient(180deg, rgba(255, 255, 255, 0.02), transparent), var(--dashboard-panel);
    box-shadow: var(--dashboard-shadow);
    overflow: hidden;

    :deep(.ant-card-head) {
        min-height: 46px;
        border-bottom: 1px solid var(--dashboard-border);
        background: linear-gradient(180deg, rgba(255, 255, 255, 0.02), rgba(255, 255, 255, 0));
    }

    :deep(.ant-card-head-title) {
        color: var(--dashboard-text);
        font-size: 14px;
        font-weight: 650;
    }

    :deep(.ant-card-body) {
        background: transparent;
    }

    :deep(.ant-select-selector) {
        border-color: var(--dashboard-border);
        background: var(--dashboard-control-bg);
    }

    &:hover {
        border-color: var(--dashboard-border-strong);
        box-shadow: var(--dashboard-shadow-hover);
    }
}

.telemetry-card {
    --telemetry-accent: #2f8cff;
    --telemetry-accent-rgb: 47, 140, 255;
    min-height: 118px;
    border: 1px solid var(--dashboard-border);
    border-radius: 8px;
    color: var(--dashboard-text);
    background:
        radial-gradient(circle at 88% 16%, rgba(var(--telemetry-accent-rgb), 0.18), transparent 34%),
        linear-gradient(180deg, rgba(var(--telemetry-accent-rgb), 0.07), rgba(var(--telemetry-accent-rgb), 0.02)),
        var(--dashboard-panel);
    box-shadow: var(--dashboard-shadow);
    transition:
        transform 0.22s ease,
        border-color 0.22s ease,
        box-shadow 0.22s ease;
    position: relative;
    overflow: hidden;

    &::before {
        content: '';
        position: absolute;
        top: 0;
        left: 0;
        width: 100%;
        height: 3px;
        background: linear-gradient(90deg, var(--telemetry-accent), rgba(var(--telemetry-accent-rgb), 0.25));
    }

    &:hover {
        transform: translateY(-3px);
        border-color: rgba(var(--telemetry-accent-rgb), 0.45);
        box-shadow: 0 18px 42px rgba(var(--telemetry-accent-rgb), 0.16);
    }
}

.telemetry-card--blue {
    --telemetry-accent: #2f8cff;
    --telemetry-accent-rgb: 47, 140, 255;
}

.telemetry-card--green {
    --telemetry-accent: #23c7b7;
    --telemetry-accent-rgb: 35, 199, 183;
}

.telemetry-card--purple {
    --telemetry-accent: #8b5cf6;
    --telemetry-accent-rgb: 139, 92, 246;
}

.telemetry-card--orange {
    --telemetry-accent: #ffb020;
    --telemetry-accent-rgb: 255, 176, 32;
}

.telemetry-card--cyan {
    --telemetry-accent: #1fb6c1;
    --telemetry-accent-rgb: 31, 182, 193;
}

.telemetry-card--magenta {
    --telemetry-accent: #f15bb5;
    --telemetry-accent-rgb: 241, 91, 181;
}

.telemetry-title {
    font-size: 13px;
    color: var(--dashboard-text-secondary);
    font-weight: 600;
    margin-bottom: 6px;
    display: flex;
    align-items: center;
}

.telemetry-value {
    color: var(--dashboard-text);
    font-size: 27px;
    font-weight: 700;
    font-feature-settings: 'tnum'; /* 等宽数字，对齐更整齐 */
    line-height: 1.1;
    margin-bottom: 6px;
}

.telemetry-unit {
    font-size: 14px;
    font-weight: normal;
    margin-left: 4px;
    color: var(--dashboard-text-tertiary);
}

.telemetry-footer {
    font-size: 11px;
    color: var(--dashboard-text-tertiary);
    border-top: 1px solid var(--dashboard-border);
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
    background-color: var(--telemetry-accent);
    border-radius: 50%;
    margin-left: 8px;
    box-shadow: 0 0 0 0 rgba(var(--telemetry-accent-rgb), 0.58);
    animation: pulse 1.6s infinite;
}

@keyframes pulse {
    0% {
        transform: scale(0.95);
        box-shadow: 0 0 0 0 rgba(var(--telemetry-accent-rgb), 0.58);
    }
    70% {
        transform: scale(1);
        box-shadow: 0 0 0 8px rgba(var(--telemetry-accent-rgb), 0);
    }
    100% {
        transform: scale(0.95);
        box-shadow: 0 0 0 0 rgba(var(--telemetry-accent-rgb), 0);
    }
}

.dashboard-ranking-panel {
    :deep(.ant-table) {
        background: transparent;
        color: var(--dashboard-text-secondary);
    }

    :deep(.ant-table-thead > tr > th) {
        border-bottom: 1px solid var(--dashboard-border);
        background: var(--dashboard-panel-soft);
        color: var(--dashboard-text-secondary);
        font-size: 12px;
        font-weight: 650;
    }

    :deep(.ant-table-tbody > tr > td) {
        border-bottom: 1px solid var(--dashboard-border);
        background: transparent;
    }

    :deep(.ant-table-tbody > tr:hover > td) {
        background: var(--dashboard-row-hover);
    }
}

.dashboard-ranking-table {
    border: 1px solid var(--dashboard-border);
    border-radius: 8px;
    overflow: hidden;
}

.dashboard-resource-panel {
    :deep(.ant-statistic-title) {
        color: var(--dashboard-text-tertiary);
        font-size: 12px;
        font-weight: 600;
    }

    :deep(.ant-statistic-content) {
        color: var(--dashboard-text);
        font-size: 24px;
        font-weight: 650;
    }
}

.resource-stat-item {
    border-radius: 8px;
    padding: 14px 8px;
    transition:
        background 0.2s ease,
        transform 0.2s ease;

    &:hover {
        background: var(--dashboard-row-hover);
        transform: translateY(-2px);
    }
}

// 快捷导航区域与标题
.quick-links-section {
    border-top: 1px solid var(--dashboard-border);
    padding-top: 16px;
    margin-top: 12px;
}

.quick-links-title {
    margin-bottom: 12px;
    font-weight: 650;
    font-size: 13px;
    color: var(--dashboard-text-tertiary);
}

// 快捷连接卡片
.quick-link-card {
    text-align: center;
    cursor: pointer;
    transition:
        transform 0.2s ease,
        border-color 0.2s ease,
        box-shadow 0.2s ease;
    margin-top: 8px;
    border: 1px solid var(--dashboard-border);
    border-radius: 8px;
    background: var(--dashboard-panel-soft);

    &:hover {
        box-shadow: 0 12px 24px rgba(47, 140, 255, 0.12);
        border-color: rgba(47, 140, 255, 0.45);
        transform: translateY(-2px);
    }
}

.quick-link-text {
    display: block;
    margin-top: 4px;
    font-size: 12px;
    color: var(--dashboard-text-secondary);
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
