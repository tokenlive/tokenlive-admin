<template>
    <a-drawer
        :open="visible"
        :title="$t('pages.provider.fetchModels.title')"
        :width="isGeminiProvider ? 1080 : 660"
        @afterOpenChange="onAfterOpen"
        @close="handleClose">
        <a-form
            ref="formRef"
            :model="formData"
            :rules="formRules"
            label-align="left"
            :colon="false"
            layout="horizontal">
            <a-form-item
                name="space_code"
                :label="$t('pages.model.form.space_code')"
                :label-col="{ flex: '110px' }"
                :wrapper-col="{ flex: 'auto' }"
                :rules="formRules.space_code">
                <a-select
                    v-model:value="formData.space_code"
                    :placeholder="$t('pages.model.form.space_code.placeholder')"
                    show-search
                    :filter-option="filterSpaceOption"
                    style="width: 100%">
                    <a-select-option
                        v-for="item in spaceOptions"
                        :key="item.code"
                        :value="item.code">
                        {{ item.name }} ({{ item.code }})
                    </a-select-option>
                </a-select>
            </a-form-item>

            <a-form-item
                name="base_url"
                :label="$t('pages.provider.fetchModels.base_url')"
                :label-col="{ flex: '110px' }"
                :wrapper-col="{ flex: 'auto' }"
                :rules="formRules.base_url">
                <a-row :gutter="8">
                    <a-col :span="18">
                        <a-input
                            v-model:value="formData.base_url"
                            :placeholder="
                                isGeminiProvider
                                    ? GEMINI_DEFAULT_BASE_URL
                                    : isCodexProvider
                                      ? 'https://chatgpt.com/backend-api/codex'
                                      : $t('pages.provider.fetchModels.base_url.placeholder')
                            " />
                    </a-col>
                    <a-col :span="6">
                        <a-button
                            type="primary"
                            :loading="fetching"
                            block
                            @click="handleFetchModels">
                            <template #icon>
                                <cloud-download-outlined />
                            </template>
                            {{ $t('pages.provider.fetchModels.fetch') }}
                        </a-button>
                    </a-col>
                </a-row>
            </a-form-item>

            <a-alert
                v-if="isCodexProvider"
                type="info"
                show-icon
                style="margin-bottom: 8px"
                message="Codex OAuth 将请求 ChatGPT backend /models，并自动携带 Bearer Token 与 Chatgpt-Account-Id。" />
        </a-form>

        <a-divider />

        <a-alert
            v-if="isGeminiProvider"
            type="info"
            show-icon
            style="margin-bottom: 12px"
            :message="$t('pages.provider.fetchModels.gemini.hint')" />

        <div class="model-table-container">
            <div
                class="model-list-header"
                style="margin-bottom: 12px">
                <span class="model-count">
                    {{
                        $t('pages.provider.fetchModels.selected', {
                            count: selectedModels.length,
                            total: modelList.length,
                        })
                    }}
                </span>
            </div>
            <a-input-search
                v-model:value="searchQuery"
                :placeholder="
                    isGeminiProvider
                        ? $t('pages.provider.fetchModels.gemini.search.placeholder')
                        : $t('pages.provider.fetchModels.search.placeholder')
                "
                style="margin-bottom: 12px; width: 100%"
                allow-clear />
            <a-table
                :columns="columns"
                :data-source="filteredModelList"
                :row-selection="rowSelection"
                :row-key="(record) => record.id"
                :pagination="false"
                :scroll="{ x: isGeminiProvider ? 980 : undefined, y: 'calc(100vh - 365px)' }"
                size="small"
                class="model-table">
            </a-table>
        </div>

        <template #footer>
            <a-space>
                <a-button @click="handleClose">{{ $t('button.cancel') }}</a-button>
                <a-button
                    type="primary"
                    :disabled="selectedModels.length === 0"
                    @click="handleConfirm">
                    {{ $t('pages.provider.fetchModels.confirm', { count: selectedModels.length }) }}
                </a-button>
            </a-space>
        </template>
    </a-drawer>
</template>

<script setup>
import { ref, computed, h } from 'vue'
import { CloudDownloadOutlined } from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { cloneDeep } from 'lodash-es'
import { GEMINI_DEFAULT_BASE_URL, isKnownTokenLimit, supportsGeminiGenerateContent } from '@/utils/provider-models'
import apis from '@/apis'
import { config } from '@/config'
import { useI18n } from 'vue-i18n'

const emit = defineEmits(['confirm'])

const CODEX_DEFAULT_BASE_URL = 'https://chatgpt.com/backend-api/codex'

