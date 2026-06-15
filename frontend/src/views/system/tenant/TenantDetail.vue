<template>
    <div class="tenant-detail">
        <!-- 头部：基本信息 -->
        <a-card
            :title="$t('pages.tenant.detail.basic_info')"
            class="info-card"
            :bordered="false">
            <a-card-grid style="width: 20%; text-align: center">
                <div class="info-item">
                    <span class="info-label">{{ $t('pages.tenant.form.name') }}</span>
                    <span class="info-value">{{ tenantData.name || '--' }}</span>
                </div>
            </a-card-grid>
            <a-card-grid style="width: 20%; text-align: center">
                <div class="info-item">
                    <span class="info-label">{{ $t('pages.tenant.form.code') }}</span>
                    <span class="info-value tenant-code-container">
                        <span class="tenant-code">{{ tenantData.code || '--' }}</span>
                        <a-tooltip
                            v-if="tenantData.code"
                            :title="$t('pages.tenant.copy.code')">
                            <copy-outlined
                                class="copy-btn-icon"
                                @click="handleCopyCode(tenantData.code)" />
                        </a-tooltip>
                    </span>
                </div>
            </a-card-grid>
            <a-card-grid style="width: 30%; text-align: center">
                <div class="info-item">
                    <span class="info-label">{{ $t('pages.tenant.form.api_key') }}</span>
                    <span class="info-value tenant-code-container">
                        <span class="tenant-code">
                            {{ apiKeyVisible ? tenantData.api_key || '--' : '••••••••••••••••••••••••••••••••' }}
                        </span>
                        <a-tooltip
                            :title="
                                apiKeyVisible
                                    ? $t('pages.tenant.detail.api_key.hide')
                                    : $t('pages.tenant.detail.api_key.show')
                            ">
                            <span
                                class="copy-btn-icon"
                                @click="apiKeyVisible = !apiKeyVisible">
                                <eye-outlined v-if="!apiKeyVisible" />
                                <eye-invisible-outlined v-else />
                            </span>
                        </a-tooltip>
                        <a-tooltip
                            v-if="tenantData.api_key"
                            :title="$t('pages.tenant.copy.api_key')">
                            <copy-outlined
                                class="copy-btn-icon"
                                @click="handleCopyCode(tenantData.api_key)" />
                        </a-tooltip>
                    </span>
                </div>
            </a-card-grid>
            <a-card-grid style="width: 10%; text-align: center">
                <div class="info-item">
                    <span class="info-label">{{ $t('pages.tenant.form.status') }}</span>
                    <span class="info-value">
                        <a-tag :color="tenantData.status === 'activated' ? 'success' : 'error'">
                            {{
                                tenantData.status === 'activated'
                                    ? $t('pages.tenant.form.status.activated')
                                    : $t('pages.tenant.form.status.freezed')
                            }}
                        </a-tag>
                    </span>
                </div>
            </a-card-grid>
            <a-card-grid style="width: 20%; text-align: center">
                <div class="info-item">
                    <span class="info-label">{{ $t('pages.tenant.form.created_at') }}</span>
                    <span class="info-value">{{ formatUtcDateTime(tenantData.created_at) || '--' }}</span>
                </div>
            </a-card-grid>
            <a-card-grid style="width: 100%">
                <div class="info-desc-item">
                    <span class="info-label">{{ $t('pages.tenant.form.description') }}:</span>
                    <span class="info-desc-value">{{
                        tenantData.description || $t('pages.tenant.detail.description.empty')
                    }}</span>
                </div>
            </a-card-grid>
        </a-card>

        <!-- Tab 区域：模型配置 -->
        <a-card
            class="detail-card"
            :bordered="false">
            <a-tabs
                v-model:activeKey="activeTab"
                class="detail-tabs">
                <a-tab-pane
                    key="model"
                    :tab="$t('pages.tenant.detail.tab.model')" />
            </a-tabs>

            <div
                v-if="activeTab === 'model'"
                class="model-config-container">
                <!-- 工具栏 -->
                <div class="tab-toolbar">
                    <a-button
                        type="primary"
                        ghost
                        @click="openAddModelModal">
                        <template #icon>
                            <plus-outlined />
                        </template>
                        {{ $t('pages.tenant.detail.model.add') }}
                    </a-button>
                    <div class="tab-toolbar-right">
                        <a-input-search
                            v-model:value="authorizedSearchKey"
                            :placeholder="$t('pages.tenant.detail.model.search.authorized')"
                            style="width: 200px"
                            allow-clear
                            @search="handleAuthorizedSearch"
                            @pressEnter="handleAuthorizedSearch" />
                        <a-button @click="loadAuthorizedModels">
                            <template #icon><reload-outlined /></template>
                        </a-button>
                    </div>
                </div>

                <!-- 已授权模型表格 -->
                <a-table
                    row-key="id"
                    :columns="authorizedColumns"
                    :data-source="filteredAuthorizedModels"
                    :loading="authorizedLoading"
                    :pagination="authorizedPagination"
                    @change="onAuthorizedTableChange">
                    <template #bodyCell="{ column, record }">
                        <template v-if="'allowed_endpoints' === column.key">
                            <span v-if="!record.allowedEndpointCount || record.allowedEndpointCount === 0">
                                <a-tag color="default">{{ $t('pages.tenant.detail.endpoint.all') }}</a-tag>
                            </span>
                            <span v-else>
                                <a-tag color="blue">{{
                                    $t('pages.tenant.detail.endpoint.count', { count: record.allowedEndpointCount })
                                }}</a-tag>
                            </span>
                        </template>
                        <template v-if="'enabled' === column.key">
                            <a-tag :color="record.enabled === 1 ? 'success' : 'default'">
                                {{
                                    record.enabled === 1
                                        ? $t('pages.tenant.detail.enabled')
                                        : $t('pages.tenant.detail.disabled')
                                }}
                            </a-tag>
                        </template>
                        <template v-if="'created_at' === column.key">
                            {{ formatUtcDateTime(record.created_at) }}
                        </template>
                        <template v-if="'action' === column.key">
                            <x-action-button
                                @click="openConfigEndpointsDrawer(record)"
                                style="margin-right: 8px">
                                <a-tooltip :title="$t('pages.tenant.detail.endpoint.select')">
                                    <setting-outlined style="color: #1890ff" />
                                </a-tooltip>
                            </x-action-button>
                            <x-action-button @click="handleRemoveModel(record)">
                                <a-tooltip :title="$t('pages.tenant.detail.model.remove')">
                                    <delete-outlined style="color: #ff4d4f" />
                                </a-tooltip>
                            </x-action-button>
                        </template>
                    </template>
                    <template #emptyText>
                        <a-empty :description="emptyAuthorizedText" />
                    </template>
                </a-table>
            </div>
        </a-card>

        <!-- 添加模型抽屉 -->
        <a-drawer
            v-model:open="addModalVisible"
            :title="$t('pages.tenant.detail.model.add_authorized')"
            :width="960"
            placement="right"
            :destroy-on-close="true"
            @close="addModalVisible = false">
            <div class="add-modal-toolbar">
                <a-input-search
                    v-model:value="modalSearchKey"
                    :placeholder="$t('pages.tenant.detail.model.search.available')"
                    style="width: 280px"
                    allow-clear
                    @search="handleModalSearch"
                    @pressEnter="handleModalSearch" />
                <span class="selected-count">
                    {{ $t('pages.tenant.detail.model.selected', { count: selectedModelKeys.length }) }}
                </span>
            </div>
            <a-table
                row-key="id"
                :columns="addModalColumns"
                :data-source="filteredAvailableModels"
                :loading="availableModelsLoading"
                :pagination="availablePagination"
                :row-selection="{
                    selectedRowKeys: selectedModelKeys,
                    onChange: onSelectChange,
                }"
                @change="onAvailableTableChange">
                <template #bodyCell="{ column, record }">
                    <template v-if="'enabled' === column.key">
                        <a-tag :color="record.enabled === 1 ? 'success' : 'default'">
                            {{
                                record.enabled === 1
                                    ? $t('pages.tenant.detail.enabled')
                                    : $t('pages.tenant.detail.disabled')
                            }}
                        </a-tag>
                    </template>
                </template>
                <template #emptyText>
                    <a-empty :description="$t('pages.tenant.detail.model.empty.available')" />
                </template>
            </a-table>
            <template #footer>
                <div class="drawer-footer">
                    <a-button @click="addModalVisible = false">{{ $t('common.cancel') }}</a-button>
                    <a-button
                        type="primary"
                        :loading="addModalLoading"
                        :disabled="selectedModelKeys.length === 0"
                        @click="handleConfirmAdd">
                        {{ $t('pages.tenant.detail.model.confirm_add') }}
                    </a-button>
                </div>
            </template>
        </a-drawer>

        <!-- 选择端点抽屉 -->
        <a-drawer
            v-model:open="configEndpointsVisible"
            :title="$t('pages.tenant.detail.endpoint.select_title')"
            :width="960"
            placement="right"
            :destroy-on-close="true"
            @close="configEndpointsVisible = false">
            <div class="endpoint-drawer-toolbar">
                <a-input-search
                    v-model:value="endpointSearchKey"
                    :placeholder="$t('pages.tenant.detail.endpoint.search')"
                    style="width: 280px"
                    allow-clear />
                <a-select
                    v-model:value="endpointProviderFilter"
                    :placeholder="$t('pages.tenant.detail.endpoint.provider_filter')"
                    allow-clear
                    style="width: 180px">
                    <a-select-option
                        v-for="p in endpointProviderOptions"
                        :key="p"
                        :value="p"
                        >{{ p }}</a-select-option
                    >
                </a-select>
                <span class="selected-count">
                    {{ $t('pages.tenant.detail.endpoint.selected', { count: selectedEndpointIds.length }) }}
                </span>
            </div>
            <a-table
                row-key="id"
                :columns="endpointDrawerColumns"
                :data-source="filteredEndpoints"
                :loading="configEndpointsLoading"
                :pagination="endpointPagination"
                :row-selection="{
                    selectedRowKeys: selectedEndpointIds,
                    onChange: onEndpointSelectChange,
                }"
                @change="onEndpointTableChange">
                <template #bodyCell="{ column, record }">
                    <template v-if="'enabled' === column.key">
                        <a-tag :color="record.enabled === 1 ? 'success' : 'default'">
                            {{
                                record.enabled === 1
                                    ? $t('pages.tenant.detail.enabled')
                                    : $t('pages.tenant.detail.disabled')
                            }}
                        </a-tag>
                    </template>
                    <template v-if="'status_points' === column.key">
                        <div
                            v-if="record.status_points && record.status_points.length > 0"
                            style="display: flex; gap: 2px; align-items: center">
                            <a-tooltip
                                v-for="(point, idx) in record.status_points"
                                :key="idx">
                                <template #title>
                                    {{ point.start_time }} ~ {{ point.end_time }}<br />
                                    {{ $t('common.success') }}: {{ point.success_count }} | {{ $t('common.failed') }}:
                                    {{ point.fail_count }}
                                </template>
                                <div :style="getPointStyle(point)"></div>
                            </a-tooltip>
                        </div>
                        <span
                            v-else
                            style="color: #bbb"
                            >--</span
                        >
                    </template>
                </template>
                <template #emptyText>
                    <a-empty :description="$t('pages.tenant.detail.endpoint.empty')" />
                </template>
            </a-table>
            <template #footer>
                <div class="drawer-footer">
                    <a-button @click="configEndpointsVisible = false">{{ $t('common.cancel') }}</a-button>
                    <a-button
                        type="primary"
                        :loading="configEndpointsSaving"
                        @click="handleSaveEndpoints">
                        {{ $t('common.save') }}
                    </a-button>
                </div>
            </template>
        </a-drawer>
    </div>
