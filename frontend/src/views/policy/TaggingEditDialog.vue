<template>
    <a-modal
        :open="modal.open"
        :title="modal.title"
        :width="800"
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
            :label-col="{ style: { width: '110px' } }">
            <!-- 策略名称 -->
            <a-form-item
                :label="$t('pages.tagging.form.name')"
                name="name">
                <a-input
                    :placeholder="$t('pages.tagging.form.name.placeholder')"
                    v-model:value="formData.name" />
            </a-form-item>

            <!-- 执行优先级 -->
            <a-form-item
                :label="$t('pages.tagging.form.order')"
                name="order">
                <a-input-number
                    v-model:value="formData.order"
                    :min="0"
                    style="width: 100%"
                    :placeholder="$t('pages.tagging.form.order.placeholder')" />
            </a-form-item>

            <!-- 多条件逻辑关系 -->
            <a-form-item
                :label="$t('pages.tagging.form.relation')"
                name="relation">
                <a-radio-group v-model:value="formData.relation">
                    <a-radio value="AND">AND (同时满足所有条件)</a-radio>
                    <a-radio value="OR">OR (满足任意一个条件)</a-radio>
                </a-radio-group>
            </a-form-item>

            <!-- 状态 -->
            <a-form-item
                :label="$t('pages.tagging.form.enabled')"
                name="enabled">
                <a-switch
                    v-model:checked="formData.enabled"
                    :checked-value="1"
                    :un-checked-value="0">
                </a-switch>
            </a-form-item>

            <!-- 描述 -->
            <a-form-item
                :label="$t('pages.tagging.form.description')"
                name="description">
                <a-textarea
                    v-model:value="formData.description"
                    :placeholder="$t('pages.tagging.form.description.placeholder')" />
            </a-form-item>

            <!-- 条件配置列表 -->
            <a-card
                class="mb-4"
                :title="$t('pages.tagging.form.conditions')"
                size="small">
                <template #extra>
                    <a-button
                        type="link"
                        size="small"
                        @click="addCondition">
                        <template #icon><plus-outlined /></template>
                        {{ $t('pages.tagging.form.conditions.add') }}
                    </a-button>
                </template>
                <div
                    v-if="!formData.conditions || formData.conditions.length === 0"
                    class="empty-tip">
                    {{ $t('pages.tagging.form.conditions.empty') }}
                </div>
                <div
                    v-else
                    class="rule-table-row"
                    v-for="(item, index) in formData.conditions"
                    :key="index">
                    <a-row
                        :gutter="8"
                        align="middle"
                        style="width: 100%">
                        <!-- 类型 -->
                        <a-col :span="4">
                            <a-select
                                v-model:value="item.type"
                                :placeholder="$t('pages.tagging.form.conditions.type.placeholder')"
                                style="width: 100%">
                                <a-select-option value="header">HEADER</a-select-option>
                                <a-select-option value="query">QUERY</a-select-option>
                                <a-select-option value="cookie">COOKIE</a-select-option>
                                <a-select-option value="system">SYSTEM</a-select-option>
                                <a-select-option value="tag">TAG</a-select-option>
                            </a-select>
                        </a-col>
                        <!-- 参数名 -->
                        <a-col :span="5">
                            <a-input
                                v-model:value="item.key"
                                :placeholder="$t('pages.tagging.form.conditions.key.placeholder')" />
                        </a-col>
                        <!-- 算子 -->
                        <a-col :span="5">
                            <a-select
                                v-model:value="item.op_type"
                                :placeholder="$t('pages.tagging.form.conditions.opType.placeholder')"
                                style="width: 100%">
                                <a-select-option value="EQUAL">EQUAL (等于)</a-select-option>
                                <a-select-option value="NOT_EQUAL">NOT_EQUAL (不等于)</a-select-option>
                                <a-select-option value="IN">IN (包含值)</a-select-option>
                                <a-select-option value="NOT_IN">NOT_IN (不包含值)</a-select-option>
                                <a-select-option value="REGULAR">REGULAR (正则匹配)</a-select-option>
                                <a-select-option value="PREFIX">PREFIX (前缀匹配)</a-select-option>
                            </a-select>
                        </a-col>
                        <!-- 值列表 -->
                        <a-col :span="8">
                            <a-select
                                v-model:value="item.values"
                                mode="tags"
                                :placeholder="$t('pages.tagging.form.conditions.values.placeholder')"
                                style="width: 100%"
                                :open="false" />
                        </a-col>
                        <!-- 删除 -->
                        <a-col
                            :span="2"
                            style="text-align: center">
                            <delete-outlined
                                class="delete-icon"
                                @click="removeCondition(index)" />
                        </a-col>
                    </a-row>
                </div>
            </a-card>

            <!-- 动作配置列表 -->
            <a-card
                :title="$t('pages.tagging.form.actions')"
                size="small">
                <template #extra>
                    <a-button
                        type="link"
                        size="small"
                        @click="addAction">
                        <template #icon><plus-outlined /></template>
                        {{ $t('pages.tagging.form.actions.add') }}
                    </a-button>
                </template>
                <div
                    v-if="!formData.actions || formData.actions.length === 0"
                    class="empty-tip text-danger">
                    {{ $t('pages.tagging.form.actions.empty') }}
                </div>
                <div
                    v-else
                    class="rule-table-row"
                    v-for="(item, index) in formData.actions"
                    :key="index">
                    <a-row
                        :gutter="8"
                        align="middle"
                        style="width: 100%">
                        <!-- 标签键 -->
                        <a-col :span="10">
                            <a-input
                                v-model:value="item.key"
                                :placeholder="$t('pages.tagging.form.actions.key.placeholder')" />
                        </a-col>
                        <!-- 标签值 -->
                        <a-col :span="12">
                            <a-input
                                v-model:value="item.value"
                                :placeholder="$t('pages.tagging.form.actions.value.placeholder')" />
                        </a-col>
                        <!-- 删除 -->
                        <a-col
                            :span="2"
                            style="text-align: center">
                            <delete-outlined
                                class="delete-icon"
                                @click="removeAction(index)" />
                        </a-col>
                    </a-row>
                </div>
            </a-card>
        </a-form>
    </a-modal>
