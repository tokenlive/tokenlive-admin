<template>
    <div class="provider-monitor">
        <div class="tab-toolbar">
            <div class="time-controls">
                <a-select
                    v-model:value="timeRange"
                    class="quick-range-select"
                    @change="handleQuickRangeChange">
                    <a-select-option value="1h">{{ $t('pages.dashboard.modelRanking.range.1h') }}</a-select-option>
                    <a-select-option value="6h">{{ $t('pages.dashboard.modelRanking.range.6h') }}</a-select-option>
                    <a-select-option value="24h">{{ $t('pages.dashboard.modelRanking.range.24h') }}</a-select-option>
                    <a-select-option value="7d">{{ $t('pages.dashboard.modelRanking.range.7d') }}</a-select-option>
                    <a-select-option value="today">{{
                        $t('pages.dashboard.modelRanking.range.today')
                    }}</a-select-option>
                    <a-select-option value="custom">{{
                        $t('pages.provider.detail.monitor.range.custom')
                    }}</a-select-option>
                </a-select>
                <a-range-picker
                    v-model:value="selectedTimeRange"
                    class="time-range-picker"
                    show-time
                    :allow-clear="false"
                    :disabled-date="disableOutOfWindowDate"
                    @change="handleCustomRangeChange" />
                <a-tooltip :title="$t('pages.provider.detail.monitor.range.limit')">
                    <info-circle-outlined class="range-hint" />
                </a-tooltip>
            </div>
            <a-button
                :loading="loading"
                @click="loadAll">
                <template #icon><reload-outlined /></template>
                {{ $t('pages.provider.detail.monitor.refresh') }}
            </a-button>
        </div>

        <a-alert
            v-if="providerBreakers.length > 0"
            class="monitor-alert"
            type="error"
            show-icon>
            <template #message>
                {{ $t('pages.provider.detail.monitor.breakers.count', { count: providerBreakers.length }) }}
                <a-tag
                    v-for="item in providerBreakers"
                    :key="item.id"
                    color="error"
                    style="margin-left: 8px">
                    {{ item.name || item.id }}
                </a-tag>
            </template>
        </a-alert>

        <a-row
            :gutter="12"
            class="kpi-row">
            <a-col
                v-for="card in kpiCards"
                :key="card.key"
                :xs="12"
                :sm="8"
                :md="6"
                :lg="3">
                <div class="kpi-card">
                    <div class="kpi-label">{{ card.label }}</div>
                    <a-tooltip v-if="card.hint">
                        <template #title>{{ card.hint }}</template>
                        <div class="kpi-value kpi-value--hint">{{ card.value }}</div>
                    </a-tooltip>
                    <div
                        v-else
                        class="kpi-value">
                        {{ card.value }}
                    </div>
                </div>
            </a-col>
        </a-row>

        <a-row :gutter="16">
            <a-col
                :xs="24"
                :lg="12">
                <div class="monitor-panel">
                    <div class="monitor-panel__title">{{ $t('pages.provider.detail.monitor.traffic') }}</div>
                    <a-empty
                        v-if="!hasTraffic"
                        :description="$t('pages.provider.detail.monitor.traffic.empty')" />
                    <x-chart
                        v-else
                        :options="trafficChartOptions"
                        height="280px"
                        :loading="loading" />
                </div>
            </a-col>
            <a-col
                :xs="24"
                :lg="12">
                <div class="monitor-panel">
                    <div class="monitor-panel__title">
                        {{ $t('pages.provider.detail.monitor.endpointTraffic') }}
                    </div>
                    <a-empty
                        v-if="!hasEndpointTraffic"
                        :description="$t('pages.provider.detail.monitor.endpointTraffic.empty')" />
                    <x-chart
                        v-else
                        :options="endpointChartOptions"
                        height="280px"
                        :loading="loading" />
                </div>
            </a-col>
        </a-row>

        <div class="monitor-panel events-panel">
            <div class="monitor-panel__title">{{ $t('pages.provider.detail.monitor.events') }}</div>
            <a-empty
                v-if="events.length === 0"
                :description="$t('pages.provider.detail.monitor.events.empty')" />
            <a-table
                v-else
                size="middle"
                row-key="id"
                :pagination="eventPagination"
                :data-source="events"
                :columns="eventColumns"
                :scroll="{ x: 900 }"
                :expand-row-by-click="true"
                @change="handleEventTableChange">
                <template #bodyCell="{ column, record }">
                    <template v-if="column.key === 'event_time'">
                        {{ formatUtcDateTime(record.event_time) }}
                    </template>
                    <template v-else-if="column.key === 'event_type'">
                        <a-tag :color="eventTypeColor(record.event_type)">
                            {{ eventTypeLabel(record.event_type) }}
                        </a-tag>
                    </template>
                    <template v-else-if="column.key === 'endpoint_code'">
                        {{ record.endpoint_code || record.endpoint_id || '--' }}
                    </template>
                </template>
                <template #expandedRowRender="{ record }">
                    <div class="event-expanded-container">
                        <a-descriptions
                            :column="2"
                            size="small"
                            bordered>
                            <a-descriptions-item
                                v-if="record.policy_id"
                                :label="$t('pages.ops.table.policy_id')">
                                <a-typography-text
                                    class="event-expanded-code"
                                    :copyable="{ text: record.policy_id }"
                                    :ellipsis="{ tooltip: true }">
                                    {{ record.policy_id }}
                                </a-typography-text>
                            </a-descriptions-item>
                            <a-descriptions-item
                                v-if="record.endpoint_id"
                                :label="$t('pages.ops.table.endpoint_id')">
                                <a-typography-text
                                    class="event-expanded-code"
                                    :copyable="{ text: record.endpoint_id }"
                                    :ellipsis="{ tooltip: true }">
                                    {{ record.endpoint_id }}
                                </a-typography-text>
                            </a-descriptions-item>
                            <a-descriptions-item
                                v-if="record.request_id"
                                :label="$t('pages.ops.table.request_id')">
                                <a-typography-text
                                    class="event-expanded-code"
                                    :copyable="{ text: record.request_id }"
                                    :ellipsis="{ tooltip: true }">
                                    {{ record.request_id }}
                                </a-typography-text>
                            </a-descriptions-item>
                            <a-descriptions-item
                                v-if="record.trace_id"
                                :label="$t('pages.ops.table.trace_id')">
                                <a-typography-text
                                    class="event-expanded-code"
                                    :copyable="{ text: record.trace_id }"
                                    :ellipsis="{ tooltip: true }">
                                    {{ record.trace_id }}
                                </a-typography-text>
                            </a-descriptions-item>
                            <a-descriptions-item
                                v-if="record.threshold != null || record.current_value != null"
                                :label="$t('pages.ops.table.threshold') + ' / ' + $t('pages.ops.table.current_value')">
                                <a-tag
                                    :color="
                                        record.current_value != null &&
                                        record.threshold != null &&
                                        record.current_value >= record.threshold
                                            ? 'red'
                                            : 'blue'
                                    ">
                                    {{ formatEventValue(record.threshold) }} /
                                    {{ formatEventValue(record.current_value) }}
                                </a-tag>
                            </a-descriptions-item>
                            <a-descriptions-item
                                v-if="record.message"
                                :label="$t('pages.ops.table.detail')"
                                :span="2">
                                <div class="event-expanded-message">{{ record.message }}</div>
                            </a-descriptions-item>
                        </a-descriptions>
                    </div>
                </template>
            </a-table>
        </div>
    </div>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import * as echarts from 'echarts'
