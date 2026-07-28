<template>
    <a-modal
        :open="modal.open"
        :title="modal.title"
        :width="640"
        :confirm-loading="modal.confirmLoading"
        :after-close="onAfterClose"
        :cancel-text="cancelText"
        :ok-text="okText"
        @ok="handleOk"
        @cancel="handleCancel">
        <a-form
            ref="formRef"
            :model="formData"
            :rules="formRules"
            layout="vertical"
            style="margin-top: 16px">
            <a-row :gutter="16">
                <a-col :span="12">
                    <a-form-item
                        :label="$t('pages.provider.form.name')"
                        name="name">
                        <a-input
                            v-model:value="formData.name"
                            :placeholder="$t('pages.provider.form.name.placeholder')" />
                    </a-form-item>
                </a-col>
                <a-col :span="12">
                    <a-form-item
                        :label="$t('pages.provider.form.code')"
                        name="code">
                        <a-input
                            v-model:value="formData.code"
                            :placeholder="$t('pages.provider.form.code.placeholder')" />
                    </a-form-item>
                </a-col>
            </a-row>

            <a-row :gutter="16">
                <a-col :span="12">
                    <a-form-item
                        :label="$t('pages.provider.form.protocol')"
                        name="protocol">
                        <a-select
                            v-model:value="formData.protocol"
                            :placeholder="$t('pages.provider.form.protocol.placeholder')">
                            <a-select-option value="openai">OpenAI</a-select-option>
                            <a-select-option value="anthropic">Anthropic</a-select-option>
                            <a-select-option value="gemini">Gemini</a-select-option>
                            <a-select-option value="joycode">JoyCode</a-select-option>
                        </a-select>
                    </a-form-item>
                </a-col>
                <a-col :span="12">
                    <a-form-item
                        :label="$t('pages.provider.form.enabled')"
                        name="enabled">
                        <a-switch
                            v-model:checked="formData.enabled"
                            :checked-value="1"
                            :un-checked-value="0"
                            style="margin-top: 4px" />
                    </a-form-item>
                </a-col>
            </a-row>

            <a-form-item
                :label="$t('pages.provider.form.url')"
                name="url">
                <a-input
                    v-model:value="formData.url"
                    :placeholder="$t('pages.provider.form.url.placeholder')" />
            </a-form-item>

            <a-form-item
                label="验证类型"
                name="auth_type">
                <a-radio-group v-model:value="formData.auth_type">
                    <a-radio value="api_key">API Key 密钥</a-radio>
                    <a-radio value="oauth_token">OAuth 凭证</a-radio>
                </a-radio-group>
            </a-form-item>

            <a-form-item
                v-if="formData.auth_type === 'api_key'"
                :label="$t('pages.provider.form.api_keys')"
                name="api_keys">
                <div
                    v-for="(item, index) in formData.api_keys"
                    :key="index"
                    style="display: flex; align-items: center; margin-bottom: 8px; gap: 8px">
                    <a-input-password
                        v-model:value="formData.api_keys[index].value"
                        :placeholder="$t('pages.provider.form.api_keys.placeholder')"
                        style="flex: 2" />
                    <a-input
                        v-model:value="formData.api_keys[index].description"
                        :placeholder="$t('pages.provider.form.api_keys.description_placeholder', '备注')"
                        style="flex: 1" />
                    <minus-circle-outlined
                        @click="removeApiKey(index)"
                        style="color: #ff4d4f; cursor: pointer; flex-shrink: 0" />
                </div>
                <a-button
                    type="dashed"
                    @click="addApiKey"
                    style="width: 100%">
                    <plus-outlined />
                    {{ $t('pages.provider.form.api_keys.add') }}
                </a-button>
            </a-form-item>

            <a-form-item
                v-if="formData.auth_type === 'oauth_token'"
                label="OAuth 凭证"
                name="api_keys">
                <div style="display: flex; flex-direction: column; gap: 12px">
                    <div style="display: flex; align-items: center; gap: 8px">
                        <a-select
                            v-model:value="oauthProvider"
                            style="width: 140px"
                            :options="oauthProviderOptions" />
                        <a-input-password
                            :value="formData.api_keys && formData.api_keys.length > 0 ? formData.api_keys[0].value : ''"
                            placeholder="未绑定，请选择提供方后点击绑定"
                            disabled
                            style="flex: 1" />
                        <a-button
                            type="primary"
                            @click="startOAuthBind"
                            :loading="oauthLoading">
                            {{ oauthBindButtonText }}
                        </a-button>
                    </div>

                    <div
                        v-if="oauthProvider === 'codex' && oauthCodexState"
                        style="display: flex; flex-direction: column; gap: 8px">
                        <a-alert
                            type="info"
                            show-icon
                            message="请在打开的页面完成 OpenAI/Codex 登录。浏览器会跳转到 localhost:1455 回调地址（可能无法打开页面），请复制地址栏完整 URL 粘贴到下方提交。" />
                        <a-textarea
                            v-model:value="oauthCallbackUrl"
                            :rows="3"
                            placeholder="粘贴回调 URL，例如 http://localhost:1455/auth/callback?code=...&state=..." />
                        <div style="display: flex; justify-content: flex-end; gap: 8px">
                            <a-button @click="cancelCodexOAuth">取消</a-button>
                            <a-button
                                type="primary"
                                :loading="oauthCompleteLoading"
                                @click="submitCodexCallback">
                                提交回调 URL
                            </a-button>
                        </div>
                    </div>

                    <div
                        v-if="formData.oauth && (formData.oauth.account_id || formData.oauth.email)"
                        class="oauth-account-card">
                        <div class="oauth-account-card__header">
                            <span class="oauth-account-card__title">已绑定账号</span>
                            <a-tag
                                v-if="oauthProvider"
                                color="processing"
                                class="oauth-account-card__provider">
                                {{
                                    oauthProvider === 'codex'
                                        ? 'Codex'
                                        : oauthProvider === 'xai'
                                          ? 'x.ai'
                                          : oauthProvider
                                }}
                            </a-tag>
                        </div>
                        <div class="oauth-account-card__body">
                            <div
                                v-if="formData.oauth.email"
                                class="oauth-account-card__row">
                                <span class="oauth-account-card__label">账号邮箱</span>
                                <span
                                    class="oauth-account-card__value"
                                    :title="formData.oauth.email">
                                    {{ formData.oauth.email }}
                                </span>
                                <a-button
                                    type="link"
                                    size="small"
                                    class="oauth-account-card__copy"
                                    @click="copyText(formData.oauth.email, '邮箱')">
                                    复制
                                </a-button>
                            </div>
                            <div
                                v-if="formData.oauth.account_id"
                                class="oauth-account-card__row">
                                <span class="oauth-account-card__label">Account ID</span>
                                <span
                                    class="oauth-account-card__value oauth-account-card__value--mono"
                                    :title="formData.oauth.account_id">
                                    {{ formData.oauth.account_id }}
                                </span>
                                <a-button
                                    type="link"
                                    size="small"
                                    class="oauth-account-card__copy"
                                    @click="copyText(formData.oauth.account_id, 'Account ID')">
                                    复制
                                </a-button>
                            </div>
                        </div>
                    </div>
                </div>
            </a-form-item>

            <a-form-item
                :label="$t('pages.provider.form.description')"
                name="description">
                <a-textarea
                    v-model:value="formData.description"
                    :rows="3"
                    :placeholder="$t('pages.provider.form.description.placeholder')" />
            </a-form-item>
        </a-form>
    </a-modal>
