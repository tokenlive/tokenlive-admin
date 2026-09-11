<template>
    <div
        class="sidebar-version"
        :class="[`sidebar-version--${theme}`, { 'sidebar-version--collapsed': collapsed }]">
        <a-tooltip
            placement="right"
            :title="collapsed ? `${$t('app.about.title')} · ${version}` : undefined">
            <button
                type="button"
                class="sidebar-version__button"
                :aria-label="`${$t('app.about.title')} · ${version}`"
                aria-haspopup="dialog"
                @click="$emit('about')">
                <span
                    v-if="!collapsed"
                    class="sidebar-version__text">
                    <span>{{ $t('app.about.version') }}</span>
                    <span class="sidebar-version__number">{{ version }}</span>
                </span>
                <info-circle-outlined aria-hidden="true" />
            </button>
        </a-tooltip>
    </div>
</template>

<script setup>
import { InfoCircleOutlined } from '@ant-design/icons-vue'
import { theme as antTheme } from 'ant-design-vue'

defineProps({
    version: { type: String, required: true },
    collapsed: { type: Boolean, default: false },
    theme: { type: String, default: 'dark' },
})
defineEmits(['about'])
const { token } = antTheme.useToken()
</script>

<style lang="less" scoped>
.sidebar-version {
    --version-text: v-bind('token.colorTextSecondary');
    --version-hover: v-bind('token.colorFillTertiary');
    --version-border: v-bind('token.colorSplit');
    padding: 8px;
    border-top: 1px solid var(--version-border);

    &--dark {
        --version-text: var(--color-sidebar-text);
        --version-hover: rgba(148, 163, 184, 0.075);
        --version-border: rgba(148, 163, 184, 0.12);
    }

    &__button {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 12px;
        width: 100%;
        min-height: 36px;
        padding: 8px 12px;
        border: 0;
        border-radius: var(--radius-md);
        background: transparent;
        color: var(--version-text);
        font: inherit;
        font-size: 12px;
        text-align: left;
        cursor: pointer;

        &:hover {
            background: var(--version-hover);
        }

        &:focus-visible {
            outline: 2px solid var(--color-primary);
            outline-offset: -2px;
        }

        > .anticon {
            flex-shrink: 0;
            font-size: 14px;
        }
    }

    &__text {
        display: flex;
        align-items: baseline;
        gap: 8px;
        min-width: 0;
        white-space: nowrap;
    }

    &__number {
        overflow: hidden;
        text-overflow: ellipsis;
        font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    }

    &--collapsed &__button {
        justify-content: center;
        padding-inline: 0;
    }
}
</style>
