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
                    class="ops-panel"
                    :title="$t('pages.ops.trend.title')"
                    :bordered="false">
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
                    class="ops-panel"
                    :title="$t('pages.ops.distribution.title')"
                    :bordered="false">
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
                    class="ops-panel"
                    :title="$t('pages.ops.tenant_ranking')"
                    :bordered="false">
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
                    class="ops-panel"
                    :title="$t('pages.ops.model_ranking')"
                    :bordered="false">
                    <x-chart
                        :options="modelRankingOptions"
                        height="280"
                        :loading="loading" />
                </a-card>
            </a-col>
        </a-row>

        <!-- 第四行：事件列表 -->
        <a-card
            class="ops-panel ops-table-panel"
            :bordered="false">
            <!-- 筛选栏 -->
            <a-form
                class="ops-filter-form"
                layout="inline"
                style="margin-bottom: 12px">
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
                    <template v-else-if="column.key === 'model_code'">
                        <a
                            v-if="record.model_code && modelMap[record.model_code]"
                            @click="goToModelDetail(record.model_code)"
                            style="cursor: pointer">
                            {{ record.model_code }}
                        </a>
                        <span v-else>{{ record.model_code || '--' }}</span>
                    </template>
                </template>
                <template #expandedRowRender="{ record }">
                    <div class="ops-expanded-container">
                        <a-descriptions
                            class="ops-expanded-descriptions"
                            :column="2"
                            size="small"
                            bordered>
                            <a-descriptions-item
                                :label="$t('pages.ops.table.policy_id')"
                                v-if="record.policy_id">
                                <a-typography-text
                                    class="ops-expanded-code"
                                    copyable
                                    :ellipsis="{ tooltip: true }">
                                    {{ record.policy_id }}
                                </a-typography-text>
                            </a-descriptions-item>
                            <a-descriptions-item
                                :label="$t('pages.ops.table.endpoint_code')"
                                v-if="record.endpoint_code">
                                <a-typography-text
                                    class="ops-expanded-code"
                                    copyable
                                    :ellipsis="{ tooltip: true }">
                                    {{ record.endpoint_code }}
                                </a-typography-text>
                            </a-descriptions-item>
                            <a-descriptions-item
                                :label="$t('pages.ops.table.endpoint_id')"
                                v-if="record.endpoint_id">
                                <a-typography-text
                                    class="ops-expanded-code"
                                    copyable
                                    :ellipsis="{ tooltip: true }">
                                    {{ record.endpoint_id }}
                                </a-typography-text>
                            </a-descriptions-item>
                            <a-descriptions-item
                                :label="$t('pages.ops.table.request_id')"
                                v-if="record.request_id">
                                <a-typography-text
                                    class="ops-expanded-code"
                                    copyable
                                    :ellipsis="{ tooltip: true }">
                                    {{ record.request_id }}
                                </a-typography-text>
                            </a-descriptions-item>
                            <a-descriptions-item
                                :label="$t('pages.ops.table.trace_id')"
                                v-if="record.trace_id">
                                <a-typography-text
                                    class="ops-expanded-code"
                                    copyable
                                    :ellipsis="{ tooltip: true }">
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
import { useRouter } from 'vue-router'
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
const router = useRouter()
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
const modelMap = ref({}) // model_code -> model_id 映射

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
    { title: t('pages.ops.table.model'), key: 'model_code', dataIndex: 'model_code', width: 120, ellipsis: true },
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
        circuit_break: opsChartPalette.error,
        rate_limit: opsChartPalette.warning,
        invocation_fail: opsChartPalette.strategy,
        lb_switch: opsChartPalette.system,
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
const opsChartPalette = {
    system: '#2f8cff',
    traffic: '#23c7b7',
    warning: '#ffb020',
    error: '#ff4d5e',
    strategy: '#8b5cf6',
}