</template>

<script setup>
import { ref, computed, onMounted, reactive } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { message, Modal } from 'ant-design-vue'
import {
    CopyOutlined,
    PlusOutlined,
    DeleteOutlined,
    ReloadOutlined,
    SettingOutlined,
    EyeOutlined,
    EyeInvisibleOutlined,
} from '@ant-design/icons-vue'
import apis from '@/apis'
import { config } from '@/config'
import { formatUtcDateTime } from '@/utils/util'

defineOptions({
    name: 'tenantDetail',
})

const route = useRoute()
const { t } = useI18n()
const tenantId = ref(route.params.id)
const tenantData = ref({})
const activeTab = ref('model')
const emptyAuthorizedText = computed(() => t('pages.tenant.detail.model.empty.authorized'))

// ========== 已授权模型表格 ==========
const authorizedLoading = ref(false)
const allAuthorizedModels = ref([]) // 已授权的模型完整信息列表
const authorizedSearchKey = ref('')
const apiKeyVisible = ref(false)
const authorizedPagination = reactive({
    current: 1,
    pageSize: 10,
    total: 0,
    showSizeChanger: true,
    showTotal: (total) => t('common.pagination.total', { total }),
})

const authorizedColumns = computed(() => [
    { title: t('pages.tenant.detail.model.name'), dataIndex: 'model_name', key: 'model_name', width: 180 },
    { title: t('pages.tenant.detail.model.code'), dataIndex: 'model_code', key: 'model_code', width: 150 },
    {
        title: t('pages.tenant.detail.endpoint.selected_column'),
        dataIndex: 'allowedEndpointCount',
        key: 'allowed_endpoints',
        width: 120,
    },
    { title: t('pages.tenant.detail.model.space'), dataIndex: 'space_code', key: 'space_code', width: 120 },
    { title: t('pages.tenant.detail.model.owner'), dataIndex: 'owner', key: 'owner', width: 100 },
    { title: t('pages.tenant.form.status'), key: 'enabled', width: 90 },
    { title: t('pages.tenant.form.created_at'), key: 'created_at', width: 160 },
    { title: t('common.action'), key: 'action', fixed: 'right', width: 120 },
])

