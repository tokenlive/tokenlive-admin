<template>
    <div class="provider-detail">
        <!-- 基本信息 -->
        <a-card
            :title="$t('pages.provider.detail.basicInfo')"
            class="info-card"
            :bordered="false">
            <template #extra>
                <a-button
                    type="primary"
                    ghost
                    size="small"
                    @click="handleEditProvider">
                    <template #icon><edit-outlined /></template>
                    {{ $t('pages.provider.edit') }}
                </a-button>
            </template>
            <a-card-grid style="width: 25%; text-align: center">
                <div class="info-item">
                    <span class="info-label">{{ $t('pages.provider.form.name') }}</span>
                    <span class="info-value">{{ providerData.name || '--' }}</span>
                </div>
            </a-card-grid>
            <a-card-grid style="width: 25%; text-align: center">
                <div class="info-item">
                    <span class="info-label">{{ $t('pages.provider.form.code') }}</span>
                    <span class="info-value">{{ providerData.code || '--' }}</span>
                </div>
            </a-card-grid>
            <a-card-grid style="width: 25%; text-align: center">
                <div class="info-item">
                    <span class="info-label">{{ $t('pages.provider.form.protocol') }}</span>
                    <span class="info-value">
                        <a-tag
                            color="blue"
                            v-if="providerData.protocol"
                            >{{ providerData.protocol }}</a-tag
                        >
                        <span v-else>--</span>
                    </span>
                </div>
            </a-card-grid>
            <a-card-grid style="width: 25%; text-align: center">
                <div class="info-item">
                    <span class="info-label">{{ $t('pages.provider.form.enabled') }}</span>
                    <span class="info-value">
                        <a-tag :color="providerData.enabled === 1 ? 'green' : 'default'">
                            {{
                                providerData.enabled === 1
                                    ? $t('pages.provider.form.enabled.active')
                                    : $t('pages.provider.form.enabled.inactive')
                            }}
                        </a-tag>
                    </span>
                </div>
            </a-card-grid>
            <a-card-grid style="width: 25%; text-align: center">
                <div class="info-item">
                    <span class="info-label">{{ $t('pages.provider.form.creator') }}</span>
                    <span class="info-value">{{ providerData.creator || '--' }}</span>
                </div>
            </a-card-grid>
            <a-card-grid style="width: 50%; text-align: center">
                <div class="info-item">
                    <span class="info-label">{{ $t('pages.provider.form.url') }}</span>
                    <span class="info-value">{{ providerData.url || '--' }}</span>
                </div>
            </a-card-grid>
            <a-card-grid style="width: 25%; text-align: center">
                <div class="info-item">
                    <span class="info-label">{{ $t('pages.provider.form.description') }}</span>
                    <span class="info-value">{{ providerData.description || '--' }}</span>
                </div>
            </a-card-grid>
        </a-card>

        <!-- OAuth 用量 -->
        <a-card
            v-if="showQuotaCard"
            class="info-card quota-card"
            :bordered="false">
            <template #title>
                <span>{{ $t('pages.provider.detail.quota.title') }}</span>
                <a-tag
                    v-if="quotaData?.provider"
                    color="processing"
                    style="margin-left: 8px">
                    {{ quotaProviderLabel }}
                </a-tag>
            </template>
            <template #extra>
                <a-button
                    size="small"
                    :loading="quotaLoading"
                    @click="loadProviderQuota">
                    <template #icon><reload-outlined /></template>
                    {{ $t('pages.provider.detail.quota.refresh') }}
                </a-button>
            </template>

            <a-spin :spinning="quotaLoading">
                <div
                    v-if="quotaError"
                    class="quota-error">
                    {{ quotaError }}
                </div>
                <template v-else-if="quotaData">
                    <div
                        v-if="quotaPlanText || quotaRenewalText || quotaExtrasText"
                        class="quota-meta">
                        <div v-if="quotaPlanText">
                            <span class="quota-meta-label">{{ $t('pages.provider.detail.quota.plan') }}</span>
                            <span class="quota-meta-value">{{ quotaPlanText }}</span>
                        </div>
                        <div v-if="quotaRenewalText">
                            <span class="quota-meta-label">{{ $t('pages.provider.detail.quota.renewal') }}</span>
                            <span class="quota-meta-value">{{ quotaRenewalText }}</span>
                        </div>
                        <div v-if="quotaExtrasText">
                            <span class="quota-meta-label">{{ $t('pages.provider.detail.quota.extra') }}</span>
                            <span class="quota-meta-value">{{ quotaExtrasText }}</span>
                        </div>
                    </div>

                    <div
                        v-if="!(quotaData.windows && quotaData.windows.length)"
                        class="quota-empty">
                        {{ $t('pages.provider.detail.quota.empty') }}
                    </div>
                    <div
                        v-for="win in quotaData.windows || []"
                        :key="win.id"
                        class="quota-row">
                        <div class="quota-row-header">
                            <span class="quota-row-label">{{ win.label }}</span>
                            <div class="quota-row-meta">
                                <span v-if="win.remaining_percent != null">
                                    {{ Math.round(win.remaining_percent) }}%
                                </span>
                                <span
                                    v-if="win.amount_label"
                                    class="quota-amount">
                                    {{ win.amount_label }}
                                </span>
                                <span
                                    v-if="win.reset_label"
                                    class="quota-reset">
                                    {{ win.reset_label }}
                                </span>
                            </div>
                        </div>
                        <a-progress
                            :percent="quotaBarPercent(win)"
                            :show-info="false"
                            size="small"
                            :stroke-color="quotaBarColor(win)" />
                    </div>
                </template>
                <div
                    v-else
                    class="quota-empty">
                    {{ $t('pages.provider.detail.quota.placeholder') }}
                </div>
            </a-spin>
        </a-card>

        <!-- Tab 区域 -->
        <a-card
            class="detail-card"
            :bordered="false">
            <a-tabs
                v-model:activeKey="activeTab"
                class="detail-tabs">
                <a-tab-pane
                    key="endpoint"
                    :tab="$t('pages.provider.detail.tab.endpoint')" />
                <a-tab-pane
                    key="member"
                    :tab="$t('pages.provider.detail.tab.member')" />
            </a-tabs>

            <!-- 端点管理 Tab 内容 -->
            <div v-if="activeTab === 'endpoint'">
                <div class="tab-toolbar">
                    <a-space>
                        <a-button
                            type="primary"
                            ghost
                            @click="$refs.endpointEditRef.handleCreate()">
                            <template #icon><plus-outlined /></template>
                            {{ $t('pages.endpoint.add') }}
                        </a-button>
                        <a-button @click="handleFetchModels">
                            <template #icon><import-outlined /></template>
                            {{ $t('pages.provider.detail.importEndpoint') }}
                        </a-button>
                    </a-space>
                    <div class="tab-toolbar-right">
                        <a-button @click="loadEndpointList">
                            <template #icon><reload-outlined /></template>
                        </a-button>
                    </div>
                </div>
                <a-table
                    :columns="endpointColumns"
                    :data-source="endpointListData"
                    :loading="endpointLoading"
                    :pagination="endpointPagination"
                    :scroll="{ x: 1200 }"
                    @change="onEndpointTableChange">
                    <template #bodyCell="{ column, record }">
                        <template v-if="'model_id' === column.key">
                            <a @click="goToModelDetail(record.model_id)">
                                {{ getModelName(record.model_id) }}
                            </a>
                        </template>
                        <template v-if="'url' === column.key">
                            <a-tooltip :title="record.url">
                                <span class="url-text">{{ record.url }}</span>
                            </a-tooltip>
                        </template>
                        <template v-if="'protocol' === column.key">
                            <a-tag
                                v-if="record.protocol"
                                color="blue"
                                >{{ record.protocol }}</a-tag
                            >
                            <a-tag
                                v-else-if="providerData.protocol"
                                color="blue"
                                style="border-style: dashed"
                                >{{ providerData.protocol }}</a-tag
                            >
                            <span
                                v-else
                                style="color: #999"
                                >{{ $t('pages.endpoint.form.protocol.inherit') }}</span
                            >
                        </template>
                        <template v-if="'real_model' === column.key">
                            {{ record.real_model || '--' }}
                        </template>
                        <template v-if="'priority' === column.key">
                            {{ record.priority ?? 0 }}
                        </template>
                        <template v-if="'enabled' === column.key">
                            <a-tag :color="record.enabled === 1 ? 'green' : 'default'">
                                {{
                                    record.enabled === 1
                                        ? $t('pages.endpoint.form.enabled.active')
                                        : $t('pages.endpoint.form.enabled.inactive')
                                }}
                            </a-tag>
                        </template>
                        <template v-if="'recent_status' === column.key">
                            <EndpointStatusStrip :points="record.status_points || []" />
                        </template>
                        <template v-if="'created_at' === column.key">
                            {{ formatUtcDateTime(record.created_at) }}
                        </template>
                        <template v-if="'action' === column.key">
                            <x-action-button
                                :disabled="testingEndpoints[record.id]"
                                @click="handleTestEndpoint(record)">
                                <a-tooltip>
                                    <template #title> {{ $t('pages.endpoint.test') }}</template>
                                    <loading-outlined v-if="testingEndpoints[record.id]" />
                                    <api-outlined v-else />
                                </a-tooltip>
                            </x-action-button>
                            <x-action-button @click="handleToggleEndpointEnabled(record)">
                                <a-tooltip>
                                    <template #title>{{
                                        record.enabled === 1
                                            ? $t('pages.endpoint.disable')
                                            : $t('pages.endpoint.enable')
                                    }}</template>
                                    <poweroff-outlined :style="{ color: record.enabled === 1 ? '#faad14' : '#52c41a' }"
                                /></a-tooltip>
                            </x-action-button>
                            <x-action-button @click="$refs.endpointEditRef.handleEdit(record)">
                                <a-tooltip>
                                    <template #title> {{ $t('pages.endpoint.edit') }}</template>
                                    <edit-outlined />
                                </a-tooltip>
                            </x-action-button>
                            <x-action-button @click="$refs.endpointEditRef.handleCopy(record)">
                                <a-tooltip>
                                    <template #title> {{ $t('pages.endpoint.copy') }}</template>
                                    <copy-outlined />
                                </a-tooltip>
                            </x-action-button>
                            <x-action-button @click="handleRemoveEndpoint(record)">
                                <a-tooltip>
                                    <template #title> {{ $t('button.delete') }}</template>
                                    <delete-outlined style="color: #ff4d4f" />
                                </a-tooltip>
                            </x-action-button>
                        </template>
                    </template>
                </a-table>
            </div>

            <!-- 成员管理 Tab 内容 -->
            <div v-else-if="activeTab === 'member'">
                <div class="tab-toolbar">
                    <a-button
                        type="primary"
                        ghost
                        @click="$refs.memberEditRef.handleCreate()">
                        <template #icon><plus-outlined /></template>
                        {{ $t('pages.member.add') }}
                    </a-button>
                    <div class="tab-toolbar-right">
                        <a-input-search
                            v-model:value="memberSearchUser"
                            :placeholder="$t('pages.member.search.placeholder')"
                            style="width: 200px"
                            allow-clear
                            @search="loadMemberList"
                            @pressEnter="loadMemberList" />
                        <a-button @click="loadMemberList">
                            <template #icon><reload-outlined /></template>
                        </a-button>
                    </div>
                </div>
                <a-table
                    :columns="memberColumns"
                    :data-source="memberListData"
                    :loading="memberLoading"
                    :pagination="memberPagination"
                    @change="onMemberTableChange">
                    <template #bodyCell="{ column, record }">
                        <template v-if="'permission' === column.key">
                            <a-tag
                                v-if="hasPermission(record.permission, 1)"
                                color="green"
                                >{{ $t('pages.member.form.permission.read') }}</a-tag
                            >
                            <a-tag
                                v-if="hasPermission(record.permission, 2)"
                                color="blue"
                                >{{ $t('pages.member.form.permission.write') }}</a-tag
                            >
                            <a-tag
                                v-if="hasPermission(record.permission, 4)"
                                color="red"
                                >{{ $t('pages.member.form.permission.delete') }}</a-tag
                            >
                        </template>
                        <template v-if="'created_at' === column.key">
                            {{ formatUtcDateTime(record.created_at) }}
                        </template>
                        <template v-if="'action' === column.key">
                            <x-action-button @click="$refs.memberEditRef.handleEdit(record)">
                                <a-tooltip>
                                    <template #title> {{ $t('pages.member.edit') }}</template>
                                    <edit-outlined />
                                </a-tooltip>
                            </x-action-button>
                            <x-action-button @click="handleRemoveMember(record)">
                                <a-tooltip>
                                    <template #title> {{ $t('button.delete') }}</template>
                                    <delete-outlined style="color: #ff4d4f" />
                                </a-tooltip>
                            </x-action-button>
                        </template>
                    </template>
                </a-table>
            </div>
        </a-card>

        <!-- 端点编辑弹窗 -->
        <endpoint-edit-dialog
            ref="endpointEditRef"
            :provider-options="providerOptions"
            :model-options="modelOptions"
            :provider-id="providerId"
            @ok="loadEndpointList" />

        <!-- 供应商编辑弹窗 -->
        <provider-edit-dialog
            ref="providerEditRef"
            @ok="loadProviderDetail" />

        <!-- 成员编辑弹窗 -->
        <provider-member-edit-dialog
            ref="memberEditRef"
            :provider-id="providerId"
            @ok="loadMemberList" />

        <!-- 获取模型抽屉 -->
        <fetch-models-drawer
            ref="fetchModelsDrawerRef"
            @confirm="onFetchModelsConfirm"></fetch-models-drawer>

        <!-- 导入映射对话框 -->
        <import-mapping-dialog
            ref="importMappingDialogRef"
            @ok="onImportEndpointsOk"></import-mapping-dialog>
    </div>
