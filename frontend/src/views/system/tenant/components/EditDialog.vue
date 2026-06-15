<template>
    <a-modal
        :open="modal.open"
        :title="modal.title"
        :width="540"
        :confirm-loading="modal.confirmLoading"
        :after-close="onAfterClose"
        :cancel-text="$t('common.cancel')"
        :ok-text="$t('common.confirm')"
        @ok="handleOk"
        @cancel="handleCancel">
        <a-form
            ref="formRef"
            :model="formData"
            :rules="formRules"
            :label-col="{ style: { width: '90px' } }">
            <a-card class="mb-8-2">
                <a-row :gutter="12">
                    <a-col :span="24">
                        <a-form-item
                            :label="$t('pages.tenant.form.code')"
                            name="code">
                            <a-input
                                :placeholder="$t('pages.tenant.form.code.placeholder')"
                                :disabled="modal.type === 'edit'"
                                v-model:value="formData.code"></a-input>
                        </a-form-item>
                    </a-col>
                </a-row>
                <a-row :gutter="12">
                    <a-col :span="24">
                        <a-form-item
                            :label="$t('pages.tenant.form.name')"
                            name="name">
                            <a-input
                                :placeholder="$t('pages.tenant.form.name.placeholder')"
                                v-model:value="formData.name"></a-input>
                        </a-form-item>
                    </a-col>
                </a-row>
                <a-row :gutter="12">
                    <a-col :span="24">
                        <a-form-item
                            :label="$t('pages.tenant.form.description')"
                            name="description">
                            <a-textarea
                                :placeholder="$t('pages.tenant.form.description.placeholder')"
                                v-model:value="formData.description"></a-textarea>
                        </a-form-item>
                    </a-col>
                </a-row>
                <a-row :gutter="12">
                    <a-col :span="24">
                        <a-form-item
                            :label="$t('pages.tenant.form.api_key')"
                            name="api_key">
                            <a-input-search
                                :placeholder="$t('pages.tenant.form.api_key.placeholder')"
                                v-model:value="formData.api_key">
                                <template #enterButton>
                                    <a-button
                                        type="primary"
                                        @click="generateKey"
                                        >{{ $t('pages.tenant.form.api_key.generate') }}</a-button
                                    >
                                </template>
                            </a-input-search>
                        </a-form-item>
                    </a-col>
                </a-row>
                <a-row :gutter="12">
                    <a-col :span="24">
                        <a-form-item
                            :label="$t('pages.tenant.form.status')"
                            name="status">
                            <a-radio-group
                                v-model:value="formData.status"
                                :options="statusOptions"></a-radio-group>
                        </a-form-item>
                    </a-col>
                </a-row>
            </a-card>
        </a-form>
    </a-modal>
</template>

<script setup>
import { computed, watchEffect } from 'vue'
import { cloneDeep } from 'lodash-es'
import { useI18n } from 'vue-i18n'
import { config } from '@/config'
import apis from '@/apis'
import { useForm, useModal } from '@/hooks'

const emit = defineEmits(['ok'])
const { t } = useI18n()
const { modal, showModal, hideModal, showLoading, hideLoading } = useModal()
const { formRecord, formData, formRef, formRules, resetForm } = useForm()

const statusOptions = computed(() => [
    { label: t('pages.tenant.form.status.activated'), value: 'activated' },
    { label: t('pages.tenant.form.status.freezed'), value: 'freezed' },
])

watchEffect(() => {
    formRules.value = {
        code: [
            { required: true, message: t('pages.tenant.form.code.required') },
            { pattern: /^[a-zA-Z0-9_-]+$/, message: t('pages.tenant.form.code.pattern') },
        ],
        name: { required: true, message: t('pages.tenant.form.name.required') },
        status: { required: true, message: t('pages.tenant.form.status.required') },
    }
})

/**
 * 随机生成密钥
 */
function generateKey() {
    const chars = '0123456789abcdef'
    let hex = ''
    for (let i = 0; i < 32; i++) {
        hex += chars[Math.floor(Math.random() * chars.length)]
    }
    formData.value.api_key = 'sk-t-' + hex
}

/**
 * 新建
 */
function handleCreate() {
    showModal({
        type: 'create',
        title: t('pages.tenant.add'),
    })
    formData.value.status = 'activated'
    formData.value.api_key = ''
}

/**
 * 编辑
 */
async function handleEdit(record = {}) {
    showModal({
        type: 'edit',
        title: t('pages.tenant.edit'),
    })
    const { data, success } = await apis.tenant.get(record.id).catch()
    if (!success) {
        hideModal()
        return
    }
    formRecord.value = data
    formData.value = cloneDeep(data)
}

/**
 * 确定保存
 */
function handleOk() {
    formRef.value
        .validateFields()
        .then(async (values) => {
            try {
                showLoading()
                let result = null
                switch (modal.value.type) {
                    case 'create':
                        result = await apis.tenant.create(values).catch(() => {
                            throw new Error()
                        })
                        break
                    case 'edit':
                        result = await apis.tenant.update(formData.value.id, values).catch(() => {
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

/**
 * 取消
 */
function handleCancel() {
    hideModal()
}

/**
 * 关闭后重置
 */
function onAfterClose() {
    resetForm()
    hideLoading()
}

defineExpose({
    handleCreate,
    handleEdit,
})
</script>

<style lang="less" scoped></style>
