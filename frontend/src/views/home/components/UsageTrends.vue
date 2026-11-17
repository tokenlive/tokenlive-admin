<template>
    <section
        v-if="panelVisible"
        class="usage-trends dashboard-panel"
        :aria-label="t('pages.dashboard.usageTrends.title')">
        <div class="trend-header">
            <div class="trend-title-group">
                <h2>{{ t('pages.dashboard.usageTrends.title') }}</h2>
                <a-button
                    v-if="state.stale"
                    type="link"
                    size="small"
                    class="trend-stale-btn"
                    :loading="state.refreshing"
                    @click="refresh"
                    >{{ t('pages.dashboard.apiKeyRanking.stale') }} ·
                    {{ t('pages.dashboard.usageTrends.retry') }}</a-button
                >
            </div>
            <div class="trend-controls">
                <a-segmented
                    v-model:value="category"
                    :options="categories"
                    :aria-label="t('pages.dashboard.usageTrends.category')" />
                <a-select
                    :value="state.timeRange"
                    :options="rangeOptions"
                    :aria-label="t('pages.dashboard.usageTrends.timeRange')"
                    @change="handleTimeRangeChange" />
            </div>
        </div>
        <div class="trend-body">
            <div
                v-if="state.phase === 'idle' || state.phase === 'loading'"
                class="trend-skeleton"
                aria-hidden="true">
                <div class="trend-summary trend-summary--skeleton">
                    <div
                        v-for="item in 3"
                        :key="item"
                        class="trend-metric">
                        <span class="trend-dot trend-dot--skeleton"></span>
                        <div class="trend-metric-skeleton-lines">
                            <span class="trend-skeleton-block trend-skeleton-block--label"></span>
                            <span class="trend-skeleton-block trend-skeleton-block--value"></span>
                        </div>
                    </div>
                </div>
                <div class="trend-chart trend-chart--skeleton">
                    <svg
                        viewBox="0 0 800 260"
                        preserveAspectRatio="none"
                        width="100%"
                        height="100%">
                        <line
                            v-for="y in [36, 92, 148, 204]"
                            :key="y"
                            x1="24"
                            :y1="y"
                            x2="780"
                            :y2="y"
                            stroke="var(--dashboard-border, rgba(15, 23, 42, 0.08))"
                            stroke-dasharray="4,4" />
                        <path
                            d="M 24 185 Q 160 120 290 165 T 540 95 T 780 128"
                            fill="none"
                            stroke="rgba(104, 133, 248, 0.22)"
                            stroke-width="3"
                            stroke-linecap="round" />
                    </svg>
                </div>
                <div class="trend-footer trend-footer--skeleton">
                    <span class="trend-skeleton-block trend-skeleton-block--hint"></span>
                    <span class="trend-skeleton-block trend-skeleton-block--window"></span>
                </div>
            </div>
            <div
                v-else-if="['error', 'forbidden'].includes(state.phase)"
                class="trend-status"
                role="status">
                <span>{{ t(`pages.dashboard.usageTrends.${state.phase}`) }}</span>
                <a-button
                    v-if="state.phase === 'error'"
                    :loading="state.refreshing"
                    @click="refresh"
                    >{{ t('pages.dashboard.usageTrends.retry') }}</a-button
                >
            </div>
            <template v-else-if="state.data">
                <div class="trend-summary">
                    <div
                        v-for="spec in series"
                        :key="spec.field"
                        class="trend-metric">
                        <span
                            class="trend-dot"
                            :style="{ background: spec.color }"></span>
                        <div class="trend-metric-content">
                            <div class="trend-label">
                                {{ t(`pages.dashboard.usageTrends.${spec.label || spec.field}`) }}
                            </div>
                            <div
                                class="trend-number"
                                :title="formatTrendValue(state.data.summary[spec.field], spec.unit, numberLocale)">
                                {{ formatTrendValue(state.data.summary[spec.field], spec.unit, numberLocale, true) }}
                            </div>
                        </div>
                    </div>
                </div>
                <div
                    class="trend-chart"
                    :aria-label="t(`pages.dashboard.usageTrends.${category}`)"
                    role="img">
                    <x-chart
                        :options="chartOptions"
                        :loading="switchingRange && state.refreshing" />
                </div>
                <div class="trend-footer">
                    <span>{{ t(`pages.dashboard.usageTrends.${category}Hint`) }}</span>
                    <span class="trend-window"
                        >{{ formatTrendTime(state.data.window.start, true) }} –
                        {{ formatTrendTime(state.data.window.end, true) }} ({{ timezone }})</span
                    >
                </div>
            </template>
        </div>
    </section>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore, useUserStore } from '@/store'
