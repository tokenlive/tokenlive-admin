<template>
    <div class="app-page">
        <a-card
            type="flex"
            class="app-card">
            <a-row
                :gutter="16"
                align="middle"
                class="mb-8-2">
                <a-col flex="none">
                    <a-button
                        v-action="'add'"
                        type="primary"
                        ghost
                        @click="$refs.editDialogRef.handleCreate()">
                        <template #icon><plus-outlined /></template>
                        新增模型目录
                    </a-button>
                </a-col>
                <a-col flex="auto"></a-col>
                <a-col flex="none">
                    <a-form
                        :model="searchFormData"
                        layout="inline">
                        <a-form-item
                            label="Slug"
                            style="margin-bottom: 0">
                            <a-input
                                v-model:value="searchFormData.slug"
                                placeholder="搜索 Slug"
                                style="width: 180px"
                                @pressEnter="getPageList" />
                        </a-form-item>
                        <a-form-item
                            label="状态"
                            style="margin-bottom: 0">
                            <a-select
                                v-model:value="searchFormData.status"
                                placeholder="全部"
                                allow-clear
                                style="width: 120px"
                                @change="handleSearch">
                                <a-select-option value="available">可用</a-select-option>
                                <a-select-option value="paused">暂停</a-select-option>
                            </a-select>
                        </a-form-item>
                        <a-form-item
                            label="可见性"
                            style="margin-bottom: 0">
                            <a-select
                                v-model:value="searchFormData.visibility"
                                placeholder="全部"
                                allow-clear
                                style="width: 120px"
                                @change="handleSearch">
                                <a-select-option value="public">公开</a-select-option>
                                <a-select-option value="private">私有</a-select-option>
                            </a-select>
                        </a-form-item>
                        <a-form-item style="margin-bottom: 0">
                            <x-filter-actions
                                @reset="handleResetSearch"
                                @search="handleSearch" />
                        </a-form-item>
                    </a-form>
                </a-col>
            </a-row>
            <a-table
                :columns="columns"
                :data-source="listData"
                :loading="loading"
                :pagination="paginationState"
                :scroll="{ x: 'max-content' }"
                @change="onTableChange">
                <template #bodyCell="{ column, record }">
                    <template v-if="'model_id' === column.key">
                        <a @click="goToDetail(record)">{{ record.model_id }}</a>
                    </template>
                    <template v-if="'slug' === column.key">
                        <a-tag color="blue">{{ record.slug }}</a-tag>
                    </template>
                    <template v-if="'status' === column.key">
                        <a-tag :color="record.status === 'available' ? 'green' : 'default'">
                            {{ record.status === 'available' ? '可用' : '暂停' }}
                        </a-tag>
                    </template>
                    <template v-if="'visibility' === column.key">
                        <a-tag :color="record.visibility === 'public' ? 'cyan' : 'orange'">
                            {{ record.visibility === 'public' ? '公开' : '私有' }}
                        </a-tag>
                    </template>
                    <template v-if="'featured' === column.key">
                        <a-tag
                            v-if="record.featured"
                            color="gold"
                            >精选</a-tag
                        >
                        <span
                            v-else
                            style="color: #999"
                            >-</span
                        >
                    </template>
                    <template v-if="'context_length' === column.key">
                        {{ record.context_length ? record.context_length.toLocaleString() : '-' }}
                    </template>
                    <template v-if="'published_at' === column.key">
                        {{ record.published_at ? dayjs(record.published_at).format('YYYY-MM-DD HH:mm') : '-' }}
                    </template>
                    <template v-if="'action' === column.key">
                        <a-space>
                            <a-tooltip title="编辑">
                                <a-button
                                    v-action="'edit'"
                                    type="link"
                                    size="small"
                                    @click="$refs.editDialogRef.handleEdit(record)">
                                    <edit-outlined />
                                </a-button>
                            </a-tooltip>
                            <a-tooltip title="发布">
                                <a-button
                                    v-action="'edit'"
                                    type="link"
                                    size="small"
                                    @click="handlePublish(record)">
                                    <rocket-outlined />
                                </a-button>
                            </a-tooltip>
                            <a-popconfirm
                                title="确认删除？"
                                @confirm="handleDelete(record)">
                                <a-button
                                    v-action="'delete'"
                                    type="link"
                                    size="small"
                                    danger>
                                    <delete-outlined />
                                </a-button>
                            </a-popconfirm>
                        </a-space>
                    </template>
                </template>
            </a-table>
        </a-card>
        <ModelCatalogEditDialog
            ref="editDialogRef"
            @success="getPageList" />
    </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Modal, message } from 'ant-design-vue'
