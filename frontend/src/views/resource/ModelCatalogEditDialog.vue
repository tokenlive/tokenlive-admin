<template>
    <a-modal
        v-model:open="visible"
        :title="isEdit ? '编辑模型目录' : '新增模型目录'"
        :width="640"
        @ok="handleSubmit"
        @cancel="handleCancel"
        :confirmLoading="submitLoading">
        <a-form
            ref="formRef"
            :model="formData"
            :rules="rules"
            layout="vertical">
            <a-row :gutter="16">
                <a-col :span="12">
                    <a-form-item
                        label="模型ID"
                        name="model_id">
                        <a-input
                            v-model:value="formData.model_id"
                            :disabled="isEdit"
                            placeholder="唯一标识，如 gpt-4o" />
                    </a-form-item>
                </a-col>
                <a-col :span="12">
                    <a-form-item
                        label="Slug"
                        name="slug">
                        <a-input
                            v-model:value="formData.slug"
                            placeholder="URL 友好标识" />
                    </a-form-item>
                </a-col>
            </a-row>
            <a-row :gutter="16">
                <a-col :span="12">
                    <a-form-item
                        label="关联模型编码"
                        name="model_code">
                        <a-input
                            v-model:value="formData.model_code"
                            placeholder="关联 admin model.model_code" />
                    </a-form-item>
                </a-col>
                <a-col :span="6">
                    <a-form-item
                        label="状态"
                        name="status">
                        <a-select v-model:value="formData.status">
                            <a-select-option value="available">可用</a-select-option>
                            <a-select-option value="paused">暂停</a-select-option>
                        </a-select>
                    </a-form-item>
                </a-col>
                <a-col :span="6">
                    <a-form-item
                        label="可见性"
                        name="visibility">
                        <a-select v-model:value="formData.visibility">
                            <a-select-option value="public">公开</a-select-option>
                            <a-select-option value="private">私有</a-select-option>
                        </a-select>
                    </a-form-item>
                </a-col>
            </a-row>
            <a-row :gutter="16">
                <a-col :span="12">
                    <a-form-item
                        label="上下文长度"
                        name="context_length">
                        <a-input-number
                            v-model:value="formData.context_length"
                            :min="0"
                            style="width: 100%"
                            placeholder="如 128000" />
                    </a-form-item>
                </a-col>
                <a-col :span="12">
                    <a-form-item
                        label="排序权重"
                        name="sort_weight">
                        <a-input-number
                            v-model:value="formData.sort_weight"
                            :min="0"
                            style="width: 100%" />
                    </a-form-item>
                </a-col>
            </a-row>
            <a-form-item
                label="Logo URL"
                name="logo_url">
                <a-input
                    v-model:value="formData.logo_url"
                    placeholder="模型 Logo 地址" />
            </a-form-item>
            <a-form-item
                label="能力标签"
                name="capabilities">
                <a-select
                    v-model:value="formData.capabilities"
                    mode="multiple"
                    placeholder="选择能力标签"
                    :options="capabilityOptions" />
            </a-form-item>
            <a-form-item
                label="输入模态"
                name="input_modalities">
                <a-select
                    v-model:value="formData.input_modalities"
                    mode="multiple"
                    placeholder="选择输入模态"
                    :options="modalityOptions" />
            </a-form-item>
            <a-form-item
                label="输出模态"
                name="output_modalities">
                <a-select
                    v-model:value="formData.output_modalities"
                    mode="multiple"
                    placeholder="选择输出模态"
                    :options="modalityOptions" />
            </a-form-item>
            <a-form-item name="featured">
                <a-checkbox v-model:checked="formData.featured">精选推荐</a-checkbox>
            </a-form-item>
        </a-form>
    </a-modal>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { message } from 'ant-design-vue'
import apis from '@/apis'

const emit = defineEmits(['success'])
const visible = ref(false)
const isEdit = ref(false)
const editId = ref('')
const submitLoading = ref(false)
const formRef = ref(null)

