<template>
    <div class="model-detail">
        <!-- 基本信息 -->
        <a-card
            :title="$t('pages.model.detail.basicInfo')"
            class="info-card"
            :bordered="false">
            <a-card-grid style="width: 25%; text-align: center">
                <div class="info-item">
                    <span class="info-label">{{ $t('pages.model.form.model_name') }}</span>
                    <span class="info-value">{{ modelData.model_name || '--' }}</span>
                </div>
            </a-card-grid>
            <a-card-grid style="width: 25%; text-align: center">
                <div class="info-item">
                    <span class="info-label">{{ $t('pages.model.form.model_code') }}</span>
                    <span class="info-value">{{ modelData.model_code || '--' }}</span>
                </div>
            </a-card-grid>
            <a-card-grid style="width: 25%; text-align: center">
                <div class="info-item">
                    <span class="info-label">{{ $t('pages.model.form.space_code') }}</span>
                    <span class="info-value">{{ modelData.space_code || '--' }}</span>
                </div>
            </a-card-grid>
            <a-card-grid style="width: 25%; text-align: center">
                <div class="info-item">
                    <span class="info-label">{{ $t('pages.model.form.owner') }}</span>
                    <span class="info-value">{{ modelData.owner || '--' }}</span>
                </div>
            </a-card-grid>
            <a-card-grid style="width: 25%; text-align: center">
                <div class="info-item">
                    <span class="info-label">{{ $t('pages.model.form.enabled') }}</span>
                    <span class="info-value">
                        <a-tag :color="modelData.enabled === 1 ? 'green' : 'default'">
                            {{
                                modelData.enabled === 1
                                    ? $t('pages.model.form.enabled.active')
                                    : $t('pages.model.form.enabled.inactive')
                            }}
                        </a-tag>
                    </span>
                </div>
            </a-card-grid>
            <a-card-grid style="width: 25%; text-align: center">
                <div class="info-item">
                    <span class="info-label">{{ $t('pages.model.form.creator') }}</span>
                    <span class="info-value">{{ modelData.creator || '--' }}</span>
                </div>
            </a-card-grid>
            <a-card-grid style="width: 25%; text-align: center">
                <div class="info-item">
                    <span class="info-label">{{ $t('pages.model.form.description') }}</span>
                    <span class="info-value">{{ modelData.description || '--' }}</span>
                </div>
            </a-card-grid>
            <a-card-grid style="width: 25%; text-align: center">
                <div class="info-item">
                    <span class="info-label">{{ $t('pages.model.form.created_at') }}</span>
                    <span class="info-value">{{ formatUtcDateTime(modelData.created_at) || '--' }}</span>
                </div>
            </a-card-grid>
            <a-card-grid style="width: 25%; text-align: center">
                <div class="info-item">
                    <span class="info-label">{{ $t('pages.model.form.input_price') }}</span>
                    <span class="info-value">{{
                        modelData.input_price !== undefined ? modelData.input_price + ' 元/M' : '--'
                    }}</span>
                </div>
            </a-card-grid>
            <a-card-grid style="width: 25%; text-align: center">
                <div class="info-item">
                    <span class="info-label">{{ $t('pages.model.form.output_price') }}</span>
                    <span class="info-value">{{
                        modelData.output_price !== undefined ? modelData.output_price + ' 元/M' : '--'
                    }}</span>
                </div>
            </a-card-grid>
            <a-card-grid style="width: 25%; text-align: center">
                <div class="info-item">
                    <span class="info-label">{{ $t('pages.model.form.cached_price') }}</span>
                    <span class="info-value">{{
                        modelData.cached_price !== undefined ? modelData.cached_price + ' 元/M' : '--'
                    }}</span>
                </div>
            </a-card-grid>
            <a-card-grid style="width: 25%; text-align: center">
                <div class="info-item">
                    <span class="info-label">{{ $t('pages.model.form.cache_creation_price') }}</span>
                    <span class="info-value">{{
                        modelData.cache_creation_price !== undefined ? modelData.cache_creation_price + ' 元/M' : '--'
                    }}</span>
                </div>
            </a-card-grid>
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
                    :tab="$t('pages.model.detail.tab.endpoint')" />
                <a-tab-pane
                    key="alias"
                    :tab="$t('pages.model.detail.tab.alias')" />
                <a-tab-pane
                    key="policy"
                    :tab="$t('pages.model.detail.tab.policy')" />
                <a-tab-pane
                    key="member"
                    :tab="$t('pages.model.detail.tab.member')" />
            </a-tabs>

            <!-- 端点管理 Tab 内容 -->
            <div v-if="activeTab === 'endpoint'">
                <div class="tab-toolbar">
                    <a-button
                        type="primary"
                        ghost
                        @click="$refs.endpointEditRef.handleCreate()">
                        {{ $t('pages.endpoint.add') }}
                    </a-button>
                    <div class="tab-toolbar-right">
                        <a-select
                            v-model:value="endpointFilterProviderId"
                            :placeholder="$t('pages.endpoint.filter.provider')"
                            style="width: 200px"
                            allow-clear
                            show-search
                            :filter-option="filterProviderOption"
                            @change="handleEndpointFilterChange">
                            <a-select-option
                                v-for="p in providerOptions"
                                :key="p.id"
                                :value="p.id">
                                {{ p.name }}
                            </a-select-option>
                        </a-select>
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
                    @change="onEndpointTableChange">
                    <template #bodyCell="{ column, record }">
                        <template v-if="'provider_id' === column.key">
                            {{ getProviderName(record.provider_id) }}
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
                                v-else-if="getInheritedProtocol(record.provider_id)"
                                color="blue"
                                style="border-style: dashed"
                                >{{ getInheritedProtocol(record.provider_id) }}</a-tag
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
                            <div style="display: flex; gap: 2px; align-items: center">
                                <a-tooltip
                                    v-for="(point, index) in record.status_points || []"
                                    :key="index">
                                    <template #title>
                                        {{ point.start_time }} ~ {{ point.end_time }}<br />
                                        {{ $t('pages.dashboard.trends.success') }}: {{ point.success_count }} |
                                        {{ $t('pages.dashboard.trends.fail') }}: {{ point.fail_count }}
                                    </template>
                                    <div :style="getPointStyle(point)"></div>
                                </a-tooltip>
                            </div>
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
                            <x-action-button @click="$refs.endpointEditRef.handleCopy(record)">
                                <a-tooltip>
                                    <template #title> {{ $t('pages.endpoint.copy') }}</template>
                                    <copy-outlined />
                                </a-tooltip>
                            </x-action-button>
                            <x-action-button @click="$refs.endpointEditRef.handleEdit(record)">
                                <a-tooltip>
                                    <template #title> {{ $t('pages.endpoint.edit') }}</template>
                                    <edit-outlined />
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

            <!-- 模型别名 Tab 内容 -->
            <div v-else-if="activeTab === 'alias'">
                <div class="tab-toolbar">
                    <a-button
                        type="primary"
                        ghost
                        @click="$refs.aliasEditRef.handleCreate()">
                        {{ $t('pages.model.alias.create') }}
                    </a-button>
                    <div class="tab-toolbar-right">
                        <a-input-search
                            v-model:value="aliasSearchName"
                            :placeholder="$t('pages.model.alias.search.placeholder')"
                            style="width: 200px"
                            allow-clear
                            @search="loadAliasList"
                            @pressEnter="loadAliasList" />
                        <a-button @click="loadAliasList">
                            <template #icon><reload-outlined /></template>
                        </a-button>
                    </div>
                </div>
                <a-table
                    :columns="aliasColumns"
                    :data-source="aliasListData"
                    :loading="aliasLoading"
                    :pagination="aliasPagination"
                    @change="onAliasTableChange">
                    <template #bodyCell="{ column, record }">
                        <template v-if="'created_at' === column.key">
                            {{ formatUtcDateTime(record.created_at) }}
                        </template>
                        <template v-if="'action' === column.key">
                            <x-action-button @click="$refs.aliasEditRef.handleEdit(record)">
                                <a-tooltip>
                                    <template #title> {{ $t('pages.model.alias.edit') }}</template>
                                    <edit-outlined />
                                </a-tooltip>
                            </x-action-button>
                            <x-action-button @click="handleRemoveAlias(record)">
                                <a-tooltip>
                                    <template #title> {{ $t('button.delete') }}</template>
                                    <delete-outlined style="color: #ff4d4f" />
                                </a-tooltip>
                            </x-action-button>
                        </template>
                    </template>
                </a-table>
            </div>

            <!-- 治理策略 Tab 内容 -->
            <div v-else-if="activeTab === 'policy'">
                <div class="tab-toolbar">
                    <a-button
                        type="primary"
                        ghost
                        @click="$refs.policyBindRef.handleCreate()">
                        {{ $t('pages.model.policy.bind') }}
                    </a-button>
                    <div class="tab-toolbar-right">
                        <a-button @click="loadPolicyBindingList">
                            <template #icon><reload-outlined /></template>
                        </a-button>
                    </div>
                </div>
                <a-table
                    :columns="policyColumns"
                    :data-source="policyListData"
                    :loading="policyLoading"
                    :pagination="policyPagination"
                    @change="onPolicyTableChange">
                    <template #bodyCell="{ column, record }">
                        <template v-if="'policy_type' === column.key">
                            <a-tag :color="getPolicyTypeColor(record.policy_type)">
                                {{ getPolicyTypeName(record.policy_type) }}
                            </a-tag>
                        </template>
                        <template v-if="'policy_name' === column.key">
                            {{ getPolicyName(record.policy_type, record.policy_id) }}
                        </template>
                        <template v-if="'tenant_code' === column.key">
                            {{ record.tenant_code || $t('pages.model.policy.form.tenant_code.placeholder') }}
                        </template>
                        <template v-if="'user_id' === column.key">
                            {{ record.user_id || $t('pages.model.policy.form.user_id.placeholder') }}
                        </template>
                        <template v-if="'enabled' === column.key">
                            <a-tag :color="record.enabled === 1 ? 'green' : 'default'">
                                {{
                                    record.enabled === 1
                                        ? $t('pages.model.form.enabled.active')
                                        : $t('pages.model.form.enabled.inactive')
                                }}
                            </a-tag>
                        </template>
                        <template v-if="'created_at' === column.key">
                            {{ formatUtcDateTime(record.created_at) }}
                        </template>
                        <template v-if="'action' === column.key">
                            <x-action-button @click="$refs.policyBindRef.handleEdit(record)">
                                <a-tooltip>
                                    <template #title> {{ $t('pages.endpoint.edit') }}</template>
                                    <edit-outlined />
                                </a-tooltip>
                            </x-action-button>
                            <x-action-button @click="handleRemovePolicy(record)">
                                <a-tooltip>
                                    <template #title> {{ $t('pages.model.policy.unbind') }}</template>
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

        <!-- 别名编辑弹窗 -->
        <model-alias-edit-dialog
            ref="aliasEditRef"
            :model-id="modelId"
            :default-space-code="modelData.space_code"
            @ok="loadAliasList" />

        <!-- 成员编辑弹窗 -->
        <model-member-edit-dialog
            ref="memberEditRef"
            :model-id="modelId"
            @ok="loadMemberList" />

        <!-- 端点编辑弹窗 -->
        <endpoint-edit-dialog
            ref="endpointEditRef"
            :provider-options="providerOptions"
            :model-options="modelOptions"
            :model-id="modelId"
            @ok="loadEndpointList" />

        <!-- 策略绑定编辑弹窗 -->
        <model-policy-bind-dialog
            v-if="modelData.model_code"
            ref="policyBindRef"
            :model-code="modelData.model_code"
            @ok="loadPolicyBindingList" />
    </div>
