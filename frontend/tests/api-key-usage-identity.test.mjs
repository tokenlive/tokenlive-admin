import test from 'node:test'
import assert from 'node:assert/strict'
import { registerHooks } from 'node:module'
import { createPinia, setActivePinia } from 'pinia'

const stub = (source) => `data:text/javascript,${encodeURIComponent(source)}`
const dependencies = new Map([
    ['@/config', stub('export const config = key => key === "http.code.success" ? true : key')],
    [
        '@/utils/storage',
        stub(`export default {local: {
        getItem: (key, fallback) => globalThis.__identityStorage.get(key) ?? fallback,
        setItem: (key, value) => globalThis.__identityStorage.set(key, value),
        removeItem: key => globalThis.__identityStorage.delete(key),
    }}`),
    ],
    ['@/apis', stub('export default {user: {getUserDetail: () => globalThis.__identityReply()}}')],
])
registerHooks({
    resolve(specifier, context, nextResolve) {
        if (dependencies.has(specifier)) return { url: dependencies.get(specifier), shortCircuit: true }
        if (
            context.parentURL?.endsWith('/store/modules/user.js') &&
            ['./app', './router', './multiTab'].includes(specifier)
        ) {
            return { url: stub('export default () => ({$reset() {}})'), shortCircuit: true }
        }
        if (specifier.startsWith('@/'))
            return nextResolve(new URL(`../src/${specifier.slice(2)}.js`, import.meta.url).href, context)
        return nextResolve(specifier, context)
    },
})
const { default: useUserStore } = await import('../src/store/modules/user.js')

function userStore() {
    globalThis.__identityStorage = new Map([
        ['storage.userInfo', { is_root: true, id: 'old-root' }],
        ['storage.token', 'old-token'],
    ])
    setActivePinia(createPinia())
    return useUserStore()
}

test('persisted Root identity is not verified until the current-user request succeeds', async () => {
    const user = userStore()
    assert.equal(user.userInfoVerified, false)
    globalThis.__identityReply = async () => ({ success: true, data: { id: 'ordinary', is_root: false } })
    await user.getUserInfo()
    assert.equal(user.userInfoVerified, true)
    assert.equal(user.userInfo.is_root, false)
    globalThis.__identityReply = async () => {
        throw new Error('offline')
    }
    await assert.rejects(user.getUserInfo())
    assert.equal(user.userInfoVerified, false)
})

test('a late profile for an old token cannot overwrite a new account identity', async () => {
    const user = userStore()
    let resolve
    globalThis.__identityReply = () =>
        new Promise((done) => {
            resolve = done
        })
    const pending = user.getUserInfo()
    user.token = 'new-token'
    user.userInfo = { id: 'new-user', is_root: false }
    user.userInfoVerified = true
    resolve({ success: true, data: { id: 'old-root', is_root: true } })
    await assert.rejects(pending)
    assert.equal(user.userInfo.id, 'new-user')
    assert.equal(user.userInfoVerified, true)
})
