<template>
    <a-modal
        :open="modal.open"
        :title="modal.title"
        :width="600"
        :confirm-loading="modal.confirmLoading"
        :after-close="onAfterClose"
        :cancel-text="cancelText"
        :ok-text="okText"
        @ok="handleOk"
        @cancel="handleCancel">
        <a-form
            ref="formRef"
            :model="formData"
            :rules="formRules"
            layout="vertical">
            <a-row :gutter="16">
                <!-- 关联用户 -->
                <a-col :span="24">
                    <a-form-item
                        :label="$t('pages.user-api-key.form.user')"
                        name="user_id">
                        <a-select
                            v-model:value="formData.user_id"
                            show-search
                            :disabled="modal.type === 'edit'"
                            :placeholder="$t('pages.user-api-key.form.user.placeholder')"
                            :filter-option="filterUserOption"
                            style="width: 100%">
                            <a-select-option
                                v-for="item in userOptions"
                                :key="item.value"
                                :value="item.value"
                                :label="item.label">
                                {{ item.label }}
                            </a-select-option>
                        </a-select>
                    </a-form-item>
                </a-col>
            </a-row>

            <a-row :gutter="16">
                <!-- 密钥名称 -->
                <a-col :span="12">
                    <a-form-item
                        :label="$t('pages.user-api-key.form.name')"
                        name="name">
                        <a-input
                            :placeholder="$t('pages.user-api-key.form.name.placeholder')"
                            v-model:value="formData.name"></a-input>
                    </a-form-item>
                </a-col>
                <!-- 状态 -->
                <a-col :span="12">
                    <a-form-item
                        :label="$t('pages.user-api-key.form.status')"
                        name="status">
                        <a-radio-group
                            v-model:value="formData.status"
                            button-style="solid">
                            <a-radio-button :value="1">{{
                                $t('pages.user-api-key.form.status.enabled')
                            }}</a-radio-button>
                            <a-radio-button :value="2">{{
                                $t('pages.user-api-key.form.status.disabled')
                            }}</a-radio-button>
                        </a-radio-group>
                    </a-form-item>
                </a-col>
            </a-row>

            <a-row :gutter="16">
                <!-- 剩余配额 -->
                <a-col :span="12">
                    <a-form-item
                        :label="$t('pages.user-api-key.form.quota')"
                        name="quota">
                        <a-input-number
                            v-model:value="formData.quota"
                            :min="-1"
                            style="width: 100%"
                            :placeholder="$t('pages.user-api-key.form.quota.placeholder')"></a-input-number>
                    </a-form-item>
                </a-col>
                <!-- 过期时间 -->
                <a-col :span="12">
                    <a-form-item
                        :label="$t('pages.user-api-key.form.expires_at')"
                        name="expires_at">
                        <a-date-picker
                            v-model:value="expiresAtDayjs"
                            show-time
                            :placeholder="$t('pages.user-api-key.form.expires_at.placeholder')"
                            style="width: 100%"></a-date-picker>
                    </a-form-item>
                </a-col>
            </a-row>

            <a-row :gutter="16">
                <!-- 备注描述 -->
                <a-col :span="24">
                    <a-form-item
                        :label="$t('pages.user-api-key.form.description')"
                        name="description">
                        <a-textarea
                            :placeholder="$t('pages.user-api-key.form.description.placeholder')"
                            v-model:value="formData.description"
                            :rows="3"></a-textarea>
                    </a-form-item>
                </a-col>
            </a-row>
        </a-form>
    </a-modal>

    <!-- 密钥创建成功展示模态框（仅展示一次） -->
    <a-modal
        v-model:open="successModal.open"
        :title="$t('pages.user-api-key.created.title')"
        :footer="null"
        :width="520"
        :mask-closable="false"
        :closable="true">
        <div class="api-key-success-container">
            <a-alert
                :message="$t('pages.user-api-key.created.warning')"
                :description="$t('pages.user-api-key.created.description')"
                type="warning"
                show-icon
                class="mb-4" />

            <div class="key-box">
                <span class="key-text">{{ createdKeyPlaintext }}</span>
                <a-button
                    type="primary"
                    size="small"
                    @click="handleCopy">
                    {{ $t('pages.user-api-key.copy.action') }}
                </a-button>
            </div>

            <div class="mt-4 align-right">
                <a-button
                    type="default"
                    @click="successModal.open = false"
                    >{{ $t('pages.user-api-key.created.saved') }}</a-button
                >
            </div>
        </div>
    </a-modal>
</template>

<script setup>
import { computed, ref, watch, watchEffect } from 'vue'
import { cloneDeep } from 'lodash-es'
import dayjs from 'dayjs'
import { useI18n } from 'vue-i18n'
import { config } from '@/config'
import apis from '@/apis'
import { useForm, useModal } from '@/hooks'
import { message } from 'ant-design-vue'

const emit = defineEmits(['ok'])
const { t } = useI18n()

const { modal, showModal, hideModal, showLoading, hideLoading } = useModal()
const { formRecord, formData, formRef, formRules, resetForm } = useForm()

const cancelText = computed(() => t('common.cancel'))
const okText = computed(() => t('common.confirm'))

const userOptions = ref([])
const expiresAtDayjs = ref(null)

// 成功创建的明文 Key 展示控制
const successModal = ref({ open: false })
const createdKeyPlaintext = ref('')

