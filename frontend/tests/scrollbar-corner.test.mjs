import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

test('global scrollbar corner is transparent when both scrollbars appear', async () => {
    const source = await readFile(new URL('../src/styles/reset.less', import.meta.url), 'utf8')
    assert.match(source, /::-webkit-scrollbar-corner\s*\{\s*background:\s*transparent;/)
})
