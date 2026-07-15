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
                <div style="display: flex; align-items: center; gap: 8px">
                    <a-input-password
                        :value="formData.api_keys && formData.api_keys.length > 0 ? formData.api_keys[0].value : ''"
                        placeholder="未绑定，请点击按钮进行绑定"
                        disabled
                        style="flex: 1" />
                    <a-button
                        type="primary"
                        @click="startXaiOAuth"
                        :loading="oauthLoading">
                        {{
                            formData.api_keys && formData.api_keys.length > 0 ? '重新绑定 (x.ai)' : '绑定 OAuth (x.ai)'
                        }}
                    </a-button>
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
import { ref } from 'vue'
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
let oauthPollTimer = null

async function startXaiOAuth() {
    try {
        oauthLoading.value = true
        const { data, success } = await apis.provider.startOAuth('xai').catch()
        if (!success || !data?.url) {
            message.error('获取授权链接失败')
            oauthLoading.value = false
            return
        }

        const authWindow = window.open(data.url, '_blank')
        if (!authWindow) {
            message.warning('弹出窗口被浏览器拦截，请允许弹出窗口后重试')
        }

        if (oauthPollTimer) {
            clearInterval(oauthPollTimer)
        }

        oauthPollTimer = setInterval(async () => {
            if (authWindow && authWindow.closed) {
                const statusRes = await apis.provider.pollOAuthStatus(data.state).catch()
                if (statusRes && statusRes.success && statusRes.data?.status === 'success') {
                    handleOAuthSuccess(statusRes.data)
                } else {
                    message.info('授权窗口已关闭，已终止绑定流程')
                    stopOAuthPolling()
                }
                return
            }

            const { data: statusData, success: statusSuccess } = await apis.provider.pollOAuthStatus(data.state).catch()
            if (statusSuccess && statusData?.status === 'success') {
                handleOAuthSuccess(statusData)
            }
        }, 3000)
    } catch (e) {
        oauthLoading.value = false
        message.error('启动授权流程失败')
    }
}

function handleOAuthSuccess(authData) {
    message.success('OAuth 凭证绑定成功！')
    formData.value.api_keys = [
        {
            value: authData.access_token,
            description: 'OAuth Token (x.ai)',
        },
    ]
    formData.value.oauth_refresh_token = authData.refresh_token
    if (authData.expires_in > 0) {
        formData.value.expires_at = new Date(Date.now() + authData.expires_in * 1000).toISOString()
    } else {
        formData.value.expires_at = null
    }
    stopOAuthPolling()
}

function stopOAuthPolling() {
    oauthLoading.value = false
    if (oauthPollTimer) {
        clearInterval(oauthPollTimer)
        oauthPollTimer = null
    }
}

function handleCreate() {
    showModal({
        type: 'create',
        title: t('pages.provider.add'),
    })
    formData.value.enabled = 1
    formData.value.auth_type = 'api_key'
    formData.value.api_keys = []
}

async function handleEdit(record = {}) {
    showModal({
        type: 'edit',
        title: t('pages.provider.edit'),
    })

    const { data, success } = await apis.provider.getProvider(record.id).catch()
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
                    oauth_refresh_token: formData.value.oauth_refresh_token,
                    expires_at: formData.value.expires_at,
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
}

defineExpose({
    handleCreate,
    handleEdit,
})
</script>

<style lang="less" scoped></style>
