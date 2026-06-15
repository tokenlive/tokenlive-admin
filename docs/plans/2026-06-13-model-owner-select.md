# Model Owner Dropdown Selector Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将模型创建/编辑时的“所属企业”（owner）输入框优化为包含主流大模型厂商（如 OpenAI、DeepSeek、Qwen 等）的自由输入下拉框（Combobox）。

**Architecture:** 前端采用 Ant Design Vue 的 `<a-select>`，配合 `mode="combobox"` 达到既有下拉预设又允许手输的效果。后端直接兼容提交的字符串。

**Tech Stack:** Vue 3, Ant Design Vue, Vite

---

### Task 1: 前端多语言占位符翻译优化

**Files:**

- Modify: `frontend/src/locales/lang/zh-CN/pages.js`
- Modify: `frontend/src/locales/lang/en-US/pages.js`

- [ ] **Step 1: 修改中文翻译文件占位符**

修改 [/Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/locales/lang/zh-CN/pages.js](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/locales/lang/zh-CN/pages.js) 中的第 273 行。
原代码：

```javascript
    'pages.model.form.owner.placeholder': '请输入所属企业',
```

修改为：

```javascript
    'pages.model.form.owner.placeholder': '请选择或输入所属企业',
```

- [ ] **Step 2: 修改英文翻译文件占位符**

修改 [/Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/locales/lang/en-US/pages.js](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/locales/lang/en-US/pages.js) 中的第 273 行。
原代码：

```javascript
    'pages.model.form.owner.placeholder': 'Please enter the owner',
```

修改为：

```javascript
    'pages.model.form.owner.placeholder': 'Please select or enter the owner',
```

- [ ] **Step 3: 提交修改**

```bash
git add frontend/src/locales/lang/zh-CN/pages.js frontend/src/locales/lang/en-US/pages.js
git commit -m "i18n: update model owner placeholder translations"
```

### Task 2: 前端大模型编辑对话框组件修改

**Files:**

- Modify: `frontend/src/views/resource/ModelEditDialog.vue`

- [ ] **Step 1: 修改模板中 owner 的 input 为 select combobox**

修改 [/Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/views/resource/ModelEditDialog.vue](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/views/resource/ModelEditDialog.vue) 中的第 94-98 行。
原代码：

```html
                <a-form-item
                    :label="$t('pages.model.form.owner')"
                    name="owner">
                    <a-input v-model:value="formData.owner"></a-input>
                </a-form-item>
```

修改为：

```html
                <a-form-item
                    :label="$t('pages.model.form.owner')"
                    name="owner">
                    <a-select
                        v-model:value="formData.owner"
                        mode="combobox"
                        :placeholder="$t('pages.model.form.owner.placeholder')"
                        :options="ownerOptions"
                        :filter-option="filterOwnerOption"
                        allow-clear>
                    </a-select>
                </a-form-item>
```

- [ ] **Step 2: 在 script setup 中增加 ownerOptions 数据和 filter 过滤算法**

在 [/Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/views/resource/ModelEditDialog.vue](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/views/resource/ModelEditDialog.vue) 的 script setup 块内合适的位置（例如第 155 行附近，`filterSpaceOption` 下方）增加以下代码。

增加代码：

```javascript
const ownerOptions = ref([
    { value: 'OpenAI', label: 'OpenAI' },
    { value: 'DeepSeek', label: 'DeepSeek' },
    { value: 'Google', label: 'Google' },
    { value: 'Anthropic', label: 'Anthropic' },
    { value: 'Qwen', label: 'Qwen (通义千问)' },
    { value: 'Zhipu AI', label: 'Zhipu AI (智谱清言)' },
    { value: 'Moonshot AI', label: 'Moonshot AI (月之暗面)' },
    { value: 'MiniMax', label: 'MiniMax' },
    { value: 'Baichuan', label: 'Baichuan (百川智能)' },
    { value: 'ByteDance', label: 'ByteDance (火山/豆包)' },
    { value: 'Tencent', label: 'Tencent (腾讯混元)' },
    { value: 'Baidu', label: 'Baidu (百度文心)' },
    { value: 'StepFun', label: 'StepFun (阶跃星辰)' },
    { value: 'Meta', label: 'Meta (Llama)' },
    { value: 'Mistral', label: 'Mistral' },
    { value: 'Ollama', label: 'Ollama' }
])

function filterOwnerOption(input, option) {
    const val = option.value || '';
    const label = option.label || '';
    return val.toLowerCase().includes(input.toLowerCase()) || 
           label.toLowerCase().includes(input.toLowerCase());
}
```

- [ ] **Step 3: 使用 prettier 对修改后的前端文件进行格式化**

根据项目规范限制，修改前端文件后，需要在 frontend 目录下运行 prettier 格式化。
在 `frontend` 目录下运行：

```bash
npx prettier --config .prettierrc --write src/views/resource/ModelEditDialog.vue
```

- [ ] **Step 4: 提交修改**

```bash
git add frontend/src/views/resource/ModelEditDialog.vue
git commit -m "feat: replace model owner input with combobox select"
```

### Task 3: 界面手动验证与确认

- [ ] **Step 1: 启动项目并进入模型管理界面**

本地启动项目开发服务器进行验证。

- [ ] **Step 2: 验证新增模型和编辑模型中的所属企业交互**

1. 确认 placeholder 为“请选择或输入所属企业”。
2. 展开下拉菜单，选中 OpenAI，确认表单字段更新为 "OpenAI"。
3. 清除选择，手输一个自定义的值 "CustomCompanyA"，确认可以直接录入。
4. 确认在过滤框中搜索 "deep" 能过滤出 "DeepSeek"。
5. 点击提交并完成创建/编辑，确认无阻碍无报错。
