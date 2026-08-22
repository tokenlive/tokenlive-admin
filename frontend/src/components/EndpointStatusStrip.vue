<template>
    <div class="endpoint-status-strip">
        <a-tooltip
            v-for="(point, index) in points"
            :key="index"
            overlay-class-name="endpoint-status-tooltip"
            :mouse-enter-delay="0.05">
            <template #title>
                <div class="endpoint-status-card">
                    <div class="endpoint-status-card__window">{{ point.start_time }} – {{ point.end_time }}</div>
                    <div class="endpoint-status-card__grid">
                        <div class="endpoint-status-card__cell">
                            <span class="endpoint-status-card__label">{{
                                $t('pages.endpoint.recent_status.success')
                            }}</span>
                            <span class="endpoint-status-card__value">{{ formatCount(point.success_count) }}</span>
                        </div>
                        <div class="endpoint-status-card__cell">
                            <span class="endpoint-status-card__label">{{
                                $t('pages.endpoint.recent_status.fail')
                            }}</span>
                            <span
                                class="endpoint-status-card__value"
                                :class="{ 'is-alert': Number(point.fail_count) > 0 }">
                                {{ formatCount(point.fail_count) }}
                            </span>
                        </div>
                        <template v-if="showPerf">
                            <div class="endpoint-status-card__cell">
                                <span class="endpoint-status-card__label">{{
                                    $t('pages.endpoint.recent_status.ttft')
                                }}</span>
                                <span class="endpoint-status-card__value is-metric">{{
                                    formatStatusTtft(point.avg_ttft_ms)
                                }}</span>
                            </div>
                            <div class="endpoint-status-card__cell">
                                <span class="endpoint-status-card__label">{{
                                    $t('pages.endpoint.recent_status.otps')
                                }}</span>
                                <span class="endpoint-status-card__value is-metric">{{
                                    formatStatusOtps(point.otps)
                                }}</span>
                            </div>
                        </template>
                    </div>
                </div>
            </template>
            <span
                class="endpoint-status-dot"
                :style="getPointStyle(point)"
                :aria-label="ariaLabel(point)" />
        </a-tooltip>
    </div>
</template>

<script setup>
defineProps({
    points: {
        type: Array,
        default: () => [],
    },
    showPerf: {
        type: Boolean,
        default: true,
    },
})

function formatCount(val) {
    const num = Number(val)
    if (!Number.isFinite(num)) return '0'
    return num.toLocaleString('en-US')
}

function formatStatusTtft(val) {
    const num = Number(val)
    if (!Number.isFinite(num) || num <= 0) return '-'
    return (num / 1000).toFixed(2) + 's'
}

function formatStatusOtps(val) {
    const num = Number(val)
    if (!Number.isFinite(num) || num <= 0) return '-'
    return (
        num.toLocaleString('en-US', {
            minimumFractionDigits: 0,
            maximumFractionDigits: 2,
        }) + ' t/s'
    )
}

function getPointStyle(point) {
    let color = '#f5f5f5'
    let border = '1px solid #d9d9d9'
    if (point.success_count > 0 && point.fail_count === 0) {
        color = '#52c41a'
        border = '1px solid #52c41a'
    } else if (point.success_count === 0 && point.fail_count > 0) {
        color = '#f5222d'
        border = '1px solid #f5222d'
    } else if (point.success_count > 0 && point.fail_count > 0) {
        color = '#fa8c16'
        border = '1px solid #fa8c16'
    }
    return {
        backgroundColor: color,
        border,
    }
}

function ariaLabel(point) {
    return `${point.start_time || ''} ${point.end_time || ''}`
}
</script>

<style lang="less" scoped>
.endpoint-status-strip {
    display: inline-flex;
    align-items: center;
    gap: 3px;
}

.endpoint-status-dot {
    width: 12px;
    height: 12px;
    padding: 0;
    border-radius: 2px;
    cursor: pointer;
    flex: none;
}
</style>

<style lang="less">
.endpoint-status-tooltip {
    .ant-tooltip-inner {
        padding: 10px 12px;
        min-width: 196px;
    }

    .endpoint-status-card {
        font-variant-numeric: tabular-nums;
    }

    .endpoint-status-card__window {
        margin-bottom: 8px;
        padding-bottom: 6px;
        border-bottom: 1px solid rgba(255, 255, 255, 0.14);
        font-size: 11px;
        letter-spacing: 0.08em;
        text-transform: uppercase;
        opacity: 0.7;
    }

    .endpoint-status-card__grid {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 8px 18px;
    }

    .endpoint-status-card__cell {
        display: flex;
        flex-direction: column;
        gap: 2px;
        min-width: 0;
    }

    .endpoint-status-card__label {
        font-size: 10px;
        letter-spacing: 0.08em;
        text-transform: uppercase;
        opacity: 0.56;
    }

    .endpoint-status-card__value {
        font-size: 13px;
        font-weight: 600;
        line-height: 1.2;
    }

    .endpoint-status-card__value.is-metric {
        font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
        font-weight: 500;
    }

    .endpoint-status-card__value.is-alert {
        color: #ffb4b4;
    }
}
</style>
