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
                <a-form-item :label="$t('pages.ops.filter.event_type')">
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
                <a-form-item :label="$t('pages.ops.filter.tenant')">
                    <a-input
                        v-model:value="filterForm.tenant_code"
                        allow-clear
                        style="width: 140px"
                        :placeholder="$t('pages.ops.filter.tenant')" />
                </a-form-item>
                <a-form-item :label="$t('pages.ops.filter.model')">
                    <a-input
                        v-model:value="filterForm.model_code"
                        allow-clear
                        style="width: 140px"
                        :placeholder="$t('pages.ops.filter.model')" />
                </a-form-item>
                <a-form-item>
                    <a-space>
                        <a-button
                            type="primary"
                            @click="handleSearch"
                            >{{ $t('common.search') }}</a-button
                        >
                        <a-button @click="handleResetSearch">{{ $t('common.reset') }}</a-button>
                    </a-space>
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
                    <template v-else-if="column.key === 'detail'">
                        <a-space
                            direction="vertical"
                            :size="2"
                            style="font-size: 12px">
                            <span v-if="record.threshold != null">
                                {{ $t('pages.ops.table.threshold') }}: {{ record.threshold }}
                                <span v-if="record.current_value != null">
                                    | {{ $t('pages.ops.table.current_value') }}: {{ record.current_value }}
                                </span>
                            </span>
                            <span
                                v-if="record.message"
                                style="color: var(--color-text-tertiary)"
                                >{{ record.message }}</span
                            >
                            <span
                                v-if="record.request_id"
                                style="color: var(--color-text-tertiary)">
                                {{ $t('pages.ops.table.request_id') }}: {{ record.request_id }}
                            </span>
                        </a-space>
                    </template>
                </template>
            </a-table>
        </a-card>
    </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/store'
import useUserStore from '@/store/modules/user'
import { AlertOutlined, WarningOutlined, ThunderboltOutlined, CloseCircleOutlined } from '@ant-design/icons-vue'
import apis from '@/apis'
import { config } from '@/config'

const { t } = useI18n()
const appStore = useAppStore()
const userStore = useUserStore()

// State
const loading = ref(false)
const tableLoading = ref(false)
const timeRange = ref('24h')
const stats = ref({})
const eventList = ref([])

const filterForm = reactive({
    event_type: undefined,
    tenant_code: '',
    model_code: '',
})

const pagination = reactive({
    current: 1,
    pageSize: 20,
    total: 0,
    showSizeChanger: true,
    showTotal: (total) => t('common.pagination.total', { total }),
})

// Filter match helper (used by WS push to avoid injecting non-matching events)
const matchesFilter = (evt) => {
    if (filterForm.event_type && evt.event_type !== filterForm.event_type) return false
    if (filterForm.tenant_code && !evt.tenant_code?.includes(filterForm.tenant_code)) return false
    if (filterForm.model_code && !evt.model_code?.includes(filterForm.model_code)) return false
    return true
}

// WebSocket reconnect state
let ws = null
let wsReconnectTimer = null
let wsReconnectDelay = 1000
let wsManualClose = false

// Table columns
const columns = computed(() => [
    { title: t('pages.ops.table.time'), key: 'event_time', dataIndex: 'event_time', width: 170 },
    { title: t('pages.ops.table.type'), key: 'event_type', dataIndex: 'event_type', width: 120 },
    { title: t('pages.ops.table.tenant'), dataIndex: 'tenant_code', width: 120, ellipsis: true },
    { title: t('pages.ops.table.model'), dataIndex: 'model_code', width: 120, ellipsis: true },
    { title: t('pages.ops.table.provider'), dataIndex: 'provider_name', width: 120, ellipsis: true },
    { title: t('pages.ops.table.policy'), dataIndex: 'policy_name', width: 140, ellipsis: true },
    { title: t('pages.ops.table.detail'), key: 'detail', width: 280 },
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
    filterForm.tenant_code = ''
    filterForm.model_code = ''
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
    const wsUrl = `${protocol}//${host}${config('http.apiBasic')}/api/v1/ops/events/ws?accessToken=${token}`

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
        grid: { left: 120, right: 20, bottom: 10, top: 10 },
        xAxis: {
            type: 'value',
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
                type: 'bar',
                barWidth: 16,
                showBackground: true,
                backgroundStyle: {
                    color: isD ? 'rgba(255, 255, 255, 0.03)' : 'rgba(0, 0, 0, 0.02)',
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
        grid: { left: 120, right: 20, bottom: 10, top: 10 },
        xAxis: {
            type: 'value',
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
                type: 'bar',
                barWidth: 16,
                showBackground: true,
                backgroundStyle: {
                    color: isD ? 'rgba(255, 255, 255, 0.03)' : 'rgba(0, 0, 0, 0.02)',
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
    color: #ffffff;
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
</style>
