<template>
    <div class="model-monitor">
        <MonitoringStatus :reading="monitoring" />
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
                        $t('pages.model.detail.monitor.range.custom')
                    }}</a-select-option>
                </a-select>
                <a-range-picker
                    v-model:value="selectedTimeRange"
                    class="time-range-picker"
                    show-time
                    :allow-clear="false"
                    :disabled-date="disableFutureDate"
                    @change="handleCustomRangeChange" />
            </div>
            <a-button
                :loading="loading"
                @click="loadAll">
                <template #icon><reload-outlined /></template>
                {{ $t('pages.model.detail.monitor.refresh') }}
            </a-button>
        </div>

        <a-alert
            v-if="modelBreakers.length > 0"
            class="monitor-alert"
            type="error"
            show-icon>
            <template #message>
                {{ $t('pages.model.detail.monitor.breakers.count', { count: modelBreakers.length }) }}
                <a-tag
                    v-for="item in modelBreakers"
                    :key="item.id || item.name"
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
                        <template #title>
                            <div
                                v-for="line in card.hint"
                                :key="line">
                                {{ line }}
                            </div>
                        </template>
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
                    <div class="monitor-panel__title">{{ $t('pages.model.detail.monitor.traffic') }}</div>
                    <MonitoringStatus :reading="readings.traffic" />
                    <a-empty
                        v-if="!hasTraffic && readings.traffic?.state === 'ok'"
                        :description="$t('pages.model.detail.monitor.empty')" />
                    <x-chart
                        v-if="hasTraffic"
                        :options="trafficChartOptions"
                        height="280px"
                        :loading="loading" />
                </div>
            </a-col>
            <a-col
                :xs="24"
                :lg="12">
                <div class="monitor-panel">
                    <div class="monitor-panel__title">{{ $t('pages.model.detail.monitor.endpointTraffic') }}</div>
                    <MonitoringStatus :reading="readings.endpointTraffic" />
                    <a-empty
                        v-if="!hasEndpointTraffic && readings.endpointTraffic?.state === 'ok'"
                        :description="$t('pages.model.detail.monitor.endpointTraffic.empty')" />
                    <x-chart
                        v-if="hasEndpointTraffic"
                        :options="endpointChartOptions"
                        height="280px"
                        :loading="loading" />
                </div>
            </a-col>
        </a-row>

        <a-row :gutter="16">
            <a-col
                :xs="24"
                :lg="12">
                <div class="monitor-panel">
                    <div class="monitor-panel__title">{{ $t('pages.model.detail.monitor.ttftTrend') }}</div>
                    <MonitoringStatus :reading="readings.performance" />
                    <a-empty
                        v-if="!hasTTFTTrend && readings.performance?.state === 'ok'"
                        :description="$t('pages.model.detail.monitor.performance.empty')" />
                    <x-chart
                        v-if="hasTTFTTrend"
                        :options="ttftChartOptions"
                        height="280px"
                        :loading="loading" />
                </div>
            </a-col>
            <a-col
                :xs="24"
                :lg="12">
                <div class="monitor-panel">
                    <div class="monitor-panel__title">{{ $t('pages.model.detail.monitor.otpsTrend') }}</div>
                    <MonitoringStatus :reading="readings.performance" />
                    <a-empty
                        v-if="!hasOTPSTrend && readings.performance?.state === 'ok'"
                        :description="$t('pages.model.detail.monitor.performance.empty')" />
                    <x-chart
                        v-if="hasOTPSTrend"
                        :options="otpsChartOptions"
                        height="280px"
                        :loading="loading" />
                </div>
            </a-col>
        </a-row>

        <div class="monitor-panel events-panel">
            <div class="monitor-panel__title">{{ $t('pages.model.detail.monitor.events') }}</div>
            <MonitoringStatus :reading="readings.events" />
            <a-empty
                v-if="
                    events.length === 0 &&
                    !eventsLoading &&
                    readings.events?.state === 'ok' &&
                    eventPagination.total === 0
                "
                :description="$t('pages.model.detail.monitor.events.empty')" />
            <a-table
                v-if="events.length > 0 || eventsLoading || eventPagination.total > 0"
                :loading="eventsLoading"
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
import MonitoringStatus from '@/components/MonitoringStatus.vue'
import { monitoringSuccessRates, monitoringStatus } from '@/utils/monitoring-snapshot'
import { computed, onBeforeUnmount, reactive, ref, shallowRef, watch } from 'vue'
import { createDetailMonitoring } from '@/utils/detail-monitoring'
import { modelObservationCards } from '@/utils/model-observations'
import { useI18n } from 'vue-i18n'
import * as echarts from 'echarts'
import dayjs from 'dayjs'
import { ReloadOutlined } from '@ant-design/icons-vue'
import { useAppStore } from '@/store'
import apis from '@/apis'
import { formatTokens, formatUtcDateTime } from '@/utils/util'

