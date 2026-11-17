<template>
    <a-card
        v-if="panelVisible"
        class="dashboard-panel api-key-usage-panel"
        :title="t('pages.dashboard.apiKeyRanking.title')"
        :bordered="false">
        <template #extra>
            <div class="usage-controls">
                <a-select
                    :value="query.time_range"
                    :options="rangeOptions"
                    :aria-label="t('pages.dashboard.apiKeyRanking.timeRange')"
                    @change="(value) => handleQueryChange({ time_range: value })" />
                <a-select
                    :value="query.sort_by"
                    :options="sortOptions"
                    :aria-label="t('pages.dashboard.apiKeyRanking.sortBy')"
                    @change="(value) => handleQueryChange({ sort_by: value })" />
                <a-select
                    :value="query.limit"
                    :options="limitOptions"
                    :aria-label="t('pages.dashboard.apiKeyRanking.limit')"
                    @change="(value) => handleQueryChange({ limit: value })" />
            </div>
        </template>

        <div class="usage-body-stage">
            <div
                v-if="state.phase === 'loading' || state.phase === 'idle'"
                class="usage-skeleton"
                aria-hidden="true">
                <div class="usage-skeleton-table">
                    <div class="usage-skeleton-header">
                        <span class="usage-skeleton-bar usage-skeleton-bar--wide"></span>
                        <span class="usage-skeleton-bar usage-skeleton-bar--mid"></span>
                        <span class="usage-skeleton-bar usage-skeleton-bar--mid"></span>
                        <span class="usage-skeleton-bar usage-skeleton-bar--short"></span>
                        <span class="usage-skeleton-bar usage-skeleton-bar--wide"></span>
                    </div>
                    <div
                        v-for="row in 6"
                        :key="row"
                        class="usage-skeleton-row">
                        <div class="usage-skeleton-cell-stack">
                            <span class="usage-skeleton-bar usage-skeleton-bar--wide"></span>
                            <span class="usage-skeleton-bar usage-skeleton-bar--short"></span>
                        </div>
                        <span class="usage-skeleton-bar usage-skeleton-bar--mid"></span>
                        <span class="usage-skeleton-bar usage-skeleton-bar--mid"></span>
                        <span class="usage-skeleton-bar usage-skeleton-bar--short"></span>
                        <span class="usage-skeleton-bar usage-skeleton-bar--wide"></span>
                    </div>
                </div>
                <div class="usage-skeleton-footer">
                    <span class="usage-skeleton-bar usage-skeleton-bar--wide"></span>
                    <span class="usage-skeleton-bar usage-skeleton-bar--mid"></span>
                </div>
            </div>
            <div
                v-else-if="statusMessage"
                class="usage-empty"
                role="status">
                {{ statusMessage }}
            </div>
            <template v-else>
                <a-alert
                    v-if="state.stale"
                    class="usage-alert"
                    type="warning"
                    show-icon
                    :message="t('pages.dashboard.apiKeyRanking.stale')" />

                <div class="usage-table-region">
                    <a-table
                        class="api-key-ranking-table"
                        :loading="switchingQuery && state.refreshing"
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
                                    <span class="usage-secondary usage-number">{{
                                        formatShare(record.token_share)
                                    }}</span>
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
                </div>

                <div
                    v-if="unknown?.request_count > 0"
                    class="usage-unattributed"
                    role="note">
                    <strong>{{ t('pages.dashboard.apiKeyRanking.unattributed') }}</strong>
                    <span
                        >{{ t('pages.dashboard.apiKeyRanking.requests') }}:
                        {{ formatCount(unknown.request_count) }}</span
                    >
                    <span
                        >{{ t('pages.dashboard.apiKeyRanking.tokens') }}: {{ formatCount(unknown.total_tokens) }}</span
                    >
                    <span
                        >{{ t('pages.dashboard.apiKeyRanking.cost') }}: {{ unknown.total_cost }}
                        {{ t('pages.dashboard.units.cost') }}</span
                    >
                    <span>{{ formatShare(unknown.token_share) }}</span>
                </div>

                <div
                    v-if="state.data"
                    class="usage-footer usage-secondary">
                    <div class="usage-totals">
                        <span
                            >{{ t('pages.dashboard.apiKeyRanking.totalRequests') }}:
                            <strong class="usage-number">{{
                                formatCount(state.data.summary.request_count)
                            }}</strong></span
                        >
                        <span
                            >{{ t('pages.dashboard.apiKeyRanking.totalTokens') }}:
                            <strong class="usage-number">{{
                                formatCount(state.data.summary.total_tokens)
                            }}</strong></span
                        >
                    </div>
                    <div>{{ t('pages.dashboard.apiKeyRanking.denominator') }}</div>
                    <div v-if="state.data.warnings?.includes('metadata_unavailable')">
                        {{ t('pages.dashboard.apiKeyRanking.metadataWarning') }}
                    </div>
                    <div class="usage-time">
                        <span
                            >{{ t('pages.dashboard.apiKeyRanking.window') }}:
                            {{ formatTime(state.data.window?.start) }} – {{ formatTime(state.data.window?.end) }}</span
                        >
                        <span
                            >{{ t('pages.dashboard.apiKeyRanking.updatedAt') }}:
                            {{ formatTime(state.data.generated_at) }}</span
                        >
                    </div>
                </div>
            </template>
        </div>
    </a-card>
</template>

<script setup>
import { computed, ref, toRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAPIKeyUsage } from '@/composables/useAPIKeyUsage'
import { isPanelRememberedDisabled, rememberPanelDisabled } from '../../../utils/dashboard-panel-memory.js'

