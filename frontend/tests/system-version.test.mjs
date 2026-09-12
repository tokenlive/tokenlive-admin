import test from 'node:test'
import assert from 'node:assert/strict'

import { copyVersionText, formatIdentity, hasAvailableUpdate } from '../src/utils/system-version.js'

const identity = {
    edition: 'professional',
    install_channel: 'release',
    build: { version: 'v1.2.3', kind: 'release' },
}
const summary = {
    identity,
    gateway: {
        status: 'ready',
        scope: 'shared',
        groups: [
            { version: 'v1.0.0', build_kind: 'release', count: 2 },
            { version: 'dev-local', build_kind: 'dev', count: 1 },
        ],
    },
    can_manage_updates: true,
}
const available = {
    enabled: true,
    components: [
        {
            component: 'admin',
            current: 'v1.2.3',
            latest: 'v1.3.0',
            state: 'available',
            source: {
                status: 'ready',
                stale: false,
                candidate: { version: 'v1.3.0', release_url: 'https://github.com/tokenlive/tokenlive-admin/releases' },
                last_attempt: '2026-09-12T00:00:00Z',
                last_success: '2026-09-12T00:00:00Z',
            },
        },
    ],
    retry_after_seconds: 0,
}

test('only a server-authorized, enabled, explicitly fresh available result creates a badge', () => {
    assert.equal(hasAvailableUpdate(summary, available), true)
    for (const can_manage_updates of [false, undefined, 'true', 1]) {
        assert.equal(hasAvailableUpdate({ ...summary, can_manage_updates }, available), false)
    }
    assert.equal(hasAvailableUpdate(summary, { ...available, enabled: false }), false)
    assert.equal(hasAvailableUpdate(null, available), false)
    assert.equal(hasAvailableUpdate(summary, null), false)
})

test('missing or nonboolean freshness is non-actionable even when state is available', () => {
    for (const source of [undefined, null, {}, { stale: true }, { stale: 0 }, { stale: 'false' }]) {
        assert.equal(
            hasAvailableUpdate(summary, { ...available, components: [{ ...available.components[0], source }] }),
            false
        )
    }
})

test('cached states are not reinterpreted as available', () => {
    for (const state of [
        'current',
        'ahead',
        'uncomparable',
        'no_candidate',
        'disabled',
        'stale',
        'unknown',
        'unavailable',
        'checking',
    ]) {
        assert.equal(
            hasAvailableUpdate(summary, { ...available, components: [{ ...available.components[0], state }] }),
            false
        )
    }
    assert.equal(hasAvailableUpdate(summary, { ...available, components: [] }), false)
    assert.equal(hasAvailableUpdate(summary, { ...available, components: null }), false)
    assert.equal(
        hasAvailableUpdate(summary, {
            ...available,
            components: [{ state: 'unknown' }, available.components[0]],
        }),
        true
    )
})

test('identity display distinguishes package versions from professional component versions', () => {
    assert.equal(formatIdentity(identity), 'Professional · Admin v1.2.3')
    assert.equal(
        formatIdentity({ ...identity, edition: 'standalone', install_channel: 'homebrew' }),
        'Standalone v1.2.3'
    )
})

test('unknown identity and missing builds never infer an edition or a release version', () => {
    assert.equal(formatIdentity(null), 'Unknown · Version unknown')
    assert.equal(formatIdentity({ edition: 'standalone' }), 'Standalone unknown')
    assert.equal(
        formatIdentity({ edition: 'future-edition', build: { version: 'local-snapshot', kind: 'dev' } }),
        'Unknown · Version local-snapshot'
    )
})

test('copy text includes edition, channel, current component versions and mixed group counts only', () => {
    assert.equal(
        copyVersionText(summary),
        'TokenLive Professional · Admin v1.2.3\nChannel: release\nBuild: release\nGateway v1.0.0 (release) × 2\nGateway dev-local (dev) × 1'
    )
    const copied = copyVersionText({
        ...summary,
        can_manage_updates: false,
        updates: available,
        gateway: { ...summary.gateway, nodes: [{ id: 'private-node' }] },
    })
    assert.doesNotMatch(copied, /v1\.3\.0|private-node|release_url/)
    assert.equal(copied, copyVersionText(summary))
})

test('standalone copies only the package and empty Gateway groups remain unknown', () => {
    assert.equal(
        copyVersionText({ ...summary, identity: { ...identity, edition: 'standalone', install_channel: 'homebrew' } }),
        'TokenLive Standalone v1.2.3\nChannel: homebrew\nBuild: release'
    )
    assert.match(copyVersionText({ ...summary, gateway: { groups: [] } }), /Gateway: unknown$/)
    assert.equal(copyVersionText(null), 'TokenLive Unknown · Version unknown\nChannel: unknown\nBuild: unknown')
})

test('optional labels localize identity and copy while keeping versions and default callers unchanged', () => {
    const labels = {
        professional: '专业版',
        standalone: '单机版',
        unknown: '未知',
        version: '版本',
        channel: '安装渠道',
        build: '构建类型',
        separator: '：',
        channelValues: { release: '发行包', homebrew: 'Homebrew', unknown: '未知' },
        buildValues: { release: '正式构建', dev: '开发构建', unknown: '未知' },
    }
    assert.equal(formatIdentity(identity, labels), '专业版 · Admin v1.2.3')
    assert.equal(
        formatIdentity({ edition: 'standalone', build: { version: '原始+build.1' } }, labels),
        '单机版 原始+build.1'
    )
    assert.equal(formatIdentity(null, labels), '未知 · 版本 未知')
    assert.equal(
        copyVersionText(summary, labels),
        'TokenLive 专业版 · Admin v1.2.3\n安装渠道：发行包\n构建类型：正式构建\nGateway v1.0.0 (正式构建) × 2\nGateway dev-local (开发构建) × 1'
    )
    assert.equal(copyVersionText(null, labels), 'TokenLive 未知 · 版本 未知\n安装渠道：未知\n构建类型：未知')
    assert.equal(formatIdentity(identity), 'Professional · Admin v1.2.3')
})