</template>

<script setup>
import { ref, onMounted, reactive } from 'vue'
import { useRoute } from 'vue-router'
import { message, Modal } from 'ant-design-vue'
import {
    ReloadOutlined,
    EditOutlined,
    DeleteOutlined,
    ApiOutlined,
    LoadingOutlined,
    CopyOutlined,
} from '@ant-design/icons-vue'
import apis from '@/apis'
import { config } from '@/config'
import { formatUtcDateTime } from '@/utils/util'
import { useI18n } from 'vue-i18n'
import ModelAliasEditDialog from './ModelAliasEditDialog.vue'
import ModelMemberEditDialog from './ModelMemberEditDialog.vue'
import EndpointEditDialog from './EndpointEditDialog.vue'
import ModelPolicyBindDialog from './ModelPolicyBindDialog.vue'

defineOptions({
    name: 'modelDetail',
})

const route = useRoute()
const { t } = useI18n()
const modelId = ref(route.params.id)
const modelData = ref({})
const activeTab = ref('endpoint')

const hasPermission = (permission, bit) => {
    return (Number(permission) & bit) === bit
}

// 模型别名
const aliasSearchName = ref('')
const aliasListData = ref([])
const aliasLoading = ref(false)
const aliasPagination = reactive({
    current: 1,
    pageSize: 10,
    total: 0,
    showSizeChanger: true,
    showTotal: (total) => `共 ${total} 条`,
})