import dayjs from 'dayjs'
import { ReloadOutlined, InfoCircleOutlined } from '@ant-design/icons-vue'
import { useAppStore } from '@/store'
import apis from '@/apis'
import { formatTokens, formatUtcDateTime } from '@/utils/util'

// 供应商监控的时间范围上限，与后端 providerMetricsMaxRange 对齐。
// 内存 store 的分钟级指标只保留 7 天，超出范围会静默返回不完整数据。详见 ADR-0008。
const MAX_RANGE_DAYS = 7

const props = defineProps({
    providerId: {
        type: String,
        required: true,
    },
    providerCode: {
        type: String,
        default: '',
    },
    providerName: {
        type: String,
        default: '',
    },
    active: {
        type: Boolean,
        default: false,
    },
})

const { t } = useI18n()
const appStore = useAppStore()

const timeRange = ref('1h')
const selectedTimeRange = ref([dayjs().subtract(1, 'hour'), dayjs()])
const loading = ref(false)
const ranking = ref(null)
const traffic = ref({ times: [], series: [] })
const endpointTraffic = ref({ times: [], series: [] })
const endpoints = ref([])
const breakers = ref([])
const events = ref([])
const eventPagination = reactive({
    current: 1,
    pageSize: 20,
    total: 0,
    showSizeChanger: true,
    showTotal: (total) => t('common.pagination.total', { total }),
})