</template>

<script setup>
import { cloneDeep } from 'lodash-es'
import { message } from 'ant-design-vue'
import { computed, ref } from 'vue'
import { PlusOutlined, MinusCircleOutlined } from '@ant-design/icons-vue'
import { config } from '@/config'
import apis from '@/apis'
import { useForm, useModal } from '@/hooks'

const emit = defineEmits(['ok'])
import { useI18n } from 'vue-i18n'
const { modal, showModal, hideModal, showLoading, hideLoading } = useModal()
const { formRecord, formData, formRef, formRules, resetForm } = useForm()
const { t } = useI18n()
const cancelText = ref(t('button.cancel'))
const okText = ref(t('button.confirm'))
formRules.value = {
    code: { required: true, message: t('pages.provider.form.code.placeholder') },
    name: { required: true, message: t('pages.provider.form.name.placeholder') },
    protocol: { required: true, message: t('pages.provider.form.protocol.placeholder') },
}

function addApiKey() {
    if (!formData.value.api_keys) {
        formData.value.api_keys = []
    }
    formData.value.api_keys.push({ value: '', description: '' })
}

function removeApiKey(index) {
    formData.value.api_keys.splice(index, 1)
}

const oauthLoading = ref(false)
const oauthCompleteLoading = ref(false)
const oauthProvider = ref('xai')
const oauthProviderOptions = [
    { label: 'x.ai', value: 'xai' },
    { label: 'Codex', value: 'codex' },
]
const oauthCodexState = ref('')
const oauthCallbackUrl = ref('')
let oauthPollTimer = null
let oauthPollDeadline = 0

const oauthBindButtonText = computed(() => {
    const bound = formData.value.api_keys && formData.value.api_keys.length > 0
    const name = oauthProvider.value === 'codex' ? 'Codex' : 'x.ai'
    return bound ? `重新绑定 (${name})` : `绑定 OAuth (${name})`
})

