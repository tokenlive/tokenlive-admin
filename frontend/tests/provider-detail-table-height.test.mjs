import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const source = await readFile(new URL('../src/views/resource/ProviderDetail.vue', import.meta.url), 'utf8')

test('provider detail tables use the shared adaptive-height layout', () => {
    assert.match(source, /:style="\{ height: appStore\.mainHeight \}"/)
    assert.match(source, /useTableAutoScrollY/)
    assert.match(source, /ref="endpointTableContainerRef"/)
    assert.match(source, /ref="memberTableContainerRef"/)
    assert.equal((source.match(/y: endpointTableScrollY \|\| undefined/g) || []).length, 1)
    assert.equal((source.match(/y: memberTableScrollY \|\| undefined/g) || []).length, 1)
})