</template>

<script setup>
import { ref, onMounted, reactive, h, watch, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message, Modal, Radio } from 'ant-design-vue'
import {
    ReloadOutlined,
    EditOutlined,
    CopyOutlined,
    DeleteOutlined,
    ApiOutlined,
    LoadingOutlined,
    PoweroffOutlined,
    ImportOutlined,
    PlusOutlined,
} from '@ant-design/icons-vue'
import apis from '@/apis'
import { config } from '@/config'
import { formatUtcDateTime } from '@/utils/util'
import { useI18n } from 'vue-i18n'
import EndpointEditDialog from './EndpointEditDialog.vue'
import EndpointStatusStrip from '@/components/EndpointStatusStrip.vue'
import ProviderEditDialog from './ProviderEditDialog.vue'
import ProviderMemberEditDialog from './ProviderMemberEditDialog.vue'
import FetchModelsDrawer from './ProviderFetchModelsDrawer.vue'
import ImportMappingDialog from './ProviderImportMappingDialog.vue'

defineOptions({
    name: 'providerDetail',
})

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const providerId = ref(route.params.id)
const providerData = ref({})
const activeTab = ref('endpoint')
const quotaLoading = ref(false)
const quotaError = ref('')
const quotaData = ref(null)
const providerEditRef = ref(null)
const memberEditRef = ref(null)
const fetchModelsDrawerRef = ref(null)
const importMappingDialogRef = ref(null)

