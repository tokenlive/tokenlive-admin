<template>
    <div class="app-page">
        <a-spin :spinning="loading">
            <!-- 基本信息卡片 -->
            <a-card
                :title="'模型目录: ' + (catalog.slug || catalog.model_id)"
                :bordered="false"
                style="margin-bottom: 16px; border-radius: 8px">
                <template #extra>
                    <a-space>
                        <a-tag :color="catalog.status === 'available' ? 'green' : 'default'">{{
                            catalog.status === 'available' ? '可用' : '暂停'
                        }}</a-tag>
                        <a-tag :color="catalog.visibility === 'public' ? 'cyan' : 'orange'">{{
                            catalog.visibility === 'public' ? '公开' : '私有'
                        }}</a-tag>
                        <a-tag
                            v-if="catalog.featured"
                            color="gold"
                            >精选</a-tag
                        >
                        <a-button
                            type="primary"
                            size="small"
                            @click="handleEdit"
                            >编辑</a-button
                        >
                    </a-space>
                </template>
                <a-descriptions
                    :column="3"
                    size="small">
                    <a-descriptions-item label="模型ID">{{ catalog.model_id }}</a-descriptions-item>
                    <a-descriptions-item label="Slug">{{ catalog.slug }}</a-descriptions-item>
                    <a-descriptions-item label="关联编码">{{ catalog.model_code || '-' }}</a-descriptions-item>
                    <a-descriptions-item label="上下文长度">{{
                        catalog.context_length?.toLocaleString() || '-'
                    }}</a-descriptions-item>
                    <a-descriptions-item label="排序权重">{{ catalog.sort_weight }}</a-descriptions-item>
                    <a-descriptions-item label="发布时间">{{
                        catalog.published_at ? dayjs(catalog.published_at).format('YYYY-MM-DD HH:mm') : '-'
                    }}</a-descriptions-item>
                    <a-descriptions-item
                        label="Logo"
                        :span="3">
                        <a-image
                            v-if="catalog.logo_url"
                            :src="catalog.logo_url"
                            :width="48"
                            :height="48"
                            style="border-radius: 8px" />
                        <span v-else>-</span>
                    </a-descriptions-item>
                    <a-descriptions-item
                        label="能力标签"
                        :span="3">
                        <a-tag
                            v-for="cap in parseJsonArray(catalog.capabilities)"
                            :key="cap"
                            color="blue"
                            >{{ cap }}</a-tag
                        >
                        <span v-if="!catalog.capabilities">-</span>
                    </a-descriptions-item>
                </a-descriptions>
            </a-card>

            <!-- Tab 页 -->
            <a-card
                :bordered="false"
                style="border-radius: 8px">
                <a-tabs v-model:activeKey="activeTab">
                    <!-- 多语言 Tab -->
                    <a-tab-pane
                        key="i18n"
                        tab="多语言内容">
                        <div style="margin-bottom: 12px">
                            <a-button
                                type="primary"
                                size="small"
                                @click="handleAddI18n"
                                >新增语言</a-button
                            >
                        </div>
                        <a-table
                            :columns="i18nColumns"
                            :data-source="i18nData"
                            :loading="i18nLoading"
                            size="small"
                            :pagination="false">
                            <template #bodyCell="{ column, record }">
                                <template v-if="'action' === column.key">
                                    <a-space>
                                        <a-button
                                            type="link"
                                            size="small"
                                            @click="handleEditI18n(record)"
                                            >编辑</a-button
                                        >
                                        <a-popconfirm
                                            title="确认删除？"
                                            @confirm="handleDeleteI18n(record)">
                                            <a-button
                                                type="link"
                                                size="small"
                                                danger
                                                >删除</a-button
                                            >
                                        </a-popconfirm>
                                    </a-space>
                                </template>
                            </template>
                        </a-table>
                    </a-tab-pane>

                    <!-- 价格版本 Tab -->
                    <a-tab-pane
                        key="prices"
                        tab="价格版本">
                        <div style="margin-bottom: 12px">
                            <a-button
                                type="primary"
                                size="small"
                                @click="handleAddPrice"
                                >新增价格版本</a-button
                            >
                        </div>
                        <a-table
                            :columns="priceColumns"
                            :data-source="priceData"
                            :loading="priceLoading"
                            size="small"
                            :pagination="false">
                            <template #bodyCell="{ column, record }">
                                <template v-if="'status' === column.key">
                                    <a-tag :color="record.status === 'active' ? 'green' : 'default'">
                                        {{ record.status === 'active' ? '生效中' : '已停用' }}
                                    </a-tag>
                                </template>
                                <template v-if="'price_display' === column.key">
                                    <div style="font-size: 12px; line-height: 1.6">
                                        <div>输入: ¥{{ formatPrice(record.input_micro_cny_per_1m_tokens) }}/M</div>
                                        <div>输出: ¥{{ formatPrice(record.output_micro_cny_per_1m_tokens) }}/M</div>
                                        <div v-if="record.cache_read_micro_cny_per_1m_tokens">
                                            缓存: ¥{{ formatPrice(record.cache_read_micro_cny_per_1m_tokens) }}/M
                                        </div>
                                    </div>
                                </template>
                                <template v-if="'action' === column.key">
                                    <a-space>
                                        <a-button
                                            type="link"
                                            size="small"
                                            @click="handleEditPrice(record)"
                                            >编辑</a-button
                                        >
                                        <a-popconfirm
                                            title="确认删除？"
                                            @confirm="handleDeletePrice(record)">
                                            <a-button
                                                type="link"
                                                size="small"
                                                danger
                                                >删除</a-button
                                            >
                                        </a-popconfirm>
                                    </a-space>
                                </template>
                            </template>
                        </a-table>
                    </a-tab-pane>

                    <!-- 服务指标 Tab -->
                    <a-tab-pane
                        key="metrics"
                        tab="服务指标">
                        <a-table
                            :columns="metricColumns"
                            :data-source="metricData"
                            :loading="metricLoading"
                            size="small"
                            :pagination="false">
                            <template #bodyCell="{ column, record }">
                                <template v-if="'availability' === column.key">
                                    {{
                                        record.availability != null ? (record.availability * 100).toFixed(2) + '%' : '-'
                                    }}
                                </template>
                                <template v-if="'success_rate' === column.key">
                                    {{
                                        record.success_rate != null ? (record.success_rate * 100).toFixed(2) + '%' : '-'
                                    }}
                                </template>
                                <template v-if="'ttft_p50_ms' === column.key">
                                    {{ record.ttft_p50_ms != null ? record.ttft_p50_ms + 'ms' : '-' }}
                                </template>
                                <template v-if="'ttft_p95_ms' === column.key">
                                    {{ record.ttft_p95_ms != null ? record.ttft_p95_ms + 'ms' : '-' }}
                                </template>
                            </template>
                        </a-table>
                    </a-tab-pane>
                </a-tabs>
            </a-card>
        </a-spin>

        <!-- 编辑目录弹窗 -->
        <ModelCatalogEditDialog
            ref="editDialogRef"
            @success="fetchCatalog" />

        <!-- 多语言编辑弹窗 -->
        <a-modal
            v-model:open="i18nVisible"
            :title="i18nIsEdit ? '编辑多语言' : '新增多语言'"
            :width="560"
            @ok="handleSubmitI18n"
            :confirmLoading="i18nSubmitting">
            <a-form
                :model="i18nForm"
                layout="vertical">
                <a-row :gutter="16">
                    <a-col :span="8">
                        <a-form-item
                            label="语言"
                            required>
                            <a-select
                                v-model:value="i18nForm.locale"
                                :disabled="i18nIsEdit">
                                <a-select-option value="zh-CN">中文</a-select-option>
                                <a-select-option value="en-US">English</a-select-option>
                                <a-select-option value="ja-JP">日本語</a-select-option>
                            </a-select>
                        </a-form-item>
                    </a-col>
                    <a-col :span="16">
                        <a-form-item
                            label="展示名称"
                            required>
                            <a-input v-model:value="i18nForm.display_name" />
                        </a-form-item>
                    </a-col>
                </a-row>
                <a-form-item label="短描述">
                    <a-textarea
                        v-model:value="i18nForm.short_description"
                        :rows="2" />
                </a-form-item>
                <a-form-item label="长描述 (Markdown)">
                    <a-textarea
                        v-model:value="i18nForm.long_description"
                        :rows="4" />
                </a-form-item>
                <a-form-item label="展示标签">
                    <a-textarea
                        v-model:value="i18nForm.tags"
                        :rows="1"
                        placeholder='JSON 数组，如 ["最新","高性价比"]' />
                </a-form-item>
            </a-form>
        </a-modal>

        <!-- 价格版本编辑弹窗 -->
        <a-modal
            v-model:open="priceVisible"
            :title="priceIsEdit ? '编辑价格版本' : '新增价格版本'"
            :width="560"
            @ok="handleSubmitPrice"
            :confirmLoading="priceSubmitting">
            <a-form
                :model="priceForm"
                layout="vertical">
                <a-row :gutter="16">
                    <a-col :span="8">
                        <a-form-item
                            label="货币"
                            required>
                            <a-select v-model:value="priceForm.currency">
                                <a-select-option value="CNY">CNY</a-select-option>
                                <a-select-option value="USD">USD</a-select-option>
                            </a-select>
                        </a-form-item>
                    </a-col>
                    <a-col :span="16">
                        <a-form-item
                            label="生效时间"
                            required>
                            <a-date-picker
                                v-model:value="priceForm.effective_from"
                                show-time
                                style="width: 100%" />
                        </a-form-item>
                    </a-col>
                </a-row>
                <a-row :gutter="16">
                    <a-col :span="8">
                        <a-form-item
                            label="输入价格 (微分/M)"
                            required>
                            <a-input-number
                                v-model:value="priceForm.input_micro_cny_per_1m_tokens"
                                :min="0"
                                style="width: 100%" />
                        </a-form-item>
                    </a-col>
                    <a-col :span="8">
                        <a-form-item
                            label="输出价格 (微分/M)"
                            required>
                            <a-input-number
                                v-model:value="priceForm.output_micro_cny_per_1m_tokens"
                                :min="0"
                                style="width: 100%" />
                        </a-form-item>
                    </a-col>
                    <a-col :span="8">
                        <a-form-item label="缓存价格 (微分/M)">
                            <a-input-number
                                v-model:value="priceForm.cache_read_micro_cny_per_1m_tokens"
                                :min="0"
                                style="width: 100%" />
                        </a-form-item>
                    </a-col>
                </a-row>
            </a-form>
        </a-modal>
    </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import dayjs from 'dayjs'
