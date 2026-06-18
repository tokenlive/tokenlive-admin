<template>
    <a-drawer
        :open="modal.open"
        :title="modal.title"
        :width="720"
        :confirm-loading="modal.confirmLoading"
        @close="handleCancel"
        @afterOpenChange="handleAfterOpenChange">
        <template #footer>
            <div class="drawer-footer">
                <a-button
                    :disabled="testing"
                    @click="handleCancel"
                    >{{ cancelText }}</a-button
                >
                <a-button
                    type="dashed"
                    :loading="testing"
                    @click="handleTest"
                    >{{ $t('pages.endpoint.test.draft') }}</a-button
                >
                <a-button
                    type="primary"
                    :loading="modal.confirmLoading"
                    :disabled="testing"
                    @click="handleOk"
                    >{{ okText }}</a-button
                >
            </div>
        </template>
        <a-form
            ref="formRef"
            :model="formData"
            :rules="formRules"
            :label-col="{ style: { width: '120px' } }"
            :wrapper-col="{ flex: 1 }">
            <a-form-item
                v-if="providerId"
                name="provider_id"
                v-show="false">
                <a-input v-model:value="formData.provider_id" />
            </a-form-item>
            <a-form-item
                v-else
                :label="$t('pages.endpoint.form.provider_id')"
                name="provider_id">
                <a-select
                    :placeholder="$t('pages.endpoint.form.provider_id.placeholder')"
                    v-model:value="formData.provider_id"
                    show-search
                    :filter-option="filterProviderOption">
                    <a-select-option
                        v-for="p in providerOptions"
                        :key="p.id"
                        :value="p.id">
                        {{ p.name }}
                    </a-select-option>
                </a-select>
            </a-form-item>
            <a-form-item
                v-if="!modelId && providerId"
                :label="$t('pages.endpoint.form.model_id')"
                name="model_id">
                <a-select
                    :placeholder="$t('pages.endpoint.form.model_id.placeholder')"
                    v-model:value="formData.model_id"
                    show-search
                    :filter-option="filterModelOption">
                    <a-select-option
                        v-for="m in modelOptions"
                        :key="m.id"
                        :value="m.id">
                        {{ m.model_name }}
                    </a-select-option>
                </a-select>
            </a-form-item>
            <a-form-item
                v-else
                name="model_id"
                v-show="false">
                <a-input v-model:value="formData.model_id" />
            </a-form-item>
            <a-form-item
                :label="$t('pages.endpoint.form.real_model')"
                name="real_model">
                <a-input
                    :placeholder="$t('pages.endpoint.form.real_model.placeholder')"
                    v-model:value="formData.real_model" />
            </a-form-item>
            <a-form-item
                :label="$t('pages.endpoint.form.priority')"
                name="priority">
                <a-input-number
                    v-model:value="formData.priority"
                    :min="0"
                    :max="100"
                    style="width: 100%" />
            </a-form-item>
            <a-form-item
                :label="$t('pages.endpoint.form.url')"
                name="url">
                <a-input
                    :placeholder="$t('pages.endpoint.form.url.placeholder')"
                    v-model:value="formData.url" />
            </a-form-item>
            <a-form-item
                :label="$t('pages.endpoint.form.api_key')"
                name="api_key">
                <a-input-password
                    :placeholder="$t('pages.endpoint.form.api_key.placeholder')"
                    v-model:value="formData.api_key"
                    allow-clear />
            </a-form-item>
            <a-form-item
                :label="$t('pages.endpoint.form.protocol')"
                name="protocol">
                <a-select
                    v-model:value="formData.protocol"
                    allow-clear
                    :placeholder="$t('pages.endpoint.form.protocol.placeholder')">
                    <a-select-option value="">{{ $t('pages.endpoint.form.protocol.inherit') }}</a-select-option>
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
                :label="$t('pages.endpoint.form.weight')"
                name="weight">
                <a-input-number
                    v-model:value="formData.weight"
                    :min="0"
                    style="width: 100%" />
            </a-form-item>
            <a-form-item
                :label="$t('pages.endpoint.form.input_price')"
                name="input_price">
                <a-input-number
                    v-model:value="formData.input_price"
                    :min="0"
                    :placeholder="$t('pages.endpoint.form.price.placeholder')"
                    style="width: 100%"
                    :addon-after="$t('pages.endpoint.form.price.unit')" />
            </a-form-item>
            <a-form-item
                :label="$t('pages.endpoint.form.output_price')"
                name="output_price">
                <a-input-number
                    v-model:value="formData.output_price"
                    :min="0"
                    :placeholder="$t('pages.endpoint.form.price.placeholder')"
                    style="width: 100%"
                    :addon-after="$t('pages.endpoint.form.price.unit')" />
            </a-form-item>
            <a-form-item
                :label="$t('pages.endpoint.form.cached_price')"
                name="cached_price">
                <a-input-number
                    v-model:value="formData.cached_price"
                    :min="0"
                    :placeholder="$t('pages.endpoint.form.price.placeholder')"
                    style="width: 100%"
                    :addon-after="$t('pages.endpoint.form.price.unit')" />
            </a-form-item>
            <a-form-item
                :label="$t('pages.endpoint.form.cache_creation_price')"
                name="cache_creation_price">
                <a-input-number
                    v-model:value="formData.cache_creation_price"
                    :min="0"
                    :placeholder="$t('pages.endpoint.form.price.placeholder')"
                    style="width: 100%"
                    :addon-after="$t('pages.endpoint.form.price.unit')" />
            </a-form-item>
            <a-form-item
                :label="$t('pages.endpoint.form.enabled')"
                name="enabled">
                <a-switch
                    v-model:checked="formData.enabled"
                    :checked-children="$t('pages.endpoint.form.enabled.active')"
                    :un-checked-children="$t('pages.endpoint.form.enabled.inactive')"
                    :checked-value="1"
                    :un-checked-value="0" />
            </a-form-item>
            <a-form-item
                :label="$t('pages.endpoint.form.metadata')"
                name="metadata">
                <div class="metadata-list">
                    <div
                        v-for="(item, index) in metadataList"
                        :key="index"
                        class="metadata-row">
                        <a-input
                            v-model:value="item.key"
                            :placeholder="$t('pages.endpoint.form.metadata.key')"
                            style="width: 35%" />
                        <a-input
                            v-model:value="item.value"
                            :placeholder="$t('pages.endpoint.form.metadata.value')"
                            style="width: 50%" />
                        <delete-outlined
                            class="metadata-delete"
                            @click="removeMetadata(index)" />
                    </div>
                    <a-button
                        type="dashed"
                        block
                        @click="addMetadata">
                        <plus-outlined />
                        {{ $t('pages.endpoint.form.metadata.add') }}
                    </a-button>
                </div>
            </a-form-item>
            <a-form-item
                :label="$t('pages.endpoint.form.headers')"
                name="headers">
                <div class="metadata-list">
                    <div
                        v-for="(item, index) in headersList"
                        :key="index"
                        class="metadata-row">
                        <a-input
                            v-model:value="item.key"
                            :placeholder="$t('pages.endpoint.form.headers.key')"
                            style="width: 35%" />
                        <a-input
                            v-model:value="item.value"
                            :placeholder="$t('pages.endpoint.form.headers.value')"
                            style="width: 50%" />
                        <delete-outlined
                            class="metadata-delete"
                            @click="removeHeader(index)" />
                    </div>
                    <a-button
                        type="dashed"
                        block
                        @click="addHeader">
                        <plus-outlined />
                        {{ $t('pages.endpoint.form.headers.add') }}
                    </a-button>
                </div>
            </a-form-item>
            <a-form-item
                :label="$t('pages.endpoint.form.description')"
                name="description">
                <a-textarea
                    :placeholder="$t('pages.endpoint.form.description.placeholder')"
                    v-model:value="formData.description"
                    :rows="3" />
            </a-form-item>
        </a-form>
    </a-drawer>