const modelOptions = ref([])
const providerOptions = ref([])
const endpointListData = ref([])
const endpointLoading = ref(false)
const endpointPagination = reactive({
    current: 1,
    pageSize: 10,
    total: 0,
    showSizeChanger: true,
    showTotal: (total) => `共 ${total} 条`,
})

const endpointColumns = [
    {
        title: t('pages.endpoint.form.code'),
        dataIndex: 'code',
        ellipsis: true,
    },
    {
        title: t('pages.endpoint.form.model_id'),
        key: 'model_id',
        width: 180,
    },
    {
        title: t('pages.endpoint.form.protocol'),
        key: 'protocol',
        width: 120,
    },
    {
        title: t('pages.endpoint.form.url'),
        key: 'url',
        ellipsis: true,
    },
    {
        title: t('pages.endpoint.form.real_model'),
        key: 'real_model',
        width: 150,
    },
    {
        title: t('pages.endpoint.form.priority'),
        dataIndex: 'priority',
        key: 'priority',
        width: 90,
        sorter: (a, b) => (a.priority ?? 0) - (b.priority ?? 0),
    },
    {
        title: t('pages.endpoint.form.weight'),
        dataIndex: 'weight',
        width: 80,
    },
    {
        title: t('pages.endpoint.form.enabled'),
        key: 'enabled',
        width: 100,
    },
    {
        title: t('pages.endpoint.recent_status'),
        key: 'recent_status',
        width: 220,
    },
    {
        title: t('pages.endpoint.form.description'),
        dataIndex: 'description',
        ellipsis: true,
    },
    {
        title: t('pages.endpoint.form.created_at'),
        key: 'created_at',
        width: 180,
    },
    {
        title: t('button.action'),
        key: 'action',
        width: 220,
    },
]

