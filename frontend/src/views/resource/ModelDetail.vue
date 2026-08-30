<template>
    <div
        class="app-page"
        :style="{ height: appStore.mainHeight }">
        <div class="model-detail">
            <!-- 基本信息 -->
            <a-card
                :title="$t('pages.model.detail.basicInfo')"
                class="info-card"
                :bordered="false">
                <template #extra>
                    <a-space>
                        <a-button
                            type="text"
                            size="small"
                            @click="basicInfoCollapsed = !basicInfoCollapsed">
                            <template #icon>
                                <down-outlined v-if="basicInfoCollapsed" />
                                <up-outlined v-else />
                            </template>
                            {{
                                basicInfoCollapsed
                                    ? $t('component.tagSelect.expand')
                                    : $t('component.tagSelect.collapse')
                            }}
                        </a-button>
                        <a-button
                            type="primary"
                            ghost
                            size="small"
                            @click="handleEditModel">
                            <template #icon><edit-outlined /></template>
                            {{ $t('pages.model.edit') }}
                        </a-button>
                    </a-space>
                </template>
                <template v-if="!basicInfoCollapsed">
                    <a-card-grid style="width: 25%; text-align: center">
                        <div class="info-item">
                            <span class="info-label">{{ $t('pages.model.form.model_name') }}</span>
                            <span class="info-value">{{ modelData.model_name || '--' }}</span>
                        </div>
                    </a-card-grid>
                    <a-card-grid style="width: 25%; text-align: center">
                        <div class="info-item">
                            <span class="info-label">{{ $t('pages.model.form.model_code') }}</span>
                            <span
                                class="info-value info-value-code"
                                :class="{ 'is-copyable': !!modelData.model_code }"
                                @click="handleCopyModelCode">
                                {{ modelData.model_code || '--' }}
                            </span>
                        </div>
                    </a-card-grid>
                    <a-card-grid style="width: 25%; text-align: center">
                        <div class="info-item">
                            <span class="info-label">{{ $t('pages.model.form.space_code') }}</span>
                            <span class="info-value">{{ modelData.space_code || '--' }}</span>
                        </div>
                    </a-card-grid>
                    <a-card-grid style="width: 25%; text-align: center">
                        <div class="info-item">
                            <span class="info-label">{{ $t('pages.model.form.owner') }}</span>
                            <span class="info-value">{{ modelData.owner || '--' }}</span>
                        </div>
                    </a-card-grid>
                    <a-card-grid style="width: 25%; text-align: center">
                        <div class="info-item">
                            <span class="info-label">{{ $t('pages.model.form.enabled') }}</span>
                            <span class="info-value">
                                <a-tag :color="modelData.enabled === 1 ? 'green' : 'default'">
                                    {{
                                        modelData.enabled === 1
                                            ? $t('pages.model.form.enabled.active')
                                            : $t('pages.model.form.enabled.inactive')
                                    }}
                                </a-tag>
                            </span>
                        </div>
                    </a-card-grid>
                    <a-card-grid style="width: 25%; text-align: center">
                        <div class="info-item">
                            <span class="info-label">{{ $t('pages.model.form.creator') }}</span>
                            <span class="info-value">{{ modelData.creator || '--' }}</span>
                        </div>
                    </a-card-grid>
                    <a-card-grid style="width: 25%; text-align: center">
                        <div class="info-item">
                            <span class="info-label">{{ $t('pages.model.form.description') }}</span>
                            <a-tooltip
                                v-if="modelData.description"
                                :title="modelData.description">
                                <span class="info-value info-value-ellipsis">{{ modelData.description }}</span>
                            </a-tooltip>
                            <span
                                v-else
                                class="info-value"
                                >--</span
                            >
                        </div>
                    </a-card-grid>
                    <a-card-grid style="width: 25%; text-align: center">
                        <div class="info-item">
                            <span class="info-label">{{ $t('pages.model.form.created_at') }}</span>
                            <span class="info-value">{{ formatUtcDateTime(modelData.created_at) || '--' }}</span>
                        </div>
                    </a-card-grid>
                    <a-card-grid style="width: 25%; text-align: center">
                        <div class="info-item">
                            <span class="info-label">{{ $t('pages.model.form.input_price') }}</span>
                            <span class="info-value info-value-price">
                                <template v-if="modelData.input_price !== undefined">
                                    {{ modelData.input_price }}<span class="info-unit">元/M</span>
                                </template>
                                <template v-else>--</template>
                            </span>
                        </div>
                    </a-card-grid>
                    <a-card-grid style="width: 25%; text-align: center">
                        <div class="info-item">
                            <span class="info-label">{{ $t('pages.model.form.output_price') }}</span>
                            <span class="info-value info-value-price">
                                <template v-if="modelData.output_price !== undefined">
                                    {{ modelData.output_price }}<span class="info-unit">元/M</span>
                                </template>
                                <template v-else>--</template>
                            </span>
                        </div>
                    </a-card-grid>
                    <a-card-grid style="width: 25%; text-align: center">
                        <div class="info-item">
                            <span class="info-label">{{ $t('pages.model.form.cached_price') }}</span>
                            <span class="info-value info-value-price">
                                <template v-if="modelData.cached_price !== undefined">
                                    {{ modelData.cached_price }}<span class="info-unit">元/M</span>
                                </template>
                                <template v-else>--</template>
                            </span>
                        </div>
                    </a-card-grid>
                    <a-card-grid style="width: 25%; text-align: center">
                        <div class="info-item">
                            <span class="info-label">{{ $t('pages.model.form.cache_creation_price') }}</span>
                            <span class="info-value info-value-price">
                                <template v-if="modelData.cache_creation_price !== undefined">
                                    {{ modelData.cache_creation_price }}<span class="info-unit">元/M</span>
                                </template>
                                <template v-else>--</template>
                            </span>
                        </div>
                    </a-card-grid>
                </template>
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
                        :tab="$t('pages.model.detail.tab.endpoint')" />
                    <a-tab-pane
                        key="alias"
                        :tab="$t('pages.model.detail.tab.alias')" />
                    <a-tab-pane
                        v-for="item in policyTypeTabs"
                        :key="item.key"
                        :tab="item.label" />
                    <a-tab-pane
                        key="monitor"
                        :tab="$t('pages.model.detail.tab.monitor')" />
                    <a-tab-pane
                        key="member"
                        :tab="$t('pages.model.detail.tab.member')" />
                </a-tabs>

                <div
                    v-if="activeTab === 'monitor'"
                    class="tab-content tab-content--scroll">
                    <ModelMonitorTab
                        :model-id="modelId"
                        :model-code="modelData.model_code"
                        :active="activeTab === 'monitor'" />
                </div>

                <!-- 端点管理 Tab 内容 -->
                <div
                    v-else-if="activeTab === 'endpoint'"
                    class="tab-content">
                    <div class="tab-toolbar">
                        <a-button
                            type="primary"
                            ghost
                            @click="$refs.endpointEditRef.handleCreate()">
                            <template #icon><plus-outlined /></template>
                            {{ $t('pages.endpoint.add') }}
                        </a-button>
                        <div class="tab-toolbar-right">
                            <a-select
                                v-model:value="endpointFilterProviderId"
                                :placeholder="$t('pages.endpoint.filter.provider')"
                                style="width: 200px; margin-right: 8px"
                                allow-clear
                                show-search
                                :filter-option="filterProviderOption"
                                @change="handleEndpointFilterChange">
                                <a-select-option
                                    v-for="p in providerOptions"
                                    :key="p.id"
                                    :value="p.id">
                                    {{ p.name }}
                                </a-select-option>
                            </a-select>
                            <a-select
                                v-model:value="endpointFilterEnabled"
                                :placeholder="$t('pages.endpoint.filter.status')"
                                style="width: 150px; margin-right: 8px"
                                allow-clear
                                @change="handleEndpointFilterChange">
                                <a-select-option :value="1">
                                    {{ $t('pages.endpoint.form.enabled.active') }}
                                </a-select-option>
                                <a-select-option :value="0">
                                    {{ $t('pages.endpoint.form.enabled.inactive') }}
                                </a-select-option>
                            </a-select>
                            <a-button @click="loadEndpointList">
                                <template #icon><reload-outlined /></template>
                            </a-button>
                        </div>
                    </div>
                    <div
                        ref="endpointTableContainerRef"
                        class="table-fill-region"
                        :style="endpointTableContainerStyle">
                        <a-table
                            :columns="endpointColumns"
                            :data-source="endpointListData"
                            :loading="endpointLoading"
                            :pagination="endpointPagination"
                            :scroll="{ x: 1200, y: endpointTableScrollY || undefined }"
                            @change="onEndpointTableChange">
                            <template #bodyCell="{ column, record }">
                                <template v-if="'provider_id' === column.key">
                                    <a
                                        v-if="record.provider_id"
                                        @click.prevent="goToProviderDetail(record.provider_id)">
                                        {{ getProviderName(record.provider_id) }}
                                    </a>
                                    <span v-else>--</span>
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
                                    <a-tag
                                        v-else-if="getInheritedProtocol(record.provider_id)"
                                        color="blue"
                                        style="border-style: dashed"
                                        >{{ getInheritedProtocol(record.provider_id) }}</a-tag
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
                                    <EndpointStatusStrip :points="record.status_points || []" />
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
                                    <x-action-button @click="handleToggleEndpointEnabled(record)">
                                        <a-tooltip>
                                            <template #title>{{
                                                record.enabled === 1
                                                    ? $t('pages.endpoint.disable')
                                                    : $t('pages.endpoint.enable')
                                            }}</template>
                                            <poweroff-outlined
                                                :style="{ color: record.enabled === 1 ? '#faad14' : '#52c41a' }"
                                        /></a-tooltip>
                                    </x-action-button>
                                    <x-action-button @click="$refs.endpointEditRef.handleEdit(record)">
                                        <a-tooltip>
                                            <template #title> {{ $t('pages.endpoint.edit') }}</template>
                                            <edit-outlined />
                                        </a-tooltip>
                                    </x-action-button>
                                    <x-action-button @click="$refs.endpointEditRef.handleCopy(record)">
                                        <a-tooltip>
                                            <template #title> {{ $t('pages.endpoint.copy') }}</template>
                                            <copy-outlined />
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
                </div>

                <!-- 模型别名 Tab 内容 -->
                <div
                    v-else-if="activeTab === 'alias'"
                    class="tab-content">
                    <div class="tab-toolbar">
                        <a-button
                            type="primary"
                            ghost
                            @click="$refs.aliasEditRef.handleCreate()">
                            <template #icon><plus-outlined /></template>
                            {{ $t('pages.model.alias.create') }}
                        </a-button>
                        <div class="tab-toolbar-right">
                            <a-input-search
                                v-model:value="aliasSearchName"
                                :placeholder="$t('pages.model.alias.search.placeholder')"
                                style="width: 200px"
                                allow-clear
                                @search="loadAliasList"
                                @pressEnter="loadAliasList" />
                            <a-button @click="loadAliasList">
                                <template #icon><reload-outlined /></template>
                            </a-button>
                        </div>
                    </div>
                    <div
                        ref="aliasTableContainerRef"
                        class="table-fill-region"
                        :style="aliasTableContainerStyle">
                        <a-table
                            :columns="aliasColumns"
                            :data-source="aliasListData"
                            :loading="aliasLoading"
                            :pagination="aliasPagination"
                            :scroll="{ y: aliasTableScrollY || undefined }"
                            @change="onAliasTableChange">
                            <template #bodyCell="{ column, record }">
                                <template v-if="'created_at' === column.key">
                                    {{ formatUtcDateTime(record.created_at) }}
                                </template>
                                <template v-if="'action' === column.key">
                                    <x-action-button @click="$refs.aliasEditRef.handleEdit(record)">
                                        <a-tooltip>
                                            <template #title> {{ $t('pages.model.alias.edit') }}</template>
                                            <edit-outlined />
                                        </a-tooltip>
                                    </x-action-button>
                                    <x-action-button @click="handleRemoveAlias(record)">
                                        <a-tooltip>
                                            <template #title> {{ $t('button.delete') }}</template>
                                            <delete-outlined style="color: #ff4d4f" />
                                        </a-tooltip>
                                    </x-action-button>
                                </template>
                            </template>
                        </a-table>
                    </div>
                </div>

                <!-- 治理策略 Tab 内容 -->
                <div
                    v-else-if="isPolicyTab(activeTab)"
                    class="tab-content">
                    <div class="tab-toolbar">
                        <div class="tab-toolbar-left">
                            <a-button
                                type="primary"
                                ghost
                                @click="handleCreateModelPolicy">
                                <template #icon><plus-outlined /></template>
                                {{ $t('button.createPolicy') }}
                            </a-button>
                            <a-button @click="handleOpenCopyTemplate">
                                <template #icon><copy-outlined /></template>
                                {{ $t('button.copyFromTemplate') }}
                            </a-button>
                        </div>
                        <div class="tab-toolbar-right">
                            <a-button @click="loadModelPolicies">
                                <template #icon><reload-outlined /></template>
                            </a-button>
                        </div>
                    </div>
                    <div
                        ref="policyTableContainerRef"
                        class="table-fill-region"
                        :style="policyTableContainerStyle">
                        <a-table
                            :columns="modelPolicyColumns"
                            :data-source="currentModelPolicies"
                            :loading="policyLoading"
                            :pagination="policyPagination"
                            :scroll="{ y: policyTableScrollY || undefined }"
                            @change="onPolicyTableChange">
                            <template #bodyCell="{ column, record }">
                                <template v-if="'enabled' === column.key">
                                    <a-tag :color="record.enabled === 1 ? 'green' : 'default'">
                                        {{
                                            record.enabled === 1
                                                ? $t('pages.model.form.enabled.active')
                                                : $t('pages.model.form.enabled.inactive')
                                        }}
                                    </a-tag>
                                </template>
                                <template v-if="'scope_type' === column.key">
                                    <span v-if="record.scope_type === 'global'">
                                        {{ $t('pages.policy.form.scope_type.global') }}
                                    </span>
                                    <span v-else-if="record.scope_type === 'tenant'">
                                        {{ $t('pages.policy.form.scope_type.tenant') }}
                                    </span>
                                    <span v-else-if="record.scope_type === 'user'">
                                        {{ $t('pages.policy.form.scope_type.user') }}
                                    </span>
                                    <span v-else>-</span>
                                </template>
                                <template v-if="'created_at' === column.key">
                                    {{ formatUtcDateTime(record.created_at) }}
                                </template>
                                <template v-if="'action' === column.key">
                                    <x-action-button
                                        :disabled="togglingModelPolicies[record.id]"
                                        @click="handleToggleModelPolicyEnabled(record)">
                                        <a-tooltip>
                                            <template #title>{{
                                                record.enabled === 1
                                                    ? $t('pages.endpoint.disable')
                                                    : $t('pages.endpoint.enable')
                                            }}</template>
                                            <loading-outlined v-if="togglingModelPolicies[record.id]" />
                                            <poweroff-outlined
                                                v-else
                                                :style="{ color: record.enabled === 1 ? '#faad14' : '#52c41a' }" />
                                        </a-tooltip>
                                    </x-action-button>
                                    <x-action-button @click="handleEditModelPolicy(record)">
                                        <a-tooltip>
                                            <template #title> {{ $t('pages.endpoint.edit') }}</template>
                                            <edit-outlined />
                                        </a-tooltip>
                                    </x-action-button>
                                    <x-action-button @click="handleRemoveModelPolicy(record)">
                                        <a-tooltip>
                                            <template #title> {{ $t('button.delete') }}</template>
                                            <delete-outlined style="color: #ff4d4f" />
                                        </a-tooltip>
                                    </x-action-button>
                                </template>
                            </template>
                        </a-table>
                    </div>
                </div>

                <!-- 成员管理 Tab 内容 -->
                <div
                    v-else-if="activeTab === 'member'"
                    class="tab-content">
                    <div class="tab-toolbar">
                        <a-button
                            type="primary"
                            ghost
                            @click="$refs.memberEditRef.handleCreate()">
                            <template #icon><plus-outlined /></template>
                            {{ $t('pages.member.add') }}
                        </a-button>
                        <div class="tab-toolbar-right">
                            <a-input-search
                                v-model:value="memberSearchUser"
                                :placeholder="$t('pages.member.search.placeholder')"
                                style="width: 200px"
                                allow-clear
                                @search="loadMemberList"
                                @pressEnter="loadMemberList" />
                            <a-button @click="loadMemberList">
                                <template #icon><reload-outlined /></template>
                            </a-button>
                        </div>
                    </div>
                    <div
                        ref="memberTableContainerRef"
                        class="table-fill-region"
                        :style="memberTableContainerStyle">
                        <a-table
                            :columns="memberColumns"
                            :data-source="memberListData"
                            :loading="memberLoading"
                            :pagination="memberPagination"
                            :scroll="{ y: memberTableScrollY || undefined }"
                            @change="onMemberTableChange">
                            <template #bodyCell="{ column, record }">
                                <template v-if="'permission' === column.key">
                                    <a-tag
                                        v-if="hasPermission(record.permission, 1)"
                                        color="green"
                                        >{{ $t('pages.member.form.permission.read') }}</a-tag
                                    >
                                    <a-tag
                                        v-if="hasPermission(record.permission, 2)"
                                        color="blue"
                                        >{{ $t('pages.member.form.permission.write') }}</a-tag
                                    >
                                    <a-tag
                                        v-if="hasPermission(record.permission, 4)"
                                        color="red"
                                        >{{ $t('pages.member.form.permission.delete') }}</a-tag
                                    >
                                </template>
                                <template v-if="'created_at' === column.key">
                                    {{ formatUtcDateTime(record.created_at) }}
                                </template>
                                <template v-if="'action' === column.key">
                                    <x-action-button @click="$refs.memberEditRef.handleEdit(record)">
                                        <a-tooltip>
                                            <template #title> {{ $t('pages.member.edit') }}</template>
                                            <edit-outlined />
                                        </a-tooltip>
                                    </x-action-button>
                                    <x-action-button @click="handleRemoveMember(record)">
                                        <a-tooltip>
                                            <template #title> {{ $t('button.delete') }}</template>
                                            <delete-outlined style="color: #ff4d4f" />
                                        </a-tooltip>
                                    </x-action-button>
                                </template>
                            </template>
                        </a-table>
                    </div>
                </div>
            </a-card>

            <!-- 别名编辑弹窗 -->
            <model-alias-edit-dialog
                ref="aliasEditRef"
                :model-id="modelId"
                :default-space-code="modelData.space_code"
                @ok="loadAliasList" />

            <!-- 成员编辑弹窗 -->
            <model-member-edit-dialog
                ref="memberEditRef"
                :model-id="modelId"
                @ok="loadMemberList" />

            <!-- 端点编辑弹窗 -->
            <endpoint-edit-dialog
                ref="endpointEditRef"
                :provider-options="providerOptions"
                :model-options="modelOptions"
                :model-id="modelId"
                @ok="loadEndpointList" />

            <loadbalance-edit-dialog
                ref="loadbalanceEditRef"
                @ok="loadModelPolicies" />
            <tag-route-edit-dialog
                ref="routeEditRef"
                @ok="loadModelPolicies" />
            <limit-edit-dialog
                ref="limitEditRef"
                @ok="loadModelPolicies" />
            <circuit-break-edit-dialog
                ref="circuitBreakEditRef"
                @ok="loadModelPolicies" />
            <invocation-edit-dialog
                ref="invocationEditRef"
                @ok="loadModelPolicies" />
            <tagging-edit-dialog
                ref="taggingEditRef"
                @ok="loadModelPolicies" />

            <a-modal
                v-model:open="copyTemplateModal.open"
                :title="$t('pages.model.policy.copyTemplate.title')"
                :confirm-loading="copyTemplateModal.loading"
                :ok-text="$t('button.confirm')"
                :cancel-text="$t('button.cancel')"
                @ok="handleCopyTemplateToModel">
                <a-form layout="vertical">
                    <a-form-item :label="$t('pages.model.policy.copyTemplate.templateLabel')">
                        <a-select
                            v-model:value="copyTemplateModal.templateId"
                            show-search
                            :filter-option="filterTemplateOption"
                            :placeholder="$t('pages.model.policy.copyTemplate.templatePlaceholder')">
                            <a-select-option
                                v-for="item in templateOptions"
                                :key="item.id"
                                :value="item.id">
                                {{ item.name }}
                            </a-select-option>
                        </a-select>
                    </a-form-item>
                    <a-form-item :label="$t('pages.model.policy.copyTemplate.newPolicyNameLabel')">
                        <a-input
                            v-model:value="copyTemplateModal.name"
                            :placeholder="$t('pages.model.policy.copyTemplate.newPolicyNamePlaceholder')" />
                    </a-form-item>
                    <a-form-item :label="$t('pages.policy.form.scope_type') || '适用维度'">
                        <a-select
                            v-model:value="copyTemplateModal.scope_type"
                            style="width: 100%">
                            <a-select-option value="global">{{
                                $t('pages.policy.form.scope_type.global') || '全局'
                            }}</a-select-option>
                            <a-select-option value="tenant">{{
                                $t('pages.policy.form.scope_type.tenant') || '租户'
                            }}</a-select-option>
                            <a-select-option value="user">{{
                                $t('pages.policy.form.scope_type.user') || '用户'
                            }}</a-select-option>
                        </a-select>
                    </a-form-item>
                    <a-form-item
                        v-if="copyTemplateModal.scope_type !== 'global'"
                        :label="
                            copyTemplateModal.scope_type === 'tenant'
                                ? $t('pages.policy.form.scope_code.tenant') || '适用租户'
                                : $t('pages.policy.form.scope_code.user') || '适用用户'
                        ">
                        <a-input
                            v-model:value="copyTemplateModal.scope_code"
                            :placeholder="
                                copyTemplateModal.scope_type === 'tenant'
                                    ? $t('pages.policy.form.scope_code.tenant.placeholder') || '请输入租户Code'
                                    : $t('pages.policy.form.scope_code.user.placeholder') || '请输入用户ID'
                            " />
                    </a-form-item>
                    <a-form-item :label="$t('pages.policy.form.priority') || '冲突优先级'">
                        <a-input-number
                            v-model:value="copyTemplateModal.priority"
                            :min="0"
                            style="width: 100%"
                            :placeholder="$t('pages.policy.form.priority.placeholder') || '数值越小越优先'" />
                    </a-form-item>
                </a-form>
            </a-modal>

            <!-- 模型基本信息编辑弹窗 -->
            <model-edit-dialog
                ref="modelEditRef"
                :space-options="spaceOptions"
                @ok="loadModelDetail" />
        </div>
    </div>
