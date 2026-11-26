<template>
    <div>
        <a-alert
            v-if="syncFailed"
            class="dialect-alert"
            type="warning"
            show-icon
            :message="$t('pages.model.dialect.syncFailed')" />
        <a-alert
            v-if="loadFailed"
            class="dialect-alert"
            type="error"
            show-icon
            :message="$t('pages.model.dialect.loadFailed')" />
        <div class="dialect-toolbar">
            <a-button
                type="primary"
                ghost
                :disabled="loading || busy || loadFailed || !availableImplementations.length"
                @click="openCreate">
                <template #icon><plus-outlined /></template>
                {{ $t('button.createPolicy') }}
            </a-button>
            <a-button
                :aria-label="$t('pages.model.dialect.refresh')"
                :disabled="busy"
                :loading="loading"
                @click="load">
                <template #icon><reload-outlined /></template>
            </a-button>
        </div>
        <a-table
            row-key="id"
            :columns="columns"
            :data-source="items"
            :loading="loading"
            :pagination="false">
            <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'implementation'">
                    {{
                        implementations.find((item) => item.value === record.implementation)?.label ||
                        record.implementation
                    }}
                </template>
                <template v-else-if="column.key === 'enabled'">
                    <a-tag :color="record.enabled === 1 ? 'green' : 'default'">
                        {{
                            $t(
                                record.enabled === 1
                                    ? 'pages.model.form.enabled.active'
                                    : 'pages.model.form.enabled.inactive'
                            )
                        }}
                    </a-tag>
                </template>
                <template v-else-if="column.key === 'action'">
                    <x-action-button
                        role="button"
                        :aria-label="$t(record.enabled === 1 ? 'pages.endpoint.disable' : 'pages.endpoint.enable')"
                        :aria-disabled="actionsDisabled"
                        :tabindex="actionsDisabled ? -1 : 0"
                        @click="toggle(record)"
                        @keydown.enter.prevent="toggle(record)"
                        @keydown.space.prevent="toggle(record)">
                        <a-tooltip
                            :title="$t(record.enabled === 1 ? 'pages.endpoint.disable' : 'pages.endpoint.enable')">
                            <loading-outlined v-if="togglingId === record.id" />
                            <poweroff-outlined
                                v-else
                                :style="{ color: record.enabled === 1 ? '#faad14' : '#52c41a' }" />
                        </a-tooltip>
                    </x-action-button>
                    <x-action-button
                        role="button"
                        :aria-label="$t('button.delete')"
                        :aria-disabled="actionsDisabled"
                        :tabindex="actionsDisabled ? -1 : 0"
                        @click="confirmDelete(record.id)"
                        @keydown.enter.prevent="confirmDelete(record.id)"
                        @keydown.space.prevent="confirmDelete(record.id)">
                        <a-tooltip :title="$t('button.delete')">
                            <delete-outlined style="color: #ff4d4f" />
                        </a-tooltip>
                    </x-action-button>
                </template>
            </template>
        </a-table>

        <a-modal
            v-model:open="createOpen"
            :title="$t('button.createPolicy')"
            :confirm-loading="busy"
            :ok-button-props="{ disabled: !selectedImplementation }"
            @ok="create">
            <a-form layout="vertical">
                <a-form-item :label="$t('pages.model.dialect.implementation')">
                    <a-select
                        v-model:value="selectedImplementation"
                        :options="availableImplementations"
                        :placeholder="$t('pages.model.dialect.placeholder')"
                        :disabled="busy" />
                </a-form-item>
            </a-form>
        </a-modal>
    </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { DeleteOutlined, LoadingOutlined, PlusOutlined, PoweroffOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { Modal } from 'ant-design-vue'
import { createDialect, delDialect, getDialectList, toggleDialectEnabled } from '@/apis/modules/policy'

const props = defineProps({
    modelId: { type: String, required: true },
})

const { t } = useI18n()
const items = ref([])
const loading = ref(false)
const busy = ref(false)
const togglingId = ref(null)
const loadFailed = ref(false)
const syncFailed = ref(false)
const createOpen = ref(false)
const selectedImplementation = ref()
const actionsDisabled = computed(() => busy.value || loading.value || loadFailed.value)

const implementations = computed(() => [
    { value: 'chat_developer_to_system', label: t('pages.model.dialect.developer') },
    { value: 'chat_store_unsupported', label: t('pages.model.dialect.store') },
])
const availableImplementations = computed(() => {
    const configured = new Set(items.value.map((item) => item.implementation))
    return implementations.value.filter((item) => !configured.has(item.value))
})
const columns = computed(() => [
    { title: t('pages.model.dialect.implementation'), key: 'implementation' },
    { title: t('pages.model.form.enabled'), key: 'enabled', width: 120 },
    { title: t('button.action'), key: 'action', width: 120 },
])

async function load() {
    loading.value = true
    try {
        const result = await getDialectList({ model_id: props.modelId, current: 1, pageSize: 10 })
        loadFailed.value = !result.success
        if (result.success) items.value = result.data || []
    } catch {
        loadFailed.value = true
    } finally {
        loading.value = false
    }
}

function openCreate() {
    selectedImplementation.value = undefined
    createOpen.value = true
}

function create() {
    if (!selectedImplementation.value) return
    return mutate(() => createDialect({ model_id: props.modelId, implementation: selectedImplementation.value }))
}

async function toggle(record) {
    if (actionsDisabled.value) return
    togglingId.value = record.id
    try {
        await mutate(() => toggleDialectEnabled(record.id, { enabled: record.enabled === 1 ? 0 : 1 }))
    } finally {
        togglingId.value = null
    }
}

function confirmDelete(id) {
    if (actionsDisabled.value) return
    Modal.confirm({
        title: t('pages.model.dialect.deleteConfirm'),
        okText: t('button.confirm'),
        okType: 'danger',
        async onOk() {
            if (!(await mutate(() => delDialect(id)))) return Promise.reject()
        },
    })
}

async function mutate(action) {
    if (actionsDisabled.value) return false
    busy.value = true
    try {
        const { success, data } = await action()
        if (!success) return false
        if (data?.sync_status) syncFailed.value = data.sync_status === 'failed'
        createOpen.value = false
        await load()
        return true
    } catch {
        return false
    } finally {
        busy.value = false
    }
}

onMounted(load)
</script>

<style scoped>
.dialect-toolbar {
    display: flex;
    justify-content: space-between;
    margin-bottom: 12px;
}

.dialect-alert {
    margin-bottom: 12px;
}
</style>
