import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const pages = [
    '../src/views/resource/ModelDetail.vue',
    '../src/views/resource/model_catalog_detail.vue',
    '../src/views/system/tenant/TenantDetail.vue',
]

test('remaining detail pages use adaptive table height in their tab content', async () => {
    for (const page of pages) {
        const source = await readFile(new URL(page, import.meta.url), 'utf8')
        assert.match(source, /:style="\{ height: appStore\.mainHeight \}"/)
        assert.match(source, /useTableAutoScrollY/)
        assert.match(source, /class="table-fill-region"/)
        assert.match(source, /y: .*TableScrollY \|\| undefined/)
    }
})