</template>

<script setup>
import { ref, onMounted, reactive, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message, Modal } from 'ant-design-vue'
import {
    ReloadOutlined,
    EditOutlined,
    DeleteOutlined,
    ApiOutlined,
    LoadingOutlined,
    CopyOutlined,
    PoweroffOutlined,
    PlusOutlined,
    DownOutlined,
    UpOutlined,
} from '@ant-design/icons-vue'
import apis from '@/apis'
import { config } from '@/config'
import { formatUtcDateTime } from '@/utils/util'
import { useTableAutoScrollY } from '@/hooks'
import { useAppStore } from '@/store'
import { useI18n } from 'vue-i18n'
import ModelAliasEditDialog from './ModelAliasEditDialog.vue'
import ModelMemberEditDialog from './ModelMemberEditDialog.vue'
import EndpointEditDialog from './EndpointEditDialog.vue'
import EndpointStatusStrip from '@/components/EndpointStatusStrip.vue'
import ModelMonitorTab from './ModelMonitorTab.vue'
import ModelEditDialog from './ModelEditDialog.vue'
import LoadbalanceEditDialog from '@/views/policy/LoadbalanceEditDialog.vue'
import TagRouteEditDialog from '@/views/policy/TagRouteEditDialog.vue'
import LimitEditDialog from '@/views/policy/LimitEditDialog.vue'
import CircuitBreakEditDialog from '@/views/policy/CircuitBreakEditDialog.vue'
import InvocationEditDialog from '@/views/policy/InvocationEditDialog.vue'
import TaggingEditDialog from '@/views/policy/TaggingEditDialog.vue'