</template>

<script setup>
import { cloneDeep } from 'lodash-es'
import { ref } from 'vue'
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons-vue'
import { message, Modal } from 'ant-design-vue'
import { config } from '@/config'
import apis from '@/apis'
import { useForm, useModal } from '@/hooks'
import { useI18n } from 'vue-i18n'

const props = defineProps({
    providerOptions: { type: Array, default: () => [] },
    modelOptions: { type: Array, default: () => [] },
    modelId: { type: String, default: '' },
    providerId: { type: String, default: '' },
})

const emit = defineEmits(['ok'])
const { modal, showModal, hideModal, showLoading, hideLoading } = useModal()
const { formRecord, formData, formRef, formRules, resetForm } = useForm()
const { t } = useI18n()
const cancelText = ref(t('button.cancel'))
const okText = ref(t('button.confirm'))

const metadataList = ref([])
const headersList = ref([])

function addMetadata() {
    metadataList.value.push({ key: '', value: '' })
}

function removeMetadata(index) {
    metadataList.value.splice(index, 1)
}

function metadataToJSON(list) {
    const obj = {}
    list.forEach((item) => {
        if (item.key.trim()) {
            obj[item.key.trim()] = item.value
        }
    })
    return Object.keys(obj).length > 0 ? obj : null
}

