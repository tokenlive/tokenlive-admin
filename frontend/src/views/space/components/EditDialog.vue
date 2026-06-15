<template>
    <a-modal
        :open="modal.open"
        :title="modal.title"
        :width="640"
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
            :label-col="{ style: { width: '90px' } }">
            <a-card class="mb-8-2">
                <a-row :gutter="12">
                    <a-col :span="12">
                        <a-form-item
                            :label="$t('pages.space.form.code')"
                            name="code">
                            <a-input v-model:value="formData.code"></a-input>
                        </a-form-item>
                    </a-col>

                    <a-col :span="12">
                        <a-form-item
                            :label="$t('pages.space.form.name')"
                            name="name">
                            <a-input v-model:value="formData.name"></a-input>
                        </a-form-item>
                    </a-col>
                </a-row>

                <a-row :gutter="24">
                    <a-col :span="24">
                        <a-form-item
                            :label="$t('pages.space.form.description')"
                            name="description">
                            <a-textarea v-model:value="formData.description"></a-textarea>
                        </a-form-item>
                    </a-col>
                </a-row>
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

const emit = defineEmits(['ok'])
import { useI18n } from 'vue-i18n'
const { modal, showModal, hideModal, showLoading, hideLoading } = useModal()
const { formRecord, formData, formRef, formRules, resetForm } = useForm()
const { t } = useI18n()
const cancelText = ref(t('button.cancel'))
const okText = ref(t('button.confirm'))
formRules.value = {
    code: { required: true, message: t('pages.space.form.code.placeholder') },
    name: { required: true, message: t('pages.space.form.name.placeholder') },
}

function handleCreate() {
    showModal({
        type: 'create',
        title: t('pages.space.add'),
    })
}

async function handleEdit(record = {}) {
    showModal({
        type: 'edit',
        title: t('pages.space.edit'),
    })

    const { data, success } = await apis.space.getSpace(record.id).catch()
    if (!success) {
        message.error(t('component.message.error.save'))
        hideModal()
        return
    }
    formRecord.value = data
    formData.value = cloneDeep(data)
}

function handleOk() {
    formRef.value
        .validateFields()
        .then(async (values) => {
            try {
                showLoading()
                const params = {
                    ...values,
                }
                let result = null
                switch (modal.value.type) {
                    case 'create':
                        result = await apis.space.createSpace(params).catch(() => {
                            throw new Error()
                        })
                        break
                    case 'edit':
                        result = await apis.space.updateSpace(formData.value.id, params).catch(() => {
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
})
</script>

<style lang="less" scoped></style>
