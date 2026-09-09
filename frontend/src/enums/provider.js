export const PROVIDER_PROTOCOL_OPTIONS = [
    { value: 'openai', labelKey: 'pages.provider.protocol.openai' },
    { value: 'anthropic', labelKey: 'pages.provider.protocol.anthropic' },
    { value: 'gemini', labelKey: 'pages.provider.protocol.gemini' },
    { value: 'joycode', labelKey: 'pages.provider.protocol.joycode' },
]

export function getProviderProtocolOptions(t) {
    return PROVIDER_PROTOCOL_OPTIONS.map((option) => ({
        value: option.value,
        label: t(option.labelKey),
    }))
}

export function getProviderProtocolLabel(protocol, t) {
    const option = PROVIDER_PROTOCOL_OPTIONS.find((item) => item.value === protocol)
    return option ? t(option.labelKey) : protocol || ''
}
