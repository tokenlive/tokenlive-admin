<template>
    <a-card
        v-if="authorized"
        class="dashboard-panel api-key-usage-panel"
        :title="t('pages.dashboard.apiKeyRanking.title')"
        :bordered="false">
        <template #extra>
            <div class="usage-controls">
                <a-select
                    :value="query.time_range"
                    :options="rangeOptions"
                    :aria-label="t('pages.dashboard.apiKeyRanking.timeRange')"
                    @change="(value) => setQuery({ time_range: value })" />
                <a-select
                    :value="query.sort_by"
                    :options="sortOptions"
                    :aria-label="t('pages.dashboard.apiKeyRanking.sortBy')"
                    @change="(value) => setQuery({ sort_by: value })" />
                <a-select
                    :value="query.limit"
                    :options="limitOptions"
                    :aria-label="t('pages.dashboard.apiKeyRanking.limit')"
                    @change="(value) => setQuery({ limit: value })" />
            </div>
        </template>

        <a-alert
            v-if="state.stale"
            class="usage-alert"
            type="warning"
            show-icon
            :message="t('pages.dashboard.apiKeyRanking.stale')" />

        <a-skeleton
            v-if="state.phase === 'loading' || state.phase === 'idle'"
            :title="false"
            :paragraph="{ rows: 4 }" />
        <div
            v-else-if="statusMessage"
            class="usage-empty"
            role="status">
            {{ statusMessage }}
        </div>
        <template v-else>
            <a-table
                class="api-key-ranking-table"
                :data-source="rows"
                :columns="columns"
                :pagination="false"
                :locale="{ emptyText: t('pages.dashboard.apiKeyRanking.noIdentifiedKeys') }"
                :scroll="{ x: 1110 }"
                row-key="row_id"
                size="middle">
                <template #bodyCell="{ column, record }">
                    <template v-if="column.key === 'key'">
                        <div class="usage-key-name">
                            {{ record.key_name || t('pages.dashboard.apiKeyRanking.unnamed') }}
                        </div>
                        <div class="usage-secondary usage-key-display">
                            {{ record.key_display || `#${record.row_id.slice(0, 8)}` }}
                            <span v-if="isDuplicate(record)"> · #{{ record.row_id.slice(0, 8) }}</span>
                        </div>
                        <a-tag
                            v-if="record.key_status !== 'enabled'"
                            class="usage-status"
                            :color="record.key_status === 'unknown' ? 'default' : 'warning'">
                            {{ t(`pages.dashboard.apiKeyRanking.status.${record.key_status || 'unknown'}`) }}
                        </a-tag>
                    </template>
                    <template v-else-if="column.key === 'source'">
                        {{ t(`pages.dashboard.apiKeyRanking.sources.${record.source || 'unknown'}`) }}
                    </template>
                    <template v-else-if="column.key === 'owner'">
                        <span :title="record.owner.name || record.owner.id">
                            {{
                                record.owner.name ||
                                record.owner.id ||
                                t('pages.dashboard.apiKeyRanking.metadataUnavailable')
                            }}
                        </span>
                    </template>
                    <template v-else-if="column.key === 'requests'">
                        <span class="usage-number">{{ formatCount(record.request_count) }}</span>
                    </template>
                    <template v-else-if="column.key === 'tokens'">
                        <div class="usage-token-value">
                            <a-tooltip :trigger="['hover', 'focus']">
                                <template #title>
                                    <div>
                                        {{ t('pages.dashboard.apiKeyRanking.input') }}:
                                        {{ formatCount(record.input_tokens) }}
                                    </div>
                                    <div>
                                        {{ t('pages.dashboard.apiKeyRanking.output') }}:
                                        {{ formatCount(record.output_tokens) }}
                                    </div>
                                    <div>
                                        {{ t('pages.dashboard.apiKeyRanking.cached') }}:
                                        {{ formatCount(record.cached_tokens) }}
                                    </div>
                                    <div>
                                        {{ t('pages.dashboard.apiKeyRanking.cacheCreation') }}:
                                        {{ formatCount(record.cache_creation_tokens) }}
                                    </div>
                                </template>
                                <span
                                    class="usage-number usage-token-detail"
                                    tabindex="0">
                                    {{ formatCount(record.total_tokens) }}
                                </span>
                            </a-tooltip>
                            <span class="usage-secondary usage-number">{{ formatShare(record.token_share) }}</span>
                        </div>
                        <a-progress
                            :percent="record.token_share ?? 0"
                            :show-info="false"
                            :stroke-color="'var(--dashboard-accent, #2f8cff)'"
                            size="small" />
                    </template>
                    <template v-else-if="column.key === 'cost'">
                        <span class="usage-number">{{ record.total_cost ?? '—' }}</span>
                    </template>
                    <template v-else-if="column.key === 'success'">
                        <span class="usage-number">{{ formatShare(record.success_rate) }}</span>
                    </template>
                </template>
            </a-table>

            <div
                v-if="unknown?.request_count > 0"
                class="usage-unattributed"
                role="note">
                <strong>{{ t('pages.dashboard.apiKeyRanking.unattributed') }}</strong>
                <span>{{ t('pages.dashboard.apiKeyRanking.requests') }}: {{ formatCount(unknown.request_count) }}</span>
                <span>{{ t('pages.dashboard.apiKeyRanking.tokens') }}: {{ formatCount(unknown.total_tokens) }}</span>
                <span
                    >{{ t('pages.dashboard.apiKeyRanking.cost') }}: {{ unknown.total_cost }}
                    {{ t('pages.dashboard.units.cost') }}</span
                >
                <span>{{ formatShare(unknown.token_share) }}</span>
            </div>
        </template>

        <div
            v-if="state.data"
            class="usage-footer usage-secondary">
            <div class="usage-totals">
                <span
                    >{{ t('pages.dashboard.apiKeyRanking.totalRequests') }}:
                    <strong class="usage-number">{{ formatCount(state.data.summary.request_count) }}</strong></span
                >
                <span
                    >{{ t('pages.dashboard.apiKeyRanking.totalTokens') }}:
                    <strong class="usage-number">{{ formatCount(state.data.summary.total_tokens) }}</strong></span
                >
            </div>
            <div>{{ t('pages.dashboard.apiKeyRanking.denominator') }}</div>
            <div v-if="state.data.warnings?.includes('metadata_unavailable')">
                {{ t('pages.dashboard.apiKeyRanking.metadataWarning') }}
            </div>
            <div class="usage-time">
                <span
                    >{{ t('pages.dashboard.apiKeyRanking.window') }}: {{ formatTime(state.data.window?.start) }} –
                    {{ formatTime(state.data.window?.end) }}</span
                >
                <span
                    >{{ t('pages.dashboard.apiKeyRanking.updatedAt') }}: {{ formatTime(state.data.generated_at) }}</span
                >
            </div>
        </div>
    </a-card>
