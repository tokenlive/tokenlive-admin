import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

const source = await readFile(new URL('../src/views/home/index.vue', import.meta.url), 'utf8')
const selectorMatch = source.match(/<a-select(?=[^>]*v-model:value="trendsGroupBy")[^>]*>[\s\S]*?<\/a-select>/)

assert.ok(selectorMatch, 'Could not find the trendsGroupBy selector')

const optionValues = [...selectorMatch[0].matchAll(/<a-select-option\s+value="([^"]*)"/g)].map((match) => match[1])
const expectedValues = ['', 'model', 'provider']

assert.deepEqual(
    optionValues,
    expectedValues,
    `Expected dashboard trend groups ${JSON.stringify(expectedValues)}, received ${JSON.stringify(optionValues)}`
)

console.log('Dashboard trend groups: global, model, provider')