// ========== 配置端点抽屉 ==========
const configEndpointsVisible = ref(false)
const configEndpointsLoading = ref(false)
const configEndpointsSaving = ref(false)
const currentConfigModel = ref({})
const allEndpoints = ref([]) // 当前模型的所有端点
const selectedEndpointIds = ref([]) // 已选端点 ID 列表
const endpointSearchKey = ref('') // 搜索关键词
const endpointProviderFilter = ref(undefined) // 供应商筛选
const endpointPagination = reactive({
    current: 1,
    pageSize: 10,
    total: 0,
    showSizeChanger: true,
    showTotal: (total) => t('common.pagination.total', { total }),
})

const endpointDrawerColumns = computed(() => [
    { title: t('pages.tenant.detail.endpoint.provider'), dataIndex: 'provider_name', key: 'provider_name', width: 90 },
    { title: t('pages.tenant.detail.endpoint.real_model'), dataIndex: 'real_model', key: 'real_model', width: 120 },
    {
        title: t('pages.tenant.detail.endpoint.weight'),
        dataIndex: 'weight',
        key: 'weight',
        width: 70,
        sorter: (a, b) => a.weight - b.weight,
    },
    {
        title: t('pages.tenant.detail.endpoint.priority'),
        dataIndex: 'priority',
        key: 'priority',
        width: 70,
        sorter: (a, b) => a.priority - b.priority,
    },
    {
        title: t('pages.tenant.detail.endpoint.enabled'),
        key: 'enabled',
        width: 90,
        sorter: (a, b) => a.enabled - b.enabled,
    },
    { title: t('pages.tenant.detail.endpoint.recent_usage'), key: 'status_points', width: 100 },
])