// 能力标签选项
const capabilityOptions = [
    { value: 'streaming', label: '流式输出 (Streaming)' },
    { value: 'tool_use', label: '工具调用 (Tool Use)' },
    { value: 'reasoning', label: '推理能力 (Reasoning)' },
    { value: 'structured_output', label: '结构化输出 (Structured Output)' },
    { value: 'vision', label: '视觉能力 (Vision)' },
    { value: 'code_interpreter', label: '代码解释器 (Code Interpreter)' },
    { value: 'function_calling', label: '函数调用 (Function Calling)' },
]

// 模态选项
const modalityOptions = [
    { value: 'text', label: '文本 (Text)' },
    { value: 'image', label: '图像 (Image)' },
    { value: 'audio', label: '音频 (Audio)' },
    { value: 'video', label: '视频 (Video)' },
    { value: 'file', label: '文件 (File)' },
]

const defaultForm = {
    model_id: '',
    model_code: '',
    slug: '',
    status: 'available',
    visibility: 'public',
    logo_url: '',
    context_length: null,
    sort_weight: 0,
    capabilities: [],
    input_modalities: [],
    output_modalities: [],
    featured: false,
}
const formData = reactive({ ...defaultForm })

const rules = {
    model_id: [{ required: true, message: '请输入模型ID' }],
    slug: [{ required: true, message: '请输入 Slug' }],
    status: [{ required: true, message: '请选择状态' }],
    visibility: [{ required: true, message: '请选择可见性' }],
}

function handleCreate() {
    isEdit.value = false
    editId.value = ''
    Object.assign(formData, { ...defaultForm, featured: false })
    visible.value = true
}

function handleEdit(record) {
    isEdit.value = true
    editId.value = record.model_id
    const formDataValue = {
        model_id: record.model_id,
        model_code: record.model_code || '',
        slug: record.slug,
        status: record.status,
        visibility: record.visibility,
        logo_url: record.logo_url || '',
        context_length: record.context_length,
        sort_weight: record.sort_weight || 0,
        capabilities: [],
        input_modalities: [],
        output_modalities: [],
        featured: record.featured || false,
    }

    // 解析 JSON 字符串为数组
    if (record.capabilities) {
        try {
            formDataValue.capabilities =
                typeof record.capabilities === 'string' ? JSON.parse(record.capabilities) : record.capabilities
        } catch (e) {
            formDataValue.capabilities = []
        }
    }

    if (record.input_modalities) {
        try {
            formDataValue.input_modalities =
                typeof record.input_modalities === 'string'
                    ? JSON.parse(record.input_modalities)
                    : record.input_modalities
        } catch (e) {
            formDataValue.input_modalities = []
        }
    }

    if (record.output_modalities) {
        try {
            formDataValue.output_modalities =
                typeof record.output_modalities === 'string'
                    ? JSON.parse(record.output_modalities)
                    : record.output_modalities
        } catch (e) {
            formDataValue.output_modalities = []
        }
    }

    Object.assign(formData, formDataValue)
    visible.value = true
}

async function handleSubmit() {
    try {
        await formRef.value.validateFields()
        submitLoading.value = true
        // 将数组转换为 JSON 字符串
        const params = {
            ...formData,
            capabilities: JSON.stringify(formData.capabilities || []),
            input_modalities: JSON.stringify(formData.input_modalities || []),
            output_modalities: JSON.stringify(formData.output_modalities || []),
        }
        if (isEdit.value) {
            await apis.model_catalog.updateModelCatalog(editId.value, params)
            message.success('更新成功')
        } else {
            await apis.model_catalog.createModelCatalog(params)
            message.success('创建成功')
        }
        visible.value = false
        emit('success')
    } catch (e) {
        if (e?.errorFields) return
    } finally {
        submitLoading.value = false
    }
}

function handleCancel() {
    formRef.value?.resetFields()
}

defineExpose({ handleCreate, handleEdit })
</script>
