<template>
    <a-modal
        :open="modal.open"
        :title="modal.title"
        :width="580"
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
            :label-col="{ style: { width: '100px' } }">
            <a-card class="mb-8-2">
                <a-form-item
                    :label="$t('pages.model.alias.form.alias')"
                    name="alias">
                    <a-input v-model:value="formData.alias"></a-input>
                </a-form-item>

                <a-form-item
                    :label="$t('pages.model.form.description')"
                    name="description">
                    <a-textarea v-model:value="formData.description"></a-textarea>
                </a-form-item>
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

const props = defineProps({
    modelId: {
        type: [String, Number],
        required: true,
    },
    defaultSpaceCode: {
        type: String,
        default: '',
    },
})

const emit = defineEmits(['ok'])
import { useI18n } from 'vue-i18n'
const { modal, showModal, hideModal, showLoading, hideLoading } = useModal()
const { formRecord, formData, formRef, formRules, resetForm } = useForm()
const { t } = useI18n()
const cancelText = ref(t('button.cancel'))
const okText = ref(t('button.confirm'))

formRules.value = {
    alias: { required: true, message: t('pages.model.alias.form.alias.placeholder') },
}

function handleCreate() {
    showModal({
        type: 'create',
        title: t('pages.model.alias.create'),
    })
    formData.value.model_id = props.modelId
    formData.value.space_code = props.defaultSpaceCode
}

async function handleEdit(record = {}) {
    showModal({
        type: 'edit',
        title: t('pages.model.alias.edit'),
    })

    const { data, success } = await apis.model_alias.getModelAlias(record.id).catch()
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
                    model_id: props.modelId,
                    space_code: formData.value.space_code,
                }
                let result = null
                switch (modal.value.type) {
                    case 'create':
                        result = await apis.model_alias.createModelAlias(params)
                        break
                    case 'edit':
                        result = await apis.model_alias.updateModelAlias(formData.value.id, params)
                        break
                }
                hideLoading()
                if (config('http.code.success') === result?.success) {
                    hideModal()
                    emit('ok')
                } else {
                    message.error(result?.msg || t('component.message.error.save'))
                }
            } catch (error) {
                hideLoading()
                const errMsg =
                    error?.response?.data?.error?.detail ||
                    error?.response?.data?.msg ||
                    error?.message ||
                    t('component.message.error.save')
                message.error(errMsg)
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