defineOptions({
    name: 'modelDetail',
})

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const appStore = useAppStore()
const {
    scrollY: endpointTableScrollY,
    containerRef: endpointTableContainerRef,
    containerStyle: endpointTableContainerStyle,
} = useTableAutoScrollY()
const {
    scrollY: aliasTableScrollY,
    containerRef: aliasTableContainerRef,
    containerStyle: aliasTableContainerStyle,
} = useTableAutoScrollY()
const {
    scrollY: policyTableScrollY,
    containerRef: policyTableContainerRef,
    containerStyle: policyTableContainerStyle,
} = useTableAutoScrollY()
const {
    scrollY: memberTableScrollY,
    containerRef: memberTableContainerRef,
    containerStyle: memberTableContainerStyle,
} = useTableAutoScrollY()
const modelId = ref(route.params.id)
const modelData = ref({})
const activeTab = ref(route.query.tab === 'monitor' ? 'monitor' : 'endpoint')
const basicInfoCollapsed = ref(false)
const modelEditRef = ref(null)
const spaceOptions = ref([])
const loadbalanceEditRef = ref(null)
const routeEditRef = ref(null)
const limitEditRef = ref(null)
const circuitBreakEditRef = ref(null)
const invocationEditRef = ref(null)
const taggingEditRef = ref(null)
const templateOptions = ref([])
const copyTemplateModal = reactive({
    open: false,
    loading: false,
    templateId: undefined,
    name: '',
    scope_type: 'global',
    scope_code: '',
    priority: undefined,
})