const { t } = useI18n()
const visible = ref(false)
const fetching = ref(false)
const fetched = ref(false)
const isCodexProvider = ref(false)
const isGeminiProvider = ref(false)
const formRef = ref(null)
const providerId = ref('')
const providerApiKeys = ref([])

const spaceOptions = ref([])

const formData = ref({
    space_code: undefined,
    base_url: '',
    api_key: undefined,
})

const formRules = ref({
    space_code: [{ required: true, message: t('pages.model.form.space_code.placeholder') }],
    base_url: [{ required: true, message: t('pages.provider.fetchModels.base_url.required') }],
})

const modelList = ref([])
const selectedModels = ref([])
const searchQuery = ref('')
const fetchedBaseUrl = ref('')
let fetchSeq = 0

const filteredModelList = computed(() => {
    if (!searchQuery.value) return modelList.value
    const query = searchQuery.value.toLowerCase()
    return modelList.value.filter((model) => {
        const fields = isGeminiProvider.value
            ? [
                  model.id,
                  model.owned_by,
                  model.name,
                  model.display_name,
                  model.description,
                  ...(model.supported_generation_methods || []),
              ]
            : [model.id, model.owned_by]
        return fields.some((value) => typeof value === 'string' && value.toLowerCase().includes(query))
    })
})

const columns = computed(() => {
    const result = [
        {
            title: t('pages.provider.fetchModels.table.model'),
            dataIndex: 'id',
            key: 'id',
            width: 380,
            customRender: ({ text, record }) => {
                const id = h('span', { class: 'model-id' }, text)
                if (!isGeminiProvider.value) return id
                return h('div', { class: 'native-model-details' }, [
                    id,
                    record.display_name && h('div', { class: 'native-display-name' }, record.display_name),
                    record.description && h('div', { class: 'native-description' }, record.description),
                    !supportsGeminiGenerateContent(record) &&
                        h('div', { class: 'native-unsupported' }, t('pages.provider.fetchModels.gemini.unsupported')),
                ])
            },
        },
        {
            title: t('pages.provider.fetchModels.table.owned_by'),
            dataIndex: 'owned_by',
            key: 'owned_by',
            width: isGeminiProvider.value ? 120 : 180,
        },
    ]
    if (isGeminiProvider.value) {
        for (const field of ['input_token_limit', 'output_token_limit']) {
            result.push({
                title: t(`pages.provider.fetchModels.table.${field}`),
                dataIndex: field,
                key: field,
                width: 130,
                customRender: ({ text }) => (isKnownTokenLimit(text) ? text : t('pages.provider.fetchModels.unknown')),
            })
        }
        result.push({
            title: t('pages.provider.fetchModels.table.supported_generation_methods'),
            dataIndex: 'supported_generation_methods',
            key: 'supported_generation_methods',
            width: 220,
            customRender: ({ text }) =>
                Array.isArray(text) && text.length ? text.join(', ') : t('pages.provider.fetchModels.unknown'),
        })
    }
    return result
})

const rowSelection = computed(() => {
    return {
        selectedRowKeys: selectedModels.value,
        preserveSelectedRowKeys: true,
        onChange: (selectedRowKeys) => {
            selectedModels.value = selectedRowKeys
        },
        ...(isGeminiProvider.value
            ? {
                  getCheckboxProps: (model) => ({
                      disabled: !supportsGeminiGenerateContent(model),
                      title: supportsGeminiGenerateContent(model)
                          ? undefined
                          : t('pages.provider.fetchModels.gemini.unsupported'),
                  }),
              }
            : {}),
    }
})

async function handleFetchModels() {
    if (isGeminiProvider.value && !String(formData.value.base_url || '').trim()) {
        formData.value.base_url = GEMINI_DEFAULT_BASE_URL
    }
    const seq = ++fetchSeq
    try {
        await formRef.value.validateFields(['base_url'])
    } catch {
        return
    }
    if (seq !== fetchSeq || !visible.value) return

    const requestedBaseUrl = String(formData.value.base_url || '').trim()
    try {
        fetching.value = true
        const { success, data } = await apis.provider
            .fetchProviderModels(providerId.value, {
                base_url: requestedBaseUrl,
                api_key: formData.value.api_key || '',
            })
            .catch(() => {
                throw new Error()
            })
        if (seq !== fetchSeq || !visible.value) return
        fetching.value = false
        fetched.value = true
        if (config('http.code.success') === success) {
            modelList.value = data?.models || []
            const resolvedBaseUrl =
                typeof data?.base_url === 'string' && data.base_url ? data.base_url : requestedBaseUrl
            formData.value.base_url = resolvedBaseUrl
            fetchedBaseUrl.value = resolvedBaseUrl
            selectedModels.value = []
        } else {
            modelList.value = []
            selectedModels.value = []
            fetchedBaseUrl.value = ''
        }
    } catch {
        if (seq !== fetchSeq) return
        fetching.value = false
        fetched.value = true
        modelList.value = []
        selectedModels.value = []
        fetchedBaseUrl.value = ''
    }
}