// 端点搜索和筛选
const filteredEndpoints = computed(() => {
    let result = allEndpoints.value

    // 按关键词搜索
    const key = endpointSearchKey.value.toLowerCase().trim()
    if (key) {
        result = result.filter(
            (ep) =>
                (ep.provider_name || '').toLowerCase().includes(key) ||
                (ep.url || '').toLowerCase().includes(key) ||
                (ep.description || '').toLowerCase().includes(key) ||
                (ep.real_model || '').toLowerCase().includes(key)
        )
    }

    // 按供应商筛选
    if (endpointProviderFilter.value) {
        result = result.filter((ep) => ep.provider_name === endpointProviderFilter.value)
    }

    return result
})

// 供应商选项列表（去重）
const endpointProviderOptions = computed(() => {
    const names = new Set(allEndpoints.value.map((ep) => ep.provider_name).filter(Boolean))
    return Array.from(names).sort()
})

// 支持前端搜索过滤已授权的模型
const filteredAuthorizedModels = computed(() => {
    const key = authorizedSearchKey.value.toLowerCase().trim()
    if (!key) return allAuthorizedModels.value
    return allAuthorizedModels.value.filter(
        (m) => (m.model_name || '').toLowerCase().includes(key) || (m.model_code || '').toLowerCase().includes(key)
    )
})