const hasPermission = (permission, bit) => {
    return (Number(permission) & bit) === bit
}

// 模型编辑
function handleEditModel() {
    modelEditRef.value.handleEdit(modelData.value)
}

async function handleCopyModelCode() {
    const code = modelData.value.model_code
    if (!code) return
    try {
        await navigator.clipboard.writeText(code)
        message.success(t('component.message.success.copy'))
    } catch (error) {
        // ignore
    }
}

// 加载空间选项
async function loadSpaceOptions() {
    try {
        const { data, success } = await apis.space
            .getSpaceList({
                pageSize: 100,
                current: 1,
            })
            .catch(() => {
                throw new Error()
            })
        if (config('http.code.success') === success) {
            spaceOptions.value = data || []
        }
    } catch (error) {
        // ignore
    }
}

// 模型别名
const aliasSearchName = ref('')
const aliasListData = ref([])
const aliasLoading = ref(false)
const aliasPagination = reactive({
    current: 1,
    pageSize: 10,
    total: 0,
    showSizeChanger: true,
    showTotal: (total) => `共 ${total} 条`,
})

const aliasColumns = [
    {
        title: t('pages.model.alias.form.alias'),
        dataIndex: 'alias',
        width: 200,
    },
    {
        title: t('pages.model.form.description'),
        dataIndex: 'description',
        ellipsis: true,
    },
    {
        title: t('pages.model.form.created_at'),
        key: 'created_at',
        width: 180,
    },
    {
        title: t('button.action'),
        key: 'action',
        width: 120,
    },
]