const hasPermission = (permission, bit) => {
    return (Number(permission) & bit) === bit
}

// 成员管理
const memberSearchUser = ref('')
const memberListData = ref([])
const memberLoading = ref(false)
const memberPagination = reactive({
    current: 1,
    pageSize: 10,
    total: 0,
    showSizeChanger: true,
    showTotal: (total) => `共 ${total} 条`,
})

const memberColumns = [
    {
        title: t('pages.member.form.user'),
        dataIndex: 'user',
        width: 150,
    },
    {
        title: t('pages.member.form.tenant'),
        dataIndex: 'tenant',
        width: 150,
    },
    {
        title: t('pages.member.form.role'),
        dataIndex: 'role',
        width: 100,
    },
    {
        title: t('pages.member.form.permission'),
        key: 'permission',
        width: 200,
    },
    {
        title: t('pages.member.form.created_at'),
        key: 'created_at',
        width: 180,
    },
    {
        title: t('button.action'),
        key: 'action',
        width: 120,
    },
]

onMounted(() => {
    loadProviderDetail()
    loadModelOptions()
    loadProviderOptions()
    loadEndpointList()
    loadMemberList()
})

watch(
    () => route.params.id,
    (newId) => {
        if (newId && route.name === 'providerDetail') {
            providerId.value = newId
            // 重置状态以防残留旧数据
            providerData.value = {}
            endpointListData.value = []
            memberListData.value = []

            endpointPagination.current = 1
            memberPagination.current = 1

            // 重新加载
            loadProviderDetail()
            loadModelOptions()
            loadProviderOptions()
            loadEndpointList()
            loadMemberList()
        }
    }
)