function handleAuthorizedSearch() {
    authorizedPagination.current = 1
}

function onAuthorizedTableChange({ current, pageSize }) {
    authorizedPagination.current = current
    authorizedPagination.pageSize = pageSize
}

// ========== 添加模型弹窗 ==========
const addModalVisible = ref(false)
const addModalLoading = ref(false)
const availableModelsLoading = ref(false)
const allModelsCache = ref([]) // 系统中所有模型的缓存
const allProvidersCache = ref([]) // 系统中所有供应商的缓存
const selectedModelKeys = ref([])
const modalSearchKey = ref('')
const availablePagination = reactive({
    current: 1,
    pageSize: 10,
    total: 0,
    showSizeChanger: true,
    showTotal: (total) => t('common.pagination.total', { total }),
})

const addModalColumns = computed(() => [
    { title: t('pages.tenant.detail.model.name'), dataIndex: 'model_name', key: 'model_name', width: 180 },
    { title: t('pages.tenant.detail.model.code'), dataIndex: 'model_code', key: 'model_code', width: 150 },
    { title: t('pages.tenant.detail.model.space'), dataIndex: 'space_code', key: 'space_code', width: 120 },
    { title: t('pages.tenant.detail.model.owner'), dataIndex: 'owner', key: 'owner', width: 100 },
    { title: t('pages.tenant.form.status'), key: 'enabled', width: 90 },
])

// 可添加的模型 = 所有模型 - 已授权的模型
const availableModels = computed(() => {
    const authorizedIds = new Set(allAuthorizedModels.value.map((m) => m.id))
    return allModelsCache.value.filter((m) => !authorizedIds.has(m.id))
})

// 弹窗内搜索过滤
const filteredAvailableModels = computed(() => {
    const key = modalSearchKey.value.toLowerCase().trim()
    if (!key) return availableModels.value
    return availableModels.value.filter(
        (m) => (m.model_name || '').toLowerCase().includes(key) || (m.model_code || '').toLowerCase().includes(key)
    )
})

function handleModalSearch() {
    availablePagination.current = 1
}

function onAvailableTableChange({ current, pageSize }) {
    availablePagination.current = current
    availablePagination.pageSize = pageSize
}

function onSelectChange(selectedKeys) {
    selectedModelKeys.value = selectedKeys
}

// ========== 数据加载 ==========
onMounted(async () => {
    await loadTenantDetail()
})

/**
 * 加载租户详情
 */
async function loadTenantDetail() {
    try {
        const { data, success } = await apis.tenant.get(tenantId.value)
        if (success && data) {
            tenantData.value = data
            // 先加载全量模型和供应商缓存，再加载已授权列表（后者依赖前者的缓存数据）
            await Promise.all([loadAllModels(), loadAllProviders()])
            await loadAuthorizedModels()
        }
    } catch (error) {
        message.error(t('pages.tenant.detail.load.failed'))
    }
}

