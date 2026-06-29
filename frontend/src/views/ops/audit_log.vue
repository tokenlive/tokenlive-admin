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
                    <span style="font-size: 16px; font-weight: 600">审计日志</span>
                </a-col>
                <a-col flex="auto"></a-col>
                <a-col flex="none">
                    <a-form
                        :model="searchFormData"
                        layout="inline">
                        <a-form-item
                            label="操作类型"
                            style="margin-bottom: 0">
                            <a-select
                                v-model:value="searchFormData.action"
                                placeholder="全部"
                                allow-clear
                                style="width: 120px"
                                @change="handleSearch">
                                <a-select-option value="create">创建</a-select-option>
                                <a-select-option value="update">更新</a-select-option>
                                <a-select-option value="delete">删除</a-select-option>
                                <a-select-option value="enable">启用</a-select-option>
                                <a-select-option value="disable">停用</a-select-option>
                                <a-select-option value="publish">发布</a-select-option>
                            </a-select>
                        </a-form-item>
                        <a-form-item
                            label="资源类型"
                            style="margin-bottom: 0">
                            <a-select
                                v-model:value="searchFormData.resource_type"
                                placeholder="全部"
                                allow-clear
                                style="width: 140px"
                                @change="handleSearch">
                                <a-select-option value="model">模型</a-select-option>
                                <a-select-option value="endpoint">端点</a-select-option>
                                <a-select-option value="provider">供应商</a-select-option>
                                <a-select-option value="model_catalog">模型目录</a-select-option>
                                <a-select-option value="price_version">价格版本</a-select-option>
                                <a-select-option value="policy">策略</a-select-option>
                                <a-select-option value="tenant">租户</a-select-option>
                                <a-select-option value="user">用户</a-select-option>
                            </a-select>
                        </a-form-item>
                        <a-form-item
                            label="租户"
                            style="margin-bottom: 0">
                            <a-input
                                v-model:value="searchFormData.tenant_code"
                                placeholder="租户编码"
                                style="width: 140px"
                                @pressEnter="handleSearch" />
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
                    <template v-if="'action' === column.key">
                        <a-tag :color="actionColor(record.action)">{{ record.action }}</a-tag>
                    </template>
                    <template v-if="'resource_type' === column.key">
                        <a-tag>{{ record.resource_type }}</a-tag>
                    </template>
                    <template v-if="'resource_name' === column.key">
                        {{ record.resource_name || record.resource_id }}
                    </template>
                    <template v-if="'before_data' === column.key">
                        <a-tooltip v-if="record.before_data">
                            <template #title>
                                <pre style="max-width: 400px; max-height: 300px; overflow: auto; font-size: 11px">{{
                                    formatJson(record.before_data)
                                }}</pre>
                            </template>
                            <a-button
                                type="link"
                                size="small"
                                >查看</a-button
                            >
                        </a-tooltip>
                        <span
                            v-else
                            style="color: #999"
                            >-</span
                        >
                    </template>
                    <template v-if="'after_data' === column.key">
                        <a-tooltip v-if="record.after_data">
                            <template #title>
                                <pre style="max-width: 400px; max-height: 300px; overflow: auto; font-size: 11px">{{
                                    formatJson(record.after_data)
                                }}</pre>
                            </template>
                            <a-button
                                type="link"
                                size="small"
                                >查看</a-button
                            >
                        </a-tooltip>
                        <span
                            v-else
                            style="color: #999"
                            >-</span
                        >
                    </template>
                    <template v-if="'created_at' === column.key">
                        {{ record.created_at ? dayjs(record.created_at).format('YYYY-MM-DD HH:mm:ss') : '-' }}
                    </template>
                </template>
            </a-table>
        </a-card>
    </div>
</template>

<script setup>
import { onMounted } from 'vue'
import dayjs from 'dayjs'
import { usePagination } from '@/hooks'
import { config } from '@/config'
import apis from '@/apis'

const { listData, loading, showLoading, hideLoading, paginationState, searchFormData, resetPagination } =
    usePagination()
searchFormData.value = { action: undefined, resource_type: undefined, tenant_code: '' }

const columns = [
    { title: '时间', key: 'created_at', dataIndex: 'created_at', width: 200 },
    { title: '操作人', dataIndex: 'actor_name', width: 140 },
    { title: '操作', key: 'action', dataIndex: 'action', width: 80 },
    { title: '资源类型', key: 'resource_type', dataIndex: 'resource_type', width: 100 },
    { title: '资源名称', key: 'resource_name', dataIndex: 'resource_name', width: 240 },
    { title: '资源ID', dataIndex: 'resource_id', width: 140, ellipsis: true },
    { title: '租户', dataIndex: 'tenant_code', width: 100 },
    { title: '变更前', key: 'before_data', width: 80 },
    { title: '变更后', key: 'after_data', width: 80 },
    { title: 'IP', dataIndex: 'ip', width: 140 },
    { title: '消息', dataIndex: 'message', ellipsis: true },
]

function actionColor(action) {
    const map = {
        create: 'green',
        update: 'blue',
        delete: 'red',
        enable: 'cyan',
        disable: 'orange',
        publish: 'purple',
        login: 'geekblue',
        logout: 'default',
    }
    return map[action] || 'default'
}

function formatJson(str) {
    if (!str) return ''
    try {
        return JSON.stringify(JSON.parse(str), null, 2)
    } catch {
        return str
    }
}

async function getPageList() {
    try {
        showLoading()
        const { pageSize, current } = paginationState
        const { success, data, total } = await apis.audit_log.getAuditLogList({
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
    searchFormData.value = { action: undefined, resource_type: undefined, tenant_code: '' }
    resetPagination()
    getPageList()
}

onMounted(() => {
    getPageList()
})
</script>