// 供应商编辑
function handleEditProvider() {
    providerEditRef.value.handleEdit(providerData.value)
}

// 导入端点（获取上游模型列表）
function handleFetchModels() {
    fetchModelsDrawerRef.value.handleOpen(providerData.value)
}

async function onFetchModelsConfirm({ providerId, space_code, base_url, api_key, api_keys, models }) {
    if (!models || models.length === 0) return

    const provider = providerData.value
    const protocol = provider?.protocol || ''

    let keysToCreate = []

    if (Array.isArray(api_keys) && api_keys.length > 1) {
        const importMode = ref('all')
        try {
            await new Promise((resolve, reject) => {
                Modal.confirm({
                    title: t('pages.provider.fetchModels.api_keys_confirm_title', '检测到多个 API 密钥'),
                    width: 600,
                    okText: t('button.confirm', '确认'),
                    cancelText: t('button.cancel', '取消'),
                    content: () => {
                        return h('div', { style: { marginTop: '12px' } }, [
                            h(
                                'p',
                                { style: { marginBottom: '16px', color: 'var(--color-text-secondary)' } },
                                `当前供应商配置了 ${api_keys.length} 个 API 密钥。请选择端点（Endpoint）创建模式：`
                            ),
                            h(
                                Radio.Group,
                                {
                                    value: importMode.value,
                                    'onUpdate:value': (val) => {
                                        importMode.value = val
                                    },
                                },
                                [
                                    h(
                                        Radio,
                                        {
                                            value: 'current',
                                            style: { display: 'block', marginBottom: '8px' },
                                        },
                                        `仅为当前选择/输入的密钥创建端点（每个模型创建 1 个端点）`
                                    ),
                                    h(
                                        Radio,
                                        {
                                            value: 'all',
                                            style: { display: 'block' },
                                        },
                                        `为所有配置的密钥分别创建端点（每个模型创建 ${api_keys.length} 个端点）`
                                    ),
                                ]
                            ),
                        ])
                    },
                    onOk: () => {
                        if (importMode.value === 'current') {
                            keysToCreate = [api_key || '']
                        } else {
                            keysToCreate = api_keys
                        }
                        resolve()
                    },
                    onCancel: () => {
                        reject(new Error('USER_CANCEL'))
                    },
                })
            })
        } catch (err) {
            if (err?.message === 'USER_CANCEL') {
                return
            }
            throw err
        }
    } else {
        if (api_key && (!Array.isArray(api_keys) || !api_keys.includes(api_key))) {
            keysToCreate = [api_key]
        } else if (Array.isArray(api_keys) && api_keys.length > 0) {
            keysToCreate = api_keys
        } else {
            keysToCreate = [api_key || '']
        }
    }

    importMappingDialogRef.value.handleOpen({
        providerId,
        providerCode: provider?.code || providerId,
        space_code,
        base_url,
        keysToCreate,
        protocol,
        auth_type: provider?.auth_type || 'api_key',
        models,
    })
}

