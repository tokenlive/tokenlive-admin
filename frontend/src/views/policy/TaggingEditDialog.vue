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
            layout="vertical"
            style="margin-top: 16px">
            <!-- 基础设置 -->
            <div class="section-title">{{ $t('pages.tagging.section.basic') }}</div>

            <!-- 策略名称 + 执行优先级 -->
            <a-row :gutter="16">
                <a-col :span="12">
                    <a-form-item
                        :label="$t('pages.tagging.form.name')"
                        name="name">
                        <a-input
                            :placeholder="$t('pages.tagging.form.name.placeholder')"
                            v-model:value="formData.name" />
                    </a-form-item>
                </a-col>
                <a-col :span="12">
                    <a-form-item
                        :label="$t('pages.tagging.form.order')"
                        name="order">
                        <a-input-number
                            v-model:value="formData.order"
                            :min="0"
                            style="width: 100%"
                            :placeholder="$t('pages.tagging.form.order.placeholder')" />
                    </a-form-item>
                </a-col>
            </a-row>

            <!-- 多条件关系 + 生效状态 -->
            <a-row :gutter="16">
                <a-col :span="12">
                    <a-form-item
                        :label="$t('pages.tagging.form.relation')"
                        name="relation">
                        <a-radio-group v-model:value="formData.relation">
                            <a-radio value="AND">AND</a-radio>
                            <a-radio value="OR">OR</a-radio>
                        </a-radio-group>
                    </a-form-item>
                </a-col>
                <a-col :span="12">
                    <a-form-item
                        :label="$t('pages.tagging.form.enabled')"
                        name="enabled">
                        <a-switch
                            v-model:checked="formData.enabled"
                            :checked-value="1"
                            :un-checked-value="0" />
                    </a-form-item>
                </a-col>
            </a-row>

            <!-- 备注说明 -->
            <a-form-item
                :label="$t('pages.tagging.form.description')"
                name="description">
                <a-textarea
                    v-model:value="formData.description"
                    :rows="2"
                    :placeholder="$t('pages.tagging.form.description.placeholder')" />
            </a-form-item>

            <!-- 条件规则配置 -->
            <div class="section-title">{{ $t('pages.tagging.form.conditions') }}</div>
            <a-card
                size="small"
                style="margin-bottom: 16px">
                <template #extra>
                    <a-button
                        type="link"
                        size="small"
                        @click="addCondition">
                        <template #icon><plus-outlined /></template>
                        {{ $t('pages.tagging.form.conditions.add') }}
                    </a-button>
                </template>

                <!-- 表头 -->
                <div
                    v-if="formData.conditions && formData.conditions.length > 0"
                    class="rule-header">
                    <a-row :gutter="8">
                        <a-col :span="4">{{ $t('pages.tagging.form.conditions.type.placeholder') }}</a-col>
                        <a-col :span="5">{{ $t('pages.tagging.form.conditions.key.placeholder') }}</a-col>
                        <a-col :span="5">{{ $t('pages.tagging.form.conditions.opType.placeholder') }}</a-col>
                        <a-col :span="8">{{ $t('pages.tagging.form.conditions.values.placeholder') }}</a-col>
                        <a-col :span="2" />
                    </a-row>
                </div>

                <!-- 空状态 -->
                <a-empty
                    v-if="!formData.conditions || formData.conditions.length === 0"
                    :description="$t('pages.tagging.form.conditions.empty')"
                    :image-style="{ height: '40px' }" />

                <!-- 数据行 -->
                <div
                    v-else
                    class="rule-table-row"
                    v-for="(item, index) in formData.conditions"
                    :key="index">
                    <a-row
                        :gutter="8"
                        align="middle"
                        style="width: 100%">
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
                        <a-col :span="5">
                            <a-input
                                v-model:value="item.key"
                                :placeholder="$t('pages.tagging.form.conditions.key.placeholder')" />
                        </a-col>
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
                        <a-col :span="8">
                            <a-select
                                v-model:value="item.values"
                                mode="tags"
                                :placeholder="$t('pages.tagging.form.conditions.values.placeholder')"
                                style="width: 100%"
                                :open="false" />
                        </a-col>
                        <a-col
                            :span="2"
                            style="text-align: center">
                            <delete-outlined
                                style="color: #ff4d4f; cursor: pointer"
                                @click="removeCondition(index)" />
                        </a-col>
                    </a-row>
                </div>
            </a-card>

            <!-- 打标动作配置 -->
            <div class="section-title">{{ $t('pages.tagging.form.actions') }}</div>
            <a-card size="small">
                <template #extra>
                    <a-button
                        type="link"
                        size="small"
                        @click="addAction">
                        <template #icon><plus-outlined /></template>
                        {{ $t('pages.tagging.form.actions.add') }}
                    </a-button>
                </template>

                <!-- 表头 -->
                <div
                    v-if="formData.actions && formData.actions.length > 0"
                    class="rule-header">
                    <a-row :gutter="8">
                        <a-col :span="6">{{ $t('pages.tagging.form.actions.type.placeholder') }}</a-col>
                        <a-col :span="8">{{ $t('pages.tagging.form.actions.key.placeholder') }}</a-col>
                        <a-col :span="8">{{ $t('pages.tagging.form.actions.value.placeholder') }}</a-col>
                        <a-col :span="2" />
                    </a-row>
                </div>

                <!-- 空状态 -->
                <a-empty
                    v-if="!formData.actions || formData.actions.length === 0"
                    :description="$t('pages.tagging.form.actions.empty')"
                    :image-style="{ height: '40px' }" />

                <!-- 数据行 -->
                <div
                    v-else
                    class="rule-table-row"
                    v-for="(item, index) in formData.actions"
                    :key="index">
                    <a-row
                        :gutter="8"
                        align="middle">
                        <a-col :span="6">
                            <a-select
                                v-model:value="item.type"
                                :placeholder="$t('pages.tagging.form.actions.type.placeholder')"
                                style="width: 100%">
                                <a-select-option value="TAG">TAG</a-select-option>
                                <a-select-option value="REQ_HEADER">REQ_HEADER</a-select-option>
                                <a-select-option value="RSP_HEADER">RSP_HEADER</a-select-option>
                                <a-select-option value="REQ_COOKIE">REQ_COOKIE</a-select-option>
                                <a-select-option value="RSP_COOKIE">RSP_COOKIE</a-select-option>
                                <a-select-option value="REQ_BODY">REQ_BODY</a-select-option>
                            </a-select>
                        </a-col>
                        <a-col :span="8">
                            <a-input
                                v-model:value="item.key"
                                :placeholder="$t('pages.tagging.form.actions.key.placeholder')" />
                        </a-col>
                        <a-col :span="8">
                            <a-input
                                v-model:value="item.value"
                                :placeholder="$t('pages.tagging.form.actions.value.placeholder')" />
                        </a-col>
                        <a-col
                            :span="2"
                            style="text-align: center">
                            <delete-outlined
                                style="color: #ff4d4f; cursor: pointer"
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
                values: c.values || [],
            }))
        }
        if (!formData.value.actions) {
            formData.value.actions = []
        } else {
            formData.value.actions = formData.value.actions.map((a) => ({
                ...a,
                type: a.type || 'TAG',
            }))
        }
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
                values: c.values || [],
            }))
        }
        if (!formData.value.actions) {
            formData.value.actions = []
        } else {
            formData.value.actions = formData.value.actions.map((a) => ({
                ...a,
                type: a.type || 'TAG',
            }))
        }
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
    formData.value.actions.push({ type: 'TAG', key: '', value: '' })
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
            // 校验 actions 是否填完了 type, key & value
            for (const act of formData.value.actions) {
                if (!act.type || !act.key || !act.value) {
                    message.error('打标动作类型、键或值不能为空，请填写完整！')
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
.section-title {
    display: flex;
    align-items: center;
    gap: 10px;
    font-size: 12px;
    font-weight: 500;
    opacity: 0.65;
    margin: 20px 0 12px;

    &::before,
    &::after {
        content: '';
        flex: 1;
        height: 1px;
        background: rgba(128, 128, 128, 0.25);
    }

    &:first-child {
        margin-top: 0;
    }
}

.rule-header {
    padding: 4px 0 8px;
    font-size: 12px;
    opacity: 0.45;
    border-bottom: 1px solid rgba(128, 128, 128, 0.15);
    margin-bottom: 4px;
}

.rule-table-row {
    padding: 8px 0;
    border-bottom: 1px solid rgba(128, 128, 128, 0.15);

    &:last-child {
        border-bottom: none;
    }
}
</style>
