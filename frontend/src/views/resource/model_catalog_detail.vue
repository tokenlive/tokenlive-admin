<template>
    <div class="app-page">
        <a-spin :spinning="loading">
            <!-- 基本信息卡片 -->
            <a-card
                :title="$t('pages.modelCatalog.detail.title') + ': ' + (catalog.slug || catalog.model_id)"
                class="info-card"
                :bordered="false"
                style="margin-bottom: 16px">
                <template #extra>
                    <a-space>
                        <a-tag :color="catalog.status === 'available' ? 'green' : 'default'">{{
                            catalog.status === 'available'
                                ? $t('pages.modelCatalog.detail.status.available')
                                : $t('pages.modelCatalog.detail.status.paused')
                        }}</a-tag>
                        <a-tag :color="catalog.visibility === 'public' ? 'cyan' : 'orange'">{{
                            catalog.visibility === 'public'
                                ? $t('pages.modelCatalog.detail.visibility.public')
                                : $t('pages.modelCatalog.detail.visibility.private')
                        }}</a-tag>
                        <a-tag
                            v-if="catalog.featured"
                            color="gold"
                            >{{ $t('pages.modelCatalog.detail.featured') }}</a-tag
                        >
                        <a-button
                            type="primary"
                            ghost
                            size="small"
                            @click="handleEdit">
                            <template #icon><edit-outlined /></template>
                            {{ $t('button.edit') }}
                        </a-button>
                    </a-space>
                </template>
                <a-card-grid style="width: 25%; text-align: center">
                    <div class="info-item">
                        <span class="info-label">{{ $t('pages.modelCatalog.detail.field.modelId') }}</span>
                        <span class="info-value">{{ catalog.model_id || '--' }}</span>
                    </div>
                </a-card-grid>
                <a-card-grid style="width: 25%; text-align: center">
                    <div class="info-item">
                        <span class="info-label">{{ $t('pages.modelCatalog.detail.field.slug') }}</span>
                        <span class="info-value">{{ catalog.slug || '--' }}</span>
                    </div>
                </a-card-grid>
                <a-card-grid style="width: 25%; text-align: center">
                    <div class="info-item">
                        <span class="info-label">{{ $t('pages.modelCatalog.detail.field.modelCode') }}</span>
                        <span class="info-value">{{ catalog.model_code || '--' }}</span>
                    </div>
                </a-card-grid>
                <a-card-grid style="width: 25%; text-align: center">
                    <div class="info-item">
                        <span class="info-label">{{ $t('pages.modelCatalog.detail.field.contextLength') }}</span>
                        <span class="info-value">{{ catalog.context_length?.toLocaleString() || '--' }}</span>
                    </div>
                </a-card-grid>
                <a-card-grid style="width: 25%; text-align: center">
                    <div class="info-item">
                        <span class="info-label">{{ $t('pages.modelCatalog.detail.field.sortWeight') }}</span>
                        <span class="info-value">{{ catalog.sort_weight ?? '--' }}</span>
                    </div>
                </a-card-grid>
                <a-card-grid style="width: 25%; text-align: center">
                    <div class="info-item">
                        <span class="info-label">{{ $t('pages.modelCatalog.detail.field.publishedAt') }}</span>
                        <span class="info-value">{{
                            catalog.published_at ? dayjs(catalog.published_at).format('YYYY-MM-DD HH:mm') : '--'
                        }}</span>
                    </div>
                </a-card-grid>
                <a-card-grid style="width: 50%; text-align: center">
                    <div class="info-item">
                        <span class="info-label">{{ $t('pages.modelCatalog.detail.field.logo') }}</span>
                        <span class="info-value">
                            <a-image
                                v-if="catalog.logo_url"
                                :src="catalog.logo_url"
                                :width="48"
                                :height="48"
                                style="border-radius: 8px" />
                            <span v-else>--</span>
                        </span>
                    </div>
                </a-card-grid>
                <a-card-grid style="width: 50%; text-align: center">
                    <div class="info-item">
                        <span class="info-label">{{ $t('pages.modelCatalog.detail.field.capabilities') }}</span>
                        <span class="info-value">
                            <a-tag
                                v-for="cap in parseJsonArray(catalog.capabilities)"
                                :key="cap"
                                color="blue"
                                >{{ cap }}</a-tag
                            >
                            <span v-if="!catalog.capabilities">--</span>
                        </span>
                    </div>
                </a-card-grid>
            </a-card>

            <!-- Tab 页 -->
            <a-card
                :bordered="false"
                style="border-radius: 8px">
                <a-tabs v-model:activeKey="activeTab">
                    <!-- 多语言 Tab -->
                    <a-tab-pane
                        key="i18n"
                        :tab="$t('pages.modelCatalog.detail.tab.i18n')">
                        <div class="tab-toolbar">
                            <a-button
                                type="primary"
                                ghost
                                @click="handleAddI18n">
                                <template #icon><plus-outlined /></template>
                                {{ $t('pages.modelCatalog.detail.i18n.add') }}
                            </a-button>
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
                                            >{{ $t('button.edit') }}</a-button
                                        >
                                        <a-popconfirm
                                            title="确认删除？"
                                            @confirm="handleDeleteI18n(record)">
                                            <a-button
                                                type="link"
                                                size="small"
                                                danger
                                                >{{ $t('button.delete') }}</a-button
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
                        :tab="$t('pages.modelCatalog.detail.tab.prices')">
                        <div class="tab-toolbar">
                            <a-button
                                type="primary"
                                ghost
                                @click="handleAddPrice">
                                <template #icon><plus-outlined /></template>
                                {{ $t('pages.modelCatalog.detail.prices.add') }}
                            </a-button>
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
                                        {{
                                            record.status === 'active'
                                                ? $t('pages.modelCatalog.detail.prices.status.active')
                                                : $t('pages.modelCatalog.detail.prices.status.inactive')
                                        }}
                                    </a-tag>
                                </template>
                                <template v-if="'price_display' === column.key">
                                    <div style="font-size: 12px; line-height: 1.6">
                                        <div>
                                            {{ $t('pages.modelCatalog.detail.prices.input') }}: ¥{{
                                                formatPrice(record.input_price)
                                            }}/M
                                        </div>
                                        <div>
                                            {{ $t('pages.modelCatalog.detail.prices.output') }}: ¥{{
                                                formatPrice(record.output_price)
                                            }}/M
                                        </div>
                                        <div v-if="record.cached_price">
                                            {{ $t('pages.modelCatalog.detail.prices.cache') }}: ¥{{
                                                formatPrice(record.cached_price)
                                            }}/M
                                        </div>
                                        <div v-if="record.cache_creation_price">
                                            {{ $t('pages.modelCatalog.detail.prices.cacheCreation') }}: ¥{{
                                                formatPrice(record.cache_creation_price)
                                            }}/M
                                        </div>
                                    </div>
                                </template>
                                <template v-if="'action' === column.key">
                                    <a-space>
                                        <a-button
                                            type="link"
                                            size="small"
                                            @click="handleEditPrice(record)"
                                            >{{ $t('button.edit') }}</a-button
                                        >
                                        <a-popconfirm
                                            title="确认删除？"
                                            @confirm="handleDeletePrice(record)">
                                            <a-button
                                                type="link"
                                                size="small"
                                                danger
                                                >{{ $t('button.delete') }}</a-button
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
                        :tab="$t('pages.modelCatalog.detail.tab.metrics')">
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
            :title="
                i18nIsEdit
                    ? $t('pages.modelCatalog.detail.i18n.editTitle')
                    : $t('pages.modelCatalog.detail.i18n.createTitle')
            "
            :width="560"
            @ok="handleSubmitI18n"
            :confirmLoading="i18nSubmitting">
            <a-form
                :model="i18nForm"
                layout="vertical">
                <a-row :gutter="16">
                    <a-col :span="8">
                        <a-form-item
                            :label="$t('pages.modelCatalog.detail.i18n.locale')"
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
                            :label="$t('pages.modelCatalog.detail.i18n.displayName')"
                            required>
                            <a-input v-model:value="i18nForm.display_name" />
                        </a-form-item>
                    </a-col>
                </a-row>
                <a-form-item :label="$t('pages.modelCatalog.detail.i18n.shortDesc')">
                    <a-textarea
                        v-model:value="i18nForm.short_description"
                        :rows="2" />
                </a-form-item>
                <a-form-item :label="$t('pages.modelCatalog.detail.i18n.longDesc')">
                    <a-textarea
                        v-model:value="i18nForm.long_description"
                        :rows="4" />
                </a-form-item>
                <a-form-item :label="$t('pages.modelCatalog.detail.i18n.tags')">
                    <a-textarea
                        v-model:value="i18nForm.tags"
                        :rows="1"
                        :placeholder="$t('pages.modelCatalog.detail.i18n.tagsPlaceholder')" />
                </a-form-item>
            </a-form>
        </a-modal>

        <!-- 价格版本编辑弹窗 -->
        <a-modal
            v-model:open="priceVisible"
            :title="
                priceIsEdit
                    ? $t('pages.modelCatalog.detail.prices.editTitle')
                    : $t('pages.modelCatalog.detail.prices.createTitle')
            "
            :width="560"
            @ok="handleSubmitPrice"
            :confirmLoading="priceSubmitting">
            <a-form
                :model="priceForm"
                layout="vertical">
                <a-row :gutter="16">
                    <a-col :span="8">
                        <a-form-item
                            :label="$t('pages.modelCatalog.detail.prices.currency')"
                            required>
                            <a-select v-model:value="priceForm.currency">
                                <a-select-option value="CNY">CNY</a-select-option>
                                <a-select-option value="USD">USD</a-select-option>
                            </a-select>
                        </a-form-item>
                    </a-col>
                    <a-col :span="16">
                        <a-form-item
                            :label="$t('pages.modelCatalog.detail.prices.effectiveFrom')"
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
                            :label="$t('pages.modelCatalog.detail.prices.inputPrice')"
                            required>
                            <a-input-number
                                v-model:value="priceForm.input_price"
                                :min="0"
                                :step="0.001"
                                :precision="6"
                                style="width: 100%" />
                        </a-form-item>
                    </a-col>
                    <a-col :span="8">
                        <a-form-item
                            :label="$t('pages.modelCatalog.detail.prices.outputPrice')"
                            required>
                            <a-input-number
                                v-model:value="priceForm.output_price"
                                :min="0"
                                :step="0.001"
                                :precision="6"
                                style="width: 100%" />
                        </a-form-item>
                    </a-col>
                    <a-col :span="8">
                        <a-form-item :label="$t('pages.modelCatalog.detail.prices.cachePrice')">
                            <a-input-number
                                v-model:value="priceForm.cached_price"
                                :min="0"
                                :step="0.001"
                                :precision="6"
                                style="width: 100%" />
                        </a-form-item>
                    </a-col>
                </a-row>
                <a-row :gutter="16">
                    <a-col :span="8">
                        <a-form-item :label="$t('pages.modelCatalog.detail.prices.cacheCreationPrice')">
                            <a-input-number
                                v-model:value="priceForm.cache_creation_price"
                                :min="0"
                                :step="0.001"
                                :precision="6"
                                style="width: 100%" />
                        </a-form-item>
                    </a-col>
                </a-row>
            </a-form>
        </a-modal>
    </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import dayjs from 'dayjs'