const eventColumns = computed(() => [
    { title: t('pages.ops.table.time'), key: 'event_time', dataIndex: 'event_time', width: 170 },
    { title: t('pages.ops.table.type'), key: 'event_type', dataIndex: 'event_type', width: 110 },
    { title: t('pages.ops.table.tenant'), key: 'tenant_code', dataIndex: 'tenant_code', width: 110, ellipsis: true },
    { title: t('pages.ops.table.model'), key: 'model_code', dataIndex: 'model_code', width: 150, ellipsis: true },
    {
        title: t('pages.ops.table.endpoint_code'),
        key: 'endpoint_code',
        dataIndex: 'endpoint_code',
        width: 150,
        ellipsis: true,
    },
    {
        title: t('pages.ops.table.policy_name'),
        key: 'policy_name',
        dataIndex: 'policy_name',
        width: 180,
        ellipsis: true,
    },
])

function formatLatency(ms) {
    if (!ms || ms === 0) return '-'
    return (ms / 1000).toFixed(2) + 's'
}

function formatOtps(val) {
    const num = Number(val)
    if (!Number.isFinite(num) || num === 0) return '-'
    return (
        num.toLocaleString('en-US', {
            minimumFractionDigits: 0,
            maximumFractionDigits: 2,
        }) + ' t/s'
    )
}

function formatSuccessRate(val) {
    if (val === undefined || val === null || Number.isNaN(Number(val))) return '-'
    return Number(val).toFixed(1) + '%'
}

function eventTypeLabel(type) {
    const key = `pages.ops.${type}`
    const label = t(key)
    return label === key ? type : label
}

function eventTypeColor(type) {
    const colors = {
        circuit_break: '#ff4d5e',
        rate_limit: '#ffb020',
        invocation_fail: '#8b5cf6',
        retry_error: '#06b6d4',
        circuit_breaker_error: '#e11d48',
        model_failover: '#2f8cff',
        endpoint_failover: '#23c7b7',
    }
    return colors[type] || 'default'
}

function formatEventValue(value) {
    if (value == null) return '-'
    return Number(value).toFixed(2).replace(/\.00$/, '')
}

function handleEventTableChange(pagination) {
    eventPagination.current = pagination.current
    eventPagination.pageSize = pagination.pageSize
    loadAll()
}

function quickRangeValue(range) {
    const now = dayjs()
    if (range === 'today') {
        return [now.startOf('day'), now]
    }
    const minutes = { '1h': 60, '6h': 360, '24h': 1440, '7d': 10080 }[range] || 60
    return [now.subtract(minutes, 'minute'), now]
}

function handleQuickRangeChange(range) {
    if (range === 'custom') return
    eventPagination.current = 1
    selectedTimeRange.value = quickRangeValue(range)
    loadAll()
}

function handleCustomRangeChange(value) {
    if (!value || value.length !== 2) return
    eventPagination.current = 1
    timeRange.value = 'custom'
    loadAll()
}

// 禁用未来日期与超出 7 天窗口的日期：后端会把超限范围夹到 7 天，
// 与其让用户选了之后拿到与预期不符的数据，不如不让选。
function disableOutOfWindowDate(current) {
    if (!current) return false
    return current.isAfter(dayjs().endOf('day')) || current.isBefore(dayjs().subtract(MAX_RANGE_DAYS, 'day'))
}

function queryTimeRange() {
    if (timeRange.value !== 'custom') return timeRange.value
    const [start, end] = selectedTimeRange.value
    const minutes = Math.max(1, Math.ceil(end.diff(start, 'minute', true)))
    return `${minutes}m`
}

const kpi = computed(() => ranking.value || {})