async function startOAuthBind() {
    if (oauthProvider.value === 'codex') {
        await startCodexOAuth()
        return
    }
    await startXaiOAuth()
}

async function startXaiOAuth() {
    try {
        cancelCodexOAuth()
        oauthLoading.value = true
        const { data, success } = await apis.provider.startOAuth('xai').catch(() => ({}))
        if (!success || !data?.url) {
            message.error('获取授权链接失败')
            oauthLoading.value = false
            return
        }

        const authWindow = window.open(data.url, '_blank')
        if (!authWindow) {
            message.warning('弹出窗口被浏览器拦截，请允许弹出窗口后重试')
        }
        if (data.user_code) {
            message.info(`请在打开的页面中确认设备码：${data.user_code}`, 8)
        }

        if (oauthPollTimer) {
            clearInterval(oauthPollTimer)
        }

        // Device-code flow: poll until the backend receives the token, or time out.
        oauthPollDeadline = Date.now() + 10 * 60 * 1000
        oauthPollTimer = setInterval(async () => {
            if (Date.now() > oauthPollDeadline) {
                message.warning('授权超时，请重试')
                stopOAuthPolling()
                return
            }
            const { data: statusData, success: statusSuccess } = await apis.provider
                .pollOAuthStatus(data.state)
                .catch(() => ({}))
            if (statusSuccess && statusData?.status === 'success') {
                handleOAuthSuccess(statusData, 'xai')
            }
        }, 3000)
    } catch (e) {
        oauthLoading.value = false
        message.error('启动授权流程失败')
    }
}

async function startCodexOAuth() {
    try {
        stopOAuthPolling()
        oauthLoading.value = true
        oauthCallbackUrl.value = ''
        oauthCodexState.value = ''
        const { data, success } = await apis.provider.startOAuth('codex').catch(() => ({}))
        if (!success || !data?.url || !data?.state) {
            message.error('获取授权链接失败')
            oauthLoading.value = false
            return
        }

        oauthCodexState.value = data.state
        const authWindow = window.open(data.url, '_blank')
        if (!authWindow) {
            message.warning('弹出窗口被浏览器拦截，请允许弹出窗口后重试，或手动打开授权链接')
        }
        message.info('请完成登录后，将回调 URL 粘贴到下方', 6)
        oauthLoading.value = false
    } catch (e) {
        oauthLoading.value = false
        message.error('启动授权流程失败')
    }
}

async function submitCodexCallback() {
    const callbackUrl = (oauthCallbackUrl.value || '').trim()
    if (!oauthCodexState.value) {
        message.warning('请先点击绑定按钮启动 Codex 授权')
        return
    }
    if (!callbackUrl) {
        message.warning('请粘贴回调 URL')
        return
    }
    try {
        oauthCompleteLoading.value = true
        const { data, success } = await apis.provider
            .completeOAuth({
                provider: 'codex',
                state: oauthCodexState.value,
                callback_url: callbackUrl,
            })
            .catch(() => ({}))
        oauthCompleteLoading.value = false
        if (!success || !data?.access_token) {
            message.error('提交回调失败，请确认 URL 完整且未过期后重试')
            return
        }
        handleOAuthSuccess(data, 'codex')
    } catch (e) {
        oauthCompleteLoading.value = false
        message.error('提交回调失败')
    }
}

function cancelCodexOAuth() {
    oauthCodexState.value = ''
    oauthCallbackUrl.value = ''
    oauthCompleteLoading.value = false
}

async function copyText(text, label = '内容') {
    const value = (text || '').trim()
    if (!value) {
        return
    }
    try {
        if (navigator?.clipboard?.writeText) {
            await navigator.clipboard.writeText(value)
        } else {
            const textarea = document.createElement('textarea')
            textarea.value = value
            textarea.setAttribute('readonly', 'readonly')
            textarea.style.position = 'fixed'
            textarea.style.left = '-9999px'
            document.body.appendChild(textarea)
            textarea.select()
            document.execCommand('copy')
            document.body.removeChild(textarea)
        }
        message.success(`${label}已复制`)
    } catch (e) {
        message.error(`${label}复制失败`)
    }
}