function jsonToMetadata(json) {
    if (!json) return []
    const obj = typeof json === 'string' ? JSON.parse(json) : json
    return Object.entries(obj).map(([key, value]) => ({ key, value: String(value) }))
}

function addHeader() {
    headersList.value.push({ key: '', value: '' })
}

function removeHeader(index) {
    headersList.value.splice(index, 1)
}

function headersToJSON(list) {
    const obj = {}
    list.forEach((item) => {
        if (item.key.trim()) {
            obj[item.key.trim()] = item.value
        }
    })
    return Object.keys(obj).length > 0 ? obj : null
}

function jsonToHeaders(json) {
    if (!json) return []
    const obj = typeof json === 'string' ? JSON.parse(json) : json
    return Object.entries(obj).map(([key, value]) => ({ key, value: String(value) }))
}

formRules.value = {
    provider_id: { required: true, message: t('pages.endpoint.form.provider_id.required') },
    model_id: { required: true, message: t('pages.endpoint.form.model_id.required') },
    url: { required: true, message: t('pages.endpoint.form.url.required') },
}

function handleCreate() {
    showModal({
        type: 'create',
        title: t('pages.endpoint.add'),
    })
    formData.value = {
        weight: 1,
        enabled: 0,
        priority: 0,
        protocol: '',
        model_id: props.modelId || undefined,
        provider_id: props.providerId || undefined,
        input_price: undefined,
        output_price: undefined,
        cached_price: undefined,
        cache_creation_price: undefined,
    }
    metadataList.value = []
    headersList.value = []
}

async function handleEdit(record = {}) {
    showModal({
        type: 'edit',
        title: t('pages.endpoint.edit'),
    })
    formRecord.value = record
    formData.value = cloneDeep(record)
    metadataList.value = jsonToMetadata(record.metadata)
    headersList.value = jsonToHeaders(record.headers)
}

async function handleCopy(record = {}) {
    showModal({
        type: 'create',
        title: t('pages.endpoint.copy'),
    })
    // 复制所有字段，但移除 id 和 api_key
    const copiedData = cloneDeep(record)
    delete copiedData.id
    delete copiedData.api_key
    delete copiedData.created_at
    delete copiedData.updated_at
    delete copiedData.status_points

    formData.value = copiedData
    metadataList.value = jsonToMetadata(record.metadata)
    headersList.value = jsonToHeaders(record.headers)
}

function handleOk() {
    formRef.value
        .validateFields()
        .then(async (values) => {
            try {
                showLoading()
                const params = {
                    ...values,
                    metadata: metadataToJSON(metadataList.value),
                    headers: headersToJSON(headersList.value),
                }
                let result = null
                switch (modal.value.type) {
                    case 'create':
                        result = await apis.endpoint.createEndpoint(params).catch(() => {
                            throw new Error()
                        })
                        break
                    case 'edit':
                        result = await apis.endpoint.updateEndpoint(formData.value.id, params).catch(() => {
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

const testing = ref(false)

function handleTest() {
    formRef.value
        .validateFields()
        .then(async (values) => {
            try {
                testing.value = true
                const params = {
                    ...values,
                    metadata: metadataToJSON(metadataList.value),
                    headers: headersToJSON(headersList.value),
                }
                const { data, success, message: errMessage } = await apis.endpoint.testEndpointDraft(params)
                if (success && data && data.success) {
                    message.success(t('pages.endpoint.test.success', { latency: data.latency_ms }))
                } else {
                    const errMsg = data ? data.detail || data.message || errMessage : errMessage
                    Modal.error({
                        title: t('pages.endpoint.test.failure'),
                        content: errMsg || '未知错误',
                        okText: t('button.confirm'),
                    })
                }
            } catch (error) {
                Modal.error({
                    title: t('pages.endpoint.test.failure'),
                    content: error.message || '网络请求错误',
                    okText: t('button.confirm'),
                })
            } finally {
                testing.value = false
            }
        })
        .catch(() => {
            // Validate fail
        })
}

function filterProviderOption(input, option) {
    return option.children?.[0]?.children?.toLowerCase().includes(input.toLowerCase())
}

function filterModelOption(input, option) {
    return option.children?.[0]?.children?.toLowerCase().includes(input.toLowerCase())
}

function onAfterClose() {
    resetForm()
    testing.value = false
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
    handleCopy,
})
</script>

<style lang="less" scoped>
.drawer-footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
}

.metadata-list {
    width: 100%;
}

.metadata-row {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;
}

.metadata-delete {
    cursor: pointer;
    color: #ff4d4f;
    flex-shrink: 0;

    &:hover {
        color: #cf1322;
    }
}
</style>
