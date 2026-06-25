<template>
    <div class="ops-dashboard">
        <!-- 第一行：统计卡片 -->
        <a-row
            :gutter="16"
            style="margin-bottom: 16px">
            <a-col
                :xs="12"
                :sm="6">
                <a-card
                    class="ops-stat-card ops-stat-card--blue"
                    :bordered="false">
                    <div class="ops-stat-card__content">
                        <div class="ops-stat-card__value">{{ stats.total_events?.toLocaleString() || 0 }}</div>
                        <div class="ops-stat-card__label">{{ $t('pages.ops.total_events') }}</div>
                    </div>
                    <alert-outlined class="ops-stat-card__icon" />
                </a-card>
            </a-col>
            <a-col
                :xs="12"
                :sm="6">
                <a-card
                    class="ops-stat-card ops-stat-card--red"
                    :bordered="false">
                    <div class="ops-stat-card__content">
                        <div class="ops-stat-card__value">{{ stats.circuit_break_count?.toLocaleString() || 0 }}</div>
                        <div class="ops-stat-card__label">{{ $t('pages.ops.circuit_break') }}</div>
                    </div>
                    <warning-outlined class="ops-stat-card__icon" />
                </a-card>
            </a-col>
            <a-col
                :xs="12"
                :sm="6">
                <a-card
                    class="ops-stat-card ops-stat-card--orange"
                    :bordered="false">
                    <div class="ops-stat-card__content">
                        <div class="ops-stat-card__value">{{ stats.rate_limit_count?.toLocaleString() || 0 }}</div>
                        <div class="ops-stat-card__label">{{ $t('pages.ops.rate_limit') }}</div>
                    </div>
                    <thunderbolt-outlined class="ops-stat-card__icon" />
                </a-card>
            </a-col>
            <a-col
                :xs="12"
                :sm="6">
                <a-card
                    class="ops-stat-card ops-stat-card--purple"
                    :bordered="false">
                    <div class="ops-stat-card__content">
                        <div class="ops-stat-card__value">{{ stats.invocation_fail_count?.toLocaleString() || 0 }}</div>
                        <div class="ops-stat-card__label">{{ $t('pages.ops.invocation_fail') }}</div>
                    </div>
                    <close-circle-outlined class="ops-stat-card__icon" />
                </a-card>
            </a-col>
        </a-row>

        <!-- 趋势图 + 类型分布 -->
        <a-row
            :gutter="16"
            style="margin-bottom: 16px">
            <a-col
                :xs="24"
                :lg="16">
                <a-card
                    :title="$t('pages.ops.trend.title')"
                    :bordered="false"
                    style="border-radius: 8px">
                    <template #extra>
                        <a-select
                            v-model:value="timeRange"
                            style="width: 140px"
                            @change="fetchData">
                            <a-select-option value="1h">{{ $t('pages.dashboard.trends.range.1h') }}</a-select-option>
                            <a-select-option value="6h">{{ $t('pages.dashboard.trends.range.6h') }}</a-select-option>
                            <a-select-option value="24h">{{ $t('pages.dashboard.trends.range.24h') }}</a-select-option>
                            <a-select-option value="7d">{{ $t('pages.dashboard.trends.range.7d') }}</a-select-option>
                            <a-select-option value="today">{{
                                $t('pages.dashboard.trends.range.today')
                            }}</a-select-option>
                        </a-select>
                    </template>
                    <x-chart
                        :options="trendChartOptions"
                        height="320"
                        :loading="loading" />
                </a-card>
            </a-col>
            <a-col
                :xs="24"
                :lg="8">
                <a-card
                    :title="$t('pages.ops.distribution.title')"
                    :bordered="false"
                    style="border-radius: 8px">
                    <x-chart
                        :options="distributionChartOptions"
                        height="320"
                        :loading="loading" />
                </a-card>
            </a-col>
        </a-row>

        <!-- 第三行：租户排行 + 模型排行 -->
        <a-row
            :gutter="16"
            style="margin-bottom: 16px">
            <a-col
                :xs="24"
                :lg="12">
                <a-card
                    :title="$t('pages.ops.tenant_ranking')"
                    :bordered="false"
                    style="border-radius: 8px">
                    <x-chart
                        :options="tenantRankingOptions"
                        height="280"
                        :loading="loading" />
                </a-card>
            </a-col>
            <a-col
                :xs="24"
                :lg="12">
                <a-card
                    :title="$t('pages.ops.model_ranking')"
                    :bordered="false"
                    style="border-radius: 8px">
                    <x-chart
                        :options="modelRankingOptions"
                        height="280"
                        :loading="loading" />
                </a-card>
            </a-col>
        </a-row>

        <!-- 第四行：事件列表 -->
        <a-card
            :bordered="false"
            style="border-radius: 8px">
            <!-- 筛选栏 -->
            <a-form
                layout="inline"
                style="margin-bottom: 16px">
                <a-form-item>
                    <a-select
                        v-model:value="filterForm.event_type"
                        allow-clear
                        style="width: 160px"
                        :placeholder="$t('pages.ops.filter.event_type')">
                        <a-select-option value="circuit_break">{{ $t('pages.ops.circuit_break') }}</a-select-option>
                        <a-select-option value="rate_limit">{{ $t('pages.ops.rate_limit') }}</a-select-option>
                        <a-select-option value="invocation_fail">{{ $t('pages.ops.invocation_fail') }}</a-select-option>
                        <a-select-option value="lb_switch">{{ $t('pages.ops.lb_switch') }}</a-select-option>
                    </a-select>
                </a-form-item>
                <a-form-item>
                    <a-input-group
                        :compact="true"
                        style="display: inline-block; vertical-align: middle">
                        <a-select
                            v-model:value="searchType2"
                            style="width: 105px">
                            <a-select-option value="tenant_code">{{ $t('pages.ops.filter.tenant') }}</a-select-option>
                            <a-select-option value="provider_name">{{
                                $t('pages.ops.filter.provider')
                            }}</a-select-option>
                        </a-select>
                        <a-input
                            v-model:value="searchValue2"
                            allow-clear
                            style="width: 150px"
                            :placeholder="
                                searchType2 === 'tenant_code'
                                    ? $t('pages.ops.filter.tenant')
                                    : $t('pages.ops.filter.provider')
                            " />
                    </a-input-group>
                </a-form-item>
                <a-form-item>
                    <a-input-group
                        :compact="true"
                        style="display: inline-block; vertical-align: middle">
                        <a-select
                            v-model:value="searchType"
                            style="width: 105px">
                            <a-select-option value="model_code">{{ $t('pages.ops.filter.model') }}</a-select-option>
                            <a-select-option value="endpoint_code">{{
                                $t('pages.ops.filter.endpoint_code')
                            }}</a-select-option>
                        </a-select>
                        <a-input
                            v-model:value="searchValue"
                            allow-clear
                            style="width: 150px"
                            :placeholder="
                                searchType === 'model_code'
                                    ? $t('pages.ops.filter.model')
                                    : $t('pages.ops.filter.endpoint_code')
                            " />
                    </a-input-group>
                </a-form-item>
                <a-form-item>
                    <a-range-picker
                        v-model:value="searchTimeRange"
                        show-time
                        allow-clear
                        style="width: 380px" />
                </a-form-item>
                <a-form-item>
                    <x-filter-actions
                        @reset="handleResetSearch"
                        @search="handleSearch" />
                </a-form-item>
            </a-form>

            <!-- 事件表格 -->
            <a-table
                :columns="columns"
                :data-source="eventList"
                :loading="tableLoading"
                :pagination="pagination"
                row-key="id"
                size="middle"
                :scroll="{ x: 1000 }"
                @change="onTableChange">
                <template #bodyCell="{ column, record }">
                    <template v-if="column.key === 'event_type'">
                        <a-tag :color="eventTypeColor(record.event_type)">
                            {{ eventTypeName(record.event_type) }}
                        </a-tag>
                    </template>
                    <template v-else-if="column.key === 'event_time'">
                        {{ formatTime(record.event_time) }}
                    </template>
                </template>
                <template #expandedRowRender="{ record }">
                    <div class="ops-expanded-container">
                        <a-descriptions
                            :column="2"
                            size="small"
                            bordered>
                            <a-descriptions-item
                                :label="$t('pages.ops.table.policy_id')"
                                v-if="record.policy_id">
                                <a-typography-text
                                    copyable
                                    :ellipsis="{ tooltip: true }"
                                    style="font-family: monospace; max-width: 200px">
                                    {{ record.policy_id }}
                                </a-typography-text>
                            </a-descriptions-item>
                            <a-descriptions-item
                                :label="$t('pages.ops.table.endpoint_code')"
                                v-if="record.endpoint_code">
                                <a-typography-text
                                    copyable
                                    :ellipsis="{ tooltip: true }"
                                    style="font-family: monospace; max-width: 200px">
                                    {{ record.endpoint_code }}
                                </a-typography-text>
                            </a-descriptions-item>
                            <a-descriptions-item
                                :label="$t('pages.ops.table.endpoint_id')"
                                v-if="record.endpoint_id">
                                <a-typography-text
                                    copyable
                                    :ellipsis="{ tooltip: true }"
                                    style="font-family: monospace; max-width: 200px">
                                    {{ record.endpoint_id }}
                                </a-typography-text>
                            </a-descriptions-item>
                            <a-descriptions-item
                                :label="$t('pages.ops.table.request_id')"
                                v-if="record.request_id">
                                <a-typography-text
                                    copyable
                                    :ellipsis="{ tooltip: true }"
                                    style="font-family: monospace; max-width: 200px">
                                    {{ record.request_id }}
                                </a-typography-text>
                            </a-descriptions-item>
                            <a-descriptions-item
                                :label="$t('pages.ops.table.trace_id')"
                                v-if="record.trace_id">
                                <a-typography-text
                                    copyable
                                    :ellipsis="{ tooltip: true }"
                                    style="font-family: monospace; max-width: 200px">
                                    {{ record.trace_id }}
                                </a-typography-text>
                            </a-descriptions-item>
                            <a-descriptions-item
                                :label="$t('pages.ops.table.threshold') + ' / ' + $t('pages.ops.table.current_value')"
                                v-if="record.threshold != null || record.current_value != null">
                                <a-tag
                                    :color="
                                        record.current_value != null &&
                                        record.threshold != null &&
                                        record.current_value >= record.threshold
                                            ? 'red'
                                            : 'blue'
                                    ">
                                    <template v-if="record.event_type === 'circuit_break'">
                                        {{
                                            record.threshold != null
                                                ? Number(record.threshold).toFixed(2).replace(/\.00$/, '') + '%'
                                                : '-'
                                        }}
                                        /
                                        {{
                                            record.current_value != null
                                                ? Number(record.current_value).toFixed(2).replace(/\.00$/, '') + '%'
                                                : '-'
                                        }}
                                    </template>
                                    <template v-else>
                                        {{ record.threshold != null ? record.threshold : '-' }} /
                                        {{ record.current_value != null ? record.current_value : '-' }}
                                    </template>
                                </a-tag>
                            </a-descriptions-item>
                            <a-descriptions-item
                                :label="$t('pages.ops.table.detail')"
                                :span="2"
                                v-if="record.message">
                                <div class="ops-expanded-error-msg">
                                    {{ record.message }}
                                </div>
                            </a-descriptions-item>
                        </a-descriptions>
                    </div>
                </template>
            </a-table>
        </a-card>

        <!-- 全局控制工具栏 -->
        <div
            v-if="isAdmin"
            class="cache-control-toolbar">
            <div style="margin-right: auto; display: flex; align-items: center">
                <sync-outlined style="font-size: 16px; color: var(--color-primary); margin-right: 8px" />
                <span class="cache-control-title">{{ $t('pages.dashboard.cache.title') }}</span>
            </div>
            <a-button
                type="primary"
                ghost
                :loading="syncing"
                @click="handleSyncRedis"
                style="border-radius: 6px; font-weight: 500">
                <template #icon><sync-outlined /></template>
                {{ $t('pages.dashboard.cache.sync') }}
            </a-button>
        </div>
    </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/store'