const aliasColumns = [
    {
        title: t('pages.model.alias.form.alias'),
        dataIndex: 'alias',
        width: 200,
    },
    {
        title: t('pages.model.form.description'),
        dataIndex: 'description',
        ellipsis: true,
    },
    {
        title: t('pages.model.form.created_at'),
        key: 'created_at',
        width: 180,
    },
    {
        title: t('button.action'),
        key: 'action',
        width: 120,
    },
]

// 治理策略关联全量缓存用于映射
const allPoliciesMap = ref({
    tagging: [],
    loadbalance: [],
    route: [],
    limit: [],
    circuit_break: [],
    invocation: [],
})

// 治理策略绑定
const policyListData = ref([])
const policyLoading = ref(false)
const policyPagination = reactive({
    current: 1,
    pageSize: 10,
    total: 0,
    showSizeChanger: true,
    showTotal: (total) => `共 ${total} 条`,
})

const policyColumns = [
    {
        title: t('pages.model.policy.form.policy_type'),
        key: 'policy_type',
        width: 130,
    },
    {
        title: t('pages.model.policy.form.policy_id'),
        key: 'policy_name',
    },
    {
        title: t('pages.model.policy.form.tenant_code'),
        key: 'tenant_code',
        width: 160,
    },
    {
        title: t('pages.model.policy.form.user_id'),
        key: 'user_id',
        width: 160,
    },
    {
        title: t('pages.model.policy.form.priority'),
        dataIndex: 'priority',
        width: 90,
    },
    {
        title: t('pages.model.form.enabled'),
        key: 'enabled',
        width: 100,
    },
    {
        title: t('pages.model.form.created_at'),
        key: 'created_at',
        width: 180,
    },
    {
        title: t('button.action'),
        key: 'action',
        width: 120,
    },
]

