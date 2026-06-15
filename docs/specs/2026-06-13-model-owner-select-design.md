# 大模型所属企业下拉选择器设计方案

## 1. 目标与背景

在创建和编辑大模型时，原有的“所属企业”（`owner`）字段使用的是普通的文本输入框（`<a-input>`）。由于大模型厂商的名字比较固定（如 OpenAI, Google, DeepSeek 等），手动输入不仅低效，而且容易因拼写不一致（例如 "OpenAI" 与 "openai"）导致数据分类混乱。
为了提升大模型维护的体验和规范性，同时保留大模型生态迅速发展下录入私有厂商或新兴厂商的灵活性，需要将此字段优化为**可联想搜索、支持自由手输**的下拉选择框。

## 2. 详细技术方案

### 2.1 后端与数据库无缝兼容

- **无须改动后端**：经过对后端源码 [model.go](file:///Users/chenzhiguo/Projects/tokenlive-admin/internal/mods/resource/schema/model.go#L23) 及 [init.sql](file:///Users/chenzhiguo/Projects/tokenlive-admin/scripts/init.sql#L156) 的调研，`owner` 字段在底层数据库中对应 `VARCHAR(64)` 类型的普通字符串，后端逻辑中并无硬编码的枚举值强校验。
- **改动范围**：仅需在前端管理台进行交互升级。

### 2.2 前端修改设计

#### 2.2.1 交互组件替换

修改 [ModelEditDialog.vue](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/views/resource/ModelEditDialog.vue) 中的表单项：

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

*注：`mode="combobox"` 使得该选择框支持联想提示列表，但又允许用户自由输入列表以外的任何值。*

#### 2.2.2 数据源定义与检索逻辑

在 [ModelEditDialog.vue](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/views/resource/ModelEditDialog.vue) 的 `<script setup>` 中：

1. 引入并定义 `ownerOptions` 响应式数据，包含主流大模型厂商：

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
```

2. 增加过滤算法 `filterOwnerOption`，支持中英文、大小写不敏感匹配：

```javascript
function filterOwnerOption(input, option) {
    const val = option.value || '';
    const label = option.label || '';
    return val.toLowerCase().includes(input.toLowerCase()) || 
           label.toLowerCase().includes(input.toLowerCase());
}
```

#### 2.2.3 多语言翻译补充

更新国际化多语言翻译：

- [zh-CN/pages.js](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/locales/lang/zh-CN/pages.js) 中的 `'pages.model.form.owner.placeholder': '请选择或输入所属企业'`
- [en-US/pages.js](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/locales/lang/en-US/pages.js) 中的 `'pages.model.form.owner.placeholder': 'Please select or enter the owner'`

## 3. 验证方案

1. **新建模型验证**：
   - 打开新增模型对话框，检查“所属企业”下拉框是否默认显示“请选择或输入所属企业”的占位符。
   - 展开下拉列表，验证主流厂商（OpenAI, DeepSeek, Qwen 等）是否展示且能够被选择。
   - 输入关键字如 `deep` 或 `智谱`，检查过滤逻辑是否工作。
   - 手动输入一个全新的名字（例如 `CustomCompany`）并创建模型，验证是否能够保存成功，并能在列表中正确回显。
2. **编辑模型验证**：
   - 打开已存在模型的编辑对话框，验证“所属企业”是否能正确回显。
