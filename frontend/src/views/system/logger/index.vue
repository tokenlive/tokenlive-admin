<template>
    <!-- 数据表格卡片 -->
    <a-row
        :gutter="8"
        :wrap="false">
        <a-col flex="auto">
            <a-card type="flex">
                <!-- 头部：操作按钮 + 搜索栏 -->
                <a-row
                    :gutter="16"
                    align="middle"
                    class="mb-8-2">
                    <a-col flex="none">
                        <a-button
                            type="primary"
                            ghost
                            @click="handleSearch">
                            <template #icon>
                                <reload-outlined></reload-outlined>
                            </template>
                            刷新日志
                        </a-button>
                    </a-col>
                    <a-col flex="auto"></a-col>
                    <a-col flex="none">
                        <a-form
                            :model="searchFormData"
                            layout="inline">
                            <a-form-item
                                :label="$t('pages.system.logger.form.level')"
                                name="level"
                                style="margin-bottom: 0">
                                <a-select
                                    v-model:value="searchFormData.level"
                                    placeholder="请选择"
                                    style="width: 120px"
                                    allow-clear>
                                    <a-select-option value="info">INFO</a-select-option>
                                    <a-select-option value="warn">WARN</a-select-option>
                                    <a-select-option value="error">ERROR</a-select-option>
                                </a-select>
                            </a-form-item>
                            <a-form-item
                                :label="$t('pages.system.logger.form.trace_id')"
                                name="trace_id"
                                style="margin-bottom: 0">
                                <a-input
                                    placeholder="输入 Trace ID"
                                    v-model:value="searchFormData.trace_id"
                                    style="width: 180px"
                                    @pressEnter="handleSearch"
                                    allow-clear></a-input>
                            </a-form-item>
                            <a-form-item style="margin-bottom: 0">
                                <a-space :size="8">
                                    <a-tooltip :title="$t('button.reset')">
                                        <a-button
                                            shape="circle"
                                            @click="handleResetForm">
                                            <template #icon>
                                                <redo-outlined />
                                            </template>
                                        </a-button>
                                    </a-tooltip>
                                    <a-tooltip :title="$t('button.search')">
                                        <a-button
                                            type="primary"
                                            ghost
                                            shape="circle"
                                            @click="handleSearch">
                                            <template #icon>
                                                <search-outlined />
                                            </template>
                                        </a-button>
                                    </a-tooltip>
                                </a-space>
                            </a-form-item>
                        </a-form>
                    </a-col>
                </a-row>

                <!-- 表格 -->
                <a-table
                    row-key="id"
                    :columns="columns"
                    :data-source="listData"
                    :loading="loading"
                    :pagination="paginationState"
                    :size="size"
                    @change="onTableChange">
                    <template #expandedRowRender="{ record }">
                        <a-row :gutter="[16, 24]">
                            <a-col
                                v-for="(item, index) in record.children"
                                :key="index"
                                class="gutter-row"
                                :span="6">
                                <div class="gutter-box">{{ index }}：{{ item }}</div>
                            </a-col>
                        </a-row>
                    </template>
                    <template #bodyCell="{ column, record }">
                        <!-- 日志级别 -->
                        <template v-if="'levels' === column.key">
                            <a-tag :color="colors[record.level]">{{ record.level }}</a-tag>
                        </template>

                        <!-- 日志标签 -->
                        <template v-if="'tags' === column.key">
                            <a-tag color="processing">{{ record.tag }}</a-tag>
                        </template>

                        <!-- 创建时间 -->
                        <template v-if="'created_at' === column.key">
                            {{ formatUtcDateTime(record.created_at) }}
                        </template>
                    </template>
                </a-table>
            </a-card>
        </a-col>
    </a-row>
</template>

<script setup>
import { ref } from 'vue'
import { ReloadOutlined, SearchOutlined, RedoOutlined } from '@ant-design/icons-vue'
import apis from '@/apis'
import { config } from '@/config'
import { usePagination } from '@/hooks'
import { formatUtcDateTime } from '@/utils/util'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

defineOptions({
    name: 'Logger',
})

const colors = ref({
    info: 'success',
    warn: 'warning',
    error: 'error',
})

const columns = [
    { title: t('pages.system.logger.form.level'), dataIndex: 'level', key: 'levels', width: 100 },
    { title: t('pages.system.logger.form.trace_id'), dataIndex: 'trace_id', width: 220 },
    { title: t('pages.system.logger.form.user_name'), dataIndex: 'user_id', width: 140 },
    { title: t('pages.system.logger.form.tag'), dataIndex: 'tag', key: 'tags', width: 120 },
    { title: t('pages.system.logger.form.message'), dataIndex: 'message' },
    { title: t('pages.system.logger.form.created_at'), dataIndex: 'created_at', key: 'created_at', width: 180 },
]

const { listData, paginationState, loading, showLoading, hideLoading, resetPagination, searchFormData } =
    usePagination()

const size = ref('default')
const startTime = ref('')
const endTime = ref('')

getPageList()

/**
 * 获取分页列表
 */
async function getPageList() {
    try {
        showLoading()
        const { pageSize, current } = paginationState
        const { success, data, total } = await apis.system
            .getLoggers({
                pageSize,
                current,
                ...searchFormData.value,
                startTime: startTime.value,
                endTime: endTime.value,
            })
            .catch(() => {
                throw new Error()
            })
        hideLoading()
        if (config('http.code.success') === success) {
            data.forEach((item) => {
                item.children = JSON.parse(item.data)
            })
            listData.value = data
            paginationState.total = total
        }
    } catch (error) {
        hideLoading()
    }
}

/**
 * 搜索
 */
function handleSearch() {
    resetPagination()
    getPageList()
}

/**
 * 表格发生改变
 */
function onTableChange({ current, pageSize }) {
    paginationState.current = current
    paginationState.pageSize = pageSize
    getPageList()
}

/**
 * 重置
 */
function handleResetForm() {
    searchFormData.value = {}
    startTime.value = ''
    endTime.value = ''
    resetPagination()
    getPageList()
}
</script>

<style lang="less" scoped>
// 行内搜索表单紧凑间距
:deep(.ant-form-inline) {
    .ant-form-item {
        margin-right: 16px;

        &:last-child {
            margin-right: 0;
        }
    }
}

// 展开子区域美化
.gutter-box {
    background: #fafafa;
    padding: 8px 12px;
    border-radius: 4px;
    border: 1px solid #f0f0f0;
    font-size: 13px;
    color: #555;
    word-break: break-all;
}

// 表格单元格 - 优化间距
:deep(.ant-table) {
    .ant-table-tbody > tr > td {
        padding: 12px 16px;
    }

    .ant-table-thead > tr > th {
        padding: 12px 16px;
        font-weight: 600;
    }
}

// 头部间距
.mb-8-2 {
    padding-bottom: 16px;
    margin-bottom: 16px;
}
</style>
