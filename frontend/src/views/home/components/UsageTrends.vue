<template>
    <section
        v-if="state.phase !== 'disabled'"
        class="usage-trends dashboard-panel"
        :aria-label="t('pages.dashboard.usageTrends.title')">
        <div class="trend-header">
            <h2>{{ t('pages.dashboard.usageTrends.title') }}</h2>
            <div class="trend-controls">
                <a-segmented
                    v-model:value="category"
                    :options="categories"
                    :aria-label="t('pages.dashboard.usageTrends.category')" />
                <a-select
                    :value="state.timeRange"
                    :options="rangeOptions"
                    :aria-label="t('pages.dashboard.usageTrends.timeRange')"
                    @change="setTimeRange" />
            </div>
        </div>
        <a-skeleton
            v-if="state.phase === 'idle' || state.phase === 'loading'"
            active
            :title="false"
            :paragraph="{ rows: 6 }" />
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
                    <div>
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
                <x-chart :options="chartOptions" />
            </div>
            <div class="trend-footer">
                <span>{{ t(`pages.dashboard.usageTrends.${category}Hint`) }}</span>
                <span class="trend-window"
                    >{{ formatTrendTime(state.data.window.start, true) }} –
                    {{ formatTrendTime(state.data.window.end, true) }} ({{ timezone }})</span
                >
            </div>
        </template>
    </section>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore, useUserStore } from '@/store'
import { useUsageTrends } from '@/composables/useUsageTrends'
import { buildUsageTrendsChart, formatTrendTime, formatTrendValue, trendSeries } from '@/utils/usage-trends-chart'

const { t, locale } = useI18n()
const app = useAppStore()
const user = useUserStore()
const { state, refresh, setTimeRange } = useUsageTrends(computed(() => user.isLogin))
const category = ref('requests')
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
}
.trend-header,
.trend-controls,
.trend-footer {
    display: flex;
    align-items: center;
    gap: 16px;
}
.trend-header {
    justify-content: space-between;
    margin-bottom: 20px;
    flex-wrap: wrap;
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
    background: var(--dashboard-panel-soft);
    border-radius: 10px;
    padding: 16px 20px;
}
.trend-metric {
    display: flex;
    align-items: center;
    gap: 10px;
    min-width: 0;
}
.trend-dot {
    flex: 0 0 8px;
    height: 8px;
    border-radius: 50%;
}
.trend-label {
    font-size: 12px;
    color: var(--dashboard-text-secondary);
    margin-bottom: 2px;
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
.trend-footer {
    justify-content: space-between;
    flex-wrap: wrap;
    margin-top: 12px;
    font-size: 12px;
    color: var(--dashboard-text-secondary);
}
.trend-window {
    font-variant-numeric: tabular-nums;
}
.trend-status {
    min-height: 280px;
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
    .trend-summary {
        grid-template-columns: repeat(2, minmax(0, 1fr));
        padding: 14px;
        gap: 16px;
    }
    .trend-number {
        font-size: 20px;
    }
    .trend-chart {
        height: 240px;
    }
}
</style>
