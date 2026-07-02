<template>
    <a-drawer
        :open="modal.open"
        :title="modal.title"
        :width="720"
        :confirm-loading="modal.confirmLoading"
        @close="handleCancel"
        @afterOpenChange="handleAfterOpenChange">
        <a-form
            ref="formRef"
            :model="formData"
            :rules="formRules"
            :label-col="{ style: { width: '120px' } }">
            <a-form-item
                :label="$t('pages.model.form.model_name')"
                name="model_name">
                <a-input v-model:value="formData.model_name"></a-input>
            </a-form-item>

            <a-form-item
                :label="$t('pages.model.form.model_code')"
                name="model_code">
                <a-input v-model:value="formData.model_code"></a-input>
            </a-form-item>

            <a-form-item
                :label="$t('pages.model.form.space_code')"
                name="space_code">
                <a-select
                    v-model:value="formData.space_code"
                    show-search
                    :filter-option="filterSpaceOption">
                    <a-select-option
                        v-for="item in props.spaceOptions"
                        :key="item.code"
                        :value="item.code">
                        {{ item.name }} ({{ item.code }})
                    </a-select-option>
                </a-select>
            </a-form-item>

            <a-form-item
                :label="$t('pages.model.form.request_types')"
                name="request_types">
                <a-select
                    v-model:value="formData.request_types"
                    mode="multiple"
                    :placeholder="$t('pages.model.form.request_types.placeholder')">
                    <a-select-option value="chat_completion">Chat Completion</a-select-option>
                    <a-select-option value="messages">Messages (Claude)</a-select-option>
                    <a-select-option value="embedding">Embedding</a-select-option>
                    <a-select-option value="image_generation">Image Generation</a-select-option>
                    <a-select-option value="audio">Audio</a-select-option>
                    <a-select-option value="responses">Responses (OpenAI Beta/Compat)</a-select-option>
                </a-select>
            </a-form-item>

            <a-form-item
                :label="$t('pages.model.form.context_length')"
                name="context_length">
                <a-input-number
                    v-model:value="formData.context_length"
                    :min="0"
                    style="width: 100%" />
            </a-form-item>

            <a-form-item
                :label="$t('pages.model.form.max_output_tokens')"
                name="max_output_tokens">
                <a-input-number
                    v-model:value="formData.max_output_tokens"
                    :min="0"
                    style="width: 100%" />
            </a-form-item>

            <a-form-item
                :label="$t('pages.model.form.abilities')"
                name="abilities">
                <a-select
                    v-model:value="formData.abilities"
                    mode="multiple"
                    :placeholder="$t('pages.model.form.abilities.placeholder')">
                    <a-select-option value="stream">流式输出 (Stream)</a-select-option>
                    <a-select-option value="tool_call">工具调用 (Tool Call)</a-select-option>
                    <a-select-option value="reasoning">思维链 (Reasoning)</a-select-option>
                    <a-select-option value="structured_output">结构化输出 (Structured Output)</a-select-option>
                </a-select>
            </a-form-item>

            <a-form-item
                :label="$t('pages.model.form.owner')"
                name="owner">
                <a-select
                    v-model:value="formData.owner"
                    mode="combobox"
                    :placeholder="$t('pages.model.form.owner.placeholder')"
                    :options="ownerOptions"
                    :filter-option="filterOwnerOption"
                    allow-clear>
                </a-select>
            </a-form-item>

            <a-form-item
                :label="$t('pages.model.form.input_price')"
                name="input_price">
                <a-input-number
                    v-model:value="formData.input_price"
                    :min="0"
                    style="width: 100%"
                    :addon-after="$t('pages.model.form.price.unit')" />
            </a-form-item>

            <a-form-item
                :label="$t('pages.model.form.output_price')"
                name="output_price">
                <a-input-number
                    v-model:value="formData.output_price"
                    :min="0"
                    style="width: 100%"
                    :addon-after="$t('pages.model.form.price.unit')" />
            </a-form-item>

            <a-form-item
                :label="$t('pages.model.form.cached_price')"
                name="cached_price">
                <a-input-number
                    v-model:value="formData.cached_price"
                    :min="0"
                    style="width: 100%"
                    :addon-after="$t('pages.model.form.price.unit')" />
            </a-form-item>

            <a-form-item
                :label="$t('pages.model.form.cache_creation_price')"
                name="cache_creation_price">
                <a-input-number
                    v-model:value="formData.cache_creation_price"
                    :min="0"
                    style="width: 100%"
                    :addon-after="$t('pages.model.form.price.unit')" />
            </a-form-item>

            <a-form-item
                :label="$t('pages.model.form.enabled')"
                name="enabled">
                <a-switch
                    v-model:checked="formData.enabled"
                    :checked-value="1"
                    :un-checked-value="0" />
            </a-form-item>

            <a-form-item
                :label="$t('pages.model.form.description')"
                name="description">
                <a-textarea v-model:value="formData.description"></a-textarea>
            </a-form-item>
        </a-form>
        <template #footer>
            <div style="display: flex; justify-content: flex-end; gap: 8px">
                <a-button @click="handleCancel">{{ cancelText }}</a-button>
                <a-button
                    type="primary"
                    :loading="modal.confirmLoading"
                    @click="handleOk"
                    >{{ okText }}</a-button
                >
            </div>
        </template>
    </a-drawer>
</template>