const props = defineProps({ authorized: { type: Boolean, required: true } })
const { t, locale } = useI18n()
const { state, query, setQuery } = useAPIKeyUsage(toRef(props, 'authorized'))
const switchingQuery = ref(false)
const rememberedDisabled = ref(isPanelRememberedDisabled('api-key-ranking'))
const panelVisible = computed(() => {
    if (!props.authorized || state.phase === 'disabled') return false
    if (rememberedDisabled.value && (state.phase === 'idle' || state.phase === 'loading')) return false
    return true
})
watch(
    () => state.phase,
    (phase) => {
        if (phase === 'disabled') {
            rememberedDisabled.value = true
            rememberPanelDisabled('api-key-ranking', true)
        } else if (phase === 'ready' || phase === 'empty') {
            rememberedDisabled.value = false
            rememberPanelDisabled('api-key-ranking', false)
        }
    }
)
watch(
    () => state.refreshing,
    (refreshing) => {
        if (!refreshing) switchingQuery.value = false
    }
)
const handleQueryChange = (patch) => {
    switchingQuery.value = true
    setQuery(patch)
}
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
    const keys = { empty: 'empty', error: 'unavailable', forbidden: 'forbidden' }
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
    contain: layout style;
}
.usage-controls {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
}
.usage-controls :deep(.ant-select) {
    min-width: 116px;
}
.dashboard-panel.api-key-usage-panel :deep(.ant-card-head) {
    min-height: 56px;
}
.api-key-usage-panel :deep(.ant-card-head-title) {
    font-size: 16px;
    font-weight: 600;
}
.usage-body-stage {
    height: 520px;
    display: flex;
    flex-direction: column;
    min-height: 0;
}
.usage-skeleton {
    flex: 1;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    min-height: 0;
    gap: 16px;
}
.usage-skeleton-table {
    flex: 1;
    border: 1px solid var(--dashboard-border, rgba(15, 23, 42, 0.08));
    border-radius: 8px;
    overflow: hidden;
    display: flex;
    flex-direction: column;
}
.usage-skeleton-header,
.usage-skeleton-row {
    display: grid;
    grid-template-columns: 2fr 1fr 1.4fr 1fr 1.6fr;
    align-items: center;
    gap: 16px;
    padding: 14px 16px;
}
.usage-skeleton-header {
    background: var(--dashboard-panel-soft, #f8fafc);
    border-bottom: 1px solid var(--dashboard-border, rgba(15, 23, 42, 0.08));
}
.usage-skeleton-row {
    flex: 1;
    border-bottom: 1px solid var(--dashboard-border, rgba(15, 23, 42, 0.08));
}
.usage-skeleton-row:last-child {
    border-bottom: none;
}
.usage-skeleton-cell-stack {
    display: flex;
    flex-direction: column;
    gap: 6px;
}
.usage-skeleton-footer {
    display: flex;
    flex-direction: column;
    gap: 8px;
}
.usage-skeleton-bar {
    display: inline-block;
    height: 14px;
    border-radius: 4px;
    background: linear-gradient(
        90deg,
        var(--dashboard-border, rgba(148, 163, 184, 0.14)) 25%,
        rgba(148, 163, 184, 0.28) 37%,
        var(--dashboard-border, rgba(148, 163, 184, 0.14)) 63%
    );
    background-size: 400% 100%;
    animation: usage-shimmer 1.4s ease infinite;
}
.usage-skeleton-bar--wide {
    width: 82%;
}
.usage-skeleton-bar--mid {
    width: 65%;
}
.usage-skeleton-bar--short {
    width: 48%;
}
@keyframes usage-shimmer {
    0% {
        background-position: 100% 50%;
    }
    100% {
        background-position: 0 50%;
    }
}
.usage-alert {
    margin-bottom: 12px;
    flex-shrink: 0;
}
.usage-empty {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 36px 8px;
    text-align: center;
    color: var(--dashboard-text-secondary, #64748b);
}
.usage-table-region {
    flex: 1 1 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
}
.usage-table-region :deep(.ant-table-wrapper),
.usage-table-region :deep(.ant-spin-nested-loading),
.usage-table-region :deep(.ant-spin-container),
.usage-table-region :deep(.ant-table),
.usage-table-region :deep(.ant-table-container) {
    height: 100%;
    min-height: 0;
}
.usage-table-region :deep(.ant-spin-container),
.usage-table-region :deep(.ant-table),
.usage-table-region :deep(.ant-table-container) {
    display: flex;
    flex-direction: column;
}
.usage-table-region :deep(.ant-table-content) {
    flex: 1 1 0;
    min-height: 0;
    overflow: auto !important;
}
.usage-table-region :deep(.ant-table-thead > tr > th) {
    position: sticky;
    top: 0;
    z-index: 2;
    background: var(--dashboard-panel-soft, #f8fafc);
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
    margin-top: 12px;
    padding: 10px 14px;
    border-radius: 6px;
    background: var(--dashboard-panel-soft, #f8fafc);
    font-size: 12px;
    flex-shrink: 0;
}
.usage-footer {
    margin-top: 12px;
    line-height: 1.7;
    flex-shrink: 0;
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
    .usage-body-stage {
        height: 560px;
    }
}
@media (prefers-reduced-motion: reduce) {
    .api-key-usage-panel :deep(.ant-progress-bg) {
        transition: none;
    }
    .usage-skeleton-bar {
        animation: none;
    }
}
</style>