const policyTypeKeys = ['tagging', 'limit', 'invocation', 'route', 'loadbalance', 'circuit_break']
const policyTypeTabs = computed(() => [
    { key: 'tagging', label: getPolicyTypeName('tagging') },
    { key: 'limit', label: getPolicyTypeName('limit') },
    { key: 'invocation', label: getPolicyTypeName('invocation') },
    { key: 'route', label: getPolicyTypeName('route') },
    { key: 'loadbalance', label: getPolicyTypeName('loadbalance') },
    { key: 'circuit_break', label: getPolicyTypeName('circuit_break') },
])
const currentPolicyType = computed(() => (isPolicyTab(activeTab.value) ? activeTab.value : 'limit'))

function isPolicyTab(tabKey) {
    return policyTypeKeys.includes(tabKey)
}

// 模型策略实例
const modelPoliciesMap = ref({
    tagging: [],
    loadbalance: [],
    route: [],
    limit: [],
    circuit_break: [],
    invocation: [],
})

// 治理策略绑定
const policyLoading = ref(false)
const policyPagination = reactive({
    current: 1,
    pageSize: 10,
    total: 0,
    showSizeChanger: true,
    showTotal: (total) => `共 ${total} 条`,
})

const modelPolicyColumns = [
    {
        title: t('pages.model.policy.form.policy_id'),
        dataIndex: 'name',
        ellipsis: true,
    },
    {
        title: t('pages.model.form.enabled'),
        key: 'enabled',
        width: 100,
    },
    {
        title: t('pages.model.form.description'),
        dataIndex: 'description',
        ellipsis: true,
    },
    {
        title: t('pages.policy.form.scope_type'),
        key: 'scope_type',
        width: 120,
    },
    {
        title: t('pages.policy.form.scope_code'),
        dataIndex: 'scope_code',
        ellipsis: true,
        width: 150,
    },
    {
        title: t('pages.policy.form.priority') || '优先级',
        dataIndex: 'priority',
        width: 80,
    },
    {
        title: t('pages.model.form.created_at'),
        key: 'created_at',
        width: 180,
    },
    {
        title: t('button.action'),
        key: 'action',
        width: 220,
    },
]

const currentModelPolicies = computed(() => modelPoliciesMap.value[currentPolicyType.value] || [])

// 模型选项（用于 endpoint 表单）
const modelOptions = ref([])

// 端点管理
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
        title: t('pages.endpoint.form.code'),
        dataIndex: 'code',
        ellipsis: true,
    },
    {
        title: t('pages.endpoint.form.provider_id'),
        key: 'provider_id',
        width: 150,
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
        title: t('pages.endpoint.recent_status'),
        key: 'recent_status',
        width: 180,
    },
    {
        title: t('pages.endpoint.form.priority'),
        dataIndex: 'priority',
        key: 'priority',
        width: 90,
        sorter: (a, b) => (a.priority ?? 0) - (b.priority ?? 0),
    },
    {
        title: t('pages.endpoint.form.weight'),
        dataIndex: 'weight',
        width: 80,
    },
    {
        title: t('pages.endpoint.form.enabled'),
        key: 'enabled',
        width: 80,
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
        width: 230,
    },
]

// 成员管理
const memberSearchUser = ref('')
const memberListData = ref([])
const memberLoading = ref(false)
const memberPagination = reactive({
    current: 1,
    pageSize: 10,
    total: 0,
    showSizeChanger: true,
    showTotal: (total) => `共 ${total} 条`,
})