const kpiCards = computed(() => [
    {
        key: 'upstreamCalls',
        label: t('pages.provider.detail.monitor.kpi.upstreamCalls'),
        value: kpi.value.upstream_call_count ? Number(kpi.value.upstream_call_count).toLocaleString('en-US') : '0',
    },
    {
        key: 'successRate',
        label: t('pages.provider.detail.monitor.kpi.successRate'),
        value: kpi.value.upstream_call_count ? formatSuccessRate(kpi.value.upstream_call_success_rate) : '-',
        hint: t('pages.provider.detail.health.successRate.hint'),
    },
    // 无 P50/P95/P99 tooltip：分位数需要直方图桶，内存 store 只有 sum/count。详见 ADR-0008。
    {
        key: 'latency',
        label: t('pages.provider.detail.monitor.kpi.latency'),
        value: formatLatency(kpi.value.avg_latency_ms),
    },
    {
        key: 'ttft',
        label: t('pages.provider.detail.monitor.kpi.ttft'),
        value: formatLatency(kpi.value.avg_ttft_ms),
    },
    {
        key: 'otps',
        label: t('pages.provider.detail.monitor.kpi.otps'),
        value: formatOtps(kpi.value.otps),
    },
    {
        key: 'tokens',
        label: t('pages.provider.detail.monitor.kpi.tokens'),
        value: kpi.value.total_tokens ? formatTokens(kpi.value.total_tokens) : '-',
    },
    {
        key: 'cost',
        label: t('pages.provider.detail.monitor.kpi.cost'),
        value: kpi.value.total_cost ? '¥' + Number(kpi.value.total_cost).toFixed(4) : '-',
    },
    {
        key: 'endpoints',
        label: t('pages.provider.detail.monitor.kpi.endpoints'),
        value: endpoints.value.length.toLocaleString('en-US'),
    },
])

const providerBreakers = computed(() => {
    const endpointIds = new Set((endpoints.value || []).map((item) => item.id))
    return (breakers.value || []).filter(
        (item) => item.provider_id === props.providerId || (item.id && endpointIds.has(item.id))
    )
})

const hasTraffic = computed(() => (traffic.value.series?.[0]?.total || []).some((value) => Number(value) > 0))

const hasEndpointTraffic = computed(() =>
    (endpointTraffic.value.series || []).some((series) => (series.total || []).some((value) => Number(value) > 0))
)

function chartTheme() {
    const isDark = appStore.config.theme === 'dark'
    return {
        isDark,
        text: isDark ? 'rgba(231, 236, 246, 0.68)' : 'rgba(31, 41, 55, 0.68)',
        tooltip: {
            backgroundColor: isDark ? 'rgba(18, 24, 38, 0.96)' : 'rgba(255, 255, 255, 0.98)',
            borderColor: isDark ? 'rgba(126, 145, 178, 0.18)' : 'rgba(15, 23, 42, 0.1)',
            textStyle: { color: isDark ? '#e7ecf6' : '#1f2937' },
            extraCssText: 'box-shadow: 0 12px 32px rgba(15, 23, 42, 0.18); border-radius: 8px;',
        },
        splitLine: {
            lineStyle: { color: isDark ? 'rgba(126, 145, 178, 0.11)' : 'rgba(15, 23, 42, 0.08)' },
        },
        axisLine: { lineStyle: { color: isDark ? 'rgba(126, 145, 178, 0.18)' : 'rgba(15, 23, 42, 0.12)' } },
    }
}

