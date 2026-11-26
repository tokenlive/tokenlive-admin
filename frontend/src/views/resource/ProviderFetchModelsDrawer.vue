<template>
    <a-drawer
        :open="visible"
        :title="$t('pages.provider.fetchModels.title')"
        :width="isGeminiProvider ? 900 : 660"
        @afterOpenChange="onAfterOpen"
        @close="handleClose">
        <a-form
            ref="formRef"
            :model="formData"
            :rules="formRules"
            label-align="left"
            :colon="false"
            layout="horizontal"
            class="fetch-form">
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

        <a-divider style="margin: 12px 0" />

        <div
            v-if="isGeminiProvider"
            class="gemini-hint-bar">
            <div class="gemini-hint-main">
                <info-circle-outlined class="gemini-hint-icon" />
                <span class="gemini-hint-text">{{ $t('pages.provider.fetchModels.gemini.hint_short') }}</span>
                <a-tooltip :title="$t('pages.provider.fetchModels.gemini.hint')">
                    <question-circle-outlined class="hint-help-icon" />
                </a-tooltip>
            </div>
            <a-tag
                v-if="geminiUnifiedOwner"
                class="provider-owner-tag">
                {{ $t('pages.provider.fetchModels.gemini.provider_badge', { owner: geminiUnifiedOwner }) }}
            </a-tag>
        </div>

        <div class="model-table-container">
            <div
                v-if="isGeminiProvider"
                class="gemini-toolbar">
                <a-radio-group
                    v-model:value="geminiSupportFilter"
                    size="small"
                    button-style="solid"
                    class="support-filter-group">
                    <a-radio-button value="supported">
                        {{ $t('pages.provider.fetchModels.gemini.filter.supported') }} ({{ supportedModelCount }})
                    </a-radio-button>
                    <a-radio-button value="all">
                        {{ $t('pages.provider.fetchModels.gemini.filter.all') }} ({{ modelList.length }})
                    </a-radio-button>
                    <a-radio-button value="unsupported">
                        {{ $t('pages.provider.fetchModels.gemini.filter.unsupported') }} ({{ unsupportedModelCount }})
                    </a-radio-button>
                </a-radio-group>

                <a-select
                    v-model:value="selectedFamily"
                    size="small"
                    class="family-filter-select">
                    <a-select-option value="__ALL__">
                        {{ $t('pages.provider.fetchModels.gemini.family.all') }} ({{ modelList.length }})
                    </a-select-option>
                    <a-select-option
                        v-for="item in familyOptions"
                        :key="item.value"
                        :value="item.value">
                        {{ item.label }} ({{ item.count }})
                    </a-select-option>
                </a-select>

                <a-input-search
                    v-model:value="searchQuery"
                    :placeholder="$t('pages.provider.fetchModels.gemini.search.placeholder')"
                    size="small"
                    class="toolbar-search-input"
                    allow-clear />
            </div>

            <a-input-search
                v-else
                v-model:value="searchQuery"
                :placeholder="$t('pages.provider.fetchModels.search.placeholder')"
                style="margin-bottom: 10px; width: 100%"
                allow-clear />

            <div class="model-list-header">
                <span class="model-count">
                    {{
                        isGeminiProvider
                            ? $t('pages.provider.fetchModels.gemini.selected', {
                                  count: selectedModels.length,
                                  total: supportedModelCount,
                              })
                            : $t('pages.provider.fetchModels.selected', {
                                  count: selectedModels.length,
                                  total: modelList.length,
                              })
                    }}
                </span>
                <div class="model-list-actions">
                    <a-checkbox
                        v-if="importedVisibleCount > 0"
                        v-model:checked="hideImported"
                        class="hide-imported-checkbox">
                        {{ $t('pages.provider.fetchModels.hide_imported') }} ({{ importedVisibleCount }})
                    </a-checkbox>
                    <a-button
                        v-if="selectedModels.length > 0"
                        type="link"
                        size="small"
                        class="clear-selection-btn"
                        @click="clearSelection">
                        {{ $t('pages.provider.fetchModels.clear_selection') }}
                    </a-button>
                </div>
            </div>

            <a-table
                :columns="columns"
                :data-source="filteredModelList"
                :row-selection="rowSelection"
                :row-key="(record) => record.id"
                :row-class-name="getRowClassName"
                :pagination="false"
                :scroll="{ y: 'calc(100vh - 345px)' }"
                size="small"
                class="model-table">
                <template #bodyCell="{ column, record }">
                    <template v-if="column.key === 'id'">
                        <div
                            v-if="!isGeminiProvider"
                            class="model-primary-line">
                            <span class="model-id">{{ record.id }}</span>
                            <a-tag
                                v-if="isModelImported(record)"
                                class="model-status-tag model-imported-tag">
                                {{ $t('pages.provider.fetchModels.imported') }}
                            </a-tag>
                        </div>
                        <div
                            v-else
                            class="native-model-details">
                            <div class="model-primary-line">
                                <span class="model-id">{{ record.id }}</span>
                                <span
                                    v-if="getDistinctGeminiDisplayName(record)"
                                    class="model-alias">
                                    {{ getDistinctGeminiDisplayName(record) }}
                                </span>
                                <a-tag
                                    v-if="isModelImported(record)"
                                    class="model-status-tag model-imported-tag">
                                    {{ $t('pages.provider.fetchModels.imported') }}
                                </a-tag>
                                <a-tooltip
                                    v-if="!supportsGeminiGenerateContent(record)"
                                    :title="$t('pages.provider.fetchModels.gemini.unsupported')">
                                    <a-tag
                                        color="warning"
                                        class="model-status-tag">
                                        {{ $t('pages.provider.fetchModels.gemini.not_importable') }}
                                    </a-tag>
                                </a-tooltip>
                            </div>
                            <a-tooltip
                                v-if="getDistinctGeminiDescription(record)"
                                :title="getDistinctGeminiDescription(record)"
                                placement="topLeft">
                                <div class="native-description">{{ getDistinctGeminiDescription(record) }}</div>
                            </a-tooltip>
                        </div>
                    </template>

                    <template v-else-if="column.key === 'token_limits'">
                        <a-tooltip
                            :title="formatTokenLimitsTooltip(record)"
                            placement="topRight">
                            <span class="token-limits-cell">
                                <span class="token-limit-in">{{
                                    formatTokenLimitCompact(record.input_token_limit, unknownLabel)
                                }}</span>
                                <span class="token-limit-sep">/</span>
                                <span class="token-limit-out">{{
                                    formatTokenLimitCompact(record.output_token_limit, unknownLabel)
                                }}</span>
                            </span>
                        </a-tooltip>
                    </template>

                    <template v-else-if="column.key === 'supported_generation_methods'">
                        <template v-if="getMethodDisplay(record).ordered.length > 0">
                            <a-tooltip
                                :title="getMethodDisplay(record).ordered.join(', ')"
                                placement="topRight">
                                <div class="methods-wrap">
                                    <a-tag
                                        :color="
                                            getMethodDisplay(record).primary === 'generateContent'
                                                ? 'success'
                                                : 'default'
                                        "
                                        class="method-tag">
                                        {{ getMethodDisplay(record).primary }}
                                    </a-tag>
                                    <a-tag
                                        v-if="getMethodDisplay(record).secondary"
                                        class="method-tag method-tag-secondary">
                                        {{ getMethodDisplay(record).secondary }}
                                    </a-tag>
                                    <a-tag
                                        v-else-if="getMethodDisplay(record).extraCount > 0"
                                        class="method-tag method-tag-more">
                                        +{{ getMethodDisplay(record).extraCount }}
                                    </a-tag>
                                </div>
                            </a-tooltip>
                        </template>
                        <span
                            v-else
                            class="method-unknown">
                            {{ unknownLabel }}
                        </span>
                    </template>
                </template>

                <template
                    v-if="isGeminiProvider && modelList.length > 0 && geminiSupportFilter !== 'all'"
                    #emptyText>
                    <div class="table-empty-state">
                        <a-button
                            type="link"
                            size="small"
                            @click="showAllGeminiModels">
                            {{ $t('pages.provider.fetchModels.gemini.show_all_models', { count: modelList.length }) }}
                        </a-button>
                    </div>
                </template>
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
import { ref, computed } from 'vue'
import { CloudDownloadOutlined, InfoCircleOutlined, QuestionCircleOutlined } from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { cloneDeep } from 'lodash-es'
import {
    GEMINI_DEFAULT_BASE_URL,
    GEMINI_OTHER_FAMILY,
    compareGeminiFamilies,
    extractGeminiModelFamily,
    formatTokenLimitCompact,
    formatTokenLimitExact,
    getDistinctGeminiDescription,
    getDistinctGeminiDisplayName,
    orderGeminiMethods,
    supportsGeminiGenerateContent,
} from '@/utils/provider-models'
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
const geminiSupportFilter = ref('supported')
const selectedFamily = ref('__ALL__')
const hideImported = ref(false)
const importedModelIds = ref(new Set())
let fetchSeq = 0
let openSeq = 0

