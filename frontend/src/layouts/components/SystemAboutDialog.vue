<template>
    <a-modal
        :open="open"
        :title="$t('app.about.title')"
        :width="440"
        :footer="null"
        :body-style="{ maxHeight: 'calc(100dvh - 180px)', overflowY: 'auto' }"
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
            <template v-if="summary">
                <dl class="system-about__details">
                    <dt>{{ $t('app.about.version') }}</dt>
                    <dd>{{ formatIdentity(summary.identity, labels) }}</dd>
                    <dt>{{ $t('app.about.channel') }}</dt>
                    <dd>{{ labels.channelValues[summary.identity?.install_channel] || $t('app.about.unknown') }}</dd>
                    <dt>{{ $t('app.about.build') }}</dt>
                    <dd>{{ labels.buildValues[summary.identity?.build?.kind] || $t('app.about.unknown') }}</dd>
                </dl>
                <section
                    v-if="summary.identity?.edition === 'professional'"
                    class="system-about__gateway">
                    <h3>{{ $t('app.about.gatewayVersions') }}</h3>
                    <p class="system-about__muted">{{ $t('app.about.gatewayWindow') }}</p>
                    <p class="system-about__muted">{{ gatewayScope }}</p>
                    <ul v-if="summary.gateway?.groups?.length">
                        <li
                            v-for="group in summary.gateway.groups"
                            :key="`${group.version}:${group.build_kind}`"
                            class="system-about__gateway-group">
                            <span class="system-about__group-version">{{
                                group.version || $t('app.about.unknown')
                            }}</span>
                            <span>{{ labels.buildValues[group.build_kind] || $t('app.about.unknown') }}</span>
                            <span class="system-about__group-count">{{
                                $t('app.about.nodeCount', { count: group.count })
                            }}</span>
                        </li>
                    </ul>
                    <p v-else>
                        {{
                            $t(
                                summary.gateway?.status === 'unavailable'
                                    ? 'app.about.gatewayUnavailable'
                                    : 'app.about.gatewayEmpty'
                            )
                        }}
                    </p>
                </section>
            </template>
            <dl
                v-else
                class="system-about__details">
                <dt>{{ $t('app.about.frontendBuild') }}</dt>
                <dd>{{ fallbackVersion }}</dd>
                <p class="system-about__muted">{{ $t('app.about.frontendBuildNotice') }}</p>
            </dl>
            <a-button
                block
                @click="copyVersion">
                <template #icon><copy-outlined aria-hidden="true" /></template>
                {{ $t('app.about.copy') }}
            </a-button>
            <section
                v-if="summary?.can_manage_updates === true"
                class="system-about__updates">
                <div class="system-about__update-heading">
                    <h3>{{ $t('app.about.updates') }}</h3>
                    <a-tag
                        v-if="hasUpdate"
                        color="blue"
                        >{{ $t('app.about.updateAvailable') }}</a-tag
                    >
                </div>
                <p
                    v-if="updates?.enabled === false"
                    class="system-about__muted">
                    {{ $t('app.about.disabledNotice') }}
                </p>
                <p
                    v-else-if="!updates"
                    class="system-about__muted">
                    {{ $t('app.about.updateUnavailable') }}
                </p>
                <p
                    v-if="summary.identity?.install_channel === 'unknown'"
                    class="system-about__muted">
                    {{ $t('app.about.unknownChannel') }}
                </p>
                <p
                    v-if="error"
                    role="alert"
                    class="system-about__error">
                    {{ errorMessage }}
                </p>
                <article
                    v-for="(item, index) in updates?.components || []"
                    :key="`${item.component}:${item.current}:${index}`"
                    class="system-about__component">
                    <div class="system-about__update-heading">
                        <strong>{{ componentLabel(item.component) }}</strong>
                        <a-tag :color="isActionable(item) ? 'blue' : undefined">{{ stateLabel(item) }}</a-tag>
                    </div>
                    <p class="system-about__current">
                        {{ $t('app.about.current') }} {{ item.current || $t('app.about.unknown') }}
                        <span v-if="item.count">{{ $t('app.about.nodeCount', { count: item.count }) }}</span>
                    </p>
                    <p
                        v-if="isHistory(item)"
                        class="system-about__muted">
                        {{ $t('app.about.historyResult', { version: item.source.candidate.version }) }}
                    </p>
                    <p v-else-if="item.latest">{{ $t('app.about.latestStable', { version: item.latest }) }}</p>
                    <p class="system-about__muted">
                        {{ $t('app.about.lastAttempt', { time: displayTime(item.source?.last_attempt) }) }}
                    </p>
                    <p class="system-about__muted">
                        {{ $t('app.about.lastSuccess', { time: displayTime(item.source?.last_success) }) }}
                    </p>
                    <template v-if="isActionable(item)">
                        <a
                            v-if="releaseURL(item)"
                            :href="releaseURL(item)"
                            target="_blank"
                            rel="noopener noreferrer"
                            >{{ $t('app.about.releaseNotes') }}</a
                        >
                        <template v-if="isHomebrew(item)">
                            <pre class="system-about__commands">{{ brewUpgrade }}</pre>
                            <p class="system-about__muted">{{ $t('app.about.homebrewRestartNotice') }}</p>
                            <pre class="system-about__commands">{{ brewRestart }}</pre>
                        </template>
                        <p
                            v-else
                            class="system-about__muted">
                            {{ $t('app.about.manualUpgradeNotice') }}
                        </p>
                        <a-button
                            size="small"
                            @click="copyText(upgradeGuidance(item))">
                            <template #icon><copy-outlined aria-hidden="true" /></template>
                            {{ $t('app.about.copyGuidance') }}
                        </a-button>
                    </template>
                </article>
                <a-button
                    block
                    class="system-about__check"
                    :loading="checking"
                    :disabled="updates?.enabled !== true || checking || retryAfterSeconds > 0"
                    @click="$emit('check')"
                    >{{ $t('app.about.checkUpdates') }}</a-button
                >
                <p
                    v-if="retryAfterSeconds > 0"
                    role="status"
                    class="system-about__muted">
                    {{ $t('app.about.retryAfter', { seconds: retryAfterSeconds }) }}
                </p>
            </section>
        </div>
    </a-modal>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { message } from 'ant-design-vue'