import { useUsageTrends } from '@/composables/useUsageTrends'
import { buildUsageTrendsChart, formatTrendTime, formatTrendValue, trendSeries } from '@/utils/usage-trends-chart'
import { isPanelRememberedDisabled, rememberPanelDisabled } from '@/utils/dashboard-panel-memory'

const { t, locale } = useI18n()
const app = useAppStore()
const user = useUserStore()
const { state, refresh, setTimeRange } = useUsageTrends(computed(() => user.isLogin))
const category = ref('requests')
const switchingRange = ref(false)
const rememberedDisabled = ref(isPanelRememberedDisabled('usage-trends'))
const panelVisible = computed(() => {
    if (state.phase === 'disabled') return false
    if (rememberedDisabled.value && (state.phase === 'idle' || state.phase === 'loading')) return false
    return true
})
watch(
    () => state.phase,
    (phase) => {
        if (phase === 'disabled') {
            rememberedDisabled.value = true
            rememberPanelDisabled('usage-trends', true)
        } else if (phase === 'ready' || phase === 'empty') {
            rememberedDisabled.value = false
            rememberPanelDisabled('usage-trends', false)
        }
    }
)
watch(
    () => state.refreshing,
    (refreshing) => {
        if (!refreshing) switchingRange.value = false
    }
)
const handleTimeRangeChange = (value) => {
    switchingRange.value = true
    setTimeRange(value)
}
const numberLocale = computed(() => (locale.value.startsWith('en') ? 'en-US' : 'zh-CN'))
const categories = computed(() =>
    Object.keys(trendSeries).map((value) => ({ value, label: t(`pages.dashboard.usageTrends.${value}`) }))
)
const rangeOptions = computed(() =>
    ['today', '1h', '6h', '24h', '7d'].map((value) => ({ value, label: t(`pages.dashboard.trends.range.${value}`) }))
)
const series = computed(() => trendSeries[category.value])
const timezone = computed(() => {
    const zone = state.data?.window?.timezone
    const offset = state.data?.window?.end?.match(/([+-]\d\d:\d\d|Z)$/)?.[0]
    return zone === 'Local' ? `UTC${offset === 'Z' ? '+00:00' : offset || ''}` : zone
})
const chartOptions = computed(() =>
    buildUsageTrendsChart({
        data: state.data,
        category: category.value,
        timeRange: state.timeRange,
        t,
        dark: app.config.theme === 'dark',
        locale: numberLocale.value,
    })
)
</script>

