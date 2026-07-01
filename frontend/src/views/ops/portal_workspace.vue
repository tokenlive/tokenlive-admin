<template>
    <div class="app-page portal-workspace-page">
        <a-card
            class="app-card"
            :bordered="false">
            <div class="portal-workspace-toolbar">
                <a-form
                    layout="inline"
                    :model="formState"
                    class="portal-workspace-form">
                    <a-form-item
                        :label="$t('pages.portalWorkspace.workspace_id')"
                        style="margin-bottom: 0">
                        <a-input
                            v-model:value="formState.workspaceId"
                            allow-clear
                            class="portal-workspace-input"
                            :placeholder="$t('pages.portalWorkspace.workspace_id.placeholder')"
                            @pressEnter="handleSearch" />
                    </a-form-item>
                    <a-form-item style="margin-bottom: 0">
                        <a-button
                            type="primary"
                            :loading="loading"
                            @click="handleSearch">
                            <template #icon>
                                <search-outlined />
                            </template>
                            {{ $t('pages.portalWorkspace.search') }}
                        </a-button>
                    </a-form-item>
                    <a-form-item style="margin-bottom: 0">
                        <a-button
                            :disabled="!currentWorkspaceId"
                            :loading="syncing"
                            @click="handleSync">
                            <template #icon>
                                <sync-outlined />
                            </template>
                            {{ $t('pages.portalWorkspace.runtime_sync') }}
                        </a-button>
                    </a-form-item>
                </a-form>
            </div>

            <a-table
                row-key="id"
                :columns="columns"
                :data-source="listData"
                :loading="loading"
                :pagination="false"
                :scroll="{ x: 'max-content' }"
                :locale="{ emptyText }">
                <template #bodyCell="{ column, record }">
                    <template v-if="'name' === column.key">
                        <span class="portal-workspace-name">{{ record.name || '-' }}</span>
                    </template>
                    <template v-if="'secret_last4' === column.key">
                        <a-tag v-if="record.secret_last4">**** {{ record.secret_last4 }}</a-tag>
                        <span v-else>-</span>
                    </template>
                    <template v-if="'status' === column.key">
                        <a-tag :color="portalAPIKeyStatusColor(record.status)">
                            {{ statusText(record.status) }}
                        </a-tag>
                    </template>
                </template>
            </a-table>
        </a-card>
    </div>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { SearchOutlined, SyncOutlined } from '@ant-design/icons-vue'
import { useI18n } from 'vue-i18n'
import apis from '@/apis'
import { config } from '@/config'
import { normalizePortalAPIKeys, portalAPIKeyStatusColor } from './portal_workspace'

defineOptions({
    name: 'portalWorkspaceOps',
})

const { t } = useI18n()
const formState = reactive({ workspaceId: '' })
const currentWorkspaceId = ref('')
const listData = ref([])
const loading = ref(false)
const syncing = ref(false)

const columns = computed(() => [
    { title: t('pages.portalWorkspace.table.name'), key: 'name', dataIndex: 'name', width: 180 },
    { title: t('pages.portalWorkspace.table.prefix'), dataIndex: 'key_prefix', width: 140 },
    { title: t('pages.portalWorkspace.table.last4'), key: 'secret_last4', width: 120 },
    { title: t('pages.portalWorkspace.table.status'), key: 'status', width: 120 },
    { title: t('pages.portalWorkspace.table.expires_at'), dataIndex: 'expires_at', width: 180 },
    { title: t('pages.portalWorkspace.table.last_used_at'), dataIndex: 'last_used_at', width: 180 },
    { title: t('pages.portalWorkspace.table.created_at'), dataIndex: 'created_at', width: 180 },
    { title: t('pages.portalWorkspace.table.updated_at'), dataIndex: 'updated_at', width: 180 },
])

const emptyText = computed(() =>
    currentWorkspaceId.value ? t('pages.portalWorkspace.empty') : t('pages.portalWorkspace.empty.initial')
)

async function handleSearch() {
    const workspaceId = formState.workspaceId.trim()
    if (!workspaceId) {
        message.warning(t('pages.portalWorkspace.workspace_id.required'))
        return
    }

    loading.value = true
    try {
        const { success, data } = await apis.ops.getPortalWorkspaceAPIKeys(workspaceId)
        if (config('http.code.success') === success) {
            currentWorkspaceId.value = workspaceId
            listData.value = normalizePortalAPIKeys(data || [])
        }
    } finally {
        loading.value = false
    }
}

function handleSync() {
    if (!currentWorkspaceId.value) return

    Modal.confirm({
        title: t('pages.portalWorkspace.runtime_sync.confirm_title'),
        content: currentWorkspaceId.value,
        okText: t('pages.portalWorkspace.runtime_sync'),
        onOk: async () => {
            syncing.value = true
            try {
                const { success } = await apis.ops.syncPortalWorkspaceRuntime(currentWorkspaceId.value)
                if (config('http.code.success') === success) {
                    message.success(t('pages.portalWorkspace.runtime_sync.success'))
                    await handleSearch()
                }
            } finally {
                syncing.value = false
            }
        },
    })
}

function statusText(status) {
    return t(`pages.portalWorkspace.status.${status || 'unknown'}`)
}
</script>

<style lang="less" scoped>
.portal-workspace-page {
    .portal-workspace-toolbar {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 12px;
        margin-bottom: 16px;
    }

    .portal-workspace-form {
        display: flex;
        flex-wrap: wrap;
        gap: 12px 0;
        width: 100%;
    }

    .portal-workspace-input {
        width: min(420px, 100%);
    }

    .portal-workspace-name {
        font-weight: 600;
        color: @color-text-heading;
    }
}
</style>