import { config } from '@/config'
import apis from '@/apis'
import { useI18n } from 'vue-i18n'
import ModelCatalogEditDialog from './ModelCatalogEditDialog.vue'
import { EditOutlined, PlusOutlined } from '@ant-design/icons-vue'

const { t } = useI18n()
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

const formatPrice = (price) => {
    if (price == null) return '-'
    return Number(price).toFixed(6)
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

const i18nColumns = computed(() => [
    { title: t('pages.modelCatalog.detail.i18n.col.locale'), dataIndex: 'locale', width: 100 },
    { title: t('pages.modelCatalog.detail.i18n.col.displayName'), dataIndex: 'display_name', width: 200 },
    { title: t('pages.modelCatalog.detail.i18n.col.shortDesc'), dataIndex: 'short_description', ellipsis: true },
    { title: t('pages.modelCatalog.detail.i18n.col.action'), key: 'action', width: 120 },
])

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
        message.warning(t('pages.modelCatalog.detail.i18n.nameRequired'))
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
        message.success(t('component.message.success.save'))
        i18nVisible.value = false
        fetchI18n()
    } finally {
        i18nSubmitting.value = false
    }
}

async function handleDeleteI18n(record) {
    await apis.model_catalog.delModelCatalogI18n(modelId, record.locale)
    message.success(t('component.message.success.delete'))
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
    input_price: 0,
    output_price: 0,
    cached_price: null,
    cache_creation_price: null,
    effective_from: null,
})