async function onImportEndpointsOk() {
    await loadEndpointList()
    await loadModelOptions()
}

const showQuotaCard = computed(() => {
    const p = providerData.value || {}
    if (p.auth_type !== 'oauth_token') return false
    const url = (p.url || '').toLowerCase()
    const endpoint = (p.oauth?.token_endpoint || '').toLowerCase()
    const desc = (p.api_keys?.[0]?.description || '').toLowerCase()
    return (
        url.includes('chatgpt.com') ||
        url.includes('x.ai') ||
        url.includes('grok.com') ||
        endpoint.includes('openai.com') ||
        endpoint.includes('x.ai') ||
        desc.includes('codex') ||
        desc.includes('x.ai')
    )
})

const quotaProviderLabel = computed(() => {
    const key = quotaData.value?.provider
    if (key === 'codex') return 'Codex'
    if (key === 'xai') return 'xAI'
    return key || ''
})

const quotaPlanText = computed(() => {
    const plan = (quotaData.value?.plan || '').toLowerCase()
    if (!plan) return ''
    const map = {
        pro: t('pages.provider.detail.quota.plan.pro'),
        plus: t('pages.provider.detail.quota.plan.plus'),
        free: t('pages.provider.detail.quota.plan.free'),
        team: t('pages.provider.detail.quota.plan.team'),
        supergrok: t('pages.provider.detail.quota.plan.supergrok'),
        supergrok_heavy: t('pages.provider.detail.quota.plan.supergrok_heavy'),
        paid: t('pages.provider.detail.quota.plan.paid'),
    }
    return map[plan] || plan
})

const quotaRenewalText = computed(() => {
    const fromQuota = (quotaData.value?.subscription_active_until || '').trim()
    if (fromQuota) return fromQuota
    return (providerData.value?.oauth?.subscription_active_until || '').trim()
})

const quotaExtrasText = computed(() => {
    const extras = quotaData.value?.extras || {}
    if (extras.mode === 'paid-health') {
        return t('pages.provider.detail.quota.paid_health')
    }
    if (extras.rate_limit_reset_credits_available != null) {
        return t('pages.provider.detail.quota.reset_credits', {
            count: extras.rate_limit_reset_credits_available,
        })
    }
    return ''
})