const unknownLabel = computed(() => t('pages.provider.fetchModels.unknown'))

const supportedModelCount = computed(() => {
    if (!isGeminiProvider.value) return modelList.value.length
    return modelList.value.filter((model) => supportsGeminiGenerateContent(model)).length
})

const unsupportedModelCount = computed(() => {
    if (!isGeminiProvider.value) return 0
    return modelList.value.length - supportedModelCount.value
})

const importedVisibleCount = computed(() => modelList.value.filter((model) => isModelImported(model)).length)

const geminiUnifiedOwner = computed(() => {
    if (!isGeminiProvider.value || modelList.value.length === 0) return ''
    const owners = new Set(modelList.value.map((model) => String(model?.owned_by || '').trim()).filter(Boolean))
    return owners.size === 1 ? [...owners][0] : ''
})

const familyOptions = computed(() => {
    if (!isGeminiProvider.value || modelList.value.length === 0) return []
    const counts = new Map()
    for (const model of modelList.value) {
        const family = extractGeminiModelFamily(model)
        counts.set(family, (counts.get(family) || 0) + 1)
    }
    return Array.from(counts.entries())
        .sort(([a], [b]) => compareGeminiFamilies(a, b))
        .map(([value, count]) => ({
            value,
            label: value === GEMINI_OTHER_FAMILY ? t('pages.provider.fetchModels.gemini.family.other') : value,
            count,
        }))
})