// 模型选项（用于 endpoint 表单）
const modelOptions = ref([])

// 端点管理
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
        title: t('pages.endpoint.form.provider_id'),
        key: 'provider_id',
        width: 150,
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
        title: t('pages.endpoint.recent_status'),
        key: 'recent_status',
        width: 180,
    },
    {
        title: t('pages.endpoint.form.weight'),
        dataIndex: 'weight',
        width: 80,
    },
    {
        title: t('pages.endpoint.form.priority'),
        key: 'priority',
        width: 80,
    },
    {
        title: t('pages.endpoint.form.enabled'),
        key: 'enabled',
        width: 80,
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
        width: 190,
    },
]

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
    loadModelDetail()
    loadAliasList()
    loadModelOptions()
    loadProviderOptions()
    loadEndpointList()
    loadMemberList()
    loadAllPoliciesForMapping()
})

const endpointFilterProviderId = ref(undefined)

function handleEndpointFilterChange() {
    endpointPagination.current = 1
    loadEndpointList()
}

function filterProviderOption(input, option) {
    return option.children?.[0]?.children?.toLowerCase().includes(input.toLowerCase())
}

function getProviderName(id) {
    if (!id) return '--'
    const p = providerOptions.value.find((item) => item.id === id)
    return p ? p.name : id
}

