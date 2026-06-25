CREATE DATABASE IF NOT EXISTS tokenlive CHARACTER SET utf8mb4 COLLATE utf8mb4_bin;

CREATE TABLE IF NOT EXISTS `menu`
(
    `id`          varchar(20) NOT NULL COMMENT 'ID',
    `code`        varchar(32)   DEFAULT NULL COMMENT '菜单编码',
    `name`        varchar(128)  DEFAULT NULL COMMENT '菜单名称',
    `description` varchar(1024) DEFAULT NULL COMMENT '描述',
    `sequence`    bigint        DEFAULT NULL COMMENT '序列',
    `type`        varchar(20)   DEFAULT NULL COMMENT '类型: page, button',
    `path`        varchar(255)  DEFAULT NULL COMMENT '路径',
    `properties`  text COMMENT '属性',
    `status`      varchar(20)   DEFAULT NULL COMMENT '状态',
    `parent_id`   varchar(20)   DEFAULT NULL COMMENT '父ID',
    `parent_path` varchar(255)  DEFAULT NULL COMMENT '父路径',
    `created_at`  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`  TIMESTAMP          DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_menu_parent_path` (`parent_path`),
    KEY `idx_menu_name` (`name`),
    KEY `idx_menu_status` (`status`),
    KEY `idx_menu_type` (`type`),
    KEY `idx_menu_parent_id` (`parent_id`),
    KEY `idx_menu_code` (`code`),
    KEY `idx_menu_sequence` (`sequence`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT = '菜单';

CREATE TABLE IF NOT EXISTS `menu_resource`
(
    `id`         varchar(20) NOT NULL COMMENT 'ID',
    `menu_id`    varchar(20)  DEFAULT NULL COMMENT '菜单ID',
    `method`     varchar(20)  DEFAULT NULL COMMENT '请求方法',
    `path`       varchar(255) DEFAULT NULL COMMENT '请求路径',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP          DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_menu_resource_menu_id` (`menu_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT = '菜单资源';

CREATE TABLE IF NOT EXISTS `menu_resource_group`
(
    `id`          varchar(20) NOT NULL COMMENT 'ID',
    `code`        varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '权限编码',
    `name`        varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '权限名称',
    `description` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '权限描述',
    `content`     json                                                   DEFAULT NULL COMMENT '资源列表[menu_id]',
    `order_num`   int                                                    DEFAULT NULL COMMENT '前端展示排序',
    `depends_on`  json                                                   DEFAULT NULL COMMENT '授权依赖其他权限编码,数组结构',
    `base_auth`   tinyint(1)                                             DEFAULT '0' COMMENT '是否基础默认权限,前端默认勾选',
    `created_at`  timestamp   NOT NULL                                   DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`  timestamp                                              DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uniq_code` (`code`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT = '菜单资源分组';

CREATE TABLE IF NOT EXISTS `role`
(
    `id`          varchar(20) NOT NULL COMMENT 'ID',
    `code`        varchar(32)   DEFAULT NULL COMMENT '角色编码',
    `name`        varchar(128)  DEFAULT NULL COMMENT '角色名称',
    `description` varchar(1024) DEFAULT NULL COMMENT '角色描述',
    `sequence`    bigint        DEFAULT NULL COMMENT '角色序列',
    `tenant`      varchar(255)  DEFAULT NULL COMMENT '租户信息',
    `status`      varchar(20)   DEFAULT NULL COMMENT '状态',
    `created_at`  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`  TIMESTAMP          DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_role_code` (`code`),
    KEY `idx_role_name` (`name`),
    KEY `idx_role_sequence` (`sequence`),
    KEY `idx_role_status` (`status`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT = '角色';

CREATE TABLE IF NOT EXISTS `role_menu`
(
    `id`            varchar(20) NOT NULL COMMENT 'ID',
    `role_id`       varchar(20) DEFAULT NULL COMMENT '角色ID',
    `menu_id` varchar(20) DEFAULT NULL COMMENT '菜单组ID',
    `created_at`    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`    TIMESTAMP          DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_role_menu_role_id` (`role_id`),
    KEY `idx_role_menu_menu_id` (`menu_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT = '角色菜单';

CREATE TABLE IF NOT EXISTS `user`
(
    `id`         varchar(20) NOT NULL COMMENT 'ID',
    `username`   varchar(64)   DEFAULT NULL COMMENT '用户名',
    `name`       varchar(64)   DEFAULT NULL COMMENT '用户名称',
    `password`   varchar(64)   DEFAULT NULL COMMENT '密码',
    `phone`      varchar(32)   DEFAULT NULL COMMENT '电话',
    `email`      varchar(128)  DEFAULT NULL COMMENT '邮件',
    `remark`     varchar(1024) DEFAULT NULL COMMENT '备注',
    `tenant`     varchar(255)  DEFAULT NULL COMMENT '租户信息',
    `status`     varchar(20)   DEFAULT NULL COMMENT '状态',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP          DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_user_username` (`username`),
    KEY `idx_user_name` (`name`),
    KEY `idx_user_status` (`status`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT = '用户';

CREATE TABLE IF NOT EXISTS `user_role`
(
    `id`         varchar(20) NOT NULL COMMENT 'ID',
    `user_id`    varchar(20) DEFAULT NULL COMMENT '用户ID',
    `role_id`    varchar(20) DEFAULT NULL COMMENT '角色ID',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP          DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_user_role_user_id` (`user_id`),
    KEY `idx_user_role_role_id` (`role_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT = '用户角色';

CREATE TABLE IF NOT EXISTS `casbin_rule`
(
    `id`    bigint(11) NOT NULL AUTO_INCREMENT COMMENT 'ID',
    `ptype` varchar(100) DEFAULT NULL COMMENT '策略类型（p 或 g）',
    `v0`    varchar(100) DEFAULT NULL COMMENT 'v0',
    `v1`    varchar(100) DEFAULT NULL COMMENT 'v1',
    `v2`    varchar(100) DEFAULT NULL COMMENT 'v2',
    `v3`    varchar(100) DEFAULT NULL COMMENT 'method',
    `v4`    varchar(100) DEFAULT NULL COMMENT 'v4',
    `v5`    varchar(100) DEFAULT NULL COMMENT 'v5',
    PRIMARY KEY (`id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT = '鉴权引擎规则';

-- 模型定义（用户视角）
CREATE TABLE IF NOT EXISTS `model`
(
    `id`                CHAR(20) PRIMARY KEY COMMENT '主键ID (XID)',
    `model_name`        VARCHAR(128)                                           NOT NULL COMMENT '模型名称',
    `model_code`        VARCHAR(64)                                            NOT NULL COMMENT '模型唯一编码',
    `space_code`        VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '模型空间编码',
    `request_types`     JSON                                                   DEFAULT NULL COMMENT '模型支持的请求类型，如 ["chat_completion", "embedding"]',
    `context_length`    BIGINT                                                 NOT NULL DEFAULT 128000 COMMENT '最大上下文窗口（Tokens）',
    `max_output_tokens` BIGINT                                                 NOT NULL DEFAULT 8192 COMMENT '最大输出Token',
    `owner`             VARCHAR(64) COMMENT '模型所属企业/厂商，如 OpenAI, Google, DeepSeek',
    `abilities`         JSON                                                   NULL COMMENT '能力列表,如:流式输出,工具调用,思维链,结构化输出等',
    `enabled`           INT                                                    NOT NULL DEFAULT 0 COMMENT '启用状态: 0-未启用，1-启用',
    `input_price`       DECIMAL(10, 6)                                         NOT NULL DEFAULT 0.002000 COMMENT '输入价格（元/百万 Tokens）',
    `output_price`      DECIMAL(10, 6)                                         NOT NULL DEFAULT 0.002000 COMMENT '输出价格（元/百万 Tokens）',
    `cached_price`      DECIMAL(10, 6)                                         NOT NULL DEFAULT 0.002000 COMMENT '缓存命中价格（元/百万 Tokens）',
    `cache_creation_price` DECIMAL(10, 6)                                      NOT NULL DEFAULT 0.002000 COMMENT '缓存创建价格（元/百万 Tokens）',
    `description`       VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin          DEFAULT NULL COMMENT '备注描述',
    `extra`             JSON                                                   NULL COMMENT '其他信息',
    `creator`           VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin          DEFAULT NULL COMMENT '创建者',
    `modifier`          VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin          DEFAULT NULL COMMENT '修改者',
    `created_at`        TIMESTAMP                                              NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`        TIMESTAMP                                                       DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`           VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin  NOT NULL DEFAULT '0' COMMENT '逻辑删除标识',
    `deleted_at`        DATETIME                                              NULL     DEFAULT NULL COMMENT '逻辑删除时间',
    UNIQUE KEY `uniq_model_name` (`model_name`) USING BTREE,
    UNIQUE KEY `uniq_model_code_deleted` (`model_code`, `deleted`) USING BTREE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT ='模型定义表，用户视角的 LLM 模型';

-- 模型别名
CREATE TABLE IF NOT EXISTS `model_alias`
(
    `id`          VARCHAR(20)  NOT NULL                                              COMMENT 'ID',
    `space_code`  VARCHAR(255) NOT NULL                                              COMMENT '模型空间编码',
    `alias`       VARCHAR(255) NOT NULL                                              COMMENT '模型别名',
    `model_id`    VARCHAR(20)  NOT NULL                                              COMMENT '所属模型ID',
    `description` VARCHAR(255)          DEFAULT NULL                                 COMMENT '备注',
    `created_at`  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP                    COMMENT '创建时间',
    `updated_at`  TIMESTAMP             DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`     VARCHAR(20)  NOT NULL DEFAULT '0'                                  COMMENT '逻辑删除标识',
    `deleted_at`  DATETIME              DEFAULT NULL                                 COMMENT '删除时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uniq_code` (`space_code`, `alias`, `deleted`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT = '模型别名';

-- Provider 定义（上游来源）
CREATE TABLE IF NOT EXISTS `provider`
(
    `id`          CHAR(20) PRIMARY KEY COMMENT '主键ID (XID)',
    `name`        VARCHAR(128)                                          NOT NULL COMMENT 'Provider名称',
    `code`        VARCHAR(128)                                          NOT NULL COMMENT 'Provider唯一标识',
    `protocol`    VARCHAR(64)                                           NOT NULL COMMENT '协议类型，决定使用哪个 ProviderFactory',
    `url`         VARCHAR(512)                                          NULL COMMENT '供应商 API 基础地址',
    `api_keys`    JSON                                                  NULL COMMENT '上游API认证密钥列表',
    `enabled`     INT                                                   NOT NULL DEFAULT 0 COMMENT '启用状态: 0-未启用，1-启用',
    `description` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '备注描述',
    `creator`     VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '创建者',
    `modifier`    VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '修改者',
    `created_at`  TIMESTAMP                                             NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`  TIMESTAMP                                                      DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`     VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '0' COMMENT '逻辑删除标识',
    `deleted_at`  DATETIME                                             NULL     DEFAULT NULL COMMENT '逻辑删除时间',
    UNIQUE KEY `uniq_provider_code` (`code`, `deleted`) USING BTREE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT ='Provider 定义表，上游 LLM 来源';

-- Provider endpoints（一个 provider 可以有多个 endpoint）
CREATE TABLE IF NOT EXISTS `endpoint`
(
    `id`          CHAR(20) PRIMARY KEY COMMENT '主键ID (XID)',
    `model_id`    CHAR(20)                                              NOT NULL COMMENT '关联 of model ID',
    `provider_id` CHAR(20)                                              NOT NULL COMMENT '关联 of provider ID',
    `url`         VARCHAR(512)                                          NOT NULL COMMENT '上游 API 地址',
    `api_key`     VARCHAR(512) COMMENT '可选，覆盖 provider 级别的 api_key',
    `protocol`    VARCHAR(64) COMMENT '可选，覆盖 provider 级别的 protocol',
    `real_model`  VARCHAR(128) COMMENT '可选，覆盖 model 级别的 real_model',
    `priority`    INT                                                   NOT NULL DEFAULT 0 COMMENT '故障转移顺序，数字越小越优先',
    `weight`      INT                                                   NOT NULL DEFAULT 1 COMMENT '负载均衡权重',
    `enabled`     INT                                                   NOT NULL DEFAULT 0 COMMENT '启用状态: 0-未启用，1-启用',
    `headers`     JSON                                                           DEFAULT NULL COMMENT '自定义请求头，如 {"X-Custom-Header": "value"}',
    `metadata`    JSON                                                           DEFAULT NULL COMMENT '元数据，用于存储标签等额外信息',
    `input_price` DECIMAL(10, 6)                                                 DEFAULT NULL COMMENT '输入价格（元/百万 Tokens），NULL表示继承模型',
    `output_price` DECIMAL(10, 6)                                                 DEFAULT NULL COMMENT '输出价格（元/百万 Tokens），NULL表示继承模型',
    `cached_price` DECIMAL(10, 6)                                                 DEFAULT NULL COMMENT '缓存命中价格（元/百万 Tokens），NULL表示继承模型',
    `cache_creation_price` DECIMAL(10, 6)                                          DEFAULT NULL COMMENT '缓存创建价格（元/百万 Tokens），NULL表示继承模型',
    `description` VARCHAR(10240) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '备注描述',
    `creator`     VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '创建者',
    `modifier`    VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '修改者',
    `created_at`  TIMESTAMP                                             NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`  TIMESTAMP                                                      DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`     VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '0' COMMENT '逻辑删除标识',
    `deleted_at`  DATETIME                                             NULL     DEFAULT NULL COMMENT '逻辑删除时间',
    KEY `idx_endpoint_route` (`model_id`, `provider_id`) USING BTREE,
    KEY `idx_provider_id` (`provider_id`, `deleted`),
    KEY `idx_model_id` (`model_id`, `deleted`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT ='Endpoint 定义表，一个 provider 可配置多个上游地址';

-- 策略绑定表（解耦多对多绑定）
CREATE TABLE IF NOT EXISTS `policy_binding`
(
    `id`          CHAR(20) PRIMARY KEY COMMENT '主键ID (XID)',
    `tenant_code` VARCHAR(64)                                           NOT NULL DEFAULT '' COMMENT '租户唯一英文编码，不限则为空字符串',
    `user_id`     CHAR(20)                                              NOT NULL DEFAULT '' COMMENT '用户唯一ID (XID)，不限则为空字符串',
    `model_code`  VARCHAR(64)                                           NOT NULL DEFAULT '' COMMENT '模型唯一编码，不限则为空字符串',
    `policy_type` VARCHAR(64)                                           NOT NULL COMMENT '策略类型：tagging / loadbalance / invocation / limit / route / circuit_break',
    `policy_id`   CHAR(20)                                              NOT NULL COMMENT '关联的具体策略表主键 ID (XID)',
    `priority`    INT                                                   NOT NULL DEFAULT 0 COMMENT '冲突合并时的优先级，数字越小越优先',
    `enabled`     INT                                                   NOT NULL DEFAULT 0 COMMENT '启用状态: 0-未启用，1-启用',
    `description` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '备注描述',
    `creator`     VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '创建者',
    `modifier`    VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '修改者',
    `created_at`  TIMESTAMP                                             NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`  TIMESTAMP                                                      DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`     VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '0' COMMENT '逻辑删除标识',
    `deleted_at`  DATETIME                                             NULL     DEFAULT NULL COMMENT '逻辑删除时间',
    UNIQUE KEY `uniq_dimensions_policy` (`tenant_code`, `user_id`, `model_code`, `policy_type`, `policy_id`, `deleted`),
    KEY `idx_pb_tenant` (`tenant_code`, `deleted`),
    KEY `idx_pb_user` (`user_id`, `deleted`),
    KEY `idx_pb_model` (`model_code`, `deleted`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT ='策略绑定表，管理策略与实体的多对多应用关系';

-- 染色打标策略表
CREATE TABLE IF NOT EXISTS `policy_tagging`
(
    `id`          CHAR(20) PRIMARY KEY COMMENT '主键ID (XID)',
    `name`        VARCHAR(128)                                          NOT NULL COMMENT '策略名称',
    `order`       INT                                                   NOT NULL DEFAULT 0 COMMENT '执行顺序，数字越小越优先',
    `relation`    VARCHAR(16)                                           NOT NULL DEFAULT 'AND' COMMENT '多条件之间的逻辑关系：AND / OR',
    `conditions`  JSON                                                  DEFAULT NULL COMMENT '匹配条件列表，嵌套 Condition 数组',
    `actions`     JSON                                                  DEFAULT NULL COMMENT '染色动作列表，嵌套 TaggingAction 数组',
    `version`     BIGINT                                                NOT NULL DEFAULT 1 COMMENT '配置版本号',
    `enabled`     INT                                                   NOT NULL DEFAULT 0 COMMENT '启用状态: 0-未启用，1-启用',
    `description` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '备注描述',
    `creator`     VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '创建者',
    `modifier`    VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '修改者',
    `created_at`  TIMESTAMP                                             NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`  TIMESTAMP                                                      DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`     VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '0' COMMENT '逻辑删除标识',
    `deleted_at`  DATETIME                                             NULL     DEFAULT NULL COMMENT '逻辑删除时间',
    UNIQUE KEY `uniq_policy_tagging_name` (`name`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT ='染色打标策略表';

-- 负载均衡策略表
CREATE TABLE IF NOT EXISTS `policy_loadbalance`
(
    `id`          CHAR(20) PRIMARY KEY COMMENT '主键ID (XID)',
    `name`        VARCHAR(128)                                          NOT NULL COMMENT '策略名称',
    `type`        VARCHAR(64)                                           NOT NULL COMMENT '负载均衡算法类型，如 ROUND_ROBIN / WEIGHTED / STICKY',
    `version`     BIGINT                                                NOT NULL DEFAULT 1 COMMENT '配置版本号',
    `enabled`     INT                                                   NOT NULL DEFAULT 0 COMMENT '启用状态: 0-未启用，1-启用',
    `params`      JSON                                                           DEFAULT NULL COMMENT '算法额外参数，如权重等',
    `description` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '备注描述',
    `creator`     VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '创建者',
    `modifier`    VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '修改者',
    `created_at`  TIMESTAMP                                             NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`  TIMESTAMP                                                      DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`     VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '0' COMMENT '逻辑删除标识',
    `deleted_at`  DATETIME                                             NULL     DEFAULT NULL COMMENT '逻辑删除时间',
    UNIQUE KEY `uniq_policy_loadbalance_name` (`name`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT ='负载均衡策略表';

-- 调用与重试策略表
CREATE TABLE IF NOT EXISTS `policy_invocation`
(
    `id`              CHAR(20) PRIMARY KEY COMMENT '主键ID (XID)',
    `name`            VARCHAR(128)                                          NOT NULL COMMENT '策略名称',
    `type`            VARCHAR(64)                                           NOT NULL DEFAULT 'failover' COMMENT '调用类型：failover,failfast',
    `retry_policy`    JSON                                                           DEFAULT NULL COMMENT '重试策略',
    `fallback_policy` JSON                                                           DEFAULT NULL COMMENT '降级策略',
    `version`         BIGINT                                                NOT NULL DEFAULT 1 COMMENT '配置版本号',
    `enabled`         INT                                                   NOT NULL DEFAULT 0 COMMENT '启用状态: 0-未启用，1-启用',
    `description`     VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '备注描述',
    `creator`         VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '创建者',
    `modifier`        VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '修改者',
    `created_at`      TIMESTAMP                                             NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`      TIMESTAMP                                                      DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`         VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '0' COMMENT '逻辑删除标识',
    `deleted_at`      DATETIME                                             NULL     DEFAULT NULL COMMENT '逻辑删除时间',
    UNIQUE KEY `uniq_policy_invocation_name` (`name`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT ='调用与重试策略表';

-- 限流策略表
CREATE TABLE IF NOT EXISTS `policy_limit`
(
    `id`              CHAR(20) PRIMARY KEY COMMENT '主键ID (XID)',
    `name`            VARCHAR(128)                                          NOT NULL COMMENT '策略名称',
    `version`         BIGINT                                                NOT NULL DEFAULT 1 COMMENT '配置版本号',
    `type`            VARCHAR(64)                                           NOT NULL COMMENT '限流维度：request / token / cost',
    `max_wait_ms`     INT                                                   NOT NULL DEFAULT 0 COMMENT '排队等待最大时间（毫秒）',
    `relation_type`   VARCHAR(16)                                           NOT NULL DEFAULT 'AND' COMMENT '多条件之间的逻辑关系：AND / OR',
    `sliding_windows` JSON                                                  DEFAULT NULL COMMENT '滑动窗口配额配置列表，嵌套 SlidingWindow 数组',
    `conditions`      JSON COMMENT '匹配条件列表，嵌套 Condition 数组',
    `estimator`       JSON COMMENT '估算器配置，包含 type 和 ratio',
    `enabled`         INT                                                   NOT NULL DEFAULT 0 COMMENT '启用状态: 0-未启用，1-启用',
    `description`     VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '备注描述',
    `creator`         VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '创建者',
    `modifier`        VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '修改者',
    `created_at`      TIMESTAMP                                             NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`      TIMESTAMP                                                      DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`         VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '0' COMMENT '逻辑删除标识',
    `deleted_at`      DATETIME                                             NULL     DEFAULT NULL COMMENT '逻辑删除时间',
    UNIQUE KEY `uniq_policy_limit_name` (`name`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT ='限流策略表';

-- 标签路由策略表
CREATE TABLE IF NOT EXISTS `policy_route`
(
    `id`          CHAR(20) PRIMARY KEY COMMENT '主键ID (XID)',
    `name`        VARCHAR(128)                                          NOT NULL COMMENT '策略名称',
    `order`       INT                                                   NOT NULL DEFAULT 0 COMMENT '执行顺序，数字越小越优先',
    `version`     BIGINT                                                NOT NULL DEFAULT 1 COMMENT '配置版本号',
    `enabled`     INT                                                   NOT NULL DEFAULT 0 COMMENT '启用状态: 0-未启用，1-启用',
    `description` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '备注描述',
    `creator`     VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '创建者',
    `modifier`    VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '修改者',
    `created_at`  TIMESTAMP                                             NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`  TIMESTAMP                                                      DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`     VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '0' COMMENT '逻辑删除标识',
    `deleted_at`  DATETIME                                             NULL     DEFAULT NULL COMMENT '逻辑删除时间',
    UNIQUE KEY `uniq_policy_route_name` (`name`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT ='标签路由策略表';

CREATE TABLE IF NOT EXISTS `policy_route_detail`
(
    `id`            varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT 'ID',
    `route_id`      varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '路由ID',
    `relation_type` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '关系类型',
    `conditions`    json                                                           DEFAULT NULL COMMENT '匹配条件',
    `destinations`  json                                                           DEFAULT NULL COMMENT '目的规则',
    `order`         int                                                   NOT NULL DEFAULT '0' COMMENT '排序值',
    `enabled`       int                                                   NOT NULL DEFAULT '0' COMMENT '启用',
    `description`   varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '备注',
    `created_at`    timestamp                                             NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`    timestamp                                                      DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`       varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '0' COMMENT '逻辑删除标识',
    `deleted_at`    DATETIME                                             NULL     DEFAULT NULL COMMENT '删除时间',
    PRIMARY KEY (`id`),
    KEY `idx_routeid` (`route_id`, `deleted`) USING BTREE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT = '标签路由详情';

-- 熔断隔离策略表
CREATE TABLE IF NOT EXISTS `policy_circuit_break`
(
    `id`                               CHAR(20) PRIMARY KEY COMMENT '主键ID (XID)',
    `name`                             VARCHAR(128)                                          NOT NULL COMMENT '策略名称',
    `level`                            VARCHAR(64)                                           NOT NULL DEFAULT 'INSTANCE' COMMENT '熔断隔离级别：SERVICE / INSTANCE',
    `sliding_window_type`              VARCHAR(16)                                           NOT NULL DEFAULT 'time' COMMENT '滑动窗口类型：time / count',
    `sliding_window_size`              INT                                                   NOT NULL DEFAULT 20 COMMENT '滑动窗口大小（次数或秒数）',
    `min_calls_threshold`              INT                                                   NOT NULL DEFAULT 5 COMMENT '熔断计算的最小调用次数',
    `failure_rate_threshold`           DECIMAL(5, 2)                                         NOT NULL DEFAULT 50.00 COMMENT '失败率阈值百分比',
    `slow_call_rate_threshold`         DECIMAL(5, 2) COMMENT '慢调用率阈值百分比',
    `slow_call_duration_threshold`     INT COMMENT '慢调用时长阈值（毫秒）',
    `slow_call_metric`                 VARCHAR(32)                                                    DEFAULT NULL COMMENT '慢调用衡量指标：TTFT 等',
    `wait_duration_in_open_state`      INT                                                   NOT NULL DEFAULT 10000 COMMENT '熔断器开启状态持续时间（毫秒）',
    `allowed_calls_in_half_open_state` INT                                                   NOT NULL DEFAULT 3 COMMENT '半开状态下允许的试探调用次数',
    `force_open`                       TINYINT(1)                                            NOT NULL DEFAULT 0 COMMENT '是否强制开启熔断：0-否，1-是',
    `outlier_max_percent`              INT                                                   NOT NULL DEFAULT 10 COMMENT '最大熔断实例比例百分比(对 INSTANCE 级有效)',
    `code_policy`                      JSON                                                           DEFAULT NULL COMMENT '响应状态码提取解析策略',
    `error_codes`                      JSON                                                           DEFAULT NULL COMMENT '触发熔断的异常状态码列表',
    `message_policy`                   JSON                                                           DEFAULT NULL COMMENT '错误消息提取解析策略',
    `error_messages`                   JSON                                                           DEFAULT NULL COMMENT '错误消息列表',
    `degrade_config`                   JSON                                                           DEFAULT NULL COMMENT '熔断降级响应配置，嵌套 DegradeConfig 结构',
    `version`                          BIGINT                                                NOT NULL DEFAULT 1 COMMENT '策略版本',
    `enabled`                          INT                                                   NOT NULL DEFAULT 0 COMMENT '启用状态: 0-未启用，1-启用',
    `description`                      VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '备注描述',
    `creator`                          VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '创建者',
    `modifier`                         VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '修改者',
    `created_at`                       TIMESTAMP                                             NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`                       TIMESTAMP                                                      DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`                          VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '0' COMMENT '逻辑删除标识',
    `deleted_at`                       DATETIME                                             NULL     DEFAULT NULL COMMENT '逻辑删除时间',
    UNIQUE KEY `uniq_policy_circuit_break_name` (`name`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT ='熔断隔离策略表';

-- =============================================================================
-- 用户 API Key 存储表，用于客户端下游认证与计量
-- =============================================================================
CREATE TABLE IF NOT EXISTS `user_api_key`
(
    `id`          CHAR(20) PRIMARY KEY COMMENT '主键ID (XID)',
    `user_id`     CHAR(20)                                              NOT NULL COMMENT '关联的用户 ID',
    `name`        VARCHAR(64)                                           NOT NULL COMMENT 'API Key 友好名称',
    `api_key`     VARCHAR(128)                                          NOT NULL COMMENT '实际的 API Key 字符串',
    `status`      INT                                                   NOT NULL DEFAULT 1 COMMENT '状态: 1-启用, 2-禁用',
    `quota`       BIGINT                                                NOT NULL DEFAULT -1 COMMENT '剩余配额: -1表示无限制',
    `expires_at`  DATETIME                                             NULL     DEFAULT NULL COMMENT '过期时间: NULL表示永不过期',
    -- 审计与管理字段
    `description` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '备注描述',
    `creator`     VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '创建者',
    `modifier`    VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '修改者',
    `created_at`  TIMESTAMP                                             NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`  TIMESTAMP                                                      DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`     VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '0' COMMENT '逻辑删除标识',
    `deleted_at`  DATETIME                                             NULL     DEFAULT NULL COMMENT '逻辑删除时间',
    UNIQUE KEY `uniq_api_key_deleted` (`api_key`, `deleted`) USING BTREE,
    KEY `idx_user_id` (`user_id`, `deleted`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT ='用户 API Key 存储表，用于客户端下游认证';

-- =============================================================================
-- 租户信息表，用于管理系统租户及多租户业务分区
-- =============================================================================
CREATE TABLE IF NOT EXISTS `tenant`
(
    `id`          CHAR(20) PRIMARY KEY COMMENT '主键ID (XID)',
    `code`        VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin  NOT NULL COMMENT '租户唯一英文编码，如 default、company-a',
    `name`        VARCHAR(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '租户名称，如 默认租户、演示组织',
    `status`      VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin  NOT NULL DEFAULT 'activated' COMMENT '状态: activated-启用, freezed-冻结',
    `api_key`     VARCHAR(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin          DEFAULT NULL COMMENT '租户 API Key (toB场景专属)',
    -- 审计与管理字段
    `description` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin          DEFAULT NULL COMMENT '备注描述',
    `creator`     VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin          DEFAULT NULL COMMENT '创建者',
    `modifier`    VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin          DEFAULT NULL COMMENT '修改者',
    `created_at`  TIMESTAMP                                              NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`  TIMESTAMP                                                       DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`     VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin  NOT NULL DEFAULT '0' COMMENT '逻辑删除标识',
    `deleted_at`  DATETIME                                              NULL     DEFAULT NULL COMMENT '逻辑删除时间',
    UNIQUE KEY `uniq_tenant_code_deleted` (`code`, `deleted`) USING BTREE,
    UNIQUE KEY `uniq_tenant_api_key` (`api_key`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT ='租户信息表';

-- =============================================================================
-- 租户大模型授权关联表，用于多租户视角下控制租户可访问的模型集
-- =============================================================================
CREATE TABLE IF NOT EXISTS `tenant_model`
(
    `id`          CHAR(20) PRIMARY KEY COMMENT '主键ID (XID)',
    `tenant_code` VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '租户唯一英文编码，关联 tenant.code',
    `model_id`    CHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin    NOT NULL COMMENT '模型ID，关联 model.id',
    `creator`     VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '创建者',
    `created_at`  TIMESTAMP                                             NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`  TIMESTAMP                                                      DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    UNIQUE KEY `uniq_tenant_model` (`tenant_code`, `model_id`) USING BTREE,
    KEY `idx_tenant_model_tenant_code` (`tenant_code`),
    KEY `idx_tenant_model_model_id` (`model_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT ='租户与模型授权关联表';

-- =============================================================================
-- 租户与端点关联表，用于多租户视角下控制租户在特定模型下能访问的端点
-- =============================================================================
CREATE TABLE IF NOT EXISTS `tenant_endpoint`
(
    `id`          CHAR(20) PRIMARY KEY COMMENT '主键ID (XID)',
    `tenant_code` VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '租户唯一英文编码，关联 tenant.code',
    `endpoint_id` CHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin    NOT NULL COMMENT '端点ID，关联 endpoint.id',
    `creator`     VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin         DEFAULT NULL COMMENT '创建者',
    `created_at`  TIMESTAMP                                             NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`  TIMESTAMP                                                      DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    UNIQUE KEY `uniq_tenant_endpoint` (`tenant_code`, `endpoint_id`) USING BTREE,
    KEY `idx_te_tenant_code` (`tenant_code`),
    KEY `idx_te_endpoint_id` (`endpoint_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT ='租户与端点关联表，用于端点级别的访问控制';

CREATE TABLE IF NOT EXISTS `data_permission`
(
    `id`         VARCHAR(20)  NOT NULL                                              COMMENT 'ID',
    `type`       VARCHAR(50)  NOT NULL                                              COMMENT '数据类型(表名)',
    `data_id`    VARCHAR(20)  NOT NULL                                              COMMENT '数据ID',
    `user`       VARCHAR(50)  NOT NULL                                              COMMENT '用户',
    `tenant`     VARCHAR(50)  NOT NULL                                              COMMENT '租户',
    `role`       VARCHAR(20)  NOT NULL                                              COMMENT '角色编码',
    `permission` INT UNSIGNED NOT NULL DEFAULT 0                                    COMMENT '数据权限位 - 格式(read,write,delete)',
    `creator`    VARCHAR(255) NOT NULL DEFAULT ''                                   COMMENT '创建者',
    `created_at` TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP                    COMMENT '创建时间',
    `updated_at` TIMESTAMP             DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`    VARCHAR(20)  NOT NULL DEFAULT '0'                                  COMMENT '逻辑删除标识',
    `deleted_at` DATETIME              DEFAULT NULL                                 COMMENT '删除时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uniq_data_permission` (`type`, `data_id`, `user`, `tenant`, `role`, `deleted`),
    KEY `idx_type` (`type`),
    KEY `idx_data_id` (`data_id`),
    KEY `idx_user` (`user`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT = '数据权限';

CREATE TABLE IF NOT EXISTS `space`
(
    `id`          VARCHAR(20)  NOT NULL                                              COMMENT 'ID',
    `code`        VARCHAR(255) NOT NULL                                              COMMENT '空间编码',
    `name`        VARCHAR(255) NOT NULL                                              COMMENT '空间名称',
    `tenant`      VARCHAR(255) NOT NULL DEFAULT ''                                   COMMENT '租户信息',
    `creator`     VARCHAR(255) NOT NULL DEFAULT ''                                   COMMENT '创建人',
    `description` VARCHAR(255) NOT NULL DEFAULT ''                                   COMMENT '描述',
    `metadata`    JSON                    DEFAULT NULL                               COMMENT '元数据',
    `created_at`  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP                    COMMENT '创建时间',
    `updated_at`  TIMESTAMP             DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted`     VARCHAR(20)  NOT NULL DEFAULT '0'                                  COMMENT '逻辑删除标识',
    `deleted_at`  DATETIME              DEFAULT NULL                                 COMMENT '逻辑删除时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uniq_code` (`code`),
    KEY `idx_space_created_at` (`created_at`),
    KEY `idx_space_updated_at` (`updated_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT = '空间管理';

-- 运维事件日志
CREATE TABLE IF NOT EXISTS `event_log`
(
    `id`            varchar(20)   NOT NULL COMMENT 'ID',
    `event_type`    varchar(32)   NOT NULL COMMENT '事件类型',
    `tenant_code`   varchar(64)   NOT NULL DEFAULT '' COMMENT '租户编码',
    `model_code`    varchar(64)   NOT NULL DEFAULT '' COMMENT '模型编码',
    `endpoint_id`   varchar(20)   NOT NULL DEFAULT '' COMMENT '端点ID',
    `endpoint_code` varchar(128)  NOT NULL DEFAULT '' COMMENT '端点Code',
    `provider_name` varchar(128)  NOT NULL DEFAULT '' COMMENT '供应商名称',
    `policy_id`     varchar(20)   NOT NULL DEFAULT '' COMMENT '策略ID',
    `policy_name`   varchar(128)  NOT NULL DEFAULT '' COMMENT '策略名称',
    `threshold`     decimal(10,2)          DEFAULT NULL COMMENT '阈值',
    `current_value` decimal(10,2)          DEFAULT NULL COMMENT '触发时当前值',
    `request_id`    varchar(64)   NOT NULL DEFAULT '' COMMENT '请求ID',
    `trace_id`      varchar(64)   NOT NULL DEFAULT '' COMMENT '追踪ID',
    `message`       varchar(10240) NOT NULL DEFAULT '' COMMENT '人机消息',
    `event_time`    datetime      NOT NULL COMMENT '事件触发时间',
    `created_at`    timestamp     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '入库时间',
    PRIMARY KEY (`id`),
    KEY `idx_el_event_type` (`event_type`),
    KEY `idx_el_tenant` (`tenant_code`),
    KEY `idx_el_model` (`model_code`),
    KEY `idx_el_endpoint` (`endpoint_id`),
    KEY `idx_el_endpoint_code` (`endpoint_code`),
    KEY `idx_el_policy` (`policy_id`),
    KEY `idx_el_time` (`event_time`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin COMMENT = '运维事件日志';