import { config } from '@/config'
import apis from '@/apis'
import ModelCatalogEditDialog from './ModelCatalogEditDialog.vue'

const route = useRoute()
const modelId = route.params.id

const loading = ref(false)
const catalog = ref({})
const activeTab = ref('i18n')
const editDialogRef = ref(null)

const parseJsonArray = (str) => {
    if (!str) return []
    try {
        return JSON.parse(str)
    } catch {
        return []
    }
}

const formatPrice = (microCny) => {
    if (microCny == null) return '-'
    return (microCny / 1000000).toFixed(6)
}

// ---- 基本信息 ----
async function fetchCatalog() {
    loading.value = true
    try {
        const { success, data } = await apis.model_catalog.getModelCatalog(modelId)
        if (config('http.code.success') === success) catalog.value = data || {}
    } finally {
        loading.value = false
    }
}

function handleEdit() {
    editDialogRef.value.handleEdit(catalog.value)
}

// ---- 多语言 ----
const i18nData = ref([])
const i18nLoading = ref(false)
const i18nVisible = ref(false)
const i18nIsEdit = ref(false)
const i18nSubmitting = ref(false)
const i18nForm = reactive({ locale: 'zh-CN', display_name: '', short_description: '', long_description: '', tags: '' })

const i18nColumns = [
    { title: '语言', dataIndex: 'locale', width: 100 },
    { title: '展示名称', dataIndex: 'display_name', width: 200 },
    { title: '短描述', dataIndex: 'short_description', ellipsis: true },
    { title: '操作', key: 'action', width: 120 },
]

