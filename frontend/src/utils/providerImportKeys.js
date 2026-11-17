import { Checkbox, message, Modal } from 'ant-design-vue'
import { computed, h, ref } from 'vue'
import { maskApiKey } from './endpointKeyMode'

/**
 * 解析并（在多 Key 场景下弹窗）选择导入端点时要绑定的供应商密钥列表。
 * 返回 [{ id, value }]；用户取消时返回 null。
 */
export async function resolveImportKeysForProvider({ provider, api_keys, provider_api_keys, t }) {
    const authType = provider?.auth_type || 'api_key'
    if (authType === 'oauth_token') {
        return [{ id: '', value: '' }]
    }

    const rawKeyItems =
        Array.isArray(provider_api_keys) && provider_api_keys.length > 0
            ? provider_api_keys
            : Array.isArray(provider?.api_keys) && provider.api_keys.length > 0
              ? provider.api_keys
              : api_keys

    const keyItems = (Array.isArray(rawKeyItems) ? rawKeyItems : [])
        .map((item) => (typeof item === 'string' ? { id: '', value: item, description: '' } : item))
        .filter((item) => item && item.value)

    if (keyItems.length <= 1) {
        // 单 Key 或无 Key：保持留空以继承 Provider 级别的 API Key
        return [{ id: '', value: '' }]
    }

    const itemIdentifier = (item) => item.id || item.value
    const selectedKeys = ref(keyItems.map(itemIdentifier))
    const checkAll = computed({
        get: () => selectedKeys.value.length === keyItems.length,
        set: (val) => {
            selectedKeys.value = val ? keyItems.map(itemIdentifier) : []
        },
    })
    const indeterminate = computed(() => selectedKeys.value.length > 0 && selectedKeys.value.length < keyItems.length)

    let confirmedKeys = null
    try {
        await new Promise((resolve, reject) => {
            Modal.confirm({
                title: t('pages.provider.fetchModels.api_keys_confirm_title', '选择端点绑定的 API 密钥'),
                width: 600,
                okText: t('button.confirm', '确认'),
                cancelText: t('button.cancel', '取消'),
                content: () =>
                    h('div', { style: { marginTop: '12px' } }, [
                        h(
                            'p',
                            { style: { marginBottom: '12px', color: 'var(--color-text-secondary)' } },
                            t('pages.provider.fetchModels.api_keys_confirm_desc', { count: keyItems.length })
                        ),
                        h(
                            'div',
                            {
                                style: {
                                    borderBottom: '1px solid var(--color-border-secondary, #f0f0f0)',
                                    paddingBottom: '8px',
                                    marginBottom: '12px',
                                },
                            },
                            [
                                h(
                                    Checkbox,
                                    {
                                        checked: checkAll.value,
                                        indeterminate: indeterminate.value,
                                        onChange: (e) => {
                                            checkAll.value = e.target.checked
                                        },
                                    },
                                    () => t('pages.provider.fetchModels.api_keys_select_all', '全选')
                                ),
                            ]
                        ),
                        h(
                            Checkbox.Group,
                            {
                                value: selectedKeys.value,
                                style: { width: '100%', maxHeight: '220px', overflowY: 'auto' },
                                'onUpdate:value': (val) => {
                                    selectedKeys.value = val
                                },
                            },
                            () =>
                                keyItems.map((item) => {
                                    const masked = maskApiKey(item.value)
                                    const label = item.description ? `${masked} (${item.description})` : masked
                                    return h(
                                        Checkbox,
                                        {
                                            value: itemIdentifier(item),
                                            style: {
                                                display: 'flex',
                                                marginLeft: 0,
                                                marginBottom: '8px',
                                                alignItems: 'center',
                                            },
                                        },
                                        () => label
                                    )
                                })
                        ),
                    ]),
                onOk: () => {
                    if (!selectedKeys.value || selectedKeys.value.length === 0) {
                        message.warning(t('pages.provider.fetchModels.api_keys_required', '请至少选择一个 API 密钥！'))
                        return Promise.reject(new Error('NO_KEY_SELECTED'))
                    }
                    const selectedSet = new Set(selectedKeys.value)
                    confirmedKeys = keyItems
                        .filter((item) => selectedSet.has(itemIdentifier(item)))
                        .map((item) => ({ id: item.id || '', value: item.value }))
                    resolve()
                },
                onCancel: () => {
                    reject(new Error('USER_CANCEL'))
                },
            })
        })
    } catch (err) {
        if (err?.message === 'USER_CANCEL') {
            return null
        }
        throw err
    }

    return confirmedKeys
}