</template>

<script setup>
import { computed, toRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAPIKeyUsage } from '@/composables/useAPIKeyUsage'

const props = defineProps({ authorized: { type: Boolean, required: true } })
const { t, locale } = useI18n()
const { state, query, setQuery } = useAPIKeyUsage(toRef(props, 'authorized'))
const rows = computed(() => state.data?.items || [])
const unknown = computed(() => state.data?.unattributed)
const rangeOptions = computed(() =>
    ['today', '1h', '6h', '24h', '7d'].map((value) => ({
        value,
        label: t(`pages.dashboard.modelRanking.range.${value}`),
    }))
)
const sortOptions = computed(() => [
    { value: 'tokens', label: t('pages.dashboard.apiKeyRanking.sortTokens') },
    { value: 'request_count', label: t('pages.dashboard.apiKeyRanking.sortRequests') },
    { value: 'cost', label: t('pages.dashboard.apiKeyRanking.sortCost') },
])
const limitOptions = [10, 20, 50].map((value) => ({ value, label: `Top ${value}` }))
const columns = computed(() => [
    { key: 'key', title: t('pages.dashboard.apiKeyRanking.key'), width: 230 },
    { key: 'source', title: t('pages.dashboard.apiKeyRanking.source'), width: 130 },
    { key: 'owner', title: t('pages.dashboard.apiKeyRanking.owner'), width: 180, ellipsis: true },
    { key: 'requests', title: t('pages.dashboard.apiKeyRanking.requests'), width: 100, align: 'right' },
    { key: 'tokens', title: t('pages.dashboard.apiKeyRanking.tokens'), width: 210, align: 'right' },
    {
        key: 'cost',
        title: `${t('pages.dashboard.apiKeyRanking.cost')} (${t('pages.dashboard.units.cost')})`,
        width: 130,
        align: 'right',
    },
    { key: 'success', title: t('pages.dashboard.apiKeyRanking.success'), width: 130, align: 'right' },
])
const statusMessage = computed(() => {
    const keys = { disabled: 'disabled', empty: 'empty', error: 'unavailable', forbidden: 'forbidden' }
    return keys[state.phase] ? t(`pages.dashboard.apiKeyRanking.${keys[state.phase]}`) : ''
})
const duplicates = computed(() => {
    const counts = new Map()
    for (const row of rows.value) {
        const key = JSON.stringify([row.key_name, row.key_display])
        counts.set(key, (counts.get(key) || 0) + 1)
    }
    return counts
})
const isDuplicate = (row) => duplicates.value.get(JSON.stringify([row.key_name, row.key_display])) > 1
const formatCount = (value) => new Intl.NumberFormat(locale.value === 'en-us' ? 'en-US' : 'zh-CN').format(value ?? 0)
const formatShare = (value) => (Number.isFinite(value) ? `${value.toFixed(2)}%` : '—')
const formatTime = (value) => (value ? value.replace('T', ' ').replace(/\.\d+(?=Z|[+-]\d\d:\d\d$)/, '') : '—')
</script>