async function fetchI18n() {
    i18nLoading.value = true
    try {
        const { success, data } = await apis.model_catalog.getModelI18nByModelId(modelId)
        if (config('http.code.success') === success) i18nData.value = data || []
    } finally {
        i18nLoading.value = false
    }
}

function handleAddI18n() {
    i18nIsEdit.value = false
    Object.assign(i18nForm, {
        locale: 'zh-CN',
        display_name: '',
        short_description: '',
        long_description: '',
        tags: '',
    })
    i18nVisible.value = true
}

function handleEditI18n(record) {
    i18nIsEdit.value = true
    Object.assign(i18nForm, {
        locale: record.locale,
        display_name: record.display_name,
        short_description: record.short_description || '',
        long_description: record.long_description || '',
        tags: record.tags || '',
    })
    i18nVisible.value = true
}

async function handleSubmitI18n() {
    if (!i18nForm.display_name) {
        message.warning('请输入展示名称')
        return
    }
    i18nSubmitting.value = true
    try {
        if (i18nIsEdit.value) {
            await apis.model_catalog.updateModelCatalogI18n(modelId, i18nForm.locale, {
                ...i18nForm,
                model_id: modelId,
            })
        } else {
            await apis.model_catalog.createModelCatalogI18n({ ...i18nForm, model_id: modelId })
        }
        message.success('保存成功')
        i18nVisible.value = false
        fetchI18n()
    } finally {
        i18nSubmitting.value = false
    }
}