function chartTextColor(isD, level = 'secondary') {
    if (level === 'primary') return isD ? 'rgba(238, 244, 255, 0.92)' : '#162033'
    return isD ? 'rgba(196, 207, 224, 0.62)' : 'rgba(22, 32, 51, 0.62)'
}

function chartTooltip(isD, trigger = 'axis') {
    return {
        trigger,
        backgroundColor: isD ? 'rgba(13, 19, 32, 0.94)' : 'rgba(255, 255, 255, 0.96)',
        borderColor: isD ? 'rgba(148, 163, 184, 0.18)' : 'rgba(22, 32, 51, 0.1)',
        borderWidth: 1,
        padding: [10, 12],
        textStyle: {
            color: chartTextColor(isD, 'primary'),
            fontSize: 12,
        },
        extraCssText: 'border-radius: 8px; box-shadow: 0 16px 36px rgba(0, 0, 0, 0.22);',
    }
}

function axisStyle(isD) {
    return {
        axisLine: { lineStyle: { color: isD ? 'rgba(148, 163, 184, 0.18)' : 'rgba(22, 32, 51, 0.12)' } },
        axisTick: { lineStyle: { color: isD ? 'rgba(148, 163, 184, 0.16)' : 'rgba(22, 32, 51, 0.12)' } },
        axisLabel: { color: chartTextColor(isD) },
    }
}

function splitLineStyle(isD) {
    return {
        lineStyle: {
            color: isD ? 'rgba(148, 163, 184, 0.08)' : 'rgba(22, 32, 51, 0.07)',
        },
    }
}

const trendChartOptions = computed(() => {
    const trend = stats.value.trend || []
    const times = trend.map((p) => p.time?.split(' ')[1] || p.time || '')
    const isD = isDark.value
    return {
        color: [opsChartPalette.error, opsChartPalette.warning, opsChartPalette.strategy, opsChartPalette.system],
        tooltip: {
            ...chartTooltip(isD),
            axisPointer: {
                type: 'line',
                lineStyle: { color: isD ? 'rgba(148, 163, 184, 0.32)' : 'rgba(47, 140, 255, 0.22)' },
            },
        },
        legend: {
            data: [
                t('pages.ops.circuit_break'),
                t('pages.ops.rate_limit'),
                t('pages.ops.invocation_fail'),
                t('pages.ops.lb_switch'),
            ],
            top: 6,
            right: 8,
            icon: 'roundRect',
            itemWidth: 12,
            itemHeight: 4,
            textStyle: { color: chartTextColor(isD), fontSize: 12 },
        },
        grid: { left: 50, right: 20, bottom: 30, top: 48 },
        xAxis: {
            type: 'category',
            data: times,
            boundaryGap: false,
            ...axisStyle(isD),
        },
        yAxis: {
            type: 'value',
            ...axisStyle(isD),
            splitLine: splitLineStyle(isD),
        },
        series: [
            {
                name: t('pages.ops.circuit_break'),
                type: 'line',
                smooth: true,
                symbol: 'circle',
                symbolSize: 5,
                data: trend.map((p) => p.circuit_break || 0),
                itemStyle: { color: opsChartPalette.error },
                lineStyle: { width: 2 },
            },
            {
                name: t('pages.ops.rate_limit'),
                type: 'line',
                smooth: true,
                symbol: 'circle',
                symbolSize: 5,
                data: trend.map((p) => p.rate_limit || 0),
                itemStyle: { color: opsChartPalette.warning },
                lineStyle: { width: 2 },
            },
            {
                name: t('pages.ops.invocation_fail'),
                type: 'line',
                smooth: true,
                symbol: 'circle',
                symbolSize: 5,
                data: trend.map((p) => p.invocation_fail || 0),
                itemStyle: { color: opsChartPalette.strategy },
                lineStyle: { width: 2.4 },
                areaStyle: {
                    color: opsChartPalette.strategy,
                    opacity: isD ? 0.12 : 0.08,
                },
            },
            {
                name: t('pages.ops.lb_switch'),
                type: 'line',
                smooth: true,
                symbol: 'circle',
                symbolSize: 5,
                data: trend.map((p) => p.lb_switch || 0),
                itemStyle: { color: opsChartPalette.system },
                lineStyle: { width: 2 },
            },
        ],
    }
})