const props = defineProps({
    modelId: {
        type: String,
        required: true,
    },
    modelCode: {
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
const eventPagination = reactive({
    current: 1,
    pageSize: 20,
    total: 0,
    showSizeChanger: true,
    showTotal: (total) => t('common.pagination.total', { total }),
})

const state = shallowRef(null)
const controller = createDetailMonitoring({
    kind: 'model',
    api: apis,
    onChange: (next) => {
        state.value = next
        eventPagination.current = next.eventPage.current
        eventPagination.pageSize = next.eventPage.pageSize
        eventPagination.total = next.eventTotal
    },
})
state.value = controller.getState()
const readings = computed(() => state.value.readings)
const monitoring = computed(() => monitoringStatus(Object.values(readings.value)))
const loading = computed(() => state.value.loading)
const eventsLoading = computed(() => state.value.eventsLoading)
const ranking = computed(() => state.value.ranking)
const traffic = computed(() => state.value.traffic)
const endpointTraffic = computed(() => state.value.endpointTraffic)
const performance = computed(() => state.value.performance)
const endpoints = computed(() => state.value.endpoints)
const breakers = computed(() => state.value.breakers)
const events = computed(() => state.value.events)

const eventColumns = computed(() => [
    { title: t('pages.ops.table.time'), key: 'event_time', dataIndex: 'event_time', width: 170 },
    { title: t('pages.ops.table.type'), key: 'event_type', dataIndex: 'event_type', width: 110 },
    { title: t('pages.ops.table.tenant'), key: 'tenant_code', dataIndex: 'tenant_code', width: 110, ellipsis: true },
    {
        title: t('pages.ops.table.endpoint_code'),
        key: 'endpoint_code',
        dataIndex: 'endpoint_code',
        width: 150,
        ellipsis: true,
    },
    {
        title: t('pages.ops.table.provider'),
        key: 'provider_name',
        dataIndex: 'provider_name',
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
    if (ms == null) return '—'
    return (ms / 1000).toFixed(2) + 's'
}

function formatOtps(val) {
    const num = Number(val)
    if (val == null || !Number.isFinite(num)) return '—'
    return (
        num.toLocaleString('en-US', {
            minimumFractionDigits: 0,
            maximumFractionDigits: 2,
        }) + ' t/s'
    )
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
    const formatted = Number(value).toFixed(2).replace(/\.00$/, '')
    return formatted
}

function handleEventTableChange(pagination) {
    return controller.pageEvents({ current: pagination.current, pageSize: pagination.pageSize })
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

function disableFutureDate(current) {
    return current && current.isAfter(dayjs().endOf('day'))
}

function queryTimeRange() {
    if (timeRange.value !== 'custom') return timeRange.value
    const [start, end] = selectedTimeRange.value
    const minutes = Math.max(1, Math.ceil(end.diff(start, 'minute', true)))
    return `${minutes}m`
}

const kpiCards = computed(() =>
    modelObservationCards(ranking.value, endpoints.value, readings.value.endpoints, t, formatTokens)
)

const modelBreakers = computed(() => {
    const endpointIds = new Set((endpoints.value || []).map((item) => item.id))
    return (breakers.value || []).filter(
        (item) => item.model_id === props.modelId || (item.id && endpointIds.has(item.id))
    )
})

const hasTraffic = computed(() => (traffic.value.series || []).some((series) => hasMetricData(series.total)))

const hasEndpointTraffic = computed(() =>
    (endpointTraffic.value.series || []).some((series) => hasMetricData(series.total))
)

function hasMetricData(values) {
    return (values || []).some((value) => value !== null && value !== undefined && Number.isFinite(Number(value)))
}

const hasTTFTTrend = computed(() => hasMetricData(performance.value.avg_ttft_ms))
const hasOTPSTrend = computed(() => hasMetricData(performance.value.otps))

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
    const successRates = monitoringSuccessRates(successData, failureData)

    return {
        tooltip: { ...theme.tooltip, trigger: 'axis', axisPointer: { type: 'cross' } },
        legend: {
            data: [
                t('pages.dashboard.trends.success_requests'),
                t('pages.dashboard.trends.failed_requests'),
                t('pages.dashboard.trends.success_rate'),
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
                name: t('pages.dashboard.trends.success_rate'),
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

const ttftChartOptions = computed(() => {
    const theme = chartTheme()
    const seriesName = t('pages.model.detail.monitor.kpi.ttft')
    return {
        tooltip: {
            ...theme.tooltip,
            trigger: 'axis',
            axisPointer: { type: 'cross' },
            valueFormatter: (value) => formatLatency(value),
        },
        grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
        xAxis: {
            type: 'category',
            boundaryGap: false,
            data: performance.value.times || [],
            axisLabel: { color: theme.text },
            axisLine: theme.axisLine,
        },
        yAxis: {
            type: 'value',
            name: t('pages.model.detail.monitor.performance.ttftUnit'),
            axisLabel: {
                color: theme.text,
                formatter: (value) => (Number(value) >= 1000 ? `${(Number(value) / 1000).toFixed(1)}s` : `${value}ms`),
            },
            splitLine: theme.splitLine,
        },
        series: [
            {
                name: seriesName,
                type: 'line',
                smooth: true,
                showSymbol: false,
                connectNulls: false,
                itemStyle: { color: '#8b5cf6' },
                lineStyle: { width: 2 },
                areaStyle: {
                    color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                        { offset: 0, color: 'rgba(139, 92, 246, 0.28)' },
                        { offset: 1, color: 'rgba(139, 92, 246, 0.02)' },
                    ]),
                },
                data: performance.value.avg_ttft_ms || [],
            },
        ],
    }
})

const otpsChartOptions = computed(() => {
    const theme = chartTheme()
    const seriesName = t('pages.model.detail.monitor.kpi.otps')
    return {
        tooltip: {
            ...theme.tooltip,
            trigger: 'axis',
            axisPointer: { type: 'cross' },
            valueFormatter: (value) => formatOtps(value),
        },
        grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
        xAxis: {
            type: 'category',
            boundaryGap: false,
            data: performance.value.times || [],
            axisLabel: { color: theme.text },
            axisLine: theme.axisLine,
        },
        yAxis: {
            type: 'value',
            name: t('pages.model.detail.monitor.performance.otpsUnit'),
            axisLabel: { color: theme.text },
            splitLine: theme.splitLine,
        },
        series: [
            {
                name: seriesName,
                type: 'line',
                smooth: true,
                showSymbol: false,
                connectNulls: false,
                itemStyle: { color: '#2f8cff' },
                lineStyle: { width: 2 },
                areaStyle: {
                    color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                        { offset: 0, color: 'rgba(47, 140, 255, 0.28)' },
                        { offset: 1, color: 'rgba(47, 140, 255, 0.02)' },
                    ]),
                },
                data: performance.value.otps || [],
            },
        ],
    }
})

function endpointSeriesLabel(label) {
    if (!label) return t('pages.model.detail.monitor.unselectedEndpoint')
    const match = endpoints.value.find((item) => item.id === label)
    return match?.code || label
}

function loadAll() {
    if (!props.active || !props.modelId || !props.modelCode) return
    if (timeRange.value !== 'custom') {
        selectedTimeRange.value = quickRangeValue(timeRange.value)
    }
    const range = queryTimeRange()
    const [start, end] = selectedTimeRange.value
    const endTime = end.format()
    return controller.refresh({
        id: props.modelId,
        code: props.modelCode,
        time_range: range,
        start_time: start.format(),
        end_time: endTime,
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    })
}

watch(
    () => [props.active, props.modelId, props.modelCode],
    ([isActive, id, code]) => {
        if (!isActive || !id || !code) {
            controller.stop()
            return
        }
        loadAll()
    },
    { immediate: true }
)
onBeforeUnmount(() => controller.stop())
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