// 表单验证规则
watchEffect(() => {
    formRules.value = {
        user_id: { required: true, message: t('pages.user-api-key.form.user.required') },
        name: { required: true, message: t('pages.user-api-key.form.name.required') },
        status: { required: true, message: t('pages.user-api-key.form.status.required') },
    }
    const superAdmin = userOptions.value.find((item) => item.value === 'root')
    if (superAdmin) {
        superAdmin.label = t('pages.user-api-key.super_admin')
    }
})

// 监听日历组件值变化同步到表单 expires_at
watch(expiresAtDayjs, (val) => {
    if (val) {
        formData.value.expires_at = val.toISOString()
    } else {
        formData.value.expires_at = null
    }
})

/**
 * 加载所有可用成员列表
 */
async function loadUsers() {
    try {
        const result = await apis.users.getUsersList({ pageSize: 100, current: 1 }).catch(() => null)
        const list = [
            {
                label: t('pages.user-api-key.super_admin'),
                value: 'root',
            },
        ]
        if (result && config('http.code.success') === result.success && result.data) {
            list.push(
                ...result.data.map((user) => ({
                    label: `${user.name} (${user.username})`,
                    value: String(user.id),
                }))
            )
        }
        userOptions.value = list
    } catch (e) {
        console.error('Failed to load user list:', e)
    }
}

/**
 * 用户下拉搜索过滤
 */
function filterUserOption(input, option) {
    const label = option.label || ''
    return label.toLowerCase().includes(input.toLowerCase())
}

// 初始化加载成员
loadUsers()

/**
 * 打开“创建”模态框
 */
function handleCreate() {
    expiresAtDayjs.value = null
    showModal({
        type: 'create',
        title: t('pages.user-api-key.add'),
    })
    // 默认表单初值
    formData.value = {
        user_id: undefined,
        name: '',
        status: 1,
        quota: -1,
        expires_at: null,
        description: '',
    }
}

/**
 * 打开“编辑”模态框
 */
async function handleEdit(record = {}) {
    showModal({
        type: 'edit',
        title: t('pages.user-api-key.edit'),
    })

    // 确保用户下拉选项已加载，避免 select 无法匹配 label
    await loadUsers()

    // 获取单条详情（前端接口获取时已脱敏）
    const { data, success } = await apis.user_api_key.get(record.id).catch(() => ({ success: false }))
    if (!success) {
        hideModal()
        return
    }

    formRecord.value = data
    formData.value = cloneDeep(data)

    if (data.expires_at) {
        expiresAtDayjs.value = dayjs(data.expires_at)
    } else {
        expiresAtDayjs.value = null
    }
}

/**
 * 提交表单
 */
function handleOk() {
    formRef.value
        .validateFields()
        .then(async () => {
            try {
                showLoading()

                // 参数转换
                const params = {
                    ...formData.value,
                    quota: Number(formData.value.quota) || -1,
                }

                let result = null
                switch (modal.value.type) {
                    case 'create':
                        result = await apis.user_api_key.create(params).catch(() => {
                            throw new Error()
                        })
                        break
                    case 'edit':
                        result = await apis.user_api_key.update(formData.value.id, params).catch(() => {
                            throw new Error()
                        })
                        break
                }

                hideLoading()
                if (config('http.code.success') === result?.success) {
                    hideModal()
                    emit('ok')

                    // 如果是创建操作，弹窗显示生成的明文 API Key
                    if (modal.value.type === 'create' && result.data?.api_key) {
                        createdKeyPlaintext.value = result.data.api_key
                        successModal.value.open = true
                    } else {
                        message.success(t('common.save.success'))
                    }
                }
            } catch (error) {
                hideLoading()
            }
        })
        .catch(() => {
            hideLoading()
        })
}

/**
 * 一键复制生成的 Key 到剪贴板
 */
function handleCopy() {
    if (navigator.clipboard) {
        navigator.clipboard
            .writeText(createdKeyPlaintext.value)
            .then(() => message.success(t('pages.user-api-key.copy.success')))
            .catch(() => message.error(t('pages.user-api-key.copy.failed')))
    } else {
        // 兼容降级
        const input = document.createElement('input')
        input.setAttribute('value', createdKeyPlaintext.value)
        document.body.appendChild(input)
        input.select()
        document.execCommand('copy')
        document.body.removeChild(input)
        message.success(t('pages.user-api-key.copy.success'))
    }
}

function handleCancel() {
    hideModal()
}

function onAfterClose() {
    resetForm()
    expiresAtDayjs.value = null
    hideLoading()
}

defineExpose({
    handleCreate,
    handleEdit,
})
</script>

<style lang="less" scoped>
.api-key-success-container {
    padding: 12px 4px;
}

.key-box {
    margin-top: 16px;
    background: #f5f5f5;
    padding: 12px 16px;
    border-radius: 6px;
    border: 1px dashed #d9d9d9;
    display: flex;
    justify-content: space-between;
    align-items: center;

    .key-text {
        font-family: Menlo, Monaco, Consolas, 'Courier New', monospace;
        font-size: 14px;
        color: #1890ff;
        font-weight: bold;
        word-break: break-all;
        margin-right: 12px;
        user-select: all;
    }
}

.mb-4 {
    margin-bottom: 16px;
}
.mt-4 {
    margin-top: 24px;
}
</style>