function quotaBarPercent(win) {
    if (win?.remaining_percent == null) return 0
    return Math.max(0, Math.min(100, Number(win.remaining_percent)))
}

function quotaBarColor(win) {
    const p = quotaBarPercent(win)
    if (p >= 70) return 'var(--color-success)'
    if (p >= 30) return 'var(--color-warning)'
    return 'var(--color-error)'
}

async function loadProviderQuota() {
    if (!showQuotaCard.value || !providerId.value) return
    quotaLoading.value = true
    quotaError.value = ''
    try {
        const res = await apis.provider.getProviderQuota(providerId.value).catch((e) => e?.response?.data || e)
        const success = res?.success === true || res?.success === config('http.code.success')
        if (success) {
            quotaData.value = res.data || null
            quotaError.value = ''
        } else {
            quotaData.value = null
            quotaError.value =
                res?.error?.detail ||
                res?.error?.message ||
                res?.msg ||
                res?.message ||
                t('pages.provider.detail.quota.load_failed')
        }
    } catch (e) {
        quotaData.value = null
        quotaError.value =
            e?.response?.data?.error?.detail || e?.message || t('pages.provider.detail.quota.load_failed')
    } finally {
        quotaLoading.value = false
    }
}

async function loadProviderDetail() {
    try {
        const { data, success } = await apis.provider.getProvider(providerId.value)
        if (success) {
            providerData.value = data || {}
            if (showQuotaCard.value) {
                loadProviderQuota()
            } else {
                quotaData.value = null
                quotaError.value = ''
            }
        }
    } catch (error) {
        message.error(t('pages.provider.detail.load.failed'))
    }
}

async function loadModelOptions() {
    try {
        const { data, success } = await apis.model.getModelList({ pageSize: 1000, current: 1 })
        if (config('http.code.success') === success) {
            modelOptions.value = data || []
        }
    } catch (error) {
        // ignore
    }
}

async function loadProviderOptions() {
    try {
        const { data, success } = await apis.provider.getProviderList({ pageSize: 1000, current: 1 })
        if (config('http.code.success') === success) {
            providerOptions.value = data || []
        }
    } catch (error) {
        // ignore
    }
}

function getModelName(id) {
    if (!id) return '--'
    const m = modelOptions.value.find((item) => item.id === id)
    return m ? m.model_name : id
}

function goToModelDetail(id) {
    if (!id) return
    router.push({ name: 'modelDetail', params: { id } })
}

async function loadEndpointList() {
    try {
        endpointLoading.value = true
        const { data, success, total } = await apis.endpoint.getEndpointsByProviderId(providerId.value).catch(() => {
            throw new Error()
        })
        endpointLoading.value = false
        if (config('http.code.success') === success) {
            endpointListData.value = data || []
            endpointPagination.total = total || (data ? data.length : 0)
        }
    } catch (error) {
        endpointLoading.value = false
    }
}

function onEndpointTableChange({ current, pageSize }) {
    endpointPagination.current = current
    endpointPagination.pageSize = pageSize
    loadEndpointList()
}

const testingEndpoints = ref({})

async function handleTestEndpoint(record) {
    if (testingEndpoints.value[record.id]) return
    testingEndpoints.value[record.id] = true
    try {
        const { data, success, message: errMessage } = await apis.endpoint.testEndpoint(record.id)
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
        testingEndpoints.value[record.id] = false
    }
}

const togglingEndpoints = ref({})

async function handleToggleEndpointEnabled(record) {
    if (togglingEndpoints.value[record.id]) return
    const nextEnabled = record.enabled === 1 ? 0 : 1
    togglingEndpoints.value[record.id] = true
    try {
        const { success } = await apis.endpoint.toggleEndpointEnabled(record.id, { enabled: nextEnabled }).catch(() => {
            throw new Error()
        })
        if (config('http.code.success') === success) {
            message.success(
                nextEnabled === 1 ? t('pages.endpoint.enable.success') : t('pages.endpoint.disable.success')
            )
            await loadEndpointList()
        }
    } catch (error) {
        // ignore, error already handled by interceptor
    } finally {
        togglingEndpoints.value[record.id] = false
    }
}

