<template>
    <a-modal
        :open="modal.open"
        :title="modal.title"
        :width="580"
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
            :label-col="{ style: { width: '100px' } }">
            <a-card class="mb-8-2">
                <a-form-item
                    :label="$t('pages.provider.form.name')"
                    name="name">
                    <a-input v-model:value="formData.name"></a-input>
                </a-form-item>

                <a-form-item
                    :label="$t('pages.provider.form.code')"
                    name="code">
                    <a-input v-model:value="formData.code"></a-input>
                </a-form-item>

                <a-form-item
                    :label="$t('pages.provider.form.protocol')"
                    name="protocol">
                    <a-select v-model:value="formData.protocol">
                        <a-select-option value="openai">OpenAI</a-select-option>
                        <a-select-option value="anthropic">Anthropic</a-select-option>
                        <a-select-option value="google">Google</a-select-option>
                        <a-select-option value="deepseek">DeepSeek</a-select-option>
                        <a-select-option value="qwen">Qwen</a-select-option>
                        <a-select-option value="ollama">Ollama</a-select-option>
                        <a-select-option value="joycode">JoyCode</a-select-option>
                    </a-select>
                </a-form-item>

                <a-form-item
                    :label="$t('pages.provider.form.url')"
                    name="url">
                    <a-input
                        :placeholder="$t('pages.provider.form.url.placeholder')"
                        v-model:value="formData.url"></a-input>
                </a-form-item>

                <a-form-item
                    :label="$t('pages.provider.form.api_keys')"
                    name="api_keys">
                    <div
                        v-for="(key, index) in formData.api_keys"
                        :key="index"
                        style="display: flex; align-items: center; margin-bottom: 8px">
                        <a-input-password
                            v-model:value="formData.api_keys[index]"
                            :placeholder="$t('pages.provider.form.api_keys.placeholder')"
                            style="flex: 1; margin-right: 8px" />
                        <minus-circle-outlined
                            @click="removeApiKey(index)"
                            style="color: #ff4d4f; cursor: pointer" />
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
                    :label="$t('pages.provider.form.enabled')"
                    name="enabled">
                    <a-switch
                        v-model:checked="formData.enabled"
                        :checked-value="1"
                        :un-checked-value="0" />
                </a-form-item>

                <a-form-item
                    :label="$t('pages.provider.form.description')"
                    name="description">
                    <a-textarea v-model:value="formData.description"></a-textarea>
                </a-form-item>
            </a-card>
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
    formData.value.api_keys.push('')
}

function removeApiKey(index) {
    formData.value.api_keys.splice(index, 1)
}

function handleCreate() {
    showModal({
        type: 'create',
        title: t('pages.provider.add'),
    })
    formData.value.enabled = 1
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
    if (!Array.isArray(formData.value.api_keys)) {
        formData.value.api_keys = formData.value.api_keys ? [formData.value.api_keys] : []
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
}

defineExpose({
    handleCreate,
    handleEdit,
})
</script>

<style lang="less" scoped></style>
