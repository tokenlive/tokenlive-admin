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
            :disabled="loadingRecord"
            :label-col="{ style: { width: '120px' } }">
            <a-form-item
                :label="$t('pages.model.form.model_name')"
                name="model_name">
                <a-input v-model:value="formData.model_name"></a-input>
            </a-form-item>

            <a-form-item
                :label="$t('pages.model.form.model_type')"
                name="model_type"
                :extra="smartTypeBlocked ? $t('pages.model.smart.conversion_blocked') : undefined">
                <a-radio-group
                    :value="formData.model_type"
                    @change="handleModelTypeChange">
                    <a-radio value="normal">{{ $t('pages.model.type.normal') }}</a-radio>
                    <a-radio
                        value="smart"
                        :disabled="smartTypeBlocked"
                        >{{ $t('pages.model.type.smart') }}</a-radio
                    >
                </a-radio-group>
            </a-form-item>

            <a-form-item
                :label="$t('pages.model.form.model_code')"
                name="model_code">
                <a-input
                    v-model:value="formData.model_code"
                    :disabled="modal.type === 'edit'"></a-input>
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
                    :disabled="isSmartModel"
                    :placeholder="$t('pages.model.form.request_types.placeholder')">
                    <a-select-option value="chat_completion">Chat Completion</a-select-option>
                    <a-select-option value="responses">Responses (OpenAI Beta/Compat)</a-select-option>
                    <a-select-option value="messages">Messages (Claude)</a-select-option>
                    <a-select-option value="gemini_generate_content">Gemini Generate Content</a-select-option>
                    <a-select-option value="embedding">Embedding</a-select-option>
                    <a-select-option value="image_generation">Image Generation</a-select-option>
                    <a-select-option value="audio">Audio</a-select-option>
                </a-select>
            </a-form-item>

            <a-form-item
                :label="$t('pages.model.form.context_length')"
                name="context_length">
                <a-auto-complete
                    v-model:value="formData.context_length"
                    :placeholder="$t('pages.model.form.context_length.placeholder')"
                    :options="contextLengthOptions"
                    :filter-option="filterContextLengthOption"
                    allow-clear
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
                    :disabled="enableBlocked"
                    :checked-value="1"
                    :un-checked-value="0" />
                <p
                    v-if="enableBlocked"
                    class="smart-disable-warning">
                    {{ $t('pages.model.smart.enable_after_config') }}
                </p>
                <a-alert
                    v-if="formData.enabled === 0 && referencedBy.length"
                    class="smart-disable-warning"
                    type="warning"
                    show-icon
                    :message="$t('pages.model.smart.disable_warning')"
                    :description="referencedBy.map((model) => model.model_name || model.model_code).join(', ')" />
            </a-form-item>

            <a-form-item
                :label="$t('pages.model.form.description')"
                name="description">
                <a-textarea v-model:value="formData.description"></a-textarea>
            </a-form-item>

            <template v-if="modal.type === 'create'">
                <a-form-item
                    :label="$t('pages.model.form.recommended_policies')"
                    :extra="$t('pages.model.form.recommended_policies.hint')">
                    <div class="recommended-policies">
                        <a-checkbox v-model:checked="formData.apply_invocation_seed">
                            {{ $t('pages.model.form.apply_invocation_seed') }}
                        </a-checkbox>
                        <a-checkbox v-model:checked="formData.apply_circuit_break_seed">
                            {{ $t('pages.model.form.apply_circuit_break_seed') }}
                        </a-checkbox>
                    </div>
                </a-form-item>
            </template>
        </a-form>
        <template #footer>
            <div style="display: flex; justify-content: flex-end; gap: 8px">
                <a-button @click="handleCancel">{{ cancelText }}</a-button>
                <a-button
                    type="primary"
                    :loading="modal.confirmLoading"
                    :disabled="loadingRecord"
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
import { computed, ref, watch } from 'vue'
import { config } from '@/config'
import apis from '@/apis'
import { useForm, useModal } from '@/hooks'
import { useI18n } from 'vue-i18n'
import { initSpaceCode, setCurrentSpaceCode } from '@/utils/spaceStorage'
import { buildModelRoutingFields, isModelEnableBlocked, getModelSaveFeedback } from '@/utils/smart-model'
import {
    CONTEXT_LENGTH_OPTIONS,
    filterContextLengthOption,
    parseContextLength,
    toContextLengthSelectValue,
} from '@/enums/model'

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
const loadingRecord = ref(false)
const isSmartModel = computed(() => formData.value.model_type === 'smart')
const referencedBy = computed(() => formRecord.value?.referenced_by || [])
const smartTypeBlocked = computed(
    () =>
        modal.value.type === 'edit' &&
        formRecord.value?.model_type !== 'smart' &&
        (Number(formRecord.value?.endpoint_count) > 0 || referencedBy.value.length > 0)
)
const enableBlocked = computed(() =>
    isModelEnableBlocked({
        ...formData.value,
        smart_routing_ready:
            formRecord.value?.model_type === 'smart' &&
            formRecord.value?.smart_routing_ready === true &&
            formData.value.space_code === formRecord.value?.space_code,
    })
)
let ordinaryRequestTypes = null

function handleModelTypeChange(event) {
    const type = event.target.value
    if (type === 'smart' && smartTypeBlocked.value) return
    if (type === 'smart') {
        ordinaryRequestTypes = [...(formData.value.request_types || [])]
        formData.value.request_types = ['chat_completion']
        if (formRecord.value?.model_type !== 'smart' || formRecord.value?.smart_routing_ready !== true) {
            formData.value.enabled = 0
        }
    } else {
        formData.value.request_types = ordinaryRequestTypes || ['chat_completion']
    }
    formData.value.model_type = type
}

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

