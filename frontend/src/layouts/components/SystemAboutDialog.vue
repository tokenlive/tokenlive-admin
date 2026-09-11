<template>
    <a-modal
        :open="open"
        :title="$t('app.about.title')"
        :width="440"
        :footer="null"
        @cancel="$emit('update:open', false)">
        <div
            ref="contentRef"
            class="system-about">
            <div class="system-about__identity">
                <img
                    :src="config('app.logo')"
                    alt=""
                    width="40"
                    height="40" />
                <div>
                    <div class="system-about__name">TokenLive</div>
                    <p class="system-about__description">{{ $t('app.about.description') }}</p>
                </div>
            </div>
            <dl class="system-about__details">
                <dt>{{ $t('app.about.version') }}</dt>
                <dd>{{ version }}</dd>
            </dl>
            <a-button
                block
                @click="copyVersion">
                <template #icon><copy-outlined /></template>
                {{ $t('app.about.copy') }}
            </a-button>
        </div>
    </a-modal>
</template>

<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { message } from 'ant-design-vue'
import { CopyOutlined } from '@ant-design/icons-vue'
import { config } from '@/config'

const props = defineProps({
    open: { type: Boolean, default: false },
    version: { type: String, required: true },
})
defineEmits(['update:open'])
const { t } = useI18n()
const contentRef = ref(null)

async function copyVersion() {
    const text = `TokenLive ${props.version}`
    try {
        if (navigator.clipboard?.writeText) {
            await navigator.clipboard.writeText(text)
        } else {
            // Keep the fallback inside the modal's focus trap for HTTP deployments.
            const textarea = document.createElement('textarea')
            textarea.value = text
            textarea.readOnly = true
            textarea.style.cssText = 'position: absolute; opacity: 0; pointer-events: none'
            const activeElement = document.activeElement
            contentRef.value.appendChild(textarea)
            try {
                textarea.select()
                if (!document.execCommand('copy')) throw new Error('Copy failed')
            } finally {
                textarea.remove()
                activeElement?.focus()
            }
        }
        message.success(t('component.message.success.copy'))
    } catch {
        message.error(t('component.message.error.copy'))
    }
}
</script>

<style lang="less" scoped>
.system-about {
    padding-top: 16px;

    &__identity {
        display: flex;
        align-items: center;
        gap: 14px;

        img {
            flex-shrink: 0;
            object-fit: contain;
        }
    }

    &__name {
        color: var(--color-text-primary);
        font-size: 20px;
        font-weight: 650;
        letter-spacing: -0.02em;
    }

    &__description {
        margin: 4px 0 0;
        color: var(--color-text-secondary);
        font-size: 13px;
    }

    &__details {
        margin: 24px 0;
        padding-block: 16px;
        border-block: 1px solid var(--color-border-secondary);

        dt {
            margin-bottom: 8px;
            color: var(--color-text-secondary);
            font-size: 12px;
        }

        dd {
            margin: 0;
            color: var(--color-text-primary);
            font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
            overflow-wrap: anywhere;
            user-select: text;
        }
    }
}
</style>
