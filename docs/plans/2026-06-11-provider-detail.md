# 供应商详情页开发实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 参照模型详情页的 UI 风格，为 `tokenlive-admin` 前端控制台开发供应商详情页，重点实现端点管理 Tab 页和端点弹窗复用。

**Architecture:** 路由配置中注册二级详情路由，前端列表页增加跳转支持。通过改造 `EndpointEditDialog.vue` 组件，添加互斥参数判断来实现“模型视角”与“提供商视角”的动态表单布局渲染。

**Tech Stack:** Vue 3, Ant Design Vue, Vue Router, Vue i18n

---

### Task 1: 国际化词条配置

**Files:**

- Modify: [zh-CN/pages.js](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/locales/lang/zh-CN/pages.js)
- Modify: [en-US/pages.js](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/locales/lang/en-US/pages.js)

- [ ] **Step 1: 新增中文翻译词条**
  在 `zh-CN/pages.js` 中适当位置（在 `pages.model.detail` 之后）追加：

  ```javascript
  'pages.provider.detail.basicInfo': '基本信息',
  'pages.provider.detail.tab.endpoint': '端点管理',
  ```

- [ ] **Step 2: 新增英文翻译词条**
  在 `en-US/pages.js` 中适当位置追加：

  ```javascript
  'pages.provider.detail.basicInfo': 'Basic Info',
  'pages.provider.detail.tab.endpoint': 'Endpoints',
  ```

- [ ] **Step 3: 使用 prettier 格式化 locale 文件**
  在 `frontend` 目录下运行：
  `npx prettier --config .prettierrc --write src/locales/lang/zh-CN/pages.js src/locales/lang/en-US/pages.js`
  确保格式化无误。

---

### Task 2: 路由注册与列表跳转

**Files:**

- Modify: [resource.js](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/router/routes/resource.js)
- Modify: [notMenuPage.js](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/router/notMenuPage.js)
- Modify: [provider.vue](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/views/resource/provider.vue)

- [ ] **Step 1: 注册供应商详情路由**
  在 `routes/resource.js` 文件的 `children` 属性中增加：

  ```javascript
  {
      path: 'provider/:id',
      name: 'providerDetail',
      component: 'resource/ProviderDetail.vue',
      meta: {
          title: '供应商详情',
          isMenu: false,
          keepAlive: false,
          permission: '*',
          active: 'providerList',
          openKeys: ['resource'],
          breadcrumb: [
              { name: 'resource', meta: { title: '基础资源' } },
              { name: 'providerList', meta: { title: '供应商管理' } },
          ],
      },
  }
  ```

- [ ] **Step 2: 将页面加入 notMenuPage 声明**
  在 `notMenuPage.js` 的列表末尾增加 `'providerDetail'`。
  修改后的文件内容：

  ```javascript
  const notMenuPage = ['setting', 'serviceDetail', 'modelDetail', 'tenantDetail', 'providerDetail']
  export default notMenuPage
  ```

- [ ] **Step 3: 改造 provider.vue 列表页，支持点击跳转**
  - 修改 `columns` 数组，将 name 字段定义由 `dataIndex: 'name'` 调整为 `key: 'name'` 以便定制渲染：

    ```javascript
    { title: t('pages.provider.form.name'), key: 'name', width: 200 },
    ```

  - 在 `<a-table>` 下的 `<template #bodyCell="{ column, record }">` 中增加 `name` 列的渲染插槽：

    ```html
    <template v-if="'name' === column.key">
        <a @click="goToDetail(record)">
            {{ record.name }}
        </a>
    </template>
    ```

  - 引入 `useRouter` 并编写 `goToDetail` 逻辑：

    ```javascript
    import { useRouter } from 'vue-router'
    const router = useRouter()
    function goToDetail(record) {
        router.push({ name: 'providerDetail', params: { id: record.id } })
    }
    ```

- [ ] **Step 4: 格式化路由与列表文件**
  在 `frontend` 目录下运行：
  `npx prettier --config .prettierrc --write src/router/routes/resource.js src/router/notMenuPage.js src/views/resource/provider.vue`

---

### Task 3: 改造端点编辑弹窗

**Files:**

- Modify: [EndpointEditDialog.vue](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/views/resource/EndpointEditDialog.vue)

- [ ] **Step 1: 扩展 props 定义**
  在 `EndpointEditDialog.vue` 中扩展 `props`，支持传入 `providerId`：

  ```javascript
  const props = defineProps({
      providerOptions: { type: Array, default: () => [] },
      modelOptions: { type: Array, default: () => [] },
      modelId: { type: String, default: '' },
      providerId: { type: String, default: '' },
  })
  ```