async function handleDeleteI18n(record) {
    await apis.model_catalog.delModelCatalogI18n(modelId, record.locale)
    message.success('删除成功')
    fetchI18n()
}

// ---- 价格版本 ----
const priceData = ref([])
const priceLoading = ref(false)
const priceVisible = ref(false)
const priceIsEdit = ref(false)
const priceSubmitting = ref(false)
const priceEditId = ref('')
const priceForm = reactive({
    currency: 'CNY',
    input_micro_cny_per_1m_tokens: 0,
    output_micro_cny_per_1m_tokens: 0,
    cache_read_micro_cny_per_1m_tokens: null,
    effective_from: null,
})

const priceColumns = [
    { title: '状态', key: 'status', dataIndex: 'status', width: 80 },
    { title: '货币', dataIndex: 'currency', width: 70 },
    { title: '价格明细', key: 'price_display', width: 220 },
    {
        title: '生效时间',
        dataIndex: 'effective_from',
        width: 160,
        customRender: ({ text }) => (text ? dayjs(text).format('YYYY-MM-DD HH:mm') : '-'),
    },
    {
        title: '停用时间',
        dataIndex: 'effective_until',
        width: 160,
        customRender: ({ text }) => (text ? dayjs(text).format('YYYY-MM-DD HH:mm') : '永久'),
    },
    { title: '操作', key: 'action', width: 120 },
]

async function fetchPrices() {
    priceLoading.value = true
    try {
        const { success, data } = await apis.model_catalog.getPricesByModelId(modelId)
        if (config('http.code.success') === success) priceData.value = data || []
    } finally {
        priceLoading.value = false
    }
}

function handleAddPrice() {
    priceIsEdit.value = false
    priceEditId.value = ''
    Object.assign(priceForm, {
        currency: 'CNY',
        input_micro_cny_per_1m_tokens: 0,
        output_micro_cny_per_1m_tokens: 0,
        cache_read_micro_cny_per_1m_tokens: null,
        effective_from: null,
    })
    priceVisible.value = true
}

function handleEditPrice(record) {
    priceIsEdit.value = true
    priceEditId.value = record.id
    Object.assign(priceForm, {
        currency: record.currency,
        input_micro_cny_per_1m_tokens: record.input_micro_cny_per_1m_tokens,
        output_micro_cny_per_1m_tokens: record.output_micro_cny_per_1m_tokens,
        cache_read_micro_cny_per_1m_tokens: record.cache_read_micro_cny_per_1m_tokens,
        effective_from: record.effective_from ? dayjs(record.effective_from) : null,
    })
    priceVisible.value = true
}

async function handleSubmitPrice() {
    priceSubmitting.value = true
    try {
        const params = { ...priceForm, model_id: modelId, effective_from: priceForm.effective_from?.toISOString() }
        if (priceIsEdit.value) {
            await apis.model_catalog.updateModelPriceVersion(priceEditId.value, params)
        } else {
            await apis.model_catalog.createModelPriceVersion(params)
        }
        message.success('保存成功')
        priceVisible.value = false
        fetchPrices()
    } finally {
        priceSubmitting.value = false
    }
}

async function handleDeletePrice(record) {
    await apis.model_catalog.delModelPriceVersion(record.id)
    message.success('删除成功')
    fetchPrices()
}

// ---- 服务指标 ----
const metricData = ref([])
const metricLoading = ref(false)

const metricColumns = [
    { title: '统计窗口', dataIndex: 'window', width: 100 },
    { title: '可用率', key: 'availability', width: 100 },
    { title: '成功率', key: 'success_rate', width: 100 },
    { title: 'TTFT P50', key: 'ttft_p50_ms', width: 100 },
    { title: 'TTFT P95', key: 'ttft_p95_ms', width: 100 },
    {
        title: '响应速度',
        dataIndex: 'response_speed',
        width: 120,
        customRender: ({ text }) => (text ? Number(text).toFixed(2) + ' t/s' : '-'),
    },
    { title: '样本数', dataIndex: 'sample_count', width: 100 },
    {
        title: '更新时间',
        dataIndex: 'updated_at',
        width: 160,
        customRender: ({ text }) => (text ? dayjs(text).format('YYYY-MM-DD HH:mm') : '-'),
    },
]

async function fetchMetrics() {
    metricLoading.value = true
    try {
        const { success, data } = await apis.model_catalog.getMetricsByModelId(modelId)
        if (config('http.code.success') === success) metricData.value = data || []
    } finally {
        metricLoading.value = false
    }
}

// ---- 初始化 ----
onMounted(() => {
    fetchCatalog()
    fetchI18n()
    fetchPrices()
    fetchMetrics()
})
</script>
