<template>
    <a-modal
        :open="visible"
        :title="$t('pages.provider.fetchModels.mapping.title', '确认导入模型与系统模型映射')"
        :width="680"
        :confirm-loading="confirmLoading"
        @ok="handleOk"
        @cancel="handleCancel">
        <div style="margin-bottom: 12px; opacity: 0.65">
            {{ $t('pages.provider.fetchModels.mapping.hint') }}
        </div>
        <a-table
            :columns="columns"
            :data-source="models"
            :row-key="(record) => record.id"
            :pagination="false"
            size="small"
            :scroll="{ y: 320 }">
            <template #bodyCell="{ column, record }">
                <template v-if="'id' === column.key">
                    <span style="font-family: monospace; font-weight: 500">{{ record.id }}</span>
                </template>
                <template v-if="'mapping' === column.key">
                    <a-select
                        v-model:value="mapping[record.id]"
                        show-search
                        :filter-option="filterModelOption"
                        style="width: 100%">
                        <a-select-option value="__NEW__">
                            <span style="color: #1890ff; font-weight: 500">🆕 创建为新系统模型</span>
                        </a-select-option>
                        <a-select-option
                            v-for="item in existingModels"
                            :key="item.id"
                            :value="item.id">
                            {{ item.model_name }} ({{ item.model_code }})
                        </a-select-option>
                    </a-select>
                </template>
                <template v-if="'new_code' === column.key">
                    <a-input
                        v-if="mapping[record.id] === '__NEW__'"
                        v-model:value="newModelCodes[record.id]"
                        :placeholder="$t('pages.provider.fetchModels.table.new_code.placeholder')"
                        style="width: 100%" />
                    <span
                        v-else
                        style="opacity: 0.25"
                        >{{ $t('pages.provider.fetchModels.table.new_code.existing') }}</span
                    >
                </template>
            </template>
        </a-table>
        <div class="recommended-policies">
            <div class="recommended-policies-title">{{ $t('pages.model.form.recommended_policies') }}</div>
            <div class="recommended-policies-hint">
                {{
                    newModelCount > 0
                        ? $t('pages.provider.fetchModels.recommended_policies.hint')
                        : $t('pages.provider.fetchModels.recommended_policies.disabled')
                }}
            </div>
            <a-checkbox
                v-model:checked="applyInvocationSeed"
                :disabled="newModelCount === 0">
                {{ $t('pages.model.form.apply_invocation_seed') }}
            </a-checkbox>
            <a-checkbox
                v-model:checked="applyCircuitBreakSeed"
                :disabled="newModelCount === 0">
                {{ $t('pages.model.form.apply_circuit_break_seed') }}
            </a-checkbox>
        </div>
    </a-modal>
</template>

<script setup>
import { computed, ref } from 'vue'
import { message } from 'ant-design-vue'
import apis from '@/apis'
import { config } from '@/config'
import { useI18n } from 'vue-i18n'

const emit = defineEmits(['ok'])
const { t } = useI18n()

const visible = ref(false)
const confirmLoading = ref(false)

const models = ref([])
const mapping = ref({})
const newModelCodes = ref({})
const existingModels = ref([])
const applyInvocationSeed = ref(true)
const applyCircuitBreakSeed = ref(true)
const newModelCount = computed(() => Object.values(mapping.value).filter((value) => value === '__NEW__').length)

const importContext = ref({
    providerId: '',
    providerCode: '',
    space_code: '',
    base_url: '',
    keysToCreate: [],
    protocol: '',
})

const columns = [
    {
        title: t('pages.provider.fetchModels.table.model', '发现的模型名称/Code'),
        dataIndex: 'id',
        key: 'id',
        width: 220,
    },
    { title: t('pages.provider.fetchModels.table.mapping'), key: 'mapping', width: 240 },
    { title: t('pages.provider.fetchModels.table.new_code'), key: 'new_code' },
]

function filterModelOption(input, option) {
    // 处理 ASelect 选项标签的过滤逻辑
    const label = option.children?.[0]?.children || option.children || ''
    return (
        option.value.toLowerCase().includes(input.toLowerCase()) ||
        String(label).toLowerCase().includes(input.toLowerCase())
    )
}

async function handleOpen(context) {
    importContext.value = {
        providerId: context.providerId,
        providerCode: context.providerCode || context.providerId,
        space_code: context.space_code,
        base_url: context.base_url,
        keysToCreate: context.keysToCreate,
        protocol: context.protocol,
        auth_type: context.auth_type,
    }
    models.value = context.models || []

    // 初始化映射，默认都为新建
    mapping.value = {}
    newModelCodes.value = {}
    applyInvocationSeed.value = true
    applyCircuitBreakSeed.value = true
    for (const m of models.value) {
        mapping.value[m.id] = '__NEW__'
        newModelCodes.value[m.id] = m.id
    }

    // 获取系统已有模型
    existingModels.value = []
    try {
        const { success, data } = await apis.model.getModelList({ pageSize: 1000 }).catch(() => ({ success: false }))
        if (config('http.code.success') === success && Array.isArray(data)) {
            existingModels.value = data
        }
    } catch (e) {
        console.error(e)
    }

    visible.value = true
}