const filteredModelList = computed(() => {
    let list = modelList.value
    if (isGeminiProvider.value) {
        if (geminiSupportFilter.value === 'supported') {
            list = list.filter((model) => supportsGeminiGenerateContent(model))
        } else if (geminiSupportFilter.value === 'unsupported') {
            list = list.filter((model) => !supportsGeminiGenerateContent(model))
        }
        if (selectedFamily.value !== '__ALL__') {
            list = list.filter((model) => extractGeminiModelFamily(model) === selectedFamily.value)
        }
    }
    if (hideImported.value) {
        list = list.filter((model) => !isModelImported(model))
    }
    const query = String(searchQuery.value || '')
        .trim()
        .toLowerCase()
    if (query) {
        list = list.filter((model) => {
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
    }
    if (isGeminiProvider.value) {
        return [...list].sort(
            (a, b) => Number(supportsGeminiGenerateContent(b)) - Number(supportsGeminiGenerateContent(a))
        )
    }
    return list
})

const columns = computed(() => {
    if (isGeminiProvider.value) {
        return [
            {
                title: t('pages.provider.fetchModels.table.model'),
                dataIndex: 'id',
                key: 'id',
            },
            {
                title: t('pages.provider.fetchModels.table.token_limits'),
                key: 'token_limits',
                width: 148,
                align: 'right',
            },
            {
                title: t('pages.provider.fetchModels.table.supported_generation_methods'),
                dataIndex: 'supported_generation_methods',
                key: 'supported_generation_methods',
                width: 224,
            },
        ]
    }
    return [
        {
            title: t('pages.provider.fetchModels.table.model'),
            dataIndex: 'id',
            key: 'id',
            width: 380,
        },
        {
            title: t('pages.provider.fetchModels.table.owned_by'),
            dataIndex: 'owned_by',
            key: 'owned_by',
            width: 180,
        },
    ]
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

function isModelImported(model) {
    const id = typeof model?.id === 'string' ? model.id.trim() : ''
    return Boolean(id && importedModelIds.value.has(id))
}

function getRowClassName(record) {
    if (isGeminiProvider.value && !supportsGeminiGenerateContent(record)) {
        return 'model-row-unsupported'
    }
    return ''
}

function formatTokenLimitsTooltip(record) {
    const unknown = unknownLabel.value
    const input = formatTokenLimitExact(record?.input_token_limit, unknown)
    const output = formatTokenLimitExact(record?.output_token_limit, unknown)
    return `${t('pages.provider.fetchModels.table.input_token_limit')}: ${input} / ${t('pages.provider.fetchModels.table.output_token_limit')}: ${output}`
}

function getMethodDisplay(record) {
    const ordered = orderGeminiMethods(record?.supported_generation_methods)
    if (ordered.length === 0) {
        return { ordered, primary: '', secondary: '', extraCount: 0 }
    }
    const primary = ordered[0]
    const secondary = ordered.length === 2 && ordered[1].length <= 14 ? ordered[1] : ''
    const extraCount = !secondary && ordered.length > 1 ? ordered.length - 1 : 0
    return { ordered, primary, secondary, extraCount }
}

function clearSelection() {
    selectedModels.value = []
}

function showAllGeminiModels() {
    geminiSupportFilter.value = 'all'
    selectedFamily.value = '__ALL__'
    hideImported.value = false
}

async function loadExistingEndpoints(targetProviderId, seq) {
    if (!targetProviderId || typeof apis.endpoint?.getEndpointsByProviderId !== 'function') {
        return
    }
    try {
        const res = await apis.endpoint.getEndpointsByProviderId(targetProviderId)
        if (seq !== openSeq || !visible.value) return
        if (config('http.code.success') === res?.success && Array.isArray(res?.data)) {
            const nextIds = new Set()
            for (const item of res.data) {
                const realModel = typeof item?.real_model === 'string' ? item.real_model.trim() : ''
                if (realModel) {
                    nextIds.add(realModel)
                }
            }
            importedModelIds.value = nextIds
        }
    } catch {
        // 静默降级：端点加载失败不影响模型拉取与导入主流程
    }
}

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
            const validFamilies = new Set(familyOptions.value.map((item) => item.value))
            if (selectedFamily.value !== '__ALL__' && !validFamilies.has(selectedFamily.value)) {
                selectedFamily.value = '__ALL__'
            }
        } else {
            modelList.value = []
            selectedModels.value = []
            fetchedBaseUrl.value = ''
            selectedFamily.value = '__ALL__'
        }
    } catch {
        if (seq !== fetchSeq) return
        fetching.value = false
        fetched.value = true
        modelList.value = []
        selectedModels.value = []
        fetchedBaseUrl.value = ''
        selectedFamily.value = '__ALL__'
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
    openSeq++
    const currentOpenSeq = openSeq
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
    geminiSupportFilter.value = 'supported'
    selectedFamily.value = '__ALL__'
    hideImported.value = false
    importedModelIds.value = new Set()
    visible.value = true
    loadExistingEndpoints(record.id, currentOpenSeq)
}

function onAfterOpen(open) {
    if (open && formData.value.base_url) {
        handleFetchModels()
    }
}

function handleClose() {
    openSeq++
    fetchSeq++
    visible.value = false
    fetching.value = false
    fetchedBaseUrl.value = ''
    modelList.value = []
    selectedModels.value = []
    searchQuery.value = ''
    geminiSupportFilter.value = 'supported'
    selectedFamily.value = '__ALL__'
    hideImported.value = false
    importedModelIds.value = new Set()
    formRef.value?.resetFields()
}

defineExpose({
    handleOpen,
})
</script>

<style lang="less" scoped>
.fetch-form :deep(.ant-form-item) {
    margin-bottom: 12px;
}

.gemini-hint-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    padding: 7px 12px;
    margin-bottom: 12px;
    border-radius: var(--radius-md, 6px);
    background: var(--color-info-bg);
    border: 1px solid var(--color-primary-border);
    font-size: 12px;
    color: var(--color-text-secondary);
}

.gemini-hint-main {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
}

.gemini-hint-icon {
    color: var(--color-info);
    font-size: 14px;
    flex-shrink: 0;
}

.gemini-hint-text {
    line-height: 1.4;
}

.hint-help-icon {
    color: var(--color-text-tertiary);
    cursor: help;
    flex-shrink: 0;
}

.provider-owner-tag {
    margin-inline-end: 0;
    flex-shrink: 0;
    font-size: 11px;
    color: var(--color-text-secondary);
    background: var(--color-bg-container);
    border-color: var(--color-border);
}

.gemini-toolbar {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 10px;
    flex-wrap: wrap;
}

.support-filter-group {
    flex-shrink: 0;
}

.family-filter-select {
    width: 150px;
    flex-shrink: 0;
}

.toolbar-search-input {
    flex: 1;
    min-width: 180px;
}

.model-list-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    margin-bottom: 8px;
    min-height: 24px;
}