import dayjs from 'dayjs'
import { usePagination } from '@/hooks'
import { config } from '@/config'
import apis from '@/apis'
import ModelCatalogEditDialog from './ModelCatalogEditDialog.vue'
import { PlusOutlined, EditOutlined, RocketOutlined, DeleteOutlined } from '@ant-design/icons-vue'

const router = useRouter()
const { listData, loading, showLoading, hideLoading, paginationState, searchFormData, resetPagination } =
    usePagination()
searchFormData.value = { slug: '', status: undefined, visibility: undefined }

const columns = [
    { title: '模型ID', key: 'model_id', dataIndex: 'model_id', width: 160 },
    { title: 'Slug', key: 'slug', dataIndex: 'slug', width: 140 },
    { title: '关联编码', dataIndex: 'model_code', width: 120 },
    { title: '状态', key: 'status', dataIndex: 'status', width: 80 },
    { title: '可见性', key: 'visibility', dataIndex: 'visibility', width: 80 },
    { title: '精选', key: 'featured', dataIndex: 'featured', width: 70 },
    { title: '上下文长度', key: 'context_length', dataIndex: 'context_length', width: 120 },
    { title: '排序权重', dataIndex: 'sort_weight', width: 100 },
    { title: '发布时间', key: 'published_at', dataIndex: 'published_at', width: 160 },
    {
        title: '创建时间',
        dataIndex: 'created_at',
        width: 160,
        customRender: ({ text }) => (text ? dayjs(text).format('YYYY-MM-DD HH:mm') : '-'),
    },
    { title: '操作', key: 'action', width: 140, fixed: 'right' },
]

async function getPageList() {
    try {
        showLoading()
        const { pageSize, current } = paginationState
        const { success, data, total } = await apis.model_catalog.getModelCatalogList({
            pageSize,
            current,
            ...searchFormData.value,
        })
        hideLoading()
        if (config('http.code.success') === success) {
            listData.value = data
            paginationState.total = total
        }
    } catch (error) {
        hideLoading()
    }
}

function onTableChange(pagination) {
    paginationState.current = pagination.current
    paginationState.pageSize = pagination.pageSize
    getPageList()
}

function handleSearch() {
    resetPagination()
    getPageList()
}

function handleResetSearch() {
    searchFormData.value = { slug: '', status: undefined, visibility: undefined }
    resetPagination()
    getPageList()
}

function goToDetail(record) {
    router.push({ name: 'modelCatalogDetail', params: { id: record.model_id } })
}

async function handlePublish(record) {
    try {
        await apis.model_catalog.publishModelCatalog(record.model_id, { visibility: 'public' })
        message.success('发布成功')
        getPageList()
    } catch (e) {
        /* handled by interceptor */
    }
}

async function handleDelete(record) {
    Modal.confirm({
        title: '确认删除',
        content: `确定删除模型目录 "${record.slug}" 吗？`,
        okText: '确认',
        okType: 'danger',
        onOk: async () => {
            const { success } = await apis.model_catalog.delModelCatalog(record.model_id)
            if (config('http.code.success') === success) {
                message.success('删除成功')
                getPageList()
            }
        },
    })
}

onMounted(() => {
    getPageList()
})
</script>