/**
 * 加载系统中所有模型（缓存，用于反查已授权模型的详细信息和弹窗可选列表）
 */
async function loadAllModels() {
    try {
        const { data, success } = await apis.model.getModelList({
            pageSize: 1000,
            current: 1,
        })
        if (success && data) {
            allModelsCache.value = data
        }
    } catch (error) {
        message.error(t('pages.tenant.detail.model.load.failed'))
    }
}

/**
 * 加载系统中所有供应商（缓存，用于限用供应商的反查名称映射）
 */
async function loadAllProviders() {
    try {
        const { data, success } = await apis.provider.getProviderList({
            pageSize: 1000,
            current: 1,
        })
        if (success && data) {
            allProvidersCache.value = data
        }
    } catch (error) {
        console.error('Failed to load provider list', error)
    }
}

let currentLoadId = 0

/**
 * 加载已授权模型列表（通过 model IDs 反查完整信息）
 */
async function loadAuthorizedModels() {
    if (!tenantData.value.code) return
    currentLoadId++
    const thisLoadId = currentLoadId
    try {
        authorizedLoading.value = true
        const { data, success } = await apis.tenant.getAuthorizedModelIds(tenantData.value.code)
        if (success && data) {
            const idSet = new Set(data)
            const models = allModelsCache.value.filter((m) => idSet.has(m.id))

            // 级联加载每个已授权模型的限制端点数量 (分批加载以限制并发，防流控或浏览器连接数超限导致接口失败)
            const batchSize = 3
            const modelsWithEndpoints = []
            for (let i = 0; i < models.length; i += batchSize) {
                const batch = models.slice(i, i + batchSize)
                const batchResults = await Promise.all(
                    batch.map(async (m) => {
                        // 1. 获取限制端点 ID 列表
                        const res = await apis.tenant
                            .getTenantModelEndpoints(tenantData.value.code, m.id)
                            .catch((err) => {
                                console.error(`Failed to load endpoint restrictions for ${m.model_name}(${m.id}):`, err)
                                return { data: [] }
                            })
                        const allowedEndpointIds = res.data || []

                        return {
                            ...m,
                            allowedEndpointCount: allowedEndpointIds.length,
                        }
                    })
                )
                modelsWithEndpoints.push(...batchResults)
            }

            // 防止竞态覆盖
            if (thisLoadId === currentLoadId) {
                allAuthorizedModels.value = modelsWithEndpoints
                authorizedPagination.total = allAuthorizedModels.value.length
            }
        }
        if (thisLoadId === currentLoadId) {
            authorizedLoading.value = false
        }
    } catch (error) {
        if (thisLoadId === currentLoadId) {
            authorizedLoading.value = false
            message.error(t('pages.tenant.detail.model.authorized.load.failed'))
        }
    }
}

/**
 * 打开配置端点抽屉
 */
async function openConfigEndpointsDrawer(record) {
    currentConfigModel.value = record
    selectedEndpointIds.value = []
    allEndpoints.value = []
    endpointSearchKey.value = ''
    endpointProviderFilter.value = undefined
    endpointPagination.current = 1
    configEndpointsVisible.value = true

    try {
        configEndpointsLoading.value = true

        // 1. 获取模型下的所有端点
        const epRes = await apis.endpoint.getEndpointsByModelId(record.id).catch(() => ({ data: [] }))
        const endpoints = epRes.data || []

        // 丰富端点信息：提取供应商名称和协议
        allEndpoints.value = endpoints.map((ep) => ({
            ...ep,
            provider_name: ep.provider?.name || '--',
            protocol: ep.protocol || ep.provider?.protocol || '--',
        }))

        // 2. 获取当前租户当前模型已授权的端点 ID 列表
        const res = await apis.tenant
            .getTenantModelEndpoints(tenantData.value.code, record.id)
            .catch(() => ({ data: [] }))
        selectedEndpointIds.value = res.data || []

        endpointPagination.total = allEndpoints.value.length
        configEndpointsLoading.value = false
    } catch (error) {
        configEndpointsLoading.value = false
        message.error(t('pages.tenant.detail.endpoint.load.failed'))
    }
}

