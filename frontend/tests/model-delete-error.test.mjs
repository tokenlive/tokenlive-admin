import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const source = await readFile(new URL('../src/views/resource/model.vue', import.meta.url), 'utf8')
const requestSource = await readFile(new URL('../src/utils/request.js', import.meta.url), 'utf8')
const zhLocaleSource = await readFile(new URL('../src/locales/lang/zh-CN/pages.js', import.meta.url), 'utf8')
const enLocaleSource = await readFile(new URL('../src/locales/lang/en-US/pages.js', import.meta.url), 'utf8')

test('model deletion surfaces backend errors directly and keeps the modal loading', () => {
    assert.match(
        source,
        /error\?\.response\?\.data\?\.error\?\.detail \|\| error\?\.message \|\| t\('component\.message\.error\.request'\)/
    )
    assert.match(source, /message\.error\(getRequestErrorMessage\(error\)\)/)
    assert.match(source, /throw error/)
    assert.doesNotMatch(source, /delModel\(id\)\.catch\(\(\) => \{\s*throw new Error\(\)\s*\}\)/)
    assert.doesNotMatch(source, /content:\s*t\('button\.confirm'\)/)
    assert.match(requestSource, /content: error\?\.detail \|\| 'Request failed'/)
})

test('model deletion confirmation warns that associated policies are deleted together', () => {
    assert.match(zhLocaleSource, /'pages\.model\.delTip':\s*'确定删除该模型吗？若存在关联策略，将一并删除这些策略。'/)
    assert.match(
        enLocaleSource,
        /'pages\.model\.delTip':\s*'Are you sure you want to delete this model\? Any associated policies will be deleted together\.'/
    )
})
