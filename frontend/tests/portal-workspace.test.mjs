import assert from 'node:assert/strict'
import { normalizePortalAPIKeys, portalAPIKeyStatusColor } from '../src/views/ops/portal_workspace.js'

const rows = normalizePortalAPIKeys([
    {
        id: 'key-1',
        name: 'Default',
        key_prefix: 'tl_live',
        secret_last4: 'abcd',
        status: 'active',
        expires_at: '2026-07-01T00:00:00Z',
        last_used_at: '2026-07-02T00:00:00Z',
        key_hash: 'must-not-render',
        plaintext: 'must-not-render',
    },
])

assert.equal(rows.length, 1)
assert.deepEqual(Object.keys(rows[0]).sort(), [
    'created_at',
    'expires_at',
    'id',
    'key_prefix',
    'last_used_at',
    'name',
    'secret_last4',
    'status',
    'updated_at',
])
assert.equal(rows[0].id, 'key-1')
assert.equal(rows[0].key_prefix, 'tl_live')
assert.equal(rows[0].secret_last4, 'abcd')
assert.equal(rows[0].expires_at, '2026-07-01 00:00:00')
assert.equal(rows[0].last_used_at, '2026-07-02 00:00:00')
assert.equal(portalAPIKeyStatusColor('active'), 'green')
assert.equal(portalAPIKeyStatusColor('disabled'), 'orange')
assert.equal(portalAPIKeyStatusColor('revoked'), 'red')
assert.equal(portalAPIKeyStatusColor('unknown'), 'default')