- [ ] **Step 2: 动态表单控制 (HTML 模板部分)**
  在 `<a-form>` 中：
  - 如果传入了 `providerId`，我们将 `provider_id` 表单项隐藏；否则正常渲染下拉框：

    ```html
    <a-form-item
        v-if="providerId"
        name="provider_id"
        v-show="false">
        <a-input v-model:value="formData.provider_id" />
    </a-form-item>
    <a-form-item
        v-else
        :label="$t('pages.endpoint.form.provider_id')"
        name="provider_id">
        <a-select
            :placeholder="$t('pages.endpoint.form.provider_id.placeholder')"
            v-model:value="formData.provider_id"
            show-search
            :filter-option="filterProviderOption">
            <a-select-option
                v-for="p in providerOptions"
                :key="p.id"
                :value="p.id">
                {{ p.name }}
            </a-select-option>
        </a-select>
    </a-form-item>
    ```

  - 如果传入了 `providerId`（且无 `modelId`），说明此时需要用户选择模型，我们在此场景下展示“模型选择”下拉框；否则仍然使用之前的隐藏输入框逻辑：

    ```html
    <a-form-item
        v-if="!modelId && providerId"
        :label="$t('pages.endpoint.form.model_id')"
        name="model_id">
        <a-select
            :placeholder="$t('pages.endpoint.form.model_id.placeholder')"
            v-model:value="formData.model_id"
            show-search
            :filter-option="filterModelOption">
            <a-select-option
                v-for="m in modelOptions"
                :key="m.id"
                :value="m.id">
                {{ m.model_name }}
            </a-select-option>
        </a-select>
    </a-form-item>
    <a-form-item
        v-else
        name="model_id"
        v-show="false">
        <a-input v-model:value="formData.model_id" />
    </a-form-item>
    ```

- [ ] **Step 3: 添加模型过滤方法与初始化逻辑**
  在 script 部分：
  - 增加模型选择框的模糊匹配方法：

    ```javascript
    function filterModelOption(input, option) {
        return option.children?.[0]?.children?.toLowerCase().includes(input.toLowerCase())
    }
    ```

  - 修改 `handleCreate` 函数，初始化 `provider_id` 的默认值：

    ```javascript
    function handleCreate() {
        showModal({
            type: 'create',
            title: t('pages.endpoint.add'),
        })
        formData.value = {
            weight: 1,
            enabled: 0,
            priority: 0,
            protocol: '',
            model_id: props.modelId || undefined,
            provider_id: props.providerId || undefined,
        }
        metadataList.value = []
        headersList.value = []
    }
    ```

- [ ] **Step 4: 格式化文件**
  在 `frontend` 目录下运行：
  `npx prettier --config .prettierrc --write src/views/resource/EndpointEditDialog.vue`

---

### Task 4: 开发供应商详情页

**Files:**

- Create: [ProviderDetail.vue](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/views/resource/ProviderDetail.vue)