const memberColumns = [
    {
        title: t('pages.member.form.user'),
        dataIndex: 'user',
        width: 150,
    },
    {
        title: t('pages.member.form.tenant'),
        dataIndex: 'tenant',
        width: 150,
    },
    {
        title: t('pages.member.form.role'),
        dataIndex: 'role',
        width: 100,
    },
    {
        title: t('pages.member.form.permission'),
        key: 'permission',
        width: 200,
    },
    {
        title: t('pages.member.form.created_at'),
        key: 'created_at',
        width: 180,
    },
    {
        title: t('button.action'),
        key: 'action',
        width: 120,
    },
]

const endpointFilterProviderId = ref(undefined)
const endpointFilterEnabled = ref(undefined)

onMounted(() => {
    loadModelDetail()
    loadAliasList()
    loadModelOptions()
    loadProviderOptions()
    loadMemberList()
    loadModelPolicies()
    loadSpaceOptions()
})

watch(
    () => route.query.tab,
    (tab) => {
        if (tab === 'monitor' || tab === 'endpoint' || tab === 'alias' || tab === 'member' || isPolicyTab(tab)) {
            activeTab.value = tab
        }
    }
)

watch(
    activeTab,
    (tab) => {
        if (tab === 'endpoint') {
            loadEndpointList()
        }
    },
    { immediate: true }
)

watch(
    () => route.params.id,
    (newId) => {
        if (newId && route.name === 'modelDetail') {
            modelId.value = newId
            activeTab.value = route.query.tab === 'monitor' ? 'monitor' : 'endpoint'
            // 重置状态以防残留旧数据
            modelData.value = {}
            aliasListData.value = []
            modelPoliciesMap.value = {
                tagging: [],
                loadbalance: [],
                route: [],
                limit: [],
                circuit_break: [],
                invocation: [],
            }
            endpointListData.value = []
            memberListData.value = []

            aliasPagination.current = 1
            policyPagination.current = 1
            endpointPagination.current = 1
            memberPagination.current = 1

            // 重新加载
            loadModelDetail()
            loadAliasList()
            loadModelOptions()
            loadProviderOptions()
            if (activeTab.value === 'endpoint') {
                loadEndpointList()
            }
            loadMemberList()
            loadModelPolicies()
            loadSpaceOptions()
        }
    }
)

function handleEndpointFilterChange() {
    endpointPagination.current = 1
    loadEndpointList()
}

function filterProviderOption(input, option) {
    return option.children?.[0]?.children?.toLowerCase().includes(input.toLowerCase())
}

function getProviderName(id) {
    if (!id) return '--'
    const p = providerOptions.value.find((item) => item.id === id)
    return p ? p.name : id
}

function goToProviderDetail(id) {
    if (!id) return
    router.push({ name: 'providerDetail', params: { id } })
}

function getInheritedProtocol(providerId) {
    if (!providerId) return ''
    const p = providerOptions.value.find((item) => item.id === providerId)
    return p ? p.protocol : ''
}

async function loadModelDetail() {
    try {
        const { data, success } = await apis.model.getModel(modelId.value)
        if (success) {
            modelData.value = data || {}
            loadModelPolicies()
        }
    } catch (error) {
        // ignore
    }
}

async function loadModelPolicies() {
    if (!modelId.value) {
        return
    }
    try {
        policyLoading.value = true
        const params = {
            pageSize: 1000,
            current: 1,
            model_id: modelId.value,
        }
        const [tg, lb, rt, lim, cb, iv] = await Promise.allSettled([
            apis.policy.getTaggingList(params),
            apis.policy.getLoadbalanceList(params),
            apis.policy.getRouteList(params),
            apis.policy.getLimitList(params),
            apis.policy.getCircuitBreakList(params),
            apis.policy.getInvocationList(params),
        ])
        if (tg.status === 'fulfilled' && tg.value?.success) modelPoliciesMap.value.tagging = tg.value.data || []
        if (lb.status === 'fulfilled' && lb.value?.success) modelPoliciesMap.value.loadbalance = lb.value.data || []
        if (rt.status === 'fulfilled' && rt.value?.success) modelPoliciesMap.value.route = rt.value.data || []
        if (lim.status === 'fulfilled' && lim.value?.success) modelPoliciesMap.value.limit = lim.value.data || []
        if (cb.status === 'fulfilled' && cb.value?.success) modelPoliciesMap.value.circuit_break = cb.value.data || []
        if (iv.status === 'fulfilled' && iv.value?.success) modelPoliciesMap.value.invocation = iv.value.data || []
        policyPagination.total = currentModelPolicies.value.length
    } catch (e) {
        // ignore
    } finally {
        policyLoading.value = false
    }
}

function onPolicyTableChange({ current, pageSize }) {
    policyPagination.current = current
    policyPagination.pageSize = pageSize
    policyPagination.total = currentModelPolicies.value.length
}

function getPolicyTypeName(type) {
    switch (type) {
        case 'tagging':
            return t('pages.dashboard.policies.tagging')
        case 'limit':
            return t('pages.dashboard.policies.limit')
        case 'invocation':
        case 'invoke':
            return t('pages.dashboard.policies.invocation')
        case 'route':
            return t('pages.dashboard.policies.route')
        case 'loadbalance':
        case 'load_balance':
            return t('pages.dashboard.policies.loadbalance')
        case 'circuit_break':
            return t('pages.dashboard.policies.circuitBreak')
        default:
            return type
    }
}

const policyEditRefMap = {
    tagging: taggingEditRef,
    limit: limitEditRef,
    invocation: invocationEditRef,
    route: routeEditRef,
    loadbalance: loadbalanceEditRef,
    circuit_break: circuitBreakEditRef,
}

const policyDeleteApiMap = {
    tagging: apis.policy.delTagging,
    limit: apis.policy.delLimit,
    invocation: apis.policy.delInvocation,
    route: apis.policy.delRoute,
    loadbalance: apis.policy.delLoadbalance,
    circuit_break: apis.policy.delCircuitBreak,
}

const policyToggleEnabledApiMap = {
    tagging: apis.policy.toggleTaggingEnabled,
    limit: apis.policy.toggleLimitEnabled,
    invocation: apis.policy.toggleInvocationEnabled,
    route: apis.policy.toggleRouteEnabled,
    loadbalance: apis.policy.toggleLoadbalanceEnabled,
    circuit_break: apis.policy.toggleCircuitBreakEnabled,
}

