<template>
    <a-modal
        :open="visible"
        :title="$t('pages.provider.fetchModels.mapping.title', '确认导入模型与系统模型映射')"
        :width="680"
        :confirm-loading="confirmLoading"
        @ok="handleOk"
        @cancel="handleCancel">
        <div style="margin-bottom: 12px; opacity: 0.65">
            请选择每个发现的模型应该在当前系统中关联的模型。如果选择已存在的模型，将不会在系统中重复创建该模型。
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
                        placeholder="输入新模型编码"
                        style="width: 100%" />
                    <span
                        v-else
                        style="opacity: 0.25"
                        >- (使用已有)</span
                    >
                </template>
            </template>
        </a-table>
    </a-modal>
</template>

<script setup>
import { ref } from 'vue'
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
    { title: t('pages.provider.fetchModels.table.mapping', '系统关联模型 (下拉搜索)'), key: 'mapping', width: 240 },
    { title: '新模型编码', key: 'new_code' },
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
    }
    models.value = context.models || []

    // 初始化映射，默认都为新建
    mapping.value = {}
    newModelCodes.value = {}
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
                const request_types = selectedModel.id.toLowerCase().includes('embed')
                    ? JSON.stringify(['embedding'])
                    : selectedModel.id.toLowerCase().includes('responses')
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
                }

                const { success: createSuccess, data: createData } = await apis.model
                    .createModel(modelPayload)
                    .catch(() => ({ success: false }))

                if (config('http.code.success') === createSuccess && createData?.id) {
                    modelId = createData.id
                    successModelCount++
                } else {
                    message.error(`模型 ${selectedModel.id} 创建失败，跳过其端点创建`)
                    continue
                }
            }

            // 循环密钥为端点创建绑定
            for (const key of importContext.value.keysToCreate) {
                const ts = Date.now()
                const epCode = `ep-${modelCodeForEp}-${importContext.value.providerCode}-${ts}`
                const endpointPayload = {
                    code: epCode,
                    provider_id: importContext.value.providerId,
                    model_id: modelId,
                    url: importContext.value.base_url,
                    api_key: key,
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

        message.success(`导入完成：新建 ${successModelCount} 个模型，关联并成功创建 ${successEndpointCount} 个端点。`)
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

function handleCancel() {
    visible.value = false
}

defineExpose({
    handleOpen,
})
</script>