</template>

<script setup>
import { cloneDeep } from 'lodash-es'
import { message } from 'ant-design-vue'
import { ref } from 'vue'
import { config } from '@/config'
import apis from '@/apis'
import { useForm, useModal } from '@/hooks'
import { useI18n } from 'vue-i18n'
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons-vue'

const emit = defineEmits(['ok'])
const { modal, showModal, hideModal, showLoading, hideLoading } = useModal()
const { formRecord, formData, formRef, formRules, resetForm } = useForm()
const { t } = useI18n()

const cancelText = ref(t('button.cancel'))
const okText = ref(t('button.confirm'))

formRules.value = {
    name: { required: true, message: t('pages.tagging.form.name.required') },
    relation: { required: true, message: t('pages.tagging.form.relation.required') },
}

function handleCreate() {
    showModal({
        type: 'create',
        title: t('pages.tagging.add'),
    })
    formData.value.enabled = 1
    formData.value.order = 0
    formData.value.relation = 'AND'
    formData.value.conditions = []
    formData.value.actions = []
}

async function handleCopy(record = {}) {
    showModal({
        type: 'create',
        title: t('pages.tagging.copy'),
    })

    try {
        const { data, success } = await apis.policy.getTagging(record.id).catch()
        if (!success) {
            message.error(t('component.message.error.save'))
            hideModal()
            return
        }
        // 复制模式：不设置 formRecord
        const cloned = cloneDeep(data)
        cloned.name = `${cloned.name} - ${t('pages.policy.copy.suffix')}`
        delete cloned.id
        delete cloned.created_at
        delete cloned.updated_at
        formData.value = cloned
        if (!formData.value.conditions) {
            formData.value.conditions = []
        } else {
            formData.value.conditions = formData.value.conditions.map((c) => ({
                ...c,
                op_type: c.op_type !== undefined ? c.op_type : c.opType,
            }))
        }
        if (!formData.value.actions) formData.value.actions = []
    } catch (e) {
        hideModal()
    }
}