/**
 * 端点表格选择变更
 */
function onEndpointSelectChange(selectedKeys) {
    selectedEndpointIds.value = selectedKeys
}

/**
 * 端点表格分页变更
 */
function onEndpointTableChange({ current, pageSize }) {
    endpointPagination.current = current
    endpointPagination.pageSize = pageSize
}

/**
 * 保存端点配置
 */
async function handleSaveEndpoints() {
    try {
        configEndpointsSaving.value = true
        const result = await apis.tenant
            .saveTenantModelEndpoints({
                tenant_code: tenantData.value.code,
                model_id: currentConfigModel.value.id,
                endpoint_ids: selectedEndpointIds.value,
            })
            .catch(() => {
                throw new Error()
            })

        configEndpointsSaving.value = false
        if (config('http.code.success') === result?.success) {
            message.success(t('pages.tenant.detail.endpoint.save.success'))
            configEndpointsVisible.value = false
            await loadAuthorizedModels()
        }
    } catch (error) {
        configEndpointsSaving.value = false
        message.error(t('pages.tenant.detail.endpoint.save.failed'))
    }
}

/**
 * 打开添加模型弹窗
 */
function openAddModelModal() {
    selectedModelKeys.value = []
    modalSearchKey.value = ''
    availablePagination.current = 1
    addModalVisible.value = true
}

/**
 * 确认添加选中的模型
 */
async function handleConfirmAdd() {
    if (selectedModelKeys.value.length === 0) {
        message.warning(t('pages.tenant.detail.model.select_required'))
        return
    }
    try {
        addModalLoading.value = true
        // 将当前已授权 + 新选中的合并后批量保存
        const existingIds = allAuthorizedModels.value.map((m) => m.id)
        const mergedIds = [...new Set([...existingIds, ...selectedModelKeys.value])]

        const result = await apis.tenant
            .saveTenantModels({
                tenant_code: tenantData.value.code,
                model_ids: mergedIds,
            })
            .catch(() => {
                throw new Error()
            })

        addModalLoading.value = false
        if (config('http.code.success') === result?.success) {
            message.success(t('pages.tenant.detail.model.add.success', { count: selectedModelKeys.value.length }))
            addModalVisible.value = false
            await loadAuthorizedModels()
        }
    } catch (error) {
        addModalLoading.value = false
        message.error(t('pages.tenant.detail.model.add.failed'))
    }
}

/**
 * 移除单个已授权模型
 */
function handleRemoveModel(record) {
    Modal.confirm({
        title: t('pages.tenant.detail.model.remove.title'),
        content: t('pages.tenant.detail.model.remove.content', { name: record.model_name }),
        okText: t('pages.tenant.detail.model.remove.confirm'),
        okType: 'danger',
        cancelText: t('common.cancel'),
        onOk: () => {
            return new Promise((resolve, reject) => {
                ;(async () => {
                    try {
                        const remainingIds = allAuthorizedModels.value
                            .filter((m) => m.id !== record.id)
                            .map((m) => m.id)

                        const result = await apis.tenant
                            .saveTenantModels({
                                tenant_code: tenantData.value.code,
                                model_ids: remainingIds,
                            })
                            .catch(() => {
                                throw new Error()
                            })

                        if (config('http.code.success') === result?.success) {
                            resolve()
                            message.success(t('pages.tenant.detail.model.remove.success'))
                            await loadAuthorizedModels()
                        } else {
                            reject()
                        }
                    } catch (error) {
                        reject()
                    }
                })()
            })
        },
    })
}

/**
 * 复制租户编码
 */
