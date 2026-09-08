export const CONTEXT_LENGTH_OPTIONS = [
    { value: '4096', label: '4K (4,096)' },
    { value: '8192', label: '8K (8,192)' },
    { value: '16384', label: '16K (16,384)' },
    { value: '32768', label: '32K (32,768)' },
    { value: '65536', label: '64K (65,536)' },
    { value: '100000', label: '100K (100,000)' },
    { value: '128000', label: '128K (128,000)' },
    { value: '200000', label: '200K (200,000)' },
    { value: '256000', label: '256K (256,000)' },
    { value: '1000000', label: '1M (1,000,000)' },
    { value: '2000000', label: '2M (2,000,000)' },
]

export function toContextLengthSelectValue(value) {
    if (value === undefined || value === null || value === '') {
        return undefined
    }
    return String(value)
}

export function parseContextLength(value) {
    if (value === undefined || value === null || value === '') {
        return null
    }
    const raw = String(value).replace(/,/g, '').trim().toLowerCase()
    if (!raw) {
        return null
    }
    const match = raw.match(/^(\d+(?:\.\d+)?)\s*(k|m)?$/)
    if (!match) {
        return null
    }
    let n = Number(match[1])
    if (match[2] === 'k') {
        n *= 1000
    } else if (match[2] === 'm') {
        n *= 1000000
    }
    if (!Number.isFinite(n) || n < 0) {
        return null
    }
    return Math.floor(n)
}

export function filterContextLengthOption(input, option) {
    const query = String(input).toLowerCase().replace(/,/g, '')
    const val = String(option.value ?? '')
    const label = String(option.label ?? '').toLowerCase()
    return val.includes(query) || label.includes(query)
}