.model-count {
    color: var(--color-text-tertiary);
    font-size: 12px;
}

.model-list-actions {
    display: flex;
    align-items: center;
    gap: 8px;
}

.hide-imported-checkbox {
    font-size: 12px;
    color: var(--color-text-secondary);
}

.clear-selection-btn {
    padding: 0 4px;
    height: auto;
    font-size: 12px;
}

.model-primary-line {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-wrap: wrap;
}

.model-id {
    font-family: 'SF Mono', Monaco, Consolas, monospace;
    font-size: 13px;
    font-weight: 600;
    color: var(--color-text-primary);
}

.model-alias {
    font-size: 12px;
    color: var(--color-text-secondary);
}

.model-status-tag {
    margin-inline-end: 0;
    font-size: 11px;
    line-height: 18px;
    padding: 0 6px;
}

.model-imported-tag {
    color: var(--color-text-secondary);
    background: var(--color-bg-active);
    border-color: var(--color-border);
}

.native-model-details {
    min-width: 0;
    overflow: hidden;
}

.native-description {
    margin-top: 2px;
    color: var(--color-text-tertiary);
    font-size: 12px;
    line-height: 1.4;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

.token-limits-cell {
    display: inline-flex;
    align-items: center;
    justify-content: flex-end;
    font-family: 'SF Mono', Monaco, Consolas, monospace;
    font-variant-numeric: tabular-nums;
    font-size: 12.5px;
    white-space: nowrap;
}

.token-limit-in {
    color: var(--color-text-primary);
    font-weight: 500;
}

.token-limit-sep {
    margin: 0 4px;
    color: var(--color-text-quaternary);
}

.token-limit-out {
    color: var(--color-text-secondary);
}

.methods-wrap {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    flex-wrap: nowrap;
}

.method-tag {
    margin-inline-end: 0;
    font-family: 'SF Mono', Monaco, Consolas, monospace;
    font-size: 11px;
    line-height: 18px;
    padding: 0 6px;
}

.method-tag-secondary,
.method-tag-more {
    color: var(--color-text-secondary);
    background: var(--color-bg-active);
    border-color: var(--color-border);
}

.method-unknown {
    color: var(--color-text-tertiary);
    font-size: 12px;
}

.table-empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    padding: 12px 0;
    color: var(--color-text-tertiary);
}

.model-table :deep(.model-row-unsupported td) {
    opacity: 0.6;
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
