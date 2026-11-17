import test from 'node:test'
import assert from 'node:assert/strict'
import { isPanelRememberedDisabled, rememberPanelDisabled } from '../src/utils/dashboard-panel-memory.js'

test('safely no-ops when window/localStorage is unavailable', () => {
    assert.equal(isPanelRememberedDisabled('usage-trends'), false)
    rememberPanelDisabled('usage-trends', true)
    assert.equal(isPanelRememberedDisabled('usage-trends'), false)
})

test('remembers and clears disabled state in localStorage', () => {
    const store = new Map()
    globalThis.window = {
        localStorage: {
            getItem: (key) => (store.has(key) ? store.get(key) : null),
            setItem: (key, value) => store.set(key, String(value)),
            removeItem: (key) => store.delete(key),
        },
    }
    try {
        assert.equal(isPanelRememberedDisabled('api-key-ranking'), false)
        rememberPanelDisabled('api-key-ranking', true)
        assert.equal(isPanelRememberedDisabled('api-key-ranking'), true)
        rememberPanelDisabled('api-key-ranking', false)
        assert.equal(isPanelRememberedDisabled('api-key-ranking'), false)
    } finally {
        delete globalThis.window
    }
})