const policyListApiMap = {
    tagging: apis.policy.getTaggingList,
    limit: apis.policy.getLimitList,
    invocation: apis.policy.getInvocationList,
    route: apis.policy.getRouteList,
    loadbalance: apis.policy.getLoadbalanceList,
    circuit_break: apis.policy.getCircuitBreakList,
}

const policyCopyApiMap = {
    tagging: apis.policy.copyTaggingToModel,
    limit: apis.policy.copyLimitToModel,
    invocation: apis.policy.copyInvocationToModel,
    route: apis.policy.copyRouteToModel,
    loadbalance: apis.policy.copyLoadbalanceToModel,
    circuit_break: apis.policy.copyCircuitBreakToModel,
}

function handleCreateModelPolicy() {
    policyEditRefMap[currentPolicyType.value]?.value?.handleCreate({
        modelId: modelId.value,
        title: t('pages.model.policy.create'),
    })
}

function handleEditModelPolicy(record) {
    policyEditRefMap[currentPolicyType.value]?.value?.handleEdit(record)
}

const togglingModelPolicies = ref({})

async function handleToggleModelPolicyEnabled(record) {
    if (togglingModelPolicies.value[record.id]) return
    const toggleApi = policyToggleEnabledApiMap[currentPolicyType.value]
    if (!toggleApi) return

    const nextEnabled = record.enabled === 1 ? 0 : 1
    togglingModelPolicies.value[record.id] = true
    try {
        const { success } = await toggleApi(record.id, {
            enabled: nextEnabled,
        }).catch(() => {
            throw new Error()
        })
        if (config('http.code.success') === success) {
            message.success(
                nextEnabled === 1 ? t('pages.endpoint.enable.success') : t('pages.endpoint.disable.success')
            )
            await loadModelPolicies()
        }
    } catch (error) {
        // ignore, error already handled by interceptor
    } finally {
        togglingModelPolicies.value[record.id] = false
    }
}

function filterTemplateOption(input, option) {
    return option.children?.[0]?.children?.toLowerCase().includes(input.toLowerCase())
}

async function handleOpenCopyTemplate() {
    try {
        const listApi = policyListApiMap[currentPolicyType.value]
        const { data, success } = await listApi({
            pageSize: 1000,
            current: 1,
        }).catch(() => {
            throw new Error()
        })
        if (config('http.code.success') === success) {
            templateOptions.value = data || []
        }
        copyTemplateModal.templateId = undefined
        copyTemplateModal.name = ''
        copyTemplateModal.scope_type = 'global'
        copyTemplateModal.scope_code = ''
        copyTemplateModal.priority = undefined
        copyTemplateModal.open = true
    } catch (error) {
        message.error(t('component.message.error.search'))
    }
}

async function handleCopyTemplateToModel() {
    if (!copyTemplateModal.templateId) {
        message.warning(t('pages.model.policy.copyTemplate.templateRequired'))
        return
    }
    try {
        copyTemplateModal.loading = true
        const copyApi = policyCopyApiMap[currentPolicyType.value]
        const { success } = await copyApi(copyTemplateModal.templateId, {
            model_id: modelId.value,
            name: copyTemplateModal.name || undefined,
            scope_type: copyTemplateModal.scope_type || 'global',
            scope_code: copyTemplateModal.scope_type === 'global' ? '' : copyTemplateModal.scope_code || '',
            priority: copyTemplateModal.priority !== undefined ? copyTemplateModal.priority : undefined,
        }).catch(() => {
            throw new Error()
        })
        if (config('http.code.success') === success) {
            message.success(t('component.message.success.save'))
            copyTemplateModal.open = false
            await loadModelPolicies()
        }
    } catch (error) {
        message.error('复制策略失败，请检查模板名称是否冲突')
    } finally {
        copyTemplateModal.loading = false
    }
}

function handleRemoveModelPolicy({ id }) {
    Modal.confirm({
        title: t('button.confirm'),
        okText: t('button.confirm'),
        okType: 'danger',
        onOk: () => {
            return new Promise((resolve, reject) => {
                ;(async () => {
                    try {
                        const deleteApi = policyDeleteApiMap[currentPolicyType.value]
                        const { success } = await deleteApi(id).catch(() => {
                            throw new Error()
                        })
                        if (config('http.code.success') === success) {
                            resolve()
                            message.success(t('component.message.success.delete'))
                            await loadModelPolicies()
                        }
                    } catch (error) {
                        reject()
                    }
                })()
            })
        },
    })
}

async function loadAliasList() {
    try {
        aliasLoading.value = true
        const { data, success, total } = await apis.model_alias
            .getModelAliasList({
                pageSize: aliasPagination.pageSize,
                current: aliasPagination.current,
                model_id: modelId.value,
                alias: aliasSearchName.value || undefined,
            })
            .catch(() => {
                throw new Error()
            })
        aliasLoading.value = false
        if (config('http.code.success') === success) {
            aliasListData.value = data || []
            aliasPagination.total = total || 0
        }
    } catch (error) {
        aliasLoading.value = false
    }
}

function onAliasTableChange({ current, pageSize }) {
    aliasPagination.current = current
    aliasPagination.pageSize = pageSize
    loadAliasList()
}

function handleRemoveAlias({ id }) {
    Modal.confirm({
        title: t('pages.model.alias.delTip'),
        okText: t('button.confirm'),
        okType: 'danger',
        onOk: () => {
            return new Promise((resolve, reject) => {
                ;(async () => {
                    try {
                        const { success } = await apis.model_alias.delModelAlias(id).catch(() => {
                            throw new Error()
                        })
                        if (config('http.code.success') === success) {
                            resolve()
                            message.success(t('component.message.success.delete'))
                            await loadAliasList()
                        }
                    } catch (error) {
                        reject()
                    }
                })()
            })
        },
    })
}

async function loadProviderOptions() {
    try {
        const { data, success } = await apis.provider
            .getProviderList({
                pageSize: 100,
                current: 1,
            })
            .catch(() => {
                throw new Error()
            })
        if (config('http.code.success') === success) {
            providerOptions.value = data || []
        }
    } catch (error) {
        // ignore
    }
}

