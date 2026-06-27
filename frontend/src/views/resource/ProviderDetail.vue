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
                    <a-button
                        type="primary"
                        ghost
                        @click="$refs.endpointEditRef.handleCreate()">
                        {{ $t('pages.endpoint.add') }}
                    </a-button>
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
                    @change="onEndpointTableChange">
                    <template #bodyCell="{ column, record }">
                        <template v-if="'model_id' === column.key">
                            {{ getModelName(record.model_id) }}
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
                            <div style="display: flex; gap: 2px; align-items: center">
                                <a-tooltip
                                    v-for="(point, index) in record.status_points || []"
                                    :key="index">
                                    <template #title>
                                        {{ point.start_time }} ~ {{ point.end_time }}<br />
                                        成功: {{ point.success_count }} | 失败: {{ point.fail_count }}
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
    PoweroffOutlined,
} from '@ant-design/icons-vue'
import apis from '@/apis'
import { config } from '@/config'
import { formatUtcDateTime } from '@/utils/util'
import { useI18n } from 'vue-i18n'
import EndpointEditDialog from './EndpointEditDialog.vue'
import ProviderEditDialog from './ProviderEditDialog.vue'
import ProviderMemberEditDialog from './ProviderMemberEditDialog.vue'

defineOptions({
    name: 'providerDetail',
})

const route = useRoute()
const { t } = useI18n()
const providerId = ref(route.params.id)
const providerData = ref({})
const activeTab = ref('endpoint')
const providerEditRef = ref(null)
const memberEditRef = ref(null)

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
        width: 200,
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

// 供应商编辑
function handleEditProvider() {
    providerEditRef.value.handleEdit(providerData.value)
}

async function loadProviderDetail() {
    try {
        const { data, success } = await apis.provider.getProvider(providerId.value)
        if (success) {
            providerData.value = data || {}
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
        if (point.success_count > point.fail_count) {
            color = '#fadb14'
            border = '1px solid #fadb14'
        } else {
            color = '#fa8c16'
            border = '1px solid #fa8c16'
        }
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
</style>
