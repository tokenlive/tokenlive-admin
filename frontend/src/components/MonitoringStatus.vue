<template>
    <a-tooltip
        v-if="hint && reading && reading.state !== 'ok'"
        :title="message">
        <question-circle-outlined
            class="monitoring-status-hint"
            :aria-label="message" />
    </a-tooltip>
    <a-alert
        v-else-if="!hint && reading && reading.state !== 'ok'"
        class="monitoring-status"
        :type="reading.state === 'error' ? 'error' : 'warning'"
        show-icon
        :message="message" />
</template>
<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { QuestionCircleOutlined } from '@ant-design/icons-vue'

const props = defineProps({
    reading: { type: Object, default: null },
    hint: { type: Boolean, default: false },
})
const { t } = useI18n()
const message = computed(() => (props.reading?.state ? t(`pages.dashboard.monitoring.${props.reading.state}`) : ''))
</script>
<style scoped>
.monitoring-status {
    margin-bottom: 12px;
}

.monitoring-status-hint {
    margin-left: 6px;
    color: var(--color-text-tertiary, rgba(31, 41, 55, 0.45));
    font-size: 14px;
    cursor: help;
}
</style>
