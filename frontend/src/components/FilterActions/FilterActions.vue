<template>
    <div class="x-filter-actions">
        <a-button
            class="x-filter-actions__reset"
            :disabled="resetDisabled || disabled"
            @click="emit('reset')">
            <template #icon>
                <redo-outlined />
            </template>
            {{ $t('button.reset') }}
        </a-button>
        <a-button
            class="x-filter-actions__search"
            :disabled="searchDisabled || disabled"
            :loading="searchLoading"
            @click="emit('search')">
            <template #icon>
                <search-outlined />
            </template>
            {{ $t('button.search') }}
        </a-button>
    </div>
</template>

<script setup>
import { RedoOutlined, SearchOutlined } from '@ant-design/icons-vue'

defineOptions({
    name: 'XFilterActions',
})

defineProps({
    disabled: {
        type: Boolean,
        default: false,
    },
    resetDisabled: {
        type: Boolean,
        default: false,
    },
    searchDisabled: {
        type: Boolean,
        default: false,
    },
    searchLoading: {
        type: Boolean,
        default: false,
    },
})

const emit = defineEmits(['reset', 'search'])
</script>

<style lang="less" scoped>
.x-filter-actions {
    display: inline-flex;
    align-items: center;
    height: 32px;
    gap: 8px;
    border-radius: 8px;
    background: transparent;

    :deep(.ant-btn) {
        height: 32px;
        border-radius: 6px;
        display: inline-flex;
        align-items: center;
        justify-content: center;
        box-shadow: none !important;
    }

    &__reset {
        min-width: 84px;
        padding: 0 12px;
        color: #334155;
        background: rgba(15, 23, 42, 0.035);
        border-color: rgba(15, 23, 42, 0.12);

        &:hover,
        &:focus {
            color: var(--color-primary, #7c5cfc);
            background: rgba(124, 92, 252, 0.08);
            border-color: rgba(124, 92, 252, 0.16);
        }
    }

    &__search {
        min-width: 84px;
        padding: 0 14px;
        font-weight: 500;
        color: var(--color-primary, #7c5cfc);
        border-color: rgba(124, 92, 252, 0.32);
        background: rgba(124, 92, 252, 0.08);

        &:hover,
        &:focus {
            color: var(--color-primary-hover, #9578ff);
            background: rgba(124, 92, 252, 0.13);
            border-color: rgba(124, 92, 252, 0.44);
        }
    }
}

:global(html[data-theme='dark'] .x-filter-actions) {
    background: transparent;
}

:global(html[data-theme='dark'] .x-filter-actions .ant-btn.x-filter-actions__reset) {
    min-width: 84px;
    color: rgba(255, 255, 255, 0.72) !important;
    background: rgba(255, 255, 255, 0.045) !important;
    border-color: rgba(255, 255, 255, 0.14) !important;
    box-shadow: none !important;
}

:global(html[data-theme='dark'] .x-filter-actions .ant-btn.x-filter-actions__reset:hover),
:global(html[data-theme='dark'] .x-filter-actions .ant-btn.x-filter-actions__reset:focus) {
    color: #e6ddff !important;
    background: rgba(124, 92, 252, 0.12) !important;
    border-color: rgba(185, 168, 255, 0.28) !important;
    box-shadow: none !important;
}

:global(html[data-theme='dark'] .x-filter-actions .ant-btn.x-filter-actions__search) {
    color: #d9d0ff !important;
    background: rgba(124, 92, 252, 0.16) !important;
    border-color: rgba(185, 168, 255, 0.32) !important;
    box-shadow: none !important;
}

:global(html[data-theme='dark'] .x-filter-actions .ant-btn.x-filter-actions__search:hover),
:global(html[data-theme='dark'] .x-filter-actions .ant-btn.x-filter-actions__search:focus) {
    color: #ffffff !important;
    background: rgba(124, 92, 252, 0.24) !important;
    border-color: rgba(216, 208, 255, 0.48) !important;
    box-shadow: none !important;
}
</style>