- [ ] **Step 1: 编写 ProviderDetail.vue 组件代码**
  创建完整页面以实现供应商基本信息的渲染以及对应名下的端点数据管理（可增删改查和延迟测试）。
  页面源码结构设计如下：

  ```vue
  <template>
      <div class="provider-detail">
          <!-- 基本信息 -->
          <a-card
              :title="$t('pages.provider.detail.basicInfo')"
              class="info-card"
              :bordered="false">
              <a-card-grid style="width: 25%; text-align: center">
                  <div class="info-item">
                      <span class="info-label">{{ $t('pages.provider.form.name') }}</span>
                      <span class="info-value">{{ providerData.name || '--' }}</span>
                  </div>
              </a-card-grid>
              <a-card-grid style="width: 25%; text-align: center">
                  <div class="info-item">
                      <span class="info-label">{{ $t('pages.provider.form.code') }}</span>
                      <span class="info-value">{{ providerData.code || '--' }}</span>
                  </div>
              </a-card-grid>
              <a-card-grid style="width: 25%; text-align: center">
                  <div class="info-item">
                      <span class="info-label">{{ $t('pages.provider.form.protocol') }}</span>
                      <span class="info-value">
                          <a-tag color="blue" v-if="providerData.protocol">{{ providerData.protocol }}</a-tag>
                          <span v-else>--</span>
                      </span>
                  </div>
              </a-card-grid>
              <a-card-grid style="width: 25%; text-align: center">
                  <div class="info-item">
                      <span class="info-label">{{ $t('pages.provider.form.enabled') }}</span>
                      <span class="info-value">
                          <a-tag :color="providerData.enabled === 1 ? 'green' : 'default'">
                              {{
                                  providerData.enabled === 1
                                      ? $t('pages.provider.form.enabled.active')
                                      : $t('pages.provider.form.enabled.inactive')
                              }}
                          </a-tag>
                      </span>
                  </div>
              </a-card-grid>
              <a-card-grid style="width: 25%; text-align: center">
                  <div class="info-item">
                      <span class="info-label">{{ $t('pages.provider.form.creator') }}</span>
                      <span class="info-value">{{ providerData.creator || '--' }}</span>
                  </div>
              </a-card-grid>
              <a-card-grid style="width: 50%; text-align: center">
                  <div class="info-item">
                      <span class="info-label">{{ $t('pages.provider.form.url') }}</span>
                      <span class="info-value">{{ providerData.url || '--' }}</span>
                  </div>
              </a-card-grid>
              <a-card-grid style="width: 25%; text-align: center">
                  <div class="info-item">
                      <span class="info-label">{{ $t('pages.provider.form.description') }}</span>
                      <span class="info-value">{{ providerData.description || '--' }}</span>
                  </div>
              </a-card-grid>
          </a-card>

          <!-- Tab 区域 -->
          <a-card
              class="detail-card"
              :bordered="false">
              <a-tabs
                  v-model:activeKey="activeTab"
                  class="detail-tabs">
                  <a-tab-pane
                      key="endpoint"
                      :tab="$t('pages.provider.detail.tab.endpoint')" />
              </a-tabs>

              <!-- 端点管理 Tab 内容 -->
              <div v-if="activeTab === 'endpoint'">
                  <div class="tab-toolbar">
                      <a-button
                          type="primary"
                          @click="$refs.endpointEditRef.handleCreate()">
                          {{ $t('pages.endpoint.add') }}
                      </a-button>
                      <div class="tab-toolbar-right">
                          <a-button @click="loadEndpointList">
                              <template #icon><reload-outlined /></template>
                          </a-button>
                      </div>
                  </div>
                  <a-table
                      :columns="endpointColumns"
                      :data-source="endpointListData"
                      :loading="endpointLoading"
                      :pagination="endpointPagination"
                      @change="onEndpointTableChange">
                      <template #bodyCell="{ column, record }">
                          <template v-if="'model_id' === column.key">
                              {{ getModelName(record.model_id) }}
                          </template>
                          <template v-if="'url' === column.key">
                              <a-tooltip :title="record.url">
                                  <span class="url-text">{{ record.url }}</span>
                              </a-tooltip>
                          </template>
                          <template v-if="'protocol' === column.key">
                              <a-tag
                                  v-if="record.protocol"
                                  color="blue"
                                  >{{ record.protocol }}</a-tag
                              >
                              <span
                                  v-else
                                  style="color: #999"
                                  >{{ $t('pages.endpoint.form.protocol.inherit') }}</span
                              >
                          </template>
                          <template v-if="'real_model' === column.key">
                              {{ record.real_model || '--' }}
                          </template>
                          <template v-if="'priority' === column.key">
                              {{ record.priority ?? 0 }}
                          </template>
                          <template v-if="'enabled' === column.key">
                              <a-tag :color="record.enabled === 1 ? 'green' : 'default'">
                                  {{
                                      record.enabled === 1
                                          ? $t('pages.endpoint.form.enabled.active')
                                          : $t('pages.endpoint.form.enabled.inactive')
                                  }}
                              </a-tag>
                          </template>
                          <template v-if="'recent_status' === column.key">
                              <div style="display: flex; gap: 2px; align-items: center">
                                  <a-tooltip
                                      v-for="(point, index) in record.status_points || []"
                                      :key="index">
                                      <template #title>
                                          {{ point.start_time }} ~ {{ point.end_time }}<br />
                                          成功: {{ point.success_count }} | 失败: {{ point.fail_count }}
                                      </template>
                                      <div :style="getPointStyle(point)"></div>
                                  </a-tooltip>
                              </div>
                          </template>
                          <template v-if="'created_at' === column.key">
                              {{ formatUtcDateTime(record.created_at) }}
                          </template>
                          <template v-if="'action' === column.key">
                              <x-action-button
                                  :disabled="testingEndpoints[record.id]"
                                  @click="handleTestEndpoint(record)">
                                  <a-tooltip>
                                      <template #title> {{ $t('pages.endpoint.test') }}</template>
                                      <loading-outlined v-if="testingEndpoints[record.id]" />
                                      <api-outlined v-else />
                                  </a-tooltip>
                              </x-action-button>
                              <x-action-button @click="$refs.endpointEditRef.handleEdit(record)">
                                  <a-tooltip>
                                      <template #title> {{ $t('pages.endpoint.edit') }}</template>
                                      <edit-outlined />
                                  </a-tooltip>
                              </x-action-button>
                              <x-action-button @click="handleRemoveEndpoint(record)">
                                  <a-tooltip>
                                      <template #title> {{ $t('button.delete') }}</template>
                                      <delete-outlined style="color: #ff4d4f" />
                                  </a-tooltip>
                              </x-action-button>
                          </template>
                      </template>
                  </a-table>
              </div>
          </a-card>

          <!-- 端点编辑弹窗 -->
          <endpoint-edit-dialog
              ref="endpointEditRef"
              :provider-options="providerOptions"
              :model-options="modelOptions"
              :provider-id="providerId"
              @ok="loadEndpointList" />
      </div>
  </template>

  <script setup>
  import { ref, onMounted, reactive } from 'vue'
  import { useRoute } from 'vue-router'
  import { message, Modal } from 'ant-design-vue'
  import { ReloadOutlined, EditOutlined, DeleteOutlined, ApiOutlined, LoadingOutlined } from '@ant-design/icons-vue'
  import apis from '@/apis'
  import { config } from '@/config'
  import { formatUtcDateTime } from '@/utils/util'
  import { useI18n } from 'vue-i18n'
  import EndpointEditDialog from './EndpointEditDialog.vue'

  defineOptions({
      name: 'providerDetail',
  })

  const route = useRoute()
  const { t } = useI18n()
  const providerId = ref(route.params.id)
  const providerData = ref({})
  const activeTab = ref('endpoint')

  const modelOptions = ref([])
  const providerOptions = ref([])
  const endpointListData = ref([])
  const endpointLoading = ref(false)
  const endpointPagination = reactive({
      current: 1,
      pageSize: 10,
      total: 0,
      showSizeChanger: true,
      showTotal: (total) => `共 ${total} 条`,
  })

  const endpointColumns = [
      {
          title: t('pages.endpoint.form.model_id'),
          key: 'model_id',
          width: 180,
      },
      {
          title: t('pages.endpoint.form.protocol'),
          key: 'protocol',
          width: 120,
      },
      {
          title: t('pages.endpoint.form.url'),
          key: 'url',
          ellipsis: true,
      },
      {
          title: t('pages.endpoint.form.real_model'),
          key: 'real_model',
          width: 150,
      },
      {
          title: t('pages.endpoint.form.weight'),
          dataIndex: 'weight',
          width: 80,
      },
      {
          title: t('pages.endpoint.form.priority'),
          key: 'priority',
          width: 80,
      },
      {
          title: t('pages.endpoint.form.enabled'),
          key: 'enabled',
          width: 100,
      },
      {
          title: t('pages.endpoint.recent_status'),
          key: 'recent_status',
          width: 220,
      },
      {
          title: t('pages.endpoint.form.description'),
          dataIndex: 'description',
          ellipsis: true,
      },
      {
          title: t('pages.endpoint.form.created_at'),
          key: 'created_at',
          width: 180,
      },
      {
          title: t('button.action'),
          key: 'action',
          width: 160,
      },
  ]

  onMounted(() => {
      loadProviderDetail()
      loadModelOptions()
      loadProviderOptions()
      loadEndpointList()
  })

  async function loadProviderDetail() {
      try {
          const { data, success } = await apis.provider.getProvider(providerId.value)
          if (success) {
              providerData.value = data || {}
          }
      } catch (error) {
          // ignore
      }
  }

  async function loadModelOptions() {
      try {
          const { data, success } = await apis.model.getModelList({ pageSize: 1000, current: 1 })
          if (config('http.code.success') === success) {
              modelOptions.value = data || []
          }
      } catch (error) {
          // ignore
      }
  }

  async function loadProviderOptions() {
      try {
          const { data, success } = await apis.provider.getProviderList({ pageSize: 1000, current: 1 })
          if (config('http.code.success') === success) {
              providerOptions.value = data || []
          }
      } catch (error) {
          // ignore
      }
  }

  function getModelName(id) {
      if (!id) return '--'
      const m = modelOptions.value.find((item) => item.id === id)
      return m ? m.model_name : id
  }

  async function loadEndpointList() {
      try {
          endpointLoading.value = true
          const { data, success, total } = await apis.endpoint
              .getEndpointsByProviderId(providerId.value)
              .catch(() => {
                  throw new Error()
              })
          endpointLoading.value = false
          if (config('http.code.success') === success) {
              endpointListData.value = data || []
              // getEndpointsByProviderId 没有分页时 total 可以用数据长度兜底
              endpointPagination.total = total || (data ? data.length : 0)
          }
      } catch (error) {
          endpointLoading.value = false
      }
  }

  function onEndpointTableChange({ current, pageSize }) {
      endpointPagination.current = current
      endpointPagination.pageSize = pageSize
      loadEndpointList()
  }

  const testingEndpoints = ref({})

  async function handleTestEndpoint(record) {
      if (testingEndpoints.value[record.id]) return
      testingEndpoints.value[record.id] = true
      try {
          const { data, success, message: errMessage } = await apis.endpoint.testEndpoint(record.id)
          if (success && data && data.success) {
              message.success(t('pages.endpoint.test.success', { latency: data.latency_ms }))
          } else {
              const errMsg = data ? data.detail || data.message || errMessage : errMessage
              Modal.error({
                  title: t('pages.endpoint.test.failure'),
                  content: errMsg || '未知错误',
                  okText: t('button.confirm'),
              })
          }
      } catch (error) {
          Modal.error({
              title: t('pages.endpoint.test.failure'),
              content: error.message || '网络请求错误',
              okText: t('button.confirm'),
          })
      } finally {
          testingEndpoints.value[record.id] = false
      }
  }

  function handleRemoveEndpoint({ id }) {
      Modal.confirm({
          title: t('pages.endpoint.delTip'),
          okText: t('button.confirm'),
          okType: 'danger',
          onOk: () => {
              return new Promise((resolve, reject) => {
                  ;(async () => {
                      try {
                          const { success } = await apis.endpoint.delEndpoint(id).catch(() => {
                              throw new Error()
                          })
                          if (config('http.code.success') === success) {
                              resolve()
                              message.success(t('component.message.success.delete'))
                              await loadEndpointList()
                          }
                      } catch (error) {
                          reject()
                      }
                  })()
              })
          },
      })
  }

  function getPointStyle(point) {
      let color = '#f5f5f5'
      let border = '1px solid #d9d9d9'
      if (point.success_count > 0 && point.fail_count === 0) {
          color = '#52c41a'
          border = '1px solid #52c41a'
      } else if (point.success_count === 0 && point.fail_count > 0) {
          color = '#f5222d'
          border = '1px solid #f5222d'
      } else if (point.success_count > 0 && point.fail_count > 0) {
          if (point.success_count > point.fail_count) {
              color = '#fadb14'
              border = '1px solid #fadb14'
          } else {
              color = '#fa8c16'
              border = '1px solid #fa8c16'
          }
      }
      return {
          width: '12px',
          height: '12px',
          backgroundColor: color,
          border: border,
          borderRadius: '2px',
          cursor: 'pointer',
      }
  }
  </script>

  <style lang="less" scoped>
  @import '@/styles/variables.less';

  .provider-detail {
      padding: 0;

      .info-card {
          margin-bottom: 16px;

          :deep(.ant-card-head) {
              border-bottom: 1px solid #f0f0f0;
          }

          .info-item {
              display: flex;
              flex-direction: column;
              align-items: center;
              gap: 8px;

              .info-label {
                  font-size: 13px;
                  color: rgba(0, 0, 0, 0.45);
              }

              .info-value {
                  font-size: 15px;
                  font-weight: 500;
                  color: rgba(0, 0, 0, 0.85);
                  max-width: 100%;
                  overflow: hidden;
                  text-overflow: ellipsis;
                  white-space: nowrap;
              }
          }
      }

      .detail-card {
          .detail-tabs {
              margin-bottom: 16px;
          }

          .tab-toolbar {
              display: flex;
              justify-content: space-between;
              align-items: center;
              margin-bottom: 16px;

              .tab-toolbar-right {
                  display: flex;
                  gap: 8px;
              }
          }

          .url-text {
              font-family: monospace;
              color: rgba(0, 0, 0, 0.65);
          }
      }
  }
  </style>
  ```

- [ ] **Step 2: 格式化新文件**
  在 `frontend` 目录下运行：
  `npx prettier --config .prettierrc --write src/views/resource/ProviderDetail.vue`

---

### Task 5: 整体编译与构建验证

**Files:**

- Test: [frontend/src/views/resource/ProviderDetail.vue](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/views/resource/ProviderDetail.vue)

- [ ] **Step 1: 运行前端热更新开发服务器测试编译**
  在 `frontend` 目录下运行（使用本地 npm）：
  `npm run dev -- --help` （先检查命令行选项）后，正常执行 `npm run dev` 或 `npm run build` 以检测是否有语法错误。
  预期：无编译报错，生成 `dist/` 资源包。