import useUserStore from '@/store/modules/user'
import {
    AlertOutlined,
    WarningOutlined,
    ThunderboltOutlined,
    CloseCircleOutlined,
    SyncOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import apis from '@/apis'
import { config } from '@/config'

const { t } = useI18n()
const appStore = useAppStore()
const userStore = useUserStore()
const isAdmin = computed(() => userStore.userInfo?.username === 'admin')
const syncing = ref(false)

async function handleSyncRedis() {
    syncing.value = true
    try {
        const res = await apis.dashboard.syncRedis()
        if (res && res.success) {
            message.success(t('pages.dashboard.cache.sync.success'))
            await Promise.all([fetchData(), fetchEvents()])
        }
    } catch (e) {
        console.error(e)
    } finally {
        syncing.value = false
    }
}

// State
const loading = ref(false)
const tableLoading = ref(false)
const timeRange = ref('24h')
const stats = ref({})
const eventList = ref([])
const searchTimeRange = ref(null)

const filterForm = reactive({
    event_type: undefined,
    tenant_code: '',
    model_code: '',
    provider_name: '',
    endpoint_code: '',
})

// 组合查询条件1：选择维度（租户/供应商）
const searchType2 = ref('tenant_code')
const searchValue2 = ref('')

watch([searchType2, searchValue2], ([type, val]) => {
    if (type === 'tenant_code') {
        filterForm.tenant_code = val ? val.trim() : ''
        filterForm.provider_name = ''
    } else {
        filterForm.provider_name = val ? val.trim() : ''
        filterForm.tenant_code = ''
    }
})

// 组合查询条件2：选择维度（模型编码/端点编码）
const searchType = ref('model_code')
const searchValue = ref('')

watch([searchType, searchValue], ([type, val]) => {
    if (type === 'model_code') {
        filterForm.model_code = val ? val.trim() : ''
        filterForm.endpoint_code = ''
    } else {
        filterForm.endpoint_code = val ? val.trim() : ''
        filterForm.model_code = ''
    }
})

const pagination = reactive({
    current: 1,
    pageSize: 20,
    total: 0,
    showSizeChanger: true,
    showTotal: (total) => t('common.pagination.total', { total }),
})

const matchesFilter = (evt) => {
    if (filterForm.event_type && evt.event_type !== filterForm.event_type) return false
    if (filterForm.tenant_code && !evt.tenant_code?.toLowerCase().includes(filterForm.tenant_code.toLowerCase()))
        return false
    if (filterForm.model_code && !evt.model_code?.toLowerCase().includes(filterForm.model_code.toLowerCase()))
        return false
    if (filterForm.provider_name && !evt.provider_name?.toLowerCase().includes(filterForm.provider_name.toLowerCase()))
        return false
    if (filterForm.endpoint_code && !evt.endpoint_code?.toLowerCase().includes(filterForm.endpoint_code.toLowerCase()))
        return false
    return true
}

// WebSocket reconnect state
let ws = null
let wsReconnectTimer = null
let wsReconnectDelay = 1000
let wsManualClose = false

// Table columns
const columns = computed(() => [
    { title: t('pages.ops.table.time'), key: 'event_time', dataIndex: 'event_time', width: 140 },
    { title: t('pages.ops.table.type'), key: 'event_type', dataIndex: 'event_type', width: 80 },
    { title: t('pages.ops.table.tenant'), dataIndex: 'tenant_code', width: 70, ellipsis: true },
    { title: t('pages.ops.table.model'), dataIndex: 'model_code', width: 120, ellipsis: true },
    { title: t('pages.ops.table.endpoint_code'), dataIndex: 'endpoint_code', width: 120, ellipsis: true },
    { title: t('pages.ops.table.provider'), dataIndex: 'provider_name', width: 120, ellipsis: true },
    { title: t('pages.ops.table.policy_name'), dataIndex: 'policy_name', width: 160, ellipsis: true },
])

// Event type helpers
const eventTypeName = (type) => {
    const map = {
        circuit_break: t('pages.ops.circuit_break'),
        rate_limit: t('pages.ops.rate_limit'),
        invocation_fail: t('pages.ops.invocation_fail'),
        lb_switch: t('pages.ops.lb_switch'),
    }
    return map[type] || type
}

const eventTypeColor = (type) => {
    const map = {
        circuit_break: 'error',
        rate_limit: 'warning',
        invocation_fail: 'purple',
        lb_switch: 'blue',
    }
    return map[type] || 'default'
}

const formatTime = (val) => {
    if (!val) return '-'
    const d = new Date(val)
    return d.toLocaleString()
}

// Data fetching
const fetchData = async () => {
    loading.value = true
    try {
        const { data, success } = await apis.ops.getEventStatistics({ time_range: timeRange.value })
        if (success && data) {
            stats.value = data
        }
    } catch (e) {
        // Error handled by interceptor
    } finally {
        loading.value = false
    }
}

const fetchEvents = async () => {
    tableLoading.value = true
    try {
        const params = {
            current: pagination.current,
            pageSize: pagination.pageSize,
            ...filterForm,
        }
        if (searchTimeRange.value && searchTimeRange.value.length === 2) {
            params.start_time = searchTimeRange.value[0].toISOString()
            params.end_time = searchTimeRange.value[1].toISOString()
        }
        // Clean empty params
        Object.keys(params).forEach((key) => {
            if (params[key] === '' || params[key] === undefined || params[key] === null) {
                delete params[key]
            }
        })
        const { data, total, success } = await apis.ops.getEvents(params)
        if (success) {
            eventList.value = data || []
            pagination.total = total || 0
        }
    } catch (e) {
        // Error handled by interceptor
    } finally {
        tableLoading.value = false
    }
}

const handleSearch = () => {
    pagination.current = 1
    fetchEvents()
}

const handleResetSearch = () => {
    filterForm.event_type = undefined
    searchType2.value = 'tenant_code'
    searchValue2.value = ''
    searchType.value = 'model_code'
    searchValue.value = ''
    searchTimeRange.value = null
    pagination.current = 1
    fetchEvents()
}

const onTableChange = ({ current, pageSize }) => {
    pagination.current = current
    pagination.pageSize = pageSize
    fetchEvents()
}

// WebSocket
const connectWebSocket = () => {
    if (ws) {
        wsManualClose = true
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
    const wsUrl = `${protocol}//${host}${apiBasic}/api/v1/ops/events/ws?accessToken=${token}`

    ws = new WebSocket(wsUrl)

    ws.onopen = () => {
        wsReconnectDelay = 1000
    }

    ws.onmessage = (event) => {
        try {
            const newEvent = JSON.parse(event.data)
            // Always update stats counters so cards stay live
            stats.value.total_events = (stats.value.total_events || 0) + 1
            const countKey = newEvent.event_type + '_count'
            if (stats.value[countKey] !== undefined) {
                stats.value[countKey]++
            }
            // Only inject into the table when viewing the first page and the
            // event matches the active filters, otherwise pagination/filtering breaks.
            if (pagination.current === 1 && matchesFilter(newEvent)) {
                eventList.value.unshift(newEvent)
                if (eventList.value.length > pagination.pageSize) {
                    eventList.value.pop()
                }
            }
        } catch (e) {
            // Ignore parse errors
        }
    }

    ws.onclose = () => {
        if (wsManualClose) return
        wsReconnectTimer = setTimeout(() => {
            wsReconnectDelay = Math.min(wsReconnectDelay * 2, 30000)
            connectWebSocket()
        }, wsReconnectDelay)
    }

    ws.onerror = () => {
        ws.close()
    }
}

// Chart options
const isDark = computed(() => appStore.config.theme === 'dark')

const trendChartOptions = computed(() => {
    const trend = stats.value.trend || []
    const times = trend.map((p) => p.time?.split(' ')[1] || p.time || '')
    const isD = isDark.value
    return {
        tooltip: { trigger: 'axis' },
        legend: {
            data: [
                t('pages.ops.circuit_break'),
                t('pages.ops.rate_limit'),
                t('pages.ops.invocation_fail'),
                t('pages.ops.lb_switch'),
            ],
            textStyle: { color: isD ? 'rgba(255, 255, 255, 0.65)' : '#333' },
        },
        grid: { left: 50, right: 20, bottom: 30, top: 40 },
        xAxis: {
            type: 'category',
            data: times,
            axisLabel: { color: isD ? 'rgba(255, 255, 255, 0.45)' : '#666' },
        },
        yAxis: {
            type: 'value',
            axisLabel: { color: isD ? 'rgba(255, 255, 255, 0.45)' : '#666' },
            splitLine: { lineStyle: { color: isD ? 'rgba(255, 255, 255, 0.06)' : 'rgba(0, 0, 0, 0.06)' } },
        },
        series: [
            {
                name: t('pages.ops.circuit_break'),
                type: 'line',
                smooth: true,
                data: trend.map((p) => p.circuit_break || 0),
                itemStyle: { color: '#ff4d4f' },
                areaStyle: { opacity: 0.1 },
            },
            {
                name: t('pages.ops.rate_limit'),
                type: 'line',
                smooth: true,
                data: trend.map((p) => p.rate_limit || 0),
                itemStyle: { color: '#faad14' },
                areaStyle: { opacity: 0.1 },
            },
            {
                name: t('pages.ops.invocation_fail'),
                type: 'line',
                smooth: true,
                data: trend.map((p) => p.invocation_fail || 0),
                itemStyle: { color: '#722ed1' },
                areaStyle: { opacity: 0.1 },
            },
            {
                name: t('pages.ops.lb_switch'),
                type: 'line',
                smooth: true,
                data: trend.map((p) => p.lb_switch || 0),
                itemStyle: { color: '#1890ff' },
                areaStyle: { opacity: 0.1 },
            },
        ],
    }
})

const distributionChartOptions = computed(() => {
    const isD = isDark.value
    return {
        tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
        legend: {
            orient: 'vertical',
            right: 10,
            top: 'center',
            textStyle: { color: isD ? 'rgba(255, 255, 255, 0.65)' : '#333' },
        },
        series: [
            {
                type: 'pie',
                radius: ['40%', '70%'],
                center: ['40%', '50%'],
                avoidLabelOverlap: false,
                label: { show: false },
                data: [
                    {
                        value: stats.value.circuit_break_count || 0,
                        name: t('pages.ops.circuit_break'),
                        itemStyle: { color: '#ff4d4f' },
                    },
                    {
                        value: stats.value.rate_limit_count || 0,
                        name: t('pages.ops.rate_limit'),
                        itemStyle: { color: '#faad14' },
                    },
                    {
                        value: stats.value.invocation_fail_count || 0,
                        name: t('pages.ops.invocation_fail'),
                        itemStyle: { color: '#722ed1' },
                    },
                    {
                        value: stats.value.lb_switch_count || 0,
                        name: t('pages.ops.lb_switch'),
                        itemStyle: { color: '#1890ff' },
                    },
                ],
            },
        ],
    }
})

const tenantRankingOptions = computed(() => {
    const ranking = stats.value.tenant_ranking || []
    const isD = isDark.value
    return {
        tooltip: { trigger: 'axis' },
        grid: { left: 120, right: 40, bottom: 10, top: 10 },
        xAxis: {
            type: 'value',
            name: t('pages.circuitBreak.form.slidingWindow.unit.count'),
            axisLabel: { color: isD ? 'rgba(255, 255, 255, 0.45)' : '#666' },
            splitLine: { lineStyle: { color: isD ? 'rgba(255, 255, 255, 0.06)' : 'rgba(0, 0, 0, 0.06)' } },
        },
        yAxis: {
            type: 'category',
            data: ranking.map((r) => r.name).reverse(),
            axisLabel: { color: isD ? 'rgba(255, 255, 255, 0.65)' : '#666' },
        },
        series: [
            {
                name: t('pages.ops.event_count'),
                type: 'bar',
                barWidth: 16,
                showBackground: true,
                backgroundStyle: {
                    color: isD ? 'rgba(255, 255, 255, 0.03)' : 'rgba(0, 0, 0, 0.02)',
                },
                label: {
                    show: true,
                    position: 'right',
                    color: isD ? 'rgba(255, 255, 255, 0.65)' : '#666',
                },
                data: ranking.map((r) => r.count).reverse(),
                itemStyle: { color: '#1890ff', borderRadius: [0, 4, 4, 0] },
            },
        ],
    }
})

const modelRankingOptions = computed(() => {
    const ranking = stats.value.model_ranking || []
    const isD = isDark.value
    return {
        tooltip: { trigger: 'axis' },
        grid: { left: 120, right: 40, bottom: 10, top: 10 },
        xAxis: {
            type: 'value',
            name: t('pages.circuitBreak.form.slidingWindow.unit.count'),
            axisLabel: { color: isD ? 'rgba(255, 255, 255, 0.45)' : '#666' },
            splitLine: { lineStyle: { color: isD ? 'rgba(255, 255, 255, 0.06)' : 'rgba(0, 0, 0, 0.06)' } },
        },
        yAxis: {
            type: 'category',
            data: ranking.map((r) => r.name).reverse(),
            axisLabel: { color: isD ? 'rgba(255, 255, 255, 0.65)' : '#666' },
        },
        series: [
            {
                name: t('pages.ops.event_count'),
                type: 'bar',
                barWidth: 16,
                showBackground: true,
                backgroundStyle: {
                    color: isD ? 'rgba(255, 255, 255, 0.03)' : 'rgba(0, 0, 0, 0.02)',
                },
                label: {
                    show: true,
                    position: 'right',
                    color: isD ? 'rgba(255, 255, 255, 0.65)' : '#666',
                },
                data: ranking.map((r) => r.count).reverse(),
                itemStyle: { color: '#722ed1', borderRadius: [0, 4, 4, 0] },
            },
        ],
    }
})

// Lifecycle
onMounted(async () => {
    await Promise.all([fetchData(), fetchEvents()])
    connectWebSocket()
})

onUnmounted(() => {
    wsManualClose = true
    if (wsReconnectTimer) {
        clearTimeout(wsReconnectTimer)
    }
    if (ws) {
        ws.close()
        ws = null
    }
})
</script>

<style scoped>
.ops-dashboard {
    padding: 0;
}

/* 统计卡片 — 现代化、轻量化与高保真暗黑模式配色 */
.ops-stat-card {
    border-radius: 12px;
    background: var(--color-bg-container, #ffffff);
    border: 1px solid var(--color-border-secondary, rgba(0, 0, 0, 0.06));
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.02);
    transition: all 0.3s cubic-bezier(0.25, 0.8, 0.25, 1);
    position: relative;
    overflow: hidden;
    margin-bottom: 16px;
}

[data-theme='dark'] .ops-stat-card {
    background: #141722;
    border-color: rgba(255, 255, 255, 0.05);
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
}

.ops-stat-card:hover {
    transform: translateY(-4px);
    box-shadow: 0 12px 24px rgba(0, 0, 0, 0.08);
}

[data-theme='dark'] .ops-stat-card:hover {
    box-shadow: 0 12px 24px rgba(0, 0, 0, 0.3);
    border-color: var(--color-primary, #7c5cfc);
}

/* 顶部加一条精致的彩色边框线 */
.ops-stat-card::before {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 4px;
}

.ops-stat-card--blue::before {
    background: linear-gradient(90deg, #1890ff, #40a9ff);
}

.ops-stat-card--red::before {
    background: linear-gradient(90deg, #ff4d4f, #ff7875);
}

.ops-stat-card--orange::before {
    background: linear-gradient(90deg, #faad14, #ffc53d);
}

.ops-stat-card--purple::before {
    background: linear-gradient(90deg, #722ed1, #b37feb);
}

.ops-stat-card :deep(.ant-card-body) {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 24px 20px;
    background: transparent;
}

.ops-stat-card__value {
    font-size: 28px;
    font-weight: 700;
    line-height: 1.2;
    color: var(--color-text-primary, #1f1f1f);
}

[data-theme='dark'] .ops-stat-card__value {
    color: #e0dbff;
}

.ops-stat-card__label {
    font-size: 14px;
    color: var(--color-text-secondary, #8c8c8c);
    margin-top: 4px;
}

/* 统计卡片的图标：右侧展示，带有圆形气泡背景 */
.ops-stat-card__icon {
    font-size: 22px;
    padding: 12px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
}

.ops-stat-card--blue .ops-stat-card__icon {
    color: #1890ff;
    background: rgba(24, 144, 255, 0.1);
}

.ops-stat-card--red .ops-stat-card__icon {
    color: #ff4d4f;
    background: rgba(255, 77, 79, 0.1);
}

.ops-stat-card--orange .ops-stat-card__icon {
    color: #faad14;
    background: rgba(250, 173, 20, 0.1);
}

.ops-stat-card--purple .ops-stat-card__icon {
    color: #722ed1;
    background: rgba(114, 46, 209, 0.1);
}

.ops-detail-text {
    color: rgba(0, 0, 0, 0.45);
}

[data-theme='dark'] .ops-detail-text {
    color: rgba(255, 255, 255, 0.45);
}

/* 行展开详情容器样式 */
.ops-expanded-container {
    padding: 16px 24px;
    background: #fafafa;
    border-radius: 6px;
    border: 1px solid rgba(0, 0, 0, 0.05);
}

[data-theme='dark'] .ops-expanded-container {
    background: #141414;
    border-color: rgba(255, 255, 255, 0.05);
}

/* 行展开 descriptions 组件暗黑模式深度覆盖 */
[data-theme='dark'] .ops-expanded-container :deep(.ant-descriptions-bordered .ant-descriptions-item-label) {
    background-color: #1c1c1e;
    color: rgba(255, 255, 255, 0.85);
    border-right-color: #303030;
}

[data-theme='dark'] .ops-expanded-container :deep(.ant-descriptions-bordered .ant-descriptions-item-content) {
    background-color: #141414;
    color: rgba(255, 255, 255, 0.65);
    border-right-color: #303030;
}

[data-theme='dark'] .ops-expanded-container :deep(.ant-descriptions-bordered .ant-descriptions-row) {
    border-bottom-color: #303030;
}

[data-theme='dark'] .ops-expanded-container :deep(.ant-descriptions-title) {
    color: rgba(255, 255, 255, 0.85);
}

/* 异常详情文字折行及样式 */
.ops-expanded-error-msg {
    max-height: 120px;
    overflow-y: auto;
    white-space: pre-wrap;
    word-break: break-all;
    color: #ff4d4f;
    font-family: monospace;
    font-size: 12px;
    line-height: 1.5;
}

[data-theme='dark'] .ops-expanded-error-msg {
    color: #ff7875;
}

/* 缓存控制工具栏 */
.cache-control-toolbar {
    display: flex;
    justify-content: flex-end;
    align-items: center;
    margin-top: 16px;
    padding: 12px 16px;
    border-radius: 8px;
    background: var(--color-bg-container);
    border: 1px solid var(--color-border);
    box-shadow: var(--shadow-sm);
}

.cache-control-title {
    font-weight: 500;
    font-size: 14px;
    color: var(--color-text-primary);
}
</style>