async function loadModelOptions() {
    try {
        const { data, success } = await apis.model
            .getModelList({
                pageSize: 100,
                current: 1,
            })
            .catch(() => {
                throw new Error()
            })
        if (config('http.code.success') === success) {
            modelOptions.value = data || []
        }
    } catch (error) {
        // ignore
    }
}

async function loadEndpointList() {
    try {
        endpointLoading.value = true
        const { data, success, total } = await apis.endpoint
            .getEndpointList({
                pageSize: endpointPagination.pageSize,
                current: endpointPagination.current,
                model_id: modelId.value,
                provider_id: endpointFilterProviderId.value || undefined,
                enabled: endpointFilterEnabled.value !== undefined ? endpointFilterEnabled.value : undefined,
            })
            .catch(() => {
                throw new Error()
            })
        endpointLoading.value = false
        if (config('http.code.success') === success) {
            endpointListData.value = data || []
            endpointPagination.total = total || 0
        }
    } catch (error) {
        endpointLoading.value = false
    }
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

const togglingEndpoints = ref({})

async function handleToggleEndpointEnabled(record) {
    if (togglingEndpoints.value[record.id]) return
    const nextEnabled = record.enabled === 1 ? 0 : 1
    togglingEndpoints.value[record.id] = true
    try {
        const { success } = await apis.endpoint.toggleEndpointEnabled(record.id, { enabled: nextEnabled }).catch(() => {
            throw new Error()
        })
        if (config('http.code.success') === success) {
            message.success(
                nextEnabled === 1 ? t('pages.endpoint.enable.success') : t('pages.endpoint.disable.success')
            )
            await loadEndpointList()
        }
    } catch (error) {
        // ignore, error already handled by interceptor
    } finally {
        togglingEndpoints.value[record.id] = false
    }
}

function onEndpointTableChange({ current, pageSize }) {
    endpointPagination.current = current
    endpointPagination.pageSize = pageSize
    loadEndpointList()
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

async function loadMemberList() {
    try {
        memberLoading.value = true
        const { data, success, total } = await apis.data_permission
            .getDataPermissionList({
                pageSize: memberPagination.pageSize,
                current: memberPagination.current,
                type: 'model',
                data_id: modelId.value,
                user: memberSearchUser.value || undefined,
            })
            .catch(() => {
                throw new Error()
            })
        memberLoading.value = false
        if (config('http.code.success') === success) {
            memberListData.value = data || []
            memberPagination.total = total || 0
        }
    } catch (error) {
        memberLoading.value = false
    }
}

function onMemberTableChange({ current, pageSize }) {
    memberPagination.current = current
    memberPagination.pageSize = pageSize
    loadMemberList()
}

function handleRemoveMember({ id }) {
    Modal.confirm({
        title: t('pages.member.delTip'),
        okText: t('button.confirm'),
        okType: 'danger',
        onOk: () => {
            return new Promise((resolve, reject) => {
                ;(async () => {
                    try {
                        const { success } = await apis.data_permission.delDataPermission(id).catch(() => {
                            throw new Error()
                        })
                        if (config('http.code.success') === success) {
                            resolve()
                            message.success(t('component.message.success.delete'))
                            await loadMemberList()
                        }
                    } catch (error) {
                        reject()
                    }
                })()
            })
        },
    })
}
</script>

<style lang="less" scoped>
.model-detail {
    height: 100%;
    padding: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
}

.info-card {
    flex: none;
    margin-bottom: 16px;

    :deep(.ant-card-head-title) {
        font-size: 14px;
    }

    :deep(.ant-card-grid) {
        padding: 12px 16px;
        box-shadow:
            1px 0 0 0 rgba(0, 0, 0, 0.06),
            0 1px 0 0 rgba(0, 0, 0, 0.06);

        &:hover {
            box-shadow:
                1px 0 0 0 rgba(0, 0, 0, 0.06),
                0 1px 0 0 rgba(0, 0, 0, 0.06);
        }
    }

    .info-item {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 4px;
        min-width: 0;
    }

    .info-label {
        opacity: 0.5;
        font-size: 12px;
        line-height: 18px;
    }

    .info-value {
        max-width: 100%;
        font-size: 14px;
        font-weight: 500;
        line-height: 22px;
    }

    .info-value-code {
        font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
        font-size: 13px;

        &.is-copyable {
            cursor: pointer;

            &:hover {
                color: var(--ant-color-primary, #1677ff);
            }
        }
    }

    .info-value-ellipsis {
        display: block;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .info-value-price {
        font-variant-numeric: tabular-nums;
    }

    .info-unit {
        margin-left: 4px;
        font-size: 12px;
        font-weight: 400;
        opacity: 0.5;
    }
}

[data-theme='dark'] .info-card {
    :deep(.ant-card-grid) {
        box-shadow:
            1px 0 0 0 rgba(255, 255, 255, 0.08),
            0 1px 0 0 rgba(255, 255, 255, 0.08);

        &:hover {
            box-shadow:
                1px 0 0 0 rgba(255, 255, 255, 0.08),
                0 1px 0 0 rgba(255, 255, 255, 0.08);
        }
    }
}

.detail-card {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;

    :deep(.ant-card-body) {
        flex: 1;
        min-height: 0;
        display: flex;
        flex-direction: column;
        overflow: hidden;
    }

    .detail-tabs {
        flex: none;
        margin-bottom: 0;
        -webkit-user-select: none;
        user-select: none;

        :deep(.ant-tabs-nav) {
            margin-bottom: 16px;
        }
    }
}

.tab-content {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
}

// 监控信息等非表格自适应内容固定的外壳无法自己撑开滚动，需允许整块纵向滚动；
// 内容中 a-row gutter 的负外边距会略微超出容器，横向需裁剪，否则会出现横向滚动条
.tab-content--scroll {
    overflow-y: auto;
    overflow-x: hidden;
}

.tab-toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
}

.tab-toolbar-left {
    display: flex;
    align-items: center;
    gap: 8px;
}

.tab-toolbar-right {
    display: flex;
    align-items: center;
    gap: 8px;
}

.url-text {
    font-family: monospace;
    display: inline-block;
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    vertical-align: middle;
}
</style>
