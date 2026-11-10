<template>
    <a-alert
        v-if="reading && reading.state !== 'ok'"
        class="monitoring-status"
        :type="reading.state === 'error' ? 'error' : 'warning'"
        show-icon
        :message="t(`pages.dashboard.monitoring.${reading.state}`)" />
</template>
<script setup>
import { useI18n } from 'vue-i18n'
defineProps({ reading: { type: Object, default: null } })
const { t } = useI18n()
</script>
<style scoped>
.monitoring-status {
    margin-bottom: 12px;
}
</style>