const contextLengthOptions = CONTEXT_LENGTH_OPTIONS

const ownerOptions = ref([
    { value: 'OpenAI', label: 'OpenAI' },
    { value: 'DeepSeek', label: 'DeepSeek' },
    { value: 'Google', label: 'Google' },
    { value: 'Anthropic', label: 'Anthropic' },
    { value: 'JD', label: 'JD (京东 JoyCode)' },
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
    { value: 'xAI', label: 'xAI' },
])

function filterOwnerOption(input, option) {
    const val = option.value || ''
    const label = option.label || ''
    return val.toLowerCase().includes(input.toLowerCase()) || label.toLowerCase().includes(input.toLowerCase())
}

formRules.value = {
    model_name: { required: true, message: t('pages.model.form.model_name.placeholder') },
    model_code: [
        { required: true, message: t('pages.model.form.model_code.placeholder') },
        { max: 64, message: t('pages.model.form.model_code.pattern') },
        {
            pattern: /^[a-zA-Z0-9]([a-zA-Z0-9.-]*[a-zA-Z0-9])?$/,
            message: t('pages.model.form.model_code.pattern'),
            trigger: ['blur', 'change'],
        },
    ],
    space_code: { required: true, message: t('pages.model.form.space_code.placeholder') },
    request_types: { required: true, message: t('pages.model.form.request_types.placeholder') },
    context_length: {
        validator: (_, value) => {
            if (value === undefined || value === null || value === '') {
                return Promise.resolve()
            }
            if (parseContextLength(value) === null) {
                return Promise.reject(t('pages.model.form.context_length.invalid'))
            }
            return Promise.resolve()
        },
        trigger: ['blur', 'change'],
    },
}

function handleCreate() {
    ordinaryRequestTypes = null
    formRecord.value = {}
    showModal({
        type: 'create',
        title: t('pages.model.add'),
    })
    formData.value.enabled = 1
    formData.value.model_type = 'normal'
    formData.value.space_code = initSpaceCode(props.spaceOptions)
    formData.value.context_length = toContextLengthSelectValue(1000000)
    formData.value.max_output_tokens = 8192
    formData.value.abilities = []
    formData.value.input_price = 3.0
    formData.value.output_price = 10.0
    formData.value.cached_price = 1.0
    formData.value.cache_creation_price = 3.0
    formData.value.apply_invocation_seed = true
    formData.value.apply_circuit_break_seed = true
}

async function handleEdit(record = {}) {
    ordinaryRequestTypes = null
    loadingRecord.value = true
    showModal({
        type: 'edit',
        title: t('pages.model.edit'),
    })

    const { data, success } = await apis.model.getModel(record.id).catch(() => ({ success: false }))
    loadingRecord.value = false
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
    data.model_type = data.model_type || 'normal'
    if (data.model_type === 'smart') {
        data.request_types = ['chat_completion']
    }
    formRecord.value = data
    formData.value = cloneDeep(data)
    delete formData.value.smart_routing
    delete formData.value.smart_routing_version
    formData.value.context_length = toContextLengthSelectValue(formData.value.context_length)
}

function handleOk() {
    if (loadingRecord.value) return
    formRef.value
        .validateFields()
        .then(async (values) => {
            try {
                showLoading()
                // Convert request_types and abilities arrays to JSON string
                const submitType = modal.value.type
                if (values.model_code && typeof values.model_code === 'string') {
                    values.model_code = values.model_code.trim()
                }
                values.context_length = parseContextLength(values.context_length)
                delete values.smart_routing
                delete values.smart_routing_version
                const params = {
                    ...values,
                    ...buildModelRoutingFields(formData.value, formRecord.value),
                }
                if (submitType === 'create') {
                    params.apply_invocation_seed = !!formData.value.apply_invocation_seed
                    params.apply_circuit_break_seed = !!formData.value.apply_circuit_break_seed
                }
                let result = null
                switch (submitType) {
                    case 'create':
                        result = await apis.model.createModel(params)
                        break
                    case 'edit':
                        result = await apis.model.updateModel(formData.value.id, params)
                        break
                }
                hideLoading()
                if (config('http.code.success') === result?.success) {
                    hideModal()
                    const feedback = getModelSaveFeedback(result?.data)
                    if (feedback) {
                        message.warning([t(`pages.model.smart.${feedback.key}`), ...feedback.warnings].join(' '))
                    } else if (submitType === 'create') {
                        showCreateResultMessage(result?.data)
                    } else {
                        message.success(t('component.message.success.save'))
                    }
                    emit('ok')
                } else {
                    message.error(result?.msg || t('component.message.error.save'))
                }
            } catch (error) {
                hideLoading()
                const errMsg =
                    error?.response?.data?.error?.detail ||
                    error?.response?.data?.msg ||
                    error?.message ||
                    t('component.message.error.request')
                message.error(errMsg)
            }
        })
        .catch(() => {
            hideLoading()
        })
}

function showCreateResultMessage(data) {
    const skipped = data?.skipped_seeds || []
    if (skipped.length > 0) {
        message.warning(t('pages.model.create.partial_policies'))
        return
    }
    const applied = data?.applied_seeds || []
    const appliedBoth = applied.includes('policy_invocation') && applied.includes('policy_circuit_break')
    if (appliedBoth) {
        message.success(t('pages.model.create.with_policies'))
        return
    }
    message.success(t('pages.model.create.success'))
}

function handleCancel() {
    hideModal()
}

function onAfterClose() {
    resetForm()
    hideLoading()
    loadingRecord.value = false
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

<style lang="less" scoped>
.recommended-policies {
    display: flex;
    flex-direction: column;
    gap: 8px;
}

.smart-disable-warning {
    margin-top: 8px;
}
</style>