function handleRemoveEndpoint({ id }) {
    Modal.confirm({
        title: t('pages.endpoint.delTip'),
        okText: t('button.confirm'),
        okType: 'danger',
        onOk: () => {
            return new Promise((resolve, reject) => {
                ;(async () => {
                    try {
                        const { success } = await apis.endpoint.delEndpoint(id).catch(() => {
                            throw new Error()
                        })
                        if (config('http.code.success') === success) {
                            resolve()
                            message.success(t('component.message.success.delete'))
                            await loadEndpointList()
                        }
                    } catch (error) {
                        reject()
                    }
                })()
            })
        },
    })
}

async function loadMemberList() {
    try {
        memberLoading.value = true
        const { data, success, total } = await apis.data_permission
            .getDataPermissionList({
                pageSize: memberPagination.pageSize,
                current: memberPagination.current,
                type: 'provider',
                data_id: providerId.value,
                user: memberSearchUser.value || undefined,
            })
            .catch(() => {
                throw new Error()
            })
        memberLoading.value = false
        if (config('http.code.success') === success) {
            memberListData.value = data || []
            memberPagination.total = total || 0
        }
    } catch (error) {
        memberLoading.value = false
    }
}

function onMemberTableChange({ current, pageSize }) {
    memberPagination.current = current
    memberPagination.pageSize = pageSize
    loadMemberList()
}

function handleRemoveMember({ id }) {
    Modal.confirm({
        title: t('pages.member.delTip'),
        okText: t('button.confirm'),
        okType: 'danger',
        onOk: () => {
            return new Promise((resolve, reject) => {
                ;(async () => {
                    try {
                        const { success } = await apis.data_permission.delDataPermission(id).catch(() => {
                            throw new Error()
                        })
                        if (config('http.code.success') === success) {
                            resolve()
                            message.success(t('component.message.success.delete'))
                            await loadMemberList()
                        }
                    } catch (error) {
                        reject()
                    }
                })()
            })
        },
    })
}
</script>

<style lang="less" scoped>
@import '@/styles/variables.less';

.provider-detail {
    padding: 0;
}

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

.quota-card {
    .quota-meta {
        display: flex;
        flex-wrap: wrap;
        gap: 16px 24px;
        margin-bottom: 12px;
        color: var(--color-text-secondary);
        font-size: 13px;
    }

    .quota-meta-label {
        color: var(--color-text-tertiary);
        margin-right: 6px;
    }

    .quota-meta-value {
        color: var(--color-text-primary);
        font-weight: 500;
    }

    .quota-row {
        margin-bottom: 12px;
    }

    .quota-row-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 12px;
        margin-bottom: 4px;
    }

    .quota-row-label {
        color: var(--color-text-primary);
        font-size: 13px;
        font-weight: 500;
    }

    .quota-row-meta {
        display: flex;
        align-items: center;
        gap: 10px;
        color: var(--color-text-secondary);
        font-size: 12px;
        white-space: nowrap;
    }

    .quota-amount {
        font-variant-numeric: tabular-nums;
    }

    .quota-reset {
        color: var(--color-text-tertiary);
    }

    .quota-empty,
    .quota-error {
        color: var(--color-text-tertiary);
        font-size: 13px;
        padding: 8px 0;
    }

    .quota-error {
        color: var(--color-error);
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

.tab-toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;

    .tab-toolbar-right {
        display: flex;
        gap: 8px;
    }
}

.url-text {
    font-family: monospace;
    display: inline-block;
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    vertical-align: middle;
}

:deep(.ant-table-tbody) {
    a {
        color: @color-primary;
        transition: color 0.2s ease;
        &:hover {
            color: #0958d9;
        }
    }
}
</style>