<script setup>
import { cloneDeep } from 'lodash-es'
import { message } from 'ant-design-vue'
import { ref } from 'vue'
import { config } from '@/config'
import apis from '@/apis'
import { useForm, useModal } from '@/hooks'
import { useI18n } from 'vue-i18n'
import { initSpaceCode, setCurrentSpaceCode } from '@/utils/spaceStorage'
import { watch } from 'vue'

const props = defineProps({
    spaceOptions: {
        type: Array,
        default: () => [],
    },
})
const emit = defineEmits(['ok'])
const { modal, showModal, hideModal, showLoading, hideLoading } = useModal()
const { formRecord, formData, formRef, formRules, resetForm } = useForm()
const { t } = useI18n()
const cancelText = ref(t('button.cancel'))
const okText = ref(t('button.confirm'))

watch(
    () => formData.value.space_code,
    (val) => {
        if (val) {
            setCurrentSpaceCode(val)
        }
    }
)

function filterSpaceOption(input, option) {
    const label = option.children?.[0]?.children || ''
    return option.value.toLowerCase().includes(input.toLowerCase()) || label.toLowerCase().includes(input.toLowerCase())
}

const ownerOptions = ref([
    { value: 'OpenAI', label: 'OpenAI' },
    { value: 'DeepSeek', label: 'DeepSeek' },
    { value: 'Google', label: 'Google' },
    { value: 'Anthropic', label: 'Anthropic' },
    { value: 'XiaoMi', label: 'XiaoMi (小米)' },
    { value: 'Qwen', label: 'Qwen (通义千问)' },
    { value: 'Zhipu AI', label: 'Zhipu AI (智谱清言)' },
    { value: 'Moonshot AI', label: 'Moonshot AI (月之暗面)' },
    { value: 'MiniMax', label: 'MiniMax' },
    { value: 'Baichuan', label: 'Baichuan (百川智能)' },
    { value: 'ByteDance', label: 'ByteDance (火山/豆包)' },
    { value: 'Tencent', label: 'Tencent (腾讯混元)' },
    { value: 'Baidu', label: 'Baidu (百度文心)' },
    { value: 'StepFun', label: 'StepFun (阶跃星辰)' },
    { value: 'Meta', label: 'Meta (Llama)' },
    { value: 'Mistral', label: 'Mistral' },
    { value: 'Ollama', label: 'Ollama' },
])

function filterOwnerOption(input, option) {
    const val = option.value || ''
    const label = option.label || ''
    return val.toLowerCase().includes(input.toLowerCase()) || label.toLowerCase().includes(input.toLowerCase())
}

formRules.value = {
    model_name: { required: true, message: t('pages.model.form.model_name.placeholder') },
    model_code: { required: true, message: t('pages.model.form.model_code.placeholder') },
    space_code: { required: true, message: t('pages.model.form.space_code.placeholder') },
    request_types: { required: true, message: t('pages.model.form.request_types.placeholder') },
}

function handleCreate() {
    showModal({
        type: 'create',
        title: t('pages.model.add'),
    })
    formData.value.enabled = 1
    formData.value.space_code = initSpaceCode(props.spaceOptions)
    formData.value.max_output_tokens = 8192
    formData.value.abilities = []
    formData.value.input_price = 3.0
    formData.value.output_price = 10.0
    formData.value.cached_price = 1.0
    formData.value.cache_creation_price = 3.0
}

async function handleEdit(record = {}) {
    showModal({
        type: 'edit',
        title: t('pages.model.edit'),
    })

    const { data, success } = await apis.model.getModel(record.id).catch()
    if (!success) {
        message.error(t('component.message.error.request'))
        hideModal()
        return
    }
    // Parse request_types from JSON string to array
    if (data.request_types && typeof data.request_types === 'string') {
        try {
            data.request_types = JSON.parse(data.request_types)
        } catch (e) {
            data.request_types = []
        }
    }
    // Parse abilities from JSON string to array
    if (data.abilities && typeof data.abilities === 'string') {
        try {
            data.abilities = JSON.parse(data.abilities)
        } catch (e) {
            data.abilities = []
        }
    } else if (!data.abilities) {
        data.abilities = []
    }
    formRecord.value = data
    formData.value = cloneDeep(data)
}

function handleOk() {
    formRef.value
        .validateFields()
        .then(async (values) => {
            try {
                showLoading()
                // Convert request_types and abilities arrays to JSON string
                const params = {
                    ...values,
                    request_types: JSON.stringify(values.request_types || []),
                    abilities: JSON.stringify(values.abilities || []),
                }
                let result = null
                switch (modal.value.type) {
                    case 'create':
                        result = await apis.model.createModel(params).catch(() => {
                            throw new Error()
                        })
                        break
                    case 'edit':
                        result = await apis.model.updateModel(formData.value.id, params).catch(() => {
                            throw new Error()
                        })
                        break
                }
                hideLoading()
                if (config('http.code.success') === result?.success) {
                    hideModal()
                    message.success(t('component.message.success.save'))
                    emit('ok')
                } else {
                    message.error(t('component.message.error.save'))
                }
            } catch (error) {
                hideLoading()
                message.error(t('component.message.error.request'))
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

function handleAfterOpenChange(open) {
    if (!open) {
        onAfterClose()
    }
}

defineExpose({
    handleCreate,
    handleEdit,
})
</script>

<style lang="less" scoped></style>