const trafficChartOptions = computed(() => {
    const theme = chartTheme()
    const series = traffic.value.series?.[0] || {}
    const successData = series.success || []
    const failureData = series.failure || []
    const successRates = successData.map((success, index) => {
        const total = Number(success) + Number(failureData[index] || 0)
        if (total === 0) return 100
        return parseFloat(((Number(success) / total) * 100).toFixed(1))
    })

    return {
        tooltip: { ...theme.tooltip, trigger: 'axis', axisPointer: { type: 'cross' } },
        legend: {
            data: [
                t('pages.dashboard.trends.success_requests'),
                t('pages.dashboard.trends.failed_requests'),
                t('pages.provider.detail.monitor.kpi.successRate'),
            ],
            textStyle: { color: theme.text },
        },
        grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
        xAxis: {
            type: 'category',
            boundaryGap: false,
            data: traffic.value.times || [],
            axisLabel: { color: theme.text },
            axisLine: theme.axisLine,
        },
        yAxis: [
            {
                type: 'value',
                name: t('pages.dashboard.trends.requests'),
                minInterval: 1,
                axisLabel: { color: theme.text },
                splitLine: theme.splitLine,
            },
            {
                type: 'value',
                name: t('pages.provider.detail.monitor.kpi.successRate'),
                min: 0,
                max: 100,
                axisLabel: { formatter: '{value} %', color: theme.text },
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
                name: t('pages.provider.detail.monitor.kpi.successRate'),
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
})

const endpointChartOptions = computed(() => {
    const theme = chartTheme()
    const colors = ['#2f8cff', '#23c7b7', '#ff4d5e', '#ffb020', '#8b5cf6', '#6ee7d8', '#f472b6', '#7bb8ff']
    return {
        tooltip: { ...theme.tooltip, trigger: 'axis', axisPointer: { type: 'cross' } },
        legend: {
            data: (endpointTraffic.value.series || []).map((series) => endpointSeriesLabel(series.label)),
            textStyle: { color: theme.text },
        },
        grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
        xAxis: {
            type: 'category',
            boundaryGap: false,
            data: endpointTraffic.value.times || [],
            axisLabel: { color: theme.text },
            axisLine: theme.axisLine,
        },
        yAxis: [
            {
                type: 'value',
                name: t('pages.dashboard.trends.requests'),
                minInterval: 1,
                axisLabel: { color: theme.text },
                splitLine: theme.splitLine,
            },
        ],
        series: (endpointTraffic.value.series || []).map((series, index) => ({
            name: endpointSeriesLabel(series.label),
            type: 'line',
            smooth: true,
            showSymbol: false,
            itemStyle: { color: colors[index % colors.length] },
            data: series.total || [],
        })),
    }
})

function endpointSeriesLabel(label) {
    if (!label) return label
    const match = endpoints.value.find((item) => item.id === label)
    return match?.code || label
}

// 供应商侧的趋势按 group_by=provider 拉取，再挑出本供应商的那条 series。
// 后端返回的 label 是网关上报的标识（ADR-0009 起为 provider code），
// 完成穿透前的历史数据可能仍是 name，故两者都认。
function pickProviderSeries(payload) {
    const series = payload?.series || []
    const candidates = [props.providerCode, props.providerName].filter(Boolean)
    const matched = series.find((item) => candidates.includes(item.label))
    return {
        times: payload?.times || [],
        series: matched ? [matched] : [],
    }
}

async function loadAll() {
    if (!props.providerId) return
    loading.value = true
    if (timeRange.value !== 'custom') {
        selectedTimeRange.value = quickRangeValue(timeRange.value)
    }
    const range = queryTimeRange()
    const [start, end] = selectedTimeRange.value
    const endTime = end.format()
    const providerRef = props.providerCode || props.providerId

    try {
        const [rankingRes, trafficRes, endpointListRes, breakerRes, eventRes] = await Promise.all([
            apis.dashboard
                .getProviderRanking({ provider: providerRef, time_range: range, end_time: endTime, limit: 1 })
                .catch(() => ({ data: [] })),
            apis.dashboard
                .getTrends({ time_range: range, end_time: endTime, group_by: 'provider' })
                .catch(() => ({ data: { times: [], series: [] } })),
            // 用专用接口而非通用列表：它预加载了 Model，端点流量趋势需要 model_code。
            apis.endpoint.getEndpointsByProviderId(props.providerId).catch(() => ({ data: [] })),
            apis.dashboard.getCircuitBreakers().catch(() => ({ data: [] })),
            apis.ops
                .getEvents({
                    provider_ref: providerRef,
                    provider_name_fallback: props.providerName || undefined,
                    start_time: start.toISOString(),
                    end_time: endTime,
                    pageSize: eventPagination.pageSize,
                    current: eventPagination.current,
                })
                .catch(() => ({ data: [] })),
        ])

        const rankingRows = rankingRes.data || []
        ranking.value = rankingRows[0] || null
        traffic.value = pickProviderSeries(trafficRes.data)
        endpoints.value = endpointListRes.data || []
        breakers.value = breakerRes.data || []
        events.value = eventRes.data || []
        eventPagination.total = eventRes.total || 0

        await loadEndpointTraffic(range, endTime)
    } finally {
        loading.value = false
    }
}

// 端点分流量需要按模型逐个拉取：trends 的 group_by=endpoint 依赖 model 参数
// 来确定端点集合，没有供应商维度的等价查询。这里改为在前端按本供应商的
// 端点 ID 过滤 series。
async function loadEndpointTraffic(range, endTime) {
    const modelCodes = [...new Set((endpoints.value || []).map((ep) => ep.model?.model_code).filter(Boolean))]
    if (modelCodes.length === 0) {
        endpointTraffic.value = { times: [], series: [] }
        return
    }

    const ownEndpointIds = new Set((endpoints.value || []).map((ep) => ep.id))
    const results = await Promise.all(
        modelCodes.map((model) =>
            apis.dashboard
                .getTrends({ model, time_range: range, end_time: endTime, group_by: 'endpoint' })
                .catch(() => ({ data: { times: [], series: [] } }))
        )
    )

    const merged = { times: [], series: [] }
    for (const res of results) {
        const payload = res.data || {}
        if (!merged.times.length && payload.times?.length) {
            merged.times = payload.times
        }
        for (const series of payload.series || []) {
            if (ownEndpointIds.has(series.label)) {
                merged.series.push(series)
            }
        }
    }
    endpointTraffic.value = merged
}

watch(
    () => [props.active, props.providerId],
    ([isActive, id]) => {
        if (isActive && id) {
            loadAll()
        }
    },
    { immediate: true }
)

defineExpose({ loadAll })
</script>

<style lang="less" scoped>
.tab-toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
    gap: 12px;
}

.time-controls {
    display: flex;
    align-items: center;
    min-width: 0;
}

.quick-range-select {
    width: 132px;
    flex: 0 0 132px;
}

.time-range-picker {
    width: 360px;
    margin-left: 8px;
}

.range-hint {
    margin-left: 8px;
    color: var(--color-text-tertiary, rgba(31, 41, 55, 0.45));
    cursor: help;
}

.monitor-alert {
    margin-bottom: 16px;
}

.kpi-row {
    margin-bottom: 16px;
}

.kpi-card {
    background: var(--color-bg-container, #fff);
    border: 1px solid var(--color-border-secondary, rgba(15, 23, 42, 0.08));
    border-radius: 8px;
    padding: 12px;
    margin-bottom: 12px;
}

.kpi-label {
    color: var(--color-text-secondary, rgba(31, 41, 55, 0.68));
    font-size: 12px;
    margin-bottom: 6px;
}

.kpi-value {
    font-size: 18px;
    font-weight: 600;
    color: var(--color-text-primary);
}

.kpi-value--hint {
    cursor: help;
    border-bottom: 1px dashed var(--color-text-tertiary, rgba(31, 41, 55, 0.4));
    display: inline-block;
}

.monitor-panel {
    border: 1px solid var(--color-border-secondary, rgba(15, 23, 42, 0.08));
    border-radius: 8px;
    padding: 12px 16px 16px;
    margin-bottom: 16px;
}

.monitor-panel__title {
    font-weight: 600;
    margin-bottom: 12px;
}

.events-panel :deep(.ant-table) {
    color: var(--color-text-secondary);
    background: transparent;
}

.events-panel :deep(.ant-table-thead > tr > th) {
    border-bottom-color: var(--color-border-secondary);
    background: var(--color-bg-hover) !important;
    color: var(--color-text-secondary);
    font-size: 12px;
    font-weight: 600;
}

.events-panel :deep(.ant-table-tbody > tr > td) {
    border-bottom-color: var(--color-border-secondary);
    background: transparent;
}

.events-panel :deep(.ant-table-tbody > tr.ant-table-row) {
    cursor: pointer;
}

.events-panel :deep(.ant-table-tbody > tr:hover > td) {
    background: var(--color-primary-bg) !important;
}

.events-panel :deep(.ant-tag) {
    border-color: transparent;
    border-radius: 5px;
    color: #fff;
    font-size: 12px;
    font-weight: 500;
}

.event-expanded-container {
    padding: 16px 24px;
    border-radius: 8px;
    background: var(--color-bg-hover);
    word-break: break-word;
}

.event-expanded-code {
    display: inline-block;
    max-width: 100%;
    vertical-align: bottom;
    font-family: monospace;
}

.event-expanded-container :deep(.ant-descriptions-bordered .ant-descriptions-item-label) {
    border-color: var(--color-border-secondary);
    background: var(--color-bg-active);
    color: var(--color-text-secondary);
}

.event-expanded-container :deep(.ant-descriptions-bordered .ant-descriptions-item-content) {
    border-color: var(--color-border-secondary);
    background: var(--color-bg-container);
    color: var(--color-text-secondary);
}

.event-expanded-container :deep(.ant-descriptions-bordered .ant-descriptions-row) {
    border-bottom-color: var(--color-border-secondary);
}

.event-expanded-message {
    max-height: 120px;
    overflow-y: auto;
    color: var(--color-error);
    font-family: monospace;
    font-size: 12px;
    line-height: 1.5;
    white-space: pre-wrap;
    word-break: break-all;
}

@media (max-width: 767px) {
    .tab-toolbar,
    .time-controls {
        align-items: stretch;
        flex-wrap: wrap;
    }

    .tab-toolbar {
        justify-content: flex-start;
    }

    .time-range-picker {
        width: 100%;
        margin: 8px 0 0;
    }
}
</style>