import { CopyOutlined } from '@ant-design/icons-vue'
import { config } from '@/config'
import { copyVersionText, formatIdentity, hasAvailableUpdate } from '@/utils/system-version'

const props = defineProps({
    open: { type: Boolean, default: false },
    summary: { type: Object, default: null },
    updates: { type: Object, default: null },
    fallbackVersion: { type: String, default: '' },
    checking: { type: Boolean, default: false },
    retryAfterSeconds: { type: Number, default: 0 },
    error: { type: Object, default: null },
    labels: { type: Object, required: true },
})
defineEmits(['update:open', 'check'])
const { t, locale } = useI18n()
const contentRef = ref(null)
const hasUpdate = computed(() => hasAvailableUpdate(props.summary, props.updates))
const gatewayScope = computed(() =>
    t(
        props.summary?.gateway?.scope === 'shared'
            ? 'app.about.scope.shared'
            : props.summary?.gateway?.scope === 'this_admin'
              ? 'app.about.scope.thisAdmin'
              : 'app.about.scope.unknown'
    )
)
const errorMessage = computed(() => {
    const response = props.error?.response
    if (response?.status === 409 && response.data?.error?.id === 'update_check_disabled') {
        return t('app.about.disabledNotice')
    }
    if (response?.status === 429 && response.data?.error?.id === 'update_check_cooldown') {
        return t('app.about.checkCooldown')
    }
    return t('app.about.checkFailed')
})
// These commands are local templates, never server-provided executable text.
const brewUpgrade = 'brew update\nbrew upgrade tokenlive'
const brewRestart = 'brew services restart tokenlive'
const knownStates = new Set([
    'available',
    'current',
    'ahead',
    'uncomparable',
    'no_candidate',
    'disabled',
    'stale',
    'unknown',
    'unavailable',
    'unchecked',
    'checking',
])