<style scoped>
.api-key-usage-panel {
    margin-bottom: 16px;
    color: var(--dashboard-text, #111827);
    background: var(--dashboard-panel, #fff);
}
.usage-controls {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
}
.usage-controls :deep(.ant-select) {
    min-width: 116px;
}
.api-key-usage-panel :deep(.ant-card-head-title) {
    font-size: 16px;
    font-weight: 600;
}
.usage-alert {
    margin-bottom: 16px;
}
.usage-empty {
    padding: 36px 8px;
    text-align: center;
    color: var(--dashboard-text-secondary, #64748b);
}
.usage-key-name {
    font-weight: 600;
    overflow-wrap: anywhere;
}
.usage-key-display {
    margin-top: 3px;
}
.usage-secondary {
    color: var(--dashboard-text-secondary, #64748b);
    font-size: 12px;
}
.usage-number {
    font-variant-numeric: tabular-nums;
}
.usage-status {
    margin-top: 5px;
}
.usage-token-value {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    gap: 12px;
}
.usage-token-detail {
    border-bottom: 1px dashed var(--dashboard-text-tertiary, #94a3b8);
}
.usage-token-detail:focus-visible {
    outline: 2px solid var(--dashboard-accent, #2f8cff);
    outline-offset: 3px;
}
.api-key-ranking-table :deep(.ant-table-row) {
    cursor: default;
}
.usage-unattributed {
    display: flex;
    flex-wrap: wrap;
    gap: 8px 20px;
    margin-top: 14px;
    padding: 12px 14px;
    border-radius: 6px;
    background: var(--dashboard-panel-soft, #f8fafc);
    font-size: 12px;
}
.usage-footer {
    margin-top: 16px;
    line-height: 1.9;
}
.usage-totals,
.usage-time {
    display: flex;
    flex-wrap: wrap;
    gap: 4px 24px;
}
.usage-totals strong {
    color: var(--dashboard-text, #111827);
}
@media (max-width: 700px) {
    .api-key-usage-panel :deep(.ant-card-head-wrapper) {
        flex-wrap: wrap;
        padding: 12px 0;
        gap: 10px;
    }
    .api-key-usage-panel :deep(.ant-card-extra) {
        margin-inline-start: 0;
        width: 100%;
    }
    .usage-controls :deep(.ant-select) {
        flex: 1 1 100px;
        min-width: 100px;
    }
    .api-key-usage-panel :deep(.ant-card-body) {
        padding: 16px;
    }
}
@media (prefers-reduced-motion: reduce) {
    .api-key-usage-panel :deep(.ant-progress-bg) {
        transition: none;
    }
}
</style>