function filterSpaceOption(input, option) {
    return (
        option.key.toLowerCase().indexOf(input.toLowerCase()) >= 0 ||
        option.value.toLowerCase().indexOf(input.toLowerCase()) >= 0
    )
}

async function loadSpaces() {
    try {
        const { success, data } = await apis.space.getSpaceList({ pageSize: 1000 })
        if (config('http.code.success') === success) {
            spaceOptions.value = data || []
            if (spaceOptions.value.length > 0 && !formData.value.space_code) {
                formData.value.space_code = spaceOptions.value[0].code
            }
        }
    } catch {
        // no-op
    }
}

function handleConfirm() {
    const currentBaseUrl = String(formData.value.base_url || '').trim()
    if (fetchedBaseUrl.value && currentBaseUrl !== fetchedBaseUrl.value) {
        message.warning(t('pages.provider.fetchModels.base_url.changed'))
        return
    }
    const models = selectedModels.value.map((id) => modelList.value.find((model) => model.id === id))
    if (isGeminiProvider.value && models.some((model) => !supportsGeminiGenerateContent(model))) {
        message.error(t('pages.provider.fetchModels.gemini.unsupported'))
        return
    }
    emit('confirm', {
        providerId: providerId.value,
        space_code: formData.value.space_code,
        base_url: fetchedBaseUrl.value || currentBaseUrl,
        api_key: formData.value.api_key,
        api_keys: providerApiKeys.value.map((item) => item.value),
        provider_api_keys: providerApiKeys.value,
        models: models.map((model, index) => cloneDeep(model || { id: selectedModels.value[index], owned_by: '' })),
    })
    handleClose()
}

function detectCodexProvider(record = {}) {
    const url = (record.url || '').toLowerCase()
    if (url.includes('chatgpt.com/backend-api/codex')) {
        return true
    }
    const endpoint = (record.oauth?.token_endpoint || '').toLowerCase()
    if (endpoint.includes('auth.openai.com') || endpoint.includes('openai.com')) {
        return true
    }
    const desc = (record.api_keys?.[0]?.description || '').toLowerCase()
    return desc.includes('codex')
}

function handleOpen(record) {
    providerId.value = record.id
    providerApiKeys.value = Array.isArray(record.api_keys) ? cloneDeep(record.api_keys) : []
    isGeminiProvider.value = record.protocol === 'gemini'
    isCodexProvider.value = !isGeminiProvider.value && detectCodexProvider(record)
    const defaultBaseUrl =
        record.url ||
        (isGeminiProvider.value ? GEMINI_DEFAULT_BASE_URL : isCodexProvider.value ? CODEX_DEFAULT_BASE_URL : '')
    const defaultApiKey = providerApiKeys.value.length > 0 ? providerApiKeys.value[0].value : undefined
    formData.value = {
        space_code: undefined,
        base_url: defaultBaseUrl,
        api_key: defaultApiKey,
    }
    loadSpaces()

    fetchSeq++
    fetching.value = false
    fetchedBaseUrl.value = ''
    modelList.value = []
    selectedModels.value = []
    searchQuery.value = ''
    fetched.value = false
    visible.value = true
}

function onAfterOpen(open) {
    if (open && formData.value.base_url) {
        handleFetchModels()
    }
}

function handleClose() {
    fetchSeq++
    visible.value = false
    fetching.value = false
    fetchedBaseUrl.value = ''
    modelList.value = []
    selectedModels.value = []
    searchQuery.value = ''
    formRef.value?.resetFields()
}

defineExpose({
    handleOpen,
})
</script>

<style lang="less" scoped>
.model-list-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;
}

.model-count {
    color: var(--color-text-tertiary);
    font-size: 13px;
}

.model-id {
    font-family: monospace;
    font-size: 13px;
}

.native-model-details {
    overflow-wrap: anywhere;
}

.native-display-name {
    margin-top: 4px;
    font-weight: 500;
}

.native-description {
    margin-top: 4px;
    color: var(--color-text-tertiary);
    font-size: 12px;
    white-space: pre-wrap;
}

.native-unsupported {
    margin-top: 4px;
    color: #d46b08;
    font-size: 12px;
}

:deep(.ant-form-item-required::before) {
    display: none !important;
}

:deep(.ant-form-item-required::after) {
    display: inline-block;
    margin-left: 4px;
    color: #ff4d4f;
    font-size: 14px;
    font-family: SimSun, sans-serif;
    line-height: 1;
    content: '*';
}
</style>