<style scoped lang="less">
.usage-trends {
    padding: 24px;
    margin-bottom: 16px;
    border-radius: 14px;
    background: var(--dashboard-panel, var(--color-bg-container));
    color: var(--dashboard-text, var(--color-text));
    contain: layout style;
}
.trend-header,
.trend-title-group,
.trend-controls,
.trend-footer {
    display: flex;
    align-items: center;
    gap: 16px;
}
.trend-header {
    justify-content: space-between;
    margin-bottom: 20px;
    min-height: 34px;
    flex-wrap: wrap;
}
.trend-title-group {
    gap: 10px;
    flex-wrap: wrap;
}
.trend-stale-btn {
    padding: 0 4px;
    height: auto;
    font-size: 12px;
    color: #d97706;
}
.trend-body {
    min-height: 411px;
    display: flex;
    flex-direction: column;
}
.trend-skeleton {
    display: flex;
    flex-direction: column;
    flex: 1;
}
h2 {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
    color: inherit;
}
.trend-controls {
    gap: 12px;
}
.trend-controls :deep(.ant-select) {
    min-width: 126px;
}
.trend-controls :deep(.ant-segmented) {
    background: var(--dashboard-panel-soft);
    padding: 4px;
}
.trend-controls :deep(.ant-segmented-item-label) {
    min-width: 58px;
    padding: 4px 12px;
}
.trend-summary {
    display: grid;
    grid-template-columns: repeat(v-bind('series.length'), minmax(0, 1fr));
    gap: 20px;
    min-height: 83px;
    background: var(--dashboard-panel-soft);
    border-radius: 10px;
    padding: 16px 20px;
}
.trend-summary--skeleton {
    grid-template-columns: repeat(3, minmax(0, 1fr));
}
.trend-metric {
    display: flex;
    align-items: center;
    gap: 10px;
    min-width: 0;
}
.trend-metric-content {
    min-width: 0;
}
.trend-metric-skeleton-lines {
    display: flex;
    flex-direction: column;
    gap: 6px;
    min-width: 0;
    flex: 1;
}
.trend-dot {
    flex: 0 0 8px;
    height: 8px;
    border-radius: 50%;
}
.trend-dot--skeleton {
    background: var(--dashboard-border, rgba(15, 23, 42, 0.14));
}
.trend-label {
    font-size: 12px;
    line-height: 1.5;
    color: var(--dashboard-text-secondary);
    margin-bottom: 2px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}
.trend-number {
    font-size: 22px;
    font-weight: 600;
    line-height: 1.35;
    font-variant-numeric: tabular-nums;
}
.trend-chart {
    height: 280px;
    margin-top: 16px;
}
.trend-chart--skeleton {
    border-radius: 10px;
    background: var(--dashboard-panel-soft);
    padding: 10px 8px;
    box-sizing: border-box;
}
.trend-footer {
    justify-content: space-between;
    flex-wrap: wrap;
    margin-top: 12px;
    min-height: 20px;
    font-size: 12px;
    color: var(--dashboard-text-secondary);
}
.trend-window {
    font-variant-numeric: tabular-nums;
}
.trend-skeleton-block {
    display: inline-block;
    border-radius: 4px;
    background: linear-gradient(
        90deg,
        var(--dashboard-border, rgba(148, 163, 184, 0.14)) 25%,
        rgba(148, 163, 184, 0.28) 37%,
        var(--dashboard-border, rgba(148, 163, 184, 0.14)) 63%
    );
    background-size: 400% 100%;
    animation: trend-shimmer 1.4s ease infinite;
}
.trend-skeleton-block--label {
    width: 64px;
    height: 12px;
}
.trend-skeleton-block--value {
    width: 92px;
    height: 24px;
}
.trend-skeleton-block--hint {
    width: 240px;
    max-width: 60%;
    height: 14px;
}
.trend-skeleton-block--window {
    width: 180px;
    max-width: 35%;
    height: 14px;
}
@keyframes trend-shimmer {
    0% {
        background-position: 100% 50%;
    }
    100% {
        background-position: 0 50%;
    }
}
.trend-status {
    flex: 1;
    min-height: 411px;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 16px;
    color: var(--dashboard-text-secondary);
}
@media (max-width: 640px) {
    .usage-trends {
        padding: 16px;
    }
    .trend-controls {
        width: 100%;
        flex-wrap: wrap;
    }
    .trend-controls :deep(.ant-segmented) {
        max-width: 100%;
    }
    .trend-controls :deep(.ant-segmented-item-label) {
        min-width: 0;
        padding: 4px 10px;
    }
    .trend-body,
    .trend-status {
        min-height: 446px;
    }
    .trend-summary {
        grid-template-columns: repeat(2, minmax(0, 1fr));
        padding: 14px;
        gap: 16px;
        min-height: 142px;
    }
    .trend-number {
        font-size: 20px;
    }
    .trend-chart {
        height: 240px;
    }
}
@media (prefers-reduced-motion: reduce) {
    .trend-skeleton-block {
        animation: none;
    }
}
</style>