const priceColumns = computed(() => [
    { title: t('pages.modelCatalog.detail.prices.col.status'), key: 'status', dataIndex: 'status', width: 80 },
    { title: t('pages.modelCatalog.detail.prices.col.currency'), dataIndex: 'currency', width: 70 },
    { title: t('pages.modelCatalog.detail.prices.col.priceDetail'), key: 'price_display', width: 220 },
    {
        title: t('pages.modelCatalog.detail.prices.col.effectiveFrom'),
        dataIndex: 'effective_from',
        width: 160,
        customRender: ({ text }) => (text ? dayjs(text).format('YYYY-MM-DD HH:mm') : '-'),
    },
    {
        title: t('pages.modelCatalog.detail.prices.col.effectiveUntil'),
        dataIndex: 'effective_until',
        width: 160,
        customRender: ({ text }) =>
            text ? dayjs(text).format('YYYY-MM-DD HH:mm') : t('pages.modelCatalog.detail.prices.untilPermanent'),
    },
    { title: t('pages.modelCatalog.detail.prices.col.action'), key: 'action', width: 120 },
])

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
        input_price: 0,
        output_price: 0,
        cached_price: null,
        cache_creation_price: null,
        effective_from: null,
    })
    priceVisible.value = true
}

function handleEditPrice(record) {
    priceIsEdit.value = true
    priceEditId.value = record.id
    Object.assign(priceForm, {
        currency: record.currency,
        input_price: record.input_price,
        output_price: record.output_price,
        cached_price: record.cached_price,
        cache_creation_price: record.cache_creation_price,
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
        message.success(t('component.message.success.save'))
        priceVisible.value = false
        fetchPrices()
    } finally {
        priceSubmitting.value = false
    }
}

async function handleDeletePrice(record) {
    await apis.model_catalog.delModelPriceVersion(record.id)
    message.success(t('component.message.success.delete'))
    fetchPrices()
}

// ---- 服务指标 ----
const metricData = ref([])
const metricLoading = ref(false)

const metricColumns = computed(() => [
    { title: t('pages.modelCatalog.detail.metrics.col.window'), dataIndex: 'window', width: 100 },
    { title: t('pages.modelCatalog.detail.metrics.col.availability'), key: 'availability', width: 100 },
    { title: t('pages.modelCatalog.detail.metrics.col.successRate'), key: 'success_rate', width: 100 },
    { title: t('pages.modelCatalog.detail.metrics.col.ttftP50'), key: 'ttft_p50_ms', width: 100 },
    { title: t('pages.modelCatalog.detail.metrics.col.ttftP95'), key: 'ttft_p95_ms', width: 100 },
    {
        title: t('pages.modelCatalog.detail.metrics.col.responseSpeed'),
        dataIndex: 'response_speed',
        width: 120,
        customRender: ({ text }) => (text ? Number(text).toFixed(2) + ' t/s' : '-'),
    },
    { title: t('pages.modelCatalog.detail.metrics.col.sampleCount'), dataIndex: 'sample_count', width: 100 },
    {
        title: t('pages.modelCatalog.detail.metrics.col.updatedAt'),
        dataIndex: 'updated_at',
        width: 160,
        customRender: ({ text }) => (text ? dayjs(text).format('YYYY-MM-DD HH:mm') : '-'),
    },
])

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

<style lang="less" scoped>
.info-card {
    margin-bottom: 16px;

    :deep(.ant-card-head-title) {
        font-size: 14px;
    }

    :deep(.ant-card-grid) {
        padding: 8px 16px;
    }

    .info-item {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 2px;

        .info-label {
            opacity: 0.6;
            font-size: 13px;
        }

        .info-value {
            font-size: 14px;
            font-weight: 500;
        }
    }
}

.tab-toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
}
</style>