function isActionable(item) {
    return hasAvailableUpdate(props.summary, { ...props.updates, components: [item] })
}
function stateLabel(item) {
    let state = knownStates.has(item.state) ? item.state : 'unknown'
    if (props.updates?.enabled === false) state = 'disabled'
    else if (state === 'available' && !isActionable(item)) state = item.source?.stale === true ? 'stale' : 'unknown'
    return t(`app.about.state.${state}`)
}
function isHistory(item) {
    return (
        item.source?.candidate &&
        (props.updates?.enabled !== true || item.source.stale !== false || item.source.status !== 'ready')
    )
}
function componentLabel(component) {
    return t(`app.about.component.${['admin', 'gateway', 'standalone'].includes(component) ? component : 'unknown'}`)
}
function displayTime(value) {
    if (!value || value.startsWith('0001-')) return t('app.about.notChecked')
    const time = new Date(value)
    return Number.isNaN(time.getTime())
        ? t('app.about.unknown')
        : time.toLocaleString(locale.value === 'en-us' ? 'en-US' : 'zh-CN')
}
function releaseURL(item) {
    if (!isActionable(item) || !['admin', 'gateway', 'standalone'].includes(item.component)) return ''
    try {
        const url = new URL(item.source?.candidate?.release_url)
        const path = `/tokenlive/tokenlive-${item.component}/releases/tag/`
        if (
            url.protocol !== 'https:' ||
            url.host !== 'github.com' ||
            url.username ||
            url.password ||
            !url.pathname.startsWith(path) ||
            !url.pathname.slice(path.length) ||
            url.search ||
            url.hash
        )
            return ''
        return url.href
    } catch {
        return ''
    }
}
function isHomebrew(item) {
    return (
        item.component === 'standalone' &&
        props.summary?.identity?.edition === 'standalone' &&
        props.summary?.identity?.install_channel === 'homebrew'
    )
}
function upgradeGuidance(item) {
    const lines = [componentLabel(item.component)]
    if (releaseURL(item)) lines.push(releaseURL(item))
    if (isHomebrew(item)) lines.push(brewUpgrade, t('app.about.homebrewRestartNotice'), brewRestart)
    else lines.push(t('app.about.manualUpgradeNotice'))
    return lines.join('\n')
}

async function copyVersion() {
    const text = props.summary
        ? copyVersionText(props.summary, props.labels)
        : `${t('app.about.frontendBuildShort', { version: props.fallbackVersion })}\n${t('app.about.frontendBuildNotice')}`
    await copyText(text)
}
async function copyText(text) {
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

        dd + dt {
            margin-top: 12px;
        }
    }

    h3 {
        margin: 0 0 8px;
        font-size: 14px;
        font-weight: 600;
    }

    &__gateway,
    &__updates {
        margin-block: 20px;
    }

    &__gateway ul {
        padding: 0;
        list-style: none;
    }

    &__gateway-group {
        display: flex;
        flex-wrap: wrap;
        align-items: baseline;
        gap: 4px 12px;
        padding-block: 8px;
        border-bottom: 1px solid var(--color-border-secondary);
        font-size: 12px;
    }

    &__group-version {
        flex-basis: 100%;
        overflow-wrap: anywhere;
        font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    }

    &__group-count {
        margin-left: auto;
    }

    &__muted {
        margin: 6px 0;
        color: var(--color-text-secondary);
        font-size: 12px;
        overflow-wrap: anywhere;
    }

    &__error {
        color: var(--color-text-primary);
        border-left: 2px solid var(--color-primary);
        padding-left: 10px;
    }

    &__update-heading {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        justify-content: space-between;
        gap: 8px;
    }

    &__component {
        padding-block: 16px;
        border-top: 1px solid var(--color-border-secondary);

        p {
            overflow-wrap: anywhere;
        }
    }

    &__current {
        margin-block: 8px;
        font-size: 13px;
    }

    &__commands {
        padding: 10px;
        margin-block: 10px;
        border: 1px solid var(--color-border-secondary);
        border-radius: var(--radius-md);
        font-size: 12px;
        white-space: pre-wrap;
        overflow-wrap: anywhere;
    }
}
</style>