async function handleOk() {
    confirmLoading.value = true
    const hideLoadingMsg = message.loading('正在保存模型与端点配置，请稍候...', 0)

    try {
        let successModelCount = 0
        let successEndpointCount = 0
        let appliedBothCount = 0
        let skippedPolicyCount = 0

        for (const selectedModel of models.value) {
            let modelId = mapping.value[selectedModel.id]

            // 确定即将导入/关联的模型 code
            let modelCodeForEp
            if (modelId === '__NEW__') {
                modelCodeForEp = newModelCodes.value[selectedModel.id] || selectedModel.id
            } else {
                // 已有模型：从 existingModels 中查找 model_code
                const existing = existingModels.value.find((m) => m.id === modelId)
                modelCodeForEp = existing?.model_code || modelId
            }

            if (modelId === '__NEW__') {
                // 如果选择新建模型，则调用接口生成
                const isCodexImport =
                    (importContext.value.base_url || '').toLowerCase().includes('chatgpt.com/backend-api/codex') ||
                    ((importContext.value.auth_type || '') === 'oauth_token' &&
                        (importContext.value.base_url || '').toLowerCase().includes('chatgpt.com'))
                const request_types =
                    importContext.value.protocol === 'gemini'
                        ? JSON.stringify(['gemini_generate_content'])
                        : selectedModel.id.toLowerCase().includes('embed')
                          ? JSON.stringify(['embedding'])
                          : isCodexImport || selectedModel.id.toLowerCase().includes('responses')
                            ? JSON.stringify(['responses'])
                            : JSON.stringify(['chat_completion'])

                const modelPayload = {
                    model_name: modelCodeForEp,
                    model_code: modelCodeForEp,
                    space_code: importContext.value.space_code,
                    request_types: request_types,
                    context_length: 8192,
                    max_output_tokens: 8192,
                    abilities: JSON.stringify(['stream', 'tool_call']),
                    owner: selectedModel.owned_by || 'system',
                    enabled: 1,
                    description: 'Imported from provider model fetch',
                    apply_invocation_seed: applyInvocationSeed.value,
                    apply_circuit_break_seed: applyCircuitBreakSeed.value,
                }

                const { success: createSuccess, data: createData } = await apis.model
                    .createModel(modelPayload)
                    .catch(() => ({ success: false }))

                if (config('http.code.success') === createSuccess && createData?.id) {
                    modelId = createData.id
                    successModelCount++
                    const skipped = createData.skipped_seeds || []
                    const applied = createData.applied_seeds || []
                    if (skipped.length > 0) {
                        skippedPolicyCount++
                    } else if (applied.includes('policy_invocation') && applied.includes('policy_circuit_break')) {
                        appliedBothCount++
                    }
                } else {
                    message.error(`模型 ${selectedModel.id} 创建失败，跳过其端点创建`)
                    continue
                }
            }

            // 循环密钥为端点创建绑定
            const endpointCodeTs = Math.floor(Date.now() / 1000)
            const isOAuth = importContext.value.auth_type === 'oauth_token'
            for (const [keyIndex, key] of importContext.value.keysToCreate.entries()) {
                const keySuffix = importContext.value.keysToCreate.length > 1 ? `-${keyIndex + 1}` : ''
                const epCode = `ep-${modelCodeForEp}-${importContext.value.providerCode}-${endpointCodeTs}${keySuffix}`
                const endpointPayload = {
                    code: epCode,
                    provider_id: importContext.value.providerId,
                    model_id: modelId,
                    url: importContext.value.base_url,
                    // OAuth 类型：认证令牌由 provider 级别统一管理，端点不绑定 api_key
                    api_key: isOAuth ? '' : key,
                    auth_type: importContext.value.auth_type || 'api_key',
                    protocol: '',
                    real_model: selectedModel.id,
                    enabled: 1,
                    priority: 1,
                    weight: 1,
                    description: 'Auto created from provider model fetch',
                }

                const { success: endpointSuccess } = await apis.endpoint
                    .createEndpoint(endpointPayload)
                    .catch(() => ({ success: false }))

                if (config('http.code.success') === endpointSuccess) {
                    successEndpointCount++
                } else {
                    message.warning(`模型 ${selectedModel.id} 的端点创建失败`)
                }
            }
        }

        showImportResultMessage({
            successModelCount,
            successEndpointCount,
            appliedBothCount,
            skippedPolicyCount,
        })
        visible.value = false
        emit('ok')
    } catch (e) {
        message.error('保存配置时发生错误')
        console.error(e)
    } finally {
        hideLoadingMsg()
        confirmLoading.value = false
    }
}

function showImportResultMessage({ successModelCount, successEndpointCount, appliedBothCount, skippedPolicyCount }) {
    if (successModelCount === 0) {
        message.success(t('pages.provider.fetchModels.import.success_endpoints', { count: successEndpointCount }))
        return
    }
    if (skippedPolicyCount > 0) {
        message.warning(
            t('pages.provider.fetchModels.import.partial_policies', {
                models: successModelCount,
                endpoints: successEndpointCount,
            })
        )
        return
    }
    if (appliedBothCount === successModelCount) {
        message.success(
            t('pages.provider.fetchModels.import.with_policies', {
                models: successModelCount,
                endpoints: successEndpointCount,
            })
        )
        return
    }
    message.success(
        t('pages.provider.fetchModels.import.success', {
            models: successModelCount,
            endpoints: successEndpointCount,
        })
    )
}

function handleCancel() {
    visible.value = false
}

defineExpose({
    handleOpen,
})
</script>

<style lang="less" scoped>
.recommended-policies {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-top: 16px;
}

.recommended-policies-title {
    font-weight: 500;
}

.recommended-policies-hint {
    opacity: 0.65;
    line-height: 1.5;
}
</style>