const distributionChartOptions = computed(() => {
    const isD = isDark.value
    return {
        color: [opsChartPalette.error, opsChartPalette.warning, opsChartPalette.strategy, opsChartPalette.system],
        tooltip: { ...chartTooltip(isD, 'item'), formatter: '{b}: {c} ({d}%)' },
        legend: {
            orient: 'vertical',
            right: 16,
            top: 'center',
            icon: 'roundRect',
            itemWidth: 12,
            itemHeight: 8,
            itemGap: 14,
            textStyle: { color: chartTextColor(isD), fontSize: 12 },
        },
        series: [
            {
                type: 'pie',
                radius: ['46%', '68%'],
                center: ['38%', '50%'],
                avoidLabelOverlap: false,
                label: { show: false },
                itemStyle: {
                    borderColor: isD ? '#111827' : '#ffffff',
                    borderWidth: 2,
                },
                data: [
                    {
                        value: stats.value.circuit_break_count || 0,
                        name: t('pages.ops.circuit_break'),
                    },
                    {
                        value: stats.value.rate_limit_count || 0,
                        name: t('pages.ops.rate_limit'),
                    },
                    {
                        value: stats.value.invocation_fail_count || 0,
                        name: t('pages.ops.invocation_fail'),
                    },
                    {
                        value: stats.value.lb_switch_count || 0,
                        name: t('pages.ops.lb_switch'),
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
        tooltip: chartTooltip(isD),
        grid: { left: 110, right: 42, bottom: 12, top: 14 },
        xAxis: {
            type: 'value',
            name: t('pages.circuitBreak.form.slidingWindow.unit.count'),
            nameTextStyle: { color: chartTextColor(isD), padding: [0, 0, 0, 8] },
            ...axisStyle(isD),
            splitLine: splitLineStyle(isD),
        },
        yAxis: {
            type: 'category',
            data: ranking.map((r) => r.name).reverse(),
            ...axisStyle(isD),
        },
        series: [
            {
                name: t('pages.ops.event_count'),
                type: 'bar',
                barWidth: 14,
                showBackground: true,
                backgroundStyle: {
                    color: isD ? 'rgba(148, 163, 184, 0.06)' : 'rgba(22, 32, 51, 0.04)',
                },
                label: {
                    show: true,
                    position: 'right',
                    color: chartTextColor(isD),
                    fontSize: 11,
                },
                data: ranking.map((r) => r.count).reverse(),
                itemStyle: { color: opsChartPalette.system, borderRadius: [0, 6, 6, 0] },
            },
        ],
    }
})

const modelRankingOptions = computed(() => {
    const ranking = stats.value.model_ranking || []
    const isD = isDark.value
    return {
        tooltip: chartTooltip(isD),
        grid: { left: 120, right: 42, bottom: 12, top: 14 },
        xAxis: {
            type: 'value',
            name: t('pages.circuitBreak.form.slidingWindow.unit.count'),
            nameTextStyle: { color: chartTextColor(isD), padding: [0, 0, 0, 8] },
            ...axisStyle(isD),
            splitLine: splitLineStyle(isD),
        },
        yAxis: {
            type: 'category',
            data: ranking.map((r) => r.name).reverse(),
            ...axisStyle(isD),
        },
        series: [
            {
                name: t('pages.ops.event_count'),
                type: 'bar',
                barWidth: 14,
                showBackground: true,
                backgroundStyle: {
                    color: isD ? 'rgba(148, 163, 184, 0.06)' : 'rgba(22, 32, 51, 0.04)',
                },
                label: {
                    show: true,
                    position: 'right',
                    color: chartTextColor(isD),
                    fontSize: 11,
                },
                data: ranking.map((r) => r.count).reverse(),
                itemStyle: { color: opsChartPalette.strategy, borderRadius: [0, 6, 6, 0] },
            },
        ],
    }
})

// Lifecycle
onMounted(async () => {
    await Promise.all([fetchData(), fetchEvents(), loadModelMap()])
    connectWebSocket()
})

// 加载模型映射（model_code -> model_id）
async function loadModelMap() {
    try {
        const { data, success } = await apis.model.getModelList({ pageSize: 1000, current: 1 })
        if (success && data) {
            const map = {}
            data.forEach((m) => {
                map[m.model_code] = m.id
            })
            modelMap.value = map
        }
    } catch (e) {
        // ignore
    }
}

// 跳转到模型详情
function goToModelDetail(modelCode) {
    const modelId = modelMap.value[modelCode]
    if (modelId) {
        router.push({ name: 'modelDetail', params: { id: modelId } })
    }
}

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
    --ops-bg-panel: #ffffff;
    --ops-bg-panel-soft: #f7f9fc;
    --ops-bg-control: rgba(248, 250, 252, 0.94);
    --ops-border: rgba(22, 32, 51, 0.08);
    --ops-border-strong: rgba(22, 32, 51, 0.14);
    --ops-text: #162033;
    --ops-text-muted: rgba(22, 32, 51, 0.58);
    --ops-filter-bg: rgba(248, 250, 252, 0.76);
    --ops-panel-glow: rgba(47, 140, 255, 0.06);
    --ops-card-shadow: rgba(22, 32, 51, 0.08);
    --ops-card-shadow-hover: rgba(22, 32, 51, 0.12);
    --ops-table-head-bg: rgba(22, 32, 51, 0.035);
    --ops-table-head-text: rgba(22, 32, 51, 0.68);
    --ops-table-row-border: rgba(22, 32, 51, 0.06);
    --ops-expanded-bg: rgba(248, 250, 252, 0.88);
    --ops-expanded-cell-bg: #ffffff;
    --ops-expanded-label-bg: rgba(47, 140, 255, 0.055);
    --ops-shine: rgba(47, 140, 255, 0.08);
    --ops-blue: #2f8cff;
    --ops-cyan: #23c7b7;
    --ops-amber: #ffb020;
    --ops-red: #ff4d5e;
    --ops-purple: #8b5cf6;
}

[data-theme='dark'] .ops-dashboard {
    --ops-bg-panel: #111827;
    --ops-bg-panel-soft: #151c2b;
    --ops-bg-control: rgba(15, 23, 42, 0.82);
    --ops-border: rgba(148, 163, 184, 0.1);
    --ops-border-strong: rgba(148, 163, 184, 0.16);
    --ops-text: #eef4ff;
    --ops-text-muted: rgba(196, 207, 224, 0.62);
    --ops-filter-bg: rgba(15, 23, 42, 0.36);
    --ops-panel-glow: rgba(255, 255, 255, 0.025);
    --ops-card-shadow: rgba(0, 0, 0, 0.16);
    --ops-card-shadow-hover: rgba(0, 0, 0, 0.24);
    --ops-table-head-bg: rgba(148, 163, 184, 0.065);
    --ops-table-head-text: rgba(238, 244, 255, 0.72);
    --ops-table-row-border: rgba(148, 163, 184, 0.07);
    --ops-expanded-bg: rgba(15, 23, 42, 0.72);
    --ops-expanded-cell-bg: rgba(15, 23, 42, 0.48);
    --ops-expanded-label-bg: rgba(148, 163, 184, 0.07);
    --ops-shine: rgba(255, 255, 255, 0.055);
}

.ops-panel {
    border: 1px solid var(--ops-border);
    border-radius: 8px;
    background: linear-gradient(180deg, var(--ops-panel-glow), transparent 52%), var(--ops-bg-panel);
    box-shadow: 0 16px 44px var(--ops-card-shadow);
    overflow: hidden;
}

.ops-panel :deep(.ant-card-head) {
    min-height: 44px;
    padding: 0 16px;
    border-bottom-color: var(--ops-border);
}

.ops-panel :deep(.ant-card-head-title) {
    color: var(--ops-text);
    font-size: 14px;
    font-weight: 650;
}

.ops-panel :deep(.ant-card-extra) {
    padding: 8px 0;
}

.ops-panel :deep(.ant-card-body) {
    background: transparent;
}

.ops-panel :deep(.ant-select-selector),
.ops-panel :deep(.ant-picker),
.ops-panel :deep(.ant-input) {
    border-color: var(--ops-border-strong) !important;
    background: var(--ops-bg-control) !important;
    color: var(--ops-text) !important;
}

.ops-panel :deep(.ant-select-selection-placeholder),
.ops-panel :deep(.ant-input::placeholder),
.ops-panel :deep(.ant-picker-input > input::placeholder) {
    color: rgba(196, 207, 224, 0.38) !important;
}

.ops-panel :deep(.ant-select-arrow),
.ops-panel :deep(.ant-picker-suffix),
.ops-panel :deep(.ant-input-clear-icon),
.ops-panel :deep(.ant-picker-clear) {
    color: var(--ops-text-muted);
}

.ops-stat-card {
    border: 1px solid var(--ops-border);
    border-radius: 8px;
    background:
        radial-gradient(circle at 82% 22%, rgba(47, 140, 255, 0.08), transparent 34%),
        linear-gradient(180deg, var(--ops-panel-glow), transparent 62%), var(--ops-bg-panel);
    box-shadow: 0 14px 34px var(--ops-card-shadow);
    transition:
        border-color 0.2s ease,
        box-shadow 0.2s ease,
        transform 0.2s ease;
    position: relative;
    overflow: hidden;
    margin-bottom: 16px;
}

.ops-stat-card:hover {
    transform: translateY(-2px);
    border-color: rgba(47, 140, 255, 0.28);
    box-shadow: 0 18px 44px var(--ops-card-shadow-hover);
}

.ops-stat-card::before {
    content: '';
    position: absolute;
    top: 16px;
    left: 18px;
    width: 34px;
    height: 2px;
    border-radius: 999px;
    opacity: 0.9;
}

.ops-stat-card::after {
    content: '';
    position: absolute;
    inset: 0;
    background: linear-gradient(110deg, transparent 0%, var(--ops-shine) 46%, transparent 58%);
    opacity: 0;
    transform: translateX(-35%);
    transition:
        opacity 0.2s ease,
        transform 0.42s ease;
}

.ops-stat-card:hover::after {
    opacity: 1;
    transform: translateX(35%);
}

.ops-stat-card--blue::before {
    background: var(--ops-blue);
}

.ops-stat-card--red::before {
    background: var(--ops-red);
}

.ops-stat-card--orange::before {
    background: var(--ops-amber);
}

.ops-stat-card--purple::before {
    background: var(--ops-purple);
}

.ops-stat-card :deep(.ant-card-body) {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 22px 20px 20px;
    background: transparent;
    position: relative;
    z-index: 1;
}

.ops-stat-card__value {
    font-size: 27px;
    font-weight: 750;
    line-height: 1.2;
    color: var(--ops-text);
    letter-spacing: 0;
}

.ops-stat-card__label {
    margin-top: 5px;
    color: var(--ops-text-muted);
    font-size: 12px;
    font-weight: 500;
}

.ops-stat-card__icon {
    width: 38px;
    height: 38px;
    padding: 0;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 18px;
    border: 1px solid currentColor;
}

.ops-stat-card--blue .ops-stat-card__icon {
    color: var(--ops-blue);
    background: rgba(47, 140, 255, 0.12);
}

.ops-stat-card--red .ops-stat-card__icon {
    color: var(--ops-red);
    background: rgba(255, 77, 94, 0.12);
}

.ops-stat-card--orange .ops-stat-card__icon {
    color: var(--ops-amber);
    background: rgba(255, 176, 32, 0.12);
}

.ops-stat-card--purple .ops-stat-card__icon {
    color: var(--ops-purple);
    background: rgba(139, 92, 246, 0.12);
}

.ops-filter-form {
    padding: 12px;
    border: 1px solid var(--ops-border);
    border-radius: 8px;
    background: var(--ops-filter-bg);
}

.ops-filter-form :deep(.ant-form-item) {
    margin-bottom: 0;
}

.ops-filter-form :deep(.ant-btn) {
    height: 32px;
    border-radius: 6px;
}

.ops-filter-form :deep(.ant-btn-primary) {
    border-color: rgba(47, 140, 255, 0.52);
    background: linear-gradient(135deg, rgba(47, 140, 255, 0.92), rgba(35, 199, 183, 0.82));
    box-shadow: 0 8px 18px rgba(47, 140, 255, 0.18);
}

.ops-filter-form :deep(.ant-btn-default) {
    border-color: var(--ops-border-strong);
    background: var(--ops-bg-control);
    color: var(--ops-text-muted);
}

.ops-table-panel :deep(.ant-table) {
    color: var(--ops-text-muted);
    background: transparent;
}

.ops-table-panel :deep(.ant-table-thead > tr > th) {
    border-bottom-color: var(--ops-border);
    background: var(--ops-table-head-bg) !important;
    color: var(--ops-table-head-text);
    font-size: 12px;
    font-weight: 650;
}

.ops-table-panel :deep(.ant-table-tbody > tr > td) {
    border-bottom-color: var(--ops-table-row-border);
    background: transparent;
}

.ops-table-panel :deep(.ant-table-tbody > tr:hover > td) {
    background: rgba(47, 140, 255, 0.07) !important;
}

.ops-table-panel :deep(.ant-pagination-item),
.ops-table-panel :deep(.ant-pagination-prev .ant-pagination-item-link),
.ops-table-panel :deep(.ant-pagination-next .ant-pagination-item-link) {
    border-color: var(--ops-border-strong);
    background: var(--ops-bg-control);
}

.ops-table-panel :deep(.ant-tag) {
    border-radius: 5px;
    font-size: 12px;
    border-color: transparent;
    color: #fff;
    font-weight: 500;
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
    background: var(--ops-expanded-bg);
    border-radius: 8px;
    border: 1px solid var(--ops-border);
}

.ops-expanded-descriptions :deep(.ant-descriptions-view table) {
    width: 100%;
    table-layout: fixed;
}

.ops-expanded-descriptions :deep(.ant-descriptions-item-label) {
    width: 16%;
    min-width: 0;
    overflow-wrap: anywhere;
    word-break: break-word;
}

.ops-expanded-descriptions :deep(.ant-descriptions-item-content) {
    width: 34%;
    min-width: 0;
    overflow-wrap: anywhere;
    word-break: break-word;
}

.ops-expanded-code {
    display: inline-block;
    max-width: 100%;
    font-family: monospace;
    vertical-align: bottom;
}

.ops-expanded-container :deep(.ant-descriptions-bordered .ant-descriptions-item-label) {
    background-color: var(--ops-expanded-label-bg);
    color: var(--ops-table-head-text);
    border-color: var(--ops-border);
}

.ops-expanded-container :deep(.ant-descriptions-bordered .ant-descriptions-item-content) {
    background-color: var(--ops-expanded-cell-bg);
    color: var(--ops-text-muted);
    border-color: var(--ops-border);
}

.ops-expanded-container :deep(.ant-descriptions-bordered .ant-descriptions-row) {
    border-bottom-color: var(--ops-border);
}

.ops-expanded-container :deep(.ant-descriptions-title) {
    color: var(--ops-text);
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
    background: var(--ops-bg-panel);
    border: 1px solid var(--ops-border);
    box-shadow: 0 12px 32px rgba(0, 0, 0, 0.16);
}

.cache-control-title {
    font-weight: 500;
    font-size: 14px;
    color: var(--ops-text);
}
</style>