function getInheritedProtocol(providerId) {
    if (!providerId) return ''
    const p = providerOptions.value.find((item) => item.id === providerId)
    return p ? p.protocol : ''
}

async function loadModelDetail() {
    try {
        const { data, success } = await apis.model.getModel(modelId.value)
        if (success) {
            modelData.value = data || {}
            loadPolicyBindingList()
        }
    } catch (error) {
        // ignore
    }
}

async function loadAllPoliciesForMapping() {
    try {
        const params = { pageSize: 1000, current: 1 }
        const [tg, lb, rt, lim, cb, iv] = await Promise.allSettled([
            apis.policy.getTaggingList(params),
            apis.policy.getLoadbalanceList(params),
            apis.policy.getRouteList(params),
            apis.policy.getLimitList(params),
            apis.policy.getCircuitBreakList(params),
            apis.policy.getInvocationList(params),
        ])
        if (tg.status === 'fulfilled' && tg.value?.success) allPoliciesMap.value.tagging = tg.value.data || []
        if (lb.status === 'fulfilled' && lb.value?.success) allPoliciesMap.value.loadbalance = lb.value.data || []
        if (rt.status === 'fulfilled' && rt.value?.success) allPoliciesMap.value.route = rt.value.data || []
        if (lim.status === 'fulfilled' && lim.value?.success) allPoliciesMap.value.limit = lim.value.data || []
        if (cb.status === 'fulfilled' && cb.value?.success) allPoliciesMap.value.circuit_break = cb.value.data || []
        if (iv.status === 'fulfilled' && iv.value?.success) allPoliciesMap.value.invocation = iv.value.data || []
    } catch (e) {
        // ignore
    }
}

function getPolicyName(policyType, policyId) {
    if (!policyType || !policyId) return '--'
    let typeKey = policyType
    if (policyType === 'load_balance') typeKey = 'loadbalance'
    if (policyType === 'invoke') typeKey = 'invocation'
    const list = allPoliciesMap.value[typeKey] || []
    const item = list.find((p) => p.id === policyId)
    return item ? item.name : policyId
}

async function loadPolicyBindingList() {
    if (!modelData.value.model_code) {
        return
    }
    try {
        policyLoading.value = true
        const { data, success, total } = await apis.policy
            .getPolicyBindingList({
                pageSize: policyPagination.pageSize,
                current: policyPagination.current,
                model_code: modelData.value.model_code,
            })
            .catch(() => {
                throw new Error()
            })
        policyLoading.value = false
        if (config('http.code.success') === success) {
            policyListData.value = data || []
            policyPagination.total = total || 0
        }
    } catch (error) {
        policyLoading.value = false
    }
}

function onPolicyTableChange({ current, pageSize }) {
    policyPagination.current = current
    policyPagination.pageSize = pageSize
    loadPolicyBindingList()
}

function getPolicyTypeName(type) {
    switch (type) {
        case 'tagging':
            return t('pages.dashboard.policies.tagging')
        case 'limit':
            return t('pages.dashboard.policies.limit')
        case 'invocation':
        case 'invoke':
            return t('pages.dashboard.policies.invocation')
        case 'route':
            return t('pages.dashboard.policies.route')
        case 'loadbalance':
        case 'load_balance':
            return t('pages.dashboard.policies.loadbalance')
        case 'circuit_break':
            return t('pages.dashboard.policies.circuitBreak')
        default:
            return type
    }
}

function getPolicyTypeColor(type) {
    switch (type) {
        case 'tagging':
            return 'green'
        case 'limit':
            return 'orange'
        case 'invocation':
        case 'invoke':
            return 'cyan'
        case 'route':
            return 'purple'
        case 'loadbalance':
        case 'load_balance':
            return 'blue'
        case 'circuit_break':
            return 'red'
        default:
            return 'default'
    }
}