function handleCopyCode(code) {
    if (navigator.clipboard) {
        navigator.clipboard
            .writeText(code)
            .then(() => message.success(t('pages.tenant.copy.success')))
            .catch(() => message.error(t('common.copy.failed')))
    } else {
        const input = document.createElement('input')
        input.setAttribute('value', code)
        document.body.appendChild(input)
        input.select()
        document.execCommand('copy')
        document.body.removeChild(input)
        message.success(t('pages.tenant.copy.success'))
    }
}

/**
 * 获取状态点样式
 */
function getPointStyle(point) {
    let color = '#f5f5f5'
    let border = '1px solid #d9d9d9'
    if (point.success_count > 0 && point.fail_count === 0) {
        color = '#52c41a'
        border = '1px solid #52c41a'
    } else if (point.success_count === 0 && point.fail_count > 0) {
        color = '#f5222d'
        border = '1px solid #f5222d'
    } else if (point.success_count > 0 && point.fail_count > 0) {
        color = '#fa8c16'
        border = '1px solid #fa8c16'
    }
    return {
        width: '10px',
        height: '10px',
        backgroundColor: color,
        border: border,
        borderRadius: '2px',
        cursor: 'pointer',
    }
}
</script>

<style lang="less" scoped>
.tenant-detail {
    padding: 0;
}

.info-card {
    margin-bottom: 16px;

    :deep(.ant-card-head-title) {
        font-size: 14px;
        font-weight: 600;
    }

    :deep(.ant-card-grid) {
        padding: 12px 16px;
    }

    .info-item {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 4px;

        .info-label {
            opacity: 0.55;
            font-size: 13px;
        }

        .info-value {
            font-size: 14px;
            font-weight: 600;
        }
    }

    .info-desc-item {
        display: flex;
        align-items: center;

        .info-label {
            opacity: 0.55;
            font-size: 13px;
            white-space: nowrap;
        }

        .info-desc-value {
            font-size: 13px;
        }
    }
}

.tenant-code-container {
    display: inline-flex;
    align-items: center;
    gap: 8px;

    .tenant-code {
        font-family: Menlo, Monaco, Consolas, monospace;
        font-size: 13px;
        background: var(--color-bg-active);
        padding: 1px 5px;
        border-radius: 4px;
        border: 1px solid var(--color-border);
        color: var(--color-text-primary);
    }

    .copy-btn-icon {
        color: var(--color-primary);
        cursor: pointer;
        font-size: 13px;
        transition:
            color 0.3s ease,
            transform 0.2s ease;

        &:hover {
            color: var(--color-primary-hover);
            transform: scale(1.15);
        }
        &:active {
            transform: scale(0.95);
        }
    }
}

.detail-card {
    .detail-tabs {
        margin-bottom: 0;

        :deep(.ant-tabs-nav) {
            margin-bottom: 16px;
        }
    }
}

.model-config-container {
    padding: 0;
}

.tab-toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
}

.tab-toolbar-right {
    display: flex;
    align-items: center;
    gap: 8px;
}

// 抽屉内工具栏
.add-modal-toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;

    .selected-count {
        font-size: 13px;
        color: var(--color-text-tertiary);

        strong {
            color: var(--color-primary);
            font-size: 14px;
        }
    }
}

// 抽屉底部按钮
.drawer-footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
}

// 端点抽屉工具栏
.endpoint-drawer-toolbar {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 16px;
    flex-wrap: wrap;

    .selected-count {
        margin-left: auto;
        font-size: 13px;
        color: var(--color-text-tertiary);

        strong {
            color: var(--color-primary);
            font-size: 14px;
        }
    }
}

// 表格优化
:deep(.ant-table) {
    .ant-table-tbody > tr > td {
        padding: 12px 16px;
    }

    .ant-table-thead > tr > th {
        padding: 12px 16px;
        font-weight: 600;
    }
}

// 操作按钮悬停优化
:deep(.x-action-button) {
    transition: all 0.2s ease;

    &:hover {
        background-color: var(--color-bg-hover);
        border-radius: 4px;
    }
}
</style>