function handleOAuthSuccess(authData, providerKey) {
    const key = providerKey || authData.provider || oauthProvider.value || 'xai'
    const label = key === 'codex' ? 'codex' : key === 'xai' ? 'x.ai' : key
    message.success('OAuth 凭证绑定成功！')
    formData.value.api_keys = [
        {
            value: authData.access_token,
            description: `OAuth Token (${label})`,
        },
    ]
    // base_url is what the model call uses -> store it into the provider url column.
    if (authData.base_url) {
        formData.value.url = authData.base_url
    }
    // refresh_token + token_endpoint + expires_at (+ codex metadata) go into oauth JSON.
    const oauth = {
        refresh_token: authData.refresh_token,
        token_endpoint: authData.token_endpoint,
    }
    if (authData.expires_in > 0) {
        oauth.expires_at = new Date(Date.now() + authData.expires_in * 1000).toISOString()
    }
    if (authData.account_id) {
        oauth.account_id = authData.account_id
    }
    if (authData.email) {
        oauth.email = authData.email
    }
    formData.value.oauth = oauth
    oauthProvider.value = key === 'codex' ? 'codex' : 'xai'
    cancelCodexOAuth()
    stopOAuthPolling()
}

function stopOAuthPolling() {
    oauthLoading.value = false
    if (oauthPollTimer) {
        clearInterval(oauthPollTimer)
        oauthPollTimer = null
    }
}

function inferOAuthProvider(record = {}) {
    const endpoint = record?.oauth?.token_endpoint || ''
    if (typeof endpoint === 'string' && endpoint.includes('openai.com')) {
        return 'codex'
    }
    if (typeof endpoint === 'string' && endpoint.includes('x.ai')) {
        return 'xai'
    }
    const desc = record?.api_keys?.[0]?.description || ''
    if (typeof desc === 'string' && desc.toLowerCase().includes('codex')) {
        return 'codex'
    }
    return 'xai'
}

function handleCreate() {
    showModal({
        type: 'create',
        title: t('pages.provider.add'),
    })
    formData.value.enabled = 1
    formData.value.auth_type = 'api_key'
    formData.value.api_keys = []
    oauthProvider.value = 'xai'
    cancelCodexOAuth()
}

async function handleEdit(record = {}) {
    showModal({
        type: 'edit',
        title: t('pages.provider.edit'),
    })

    const { data, success } = await apis.provider.getProvider(record.id).catch(() => ({}))
    if (!success) {
        message.error(t('component.message.error.save'))
        hideModal()
        return
    }
    formRecord.value = data
    formData.value = cloneDeep(data)
    if (!formData.value.auth_type) {
        formData.value.auth_type = 'api_key'
    }
    if (!Array.isArray(formData.value.api_keys)) {
        formData.value.api_keys = []
    }
    oauthProvider.value = inferOAuthProvider(formData.value)
    cancelCodexOAuth()
}

function handleOk() {
    formRef.value
        .validateFields()
        .then(async (values) => {
            try {
                showLoading()
                const params = {
                    ...values,
                    api_keys: formData.value.api_keys,
                    oauth: formData.value.oauth,
                }
                let result = null
                switch (modal.value.type) {
                    case 'create':
                        result = await apis.provider.createProvider(params).catch(() => {
                            throw new Error()
                        })
                        break
                    case 'edit':
                        result = await apis.provider.updateProvider(formData.value.id, params).catch(() => {
                            throw new Error()
                        })
                        break
                }
                hideLoading()
                if (config('http.code.success') === result?.success) {
                    hideModal()
                    emit('ok')
                }
            } catch (error) {
                hideLoading()
            }
        })
        .catch(() => {
            hideLoading()
        })
}

function handleCancel() {
    hideModal()
}

function onAfterClose() {
    resetForm()
    hideLoading()
    stopOAuthPolling()
    cancelCodexOAuth()
    oauthProvider.value = 'xai'
}

defineExpose({
    handleCreate,
    handleEdit,
})
</script>

<style lang="less" scoped>
.oauth-account-card {
    border: 1px solid var(--color-border-secondary);
    background: var(--color-bg-hover);
    border-radius: var(--radius-md);
    padding: 10px 12px;
}

.oauth-account-card__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    margin-bottom: 8px;
}

.oauth-account-card__title {
    color: var(--color-text-secondary);
    font-size: var(--font-size-sm);
    font-weight: 500;
    line-height: 1.4;
}

.oauth-account-card__provider {
    margin-inline-end: 0;
}

.oauth-account-card__body {
    display: flex;
    flex-direction: column;
    gap: 6px;
}

.oauth-account-card__row {
    display: grid;
    grid-template-columns: 84px minmax(0, 1fr) auto;
    align-items: center;
    column-gap: 8px;
    min-height: 28px;
}

.oauth-account-card__label {
    color: var(--color-text-tertiary);
    font-size: var(--font-size-xs);
    line-height: 1.4;
}

.oauth-account-card__value {
    color: var(--color-text-primary);
    font-size: var(--font-size-sm);
    line-height: 1.4;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.oauth-account-card__value--mono {
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
    font-size: 12px;
}

.oauth-account-card__copy {
    padding-inline: 4px;
    height: 24px;
}
</style>
