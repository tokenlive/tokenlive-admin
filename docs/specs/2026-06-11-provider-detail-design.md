# 供应商详情页开发设计文档

本文档定义了在 `tokenlive-admin` 控制台中，开发“供应商详情页”（支持端点管理 Tab 页）的设计规格。

## 需求说明

- **目标**：参考已有的“模型详情页”布局和交互，增加“供应商详情页”。
- **一期范围**：仅包含“端点管理” Tab 页。

## 设计细节

### 1. 路由注册与导航

- **文件**：[resource.js](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/router/routes/resource.js)
  在资源管理路由中添加 `providerDetail` 路由：
  - 路径：`provider/:id`
  - 组件：`resource/ProviderDetail.vue`
  - 面包屑：`基础资源` -> `供应商管理` -> `供应商详情`
- **文件**：[notMenuPage.js](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/router/notMenuPage.js)
  将 `'providerDetail'` 添加到不可在左侧菜单展示的页面列表中。
- **文件**：[provider.vue](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/views/resource/provider.vue)
  将供应商列表中的 `name`（或 `code`）列转换为链接形式，点击触发调用：

  ```javascript
  function goToDetail(record) {
      router.push({ name: 'providerDetail', params: { id: record.id } })
  }
  ```

### 2. 国际化翻译 (i18n)

- **中文包** ([zh-CN/pages.js](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/locales/lang/zh-CN/pages.js)):

  ```javascript
  'pages.provider.detail.basicInfo': '基本信息',
  'pages.provider.detail.tab.endpoint': '端点管理',
  ```

- **英文包** ([en-US/pages.js](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/locales/lang/en-US/pages.js)):

  ```javascript
  'pages.provider.detail.basicInfo': 'Basic Info',
  'pages.provider.detail.tab.endpoint': 'Endpoints',
  ```

### 3. 弹窗组件适配 ([EndpointEditDialog.vue](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/views/resource/EndpointEditDialog.vue))

- **属性扩展**：
  添加 `providerId` 和 `mode`（支持 `'model'` 和 `'provider'` 两个视角）。
- **表单控制**：
  - 如果是 `'model'` 模式（默认）：隐藏“模型选择”框，显示“供应商”下拉选择框；
  - 如果是 `'provider'` 模式：隐藏“供应商”下拉框（自动绑定当前 `props.providerId`），显示“模型”下拉选择框（通过 `props.modelOptions` 获取选项列表）。
- **校验规则**：
  - 动态适应视角，根据不同视角校验所填写的必须项（如 `provider_id` 或 `model_id`）。

### 4. 供应商详情页组件 ([ProviderDetail.vue](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/views/resource/ProviderDetail.vue))

- **基本信息卡片**：
  通过 `apis.provider.getProvider(id)` 获取当前供应商的基本信息并展示。
- **端点列表加载**：
  调用现成的后端接口 `apis.endpoint.getEndpointsByProviderId(providerId)` 获取关联端点列表。
- **表格列定义**：
  表格展示端点的：协议、接口地址、真实模型名称、权重、优先级、状态及启用情况等，不展示供应商列。
- **端点交互**：
  - 支持新增、编辑、删除、以及触发端点延迟测试功能。

## 验证与测试方案

- **手动测试**：在列表页点击供应商名称成功跳转详情页，能够正常加载端点列表。
- **功能测试**：添加、编辑端点时在弹窗中能正确进行模型选择与保存，删除/测试端点逻辑运行正常。