async function handleEdit(record = {}) {
    showModal({
        type: 'edit',
        title: t('pages.tagging.edit'),
    })

    try {
        const { data, success } = await apis.policy.getTagging(record.id).catch()
        if (!success) {
            message.error(t('component.message.error.save'))
            hideModal()
            return
        }
        formRecord.value = data
        formData.value = cloneDeep(data)
        if (!formData.value.conditions) {
            formData.value.conditions = []
        } else {
            formData.value.conditions = formData.value.conditions.map((c) => ({
                ...c,
                op_type: c.op_type !== undefined ? c.op_type : c.opType,
            }))
        }
        if (!formData.value.actions) formData.value.actions = []
    } catch (e) {
        hideModal()
    }
}

// 条件列表行增删
function addCondition() {
    if (!formData.value.conditions) formData.value.conditions = []
    formData.value.conditions.push({ type: 'header', key: '', op_type: 'EQUAL', values: [] })
}

function removeCondition(index) {
    formData.value.conditions.splice(index, 1)
}

// 染色动作列表行增删
function addAction() {
    if (!formData.value.actions) formData.value.actions = []
    formData.value.actions.push({ key: '', value: '' })
}

function removeAction(index) {
    formData.value.actions.splice(index, 1)
}

function handleOk() {
    formRef.value
        .validateFields()
        .then(async (values) => {
            // 前置检查 actions 列表
            if (!formData.value.actions || formData.value.actions.length === 0) {
                message.error('请至少添加一个染色注入动作！')
                return
            }
            // 校验 actions 是否填完了 key & value
            for (const act of formData.value.actions) {
                if (!act.key || !act.value) {
                    message.error('打标动作键或值不能为空，请填写完整！')
                    return
                }
            }

            try {
                showLoading()
                const params = {
                    ...values,
                    order: formData.value.order || 0,
                    conditions: formData.value.conditions,
                    actions: formData.value.actions,
                    enabled: formData.value.enabled,
                    description: formData.value.description,
                }

                let result = null
                switch (modal.value.type) {
                    case 'create':
                        result = await apis.policy.createTagging(params).catch(() => {
                            throw new Error()
                        })
                        break
                    case 'edit':
                        result = await apis.policy.updateTagging(formData.value.id, params).catch(() => {
                            throw new Error()
                        })
                        break
                }
                hideLoading()
                if (config('http.code.success') === result?.success) {
                    hideModal()
                    emit('ok')
                }
            } catch (error) {
                hideLoading()
            }
        })
        .catch(() => {
            hideLoading()
        })
}

function handleCancel() {
    hideModal()
}

function onAfterClose() {
    resetForm()
    hideLoading()
}

defineExpose({
    handleCreate,
    handleEdit,
    handleCopy,
})
</script>

<style lang="less" scoped>
.mb-4 {
    margin-bottom: 16px;
}
.rule-table-row {
    padding: 8px;
    border-bottom: 1px solid rgba(128, 128, 128, 0.1);
    &:last-child {
        border-bottom: none;
    }
}
.empty-tip {
    text-align: center;
    color: #999;
    padding: 12px 0;
    font-size: 13px;
}
.text-danger {
    color: #ff4d4f;
}
.delete-icon {
    font-size: 16px;
    color: #ff4d4f;
    cursor: pointer;
    transition: color 0.2s;
    &:hover {
        color: #ff7875;
    }
}
</style>