function handleRemovePolicy({ id }) {
    Modal.confirm({
        title: t('pages.model.policy.unbindTip'),
        okText: t('button.confirm'),
        okType: 'danger',
        onOk: () => {
            return new Promise((resolve, reject) => {
                ;(async () => {
                    try {
                        const { success } = await apis.policy.delPolicyBinding(id).catch(() => {
                            throw new Error()
                        })
                        if (config('http.code.success') === success) {
                            resolve()
                            message.success(t('component.message.success.delete'))
                            await loadPolicyBindingList()
                        }
                    } catch (error) {
                        reject()
                    }
                })()
            })
        },
    })
}

async function loadAliasList() {
    try {
        aliasLoading.value = true
        const { data, success, total } = await apis.model_alias
            .getModelAliasList({
                pageSize: aliasPagination.pageSize,
                current: aliasPagination.current,
                model_id: modelId.value,
                alias: aliasSearchName.value || undefined,
            })
            .catch(() => {
                throw new Error()
            })
        aliasLoading.value = false
        if (config('http.code.success') === success) {
            aliasListData.value = data || []
            aliasPagination.total = total || 0
        }
    } catch (error) {
        aliasLoading.value = false
    }
}

function onAliasTableChange({ current, pageSize }) {
    aliasPagination.current = current
    aliasPagination.pageSize = pageSize
    loadAliasList()
}

function handleRemoveAlias({ id }) {
    Modal.confirm({
        title: t('pages.model.alias.delTip'),
        okText: t('button.confirm'),
        okType: 'danger',
        onOk: () => {
            return new Promise((resolve, reject) => {
                ;(async () => {
                    try {
                        const { success } = await apis.model_alias.delModelAlias(id).catch(() => {
                            throw new Error()
                        })
                        if (config('http.code.success') === success) {
                            resolve()
                            message.success(t('component.message.success.delete'))
                            await loadAliasList()
                        }
                    } catch (error) {
                        reject()
                    }
                })()
            })
        },
    })
}

async function loadProviderOptions() {
    try {
        const { data, success } = await apis.provider
            .getProviderList({
                pageSize: 100,
                current: 1,
            })
            .catch(() => {
                throw new Error()
            })
        if (config('http.code.success') === success) {
            providerOptions.value = data || []
        }
    } catch (error) {
        // ignore
    }
}

async function loadModelOptions() {
    try {
        const { data, success } = await apis.model
            .getModelList({
                pageSize: 100,
                current: 1,
            })
            .catch(() => {
                throw new Error()
            })
        if (config('http.code.success') === success) {
            modelOptions.value = data || []
        }
    } catch (error) {
        // ignore
    }
}

async function loadEndpointList() {
    try {
        endpointLoading.value = true
        const { data, success, total } = await apis.endpoint
            .getEndpointList({
                pageSize: endpointPagination.pageSize,
                current: endpointPagination.current,
                model_id: modelId.value,
                provider_id: endpointFilterProviderId.value || undefined,
            })
            .catch(() => {
                throw new Error()
            })
        endpointLoading.value = false
        if (config('http.code.success') === success) {
            endpointListData.value = data || []
            endpointPagination.total = total || 0
        }
    } catch (error) {
        endpointLoading.value = false
    }
}

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
        width: '12px',
        height: '12px',
        backgroundColor: color,
        border: border,
        borderRadius: '2px',
        cursor: 'pointer',
    }
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

function onEndpointTableChange({ current, pageSize }) {
    endpointPagination.current = current
    endpointPagination.pageSize = pageSize
    loadEndpointList()
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
                type: 'model',
                data_id: modelId.value,
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
.model-detail {
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
}

.tab-toolbar-left {
    display: flex;
    align-items: center;
    gap: 8px;
}

.tab-toolbar-right {
    display: flex;
    align-items: center;
    gap: 8px;
}

.url-text {
    display: inline-block;
    max-width: 300px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}
</style>
