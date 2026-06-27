package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// AuditLog 结构化审计日志表，记录业务操作的变更前后数据。
// 与现有 Logger 表互补：Logger 偏开发调试（level, stack, message），AuditLog 偏业务审计（action, resource, before/after）。
type AuditLog struct {
	ID           string          `json:"id" gorm:"type:char(20);primaryKey;<-:create;comment:主键ID (XID);"`
	TenantCode   string          `json:"tenant_code" gorm:"type:varchar(64);default:null;index:idx_al_tenant_created,priority:1;comment:租户编码;"`
	ActorUserID  string          `json:"actor_user_id" gorm:"type:varchar(20);default:null;index:idx_al_actor_created,priority:1;comment:操作人用户ID;"`
	ActorName    string          `json:"actor_name" gorm:"type:varchar(128);default:null;comment:操作人名称（冗余快照）;"`
	Action       string          `json:"action" gorm:"type:varchar(96);not null;index:idx_al_action;comment:操作动作，如 create, update, delete, enable, disable;"`
	ResourceType string          `json:"resource_type" gorm:"type:varchar(64);not null;index:idx_al_resource,priority:1;comment:资源类型，如 model, endpoint, provider, policy;"`
	ResourceID   string          `json:"resource_id" gorm:"type:varchar(64);not null;index:idx_al_resource,priority:2;comment:资源ID;"`
	ResourceName string          `json:"resource_name" gorm:"type:varchar(255);default:null;comment:资源名称（冗余快照）;"`
	BeforeData   *string         `json:"before_data,omitempty" gorm:"type:json;default:null;comment:变更前数据快照;"`
	AfterData    *string         `json:"after_data,omitempty" gorm:"type:json;default:null;comment:变更后数据快照;"`
	IP           string          `json:"ip" gorm:"type:varchar(64);not null;default:'';comment:请求IP;"`
	UserAgent    string          `json:"user_agent" gorm:"type:varchar(512);not null;default:'';comment:User-Agent;"`
	TraceID      string          `json:"trace_id" gorm:"type:varchar(64);default:null;index:idx_al_trace_id;comment:链路追踪ID;"`
	Message      string          `json:"message" gorm:"type:varchar(1024);default:null;comment:可读描述;"`
	CreatedAt    time.Time       `json:"created_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;autoCreateTime;index:idx_al_tenant_created,priority:2;index:idx_al_actor_created,priority:2;comment:创建时间;"`
}

func (AuditLog) TableName() string {
	return config.C.FormatTableName("audit_log")
}

// AuditLogQueryParam defines the query parameters for AuditLog.
type AuditLogQueryParam struct {
	util.PaginationParam
	TenantCode   string `form:"tenant_code"`   // Filter by tenant
	ActorUserID  string `form:"actor_user_id"` // Filter by actor
	Action       string `form:"action"`        // Filter by action
	ResourceType string `form:"resource_type"` // Filter by resource type
	ResourceID   string `form:"resource_id"`   // Filter by resource ID
	TraceID      string `form:"trace_id"`      // Filter by trace ID
	StartTime    string `form:"start_time"`    // Filter by start time
	EndTime      string `form:"end_time"`      // Filter by end time
}

// AuditLogQueryOptions defines the query options for AuditLog.
type AuditLogQueryOptions struct {
	util.QueryOptions
}

// AuditLogQueryResult defines the query result for AuditLog.
type AuditLogQueryResult struct {
	Data       AuditLogs
	PageResult *util.PaginationResult
}

// AuditLogs defines a slice of AuditLog.
type AuditLogs []*AuditLog

// AuditLogForm defines the form for creating an AuditLog.
type AuditLogForm struct {
	TenantCode   string          `json:"tenant_code" binding:"max=64"`     // Tenant code
	ActorUserID  string          `json:"actor_user_id" binding:"max=20"`  // Actor user ID
	ActorName    string          `json:"actor_name" binding:"max=128"`    // Actor name
	Action       string          `json:"action" binding:"required,max=96"` // Action
	ResourceType string          `json:"resource_type" binding:"required,max=64"` // Resource type
	ResourceID   string          `json:"resource_id" binding:"required,max=64"`   // Resource ID
	ResourceName string          `json:"resource_name" binding:"max=255"` // Resource name
	BeforeData   *string         `json:"before_data"`                     // Before data
	AfterData    *string         `json:"after_data"`                      // After data
	IP           string          `json:"ip" binding:"max=64"`             // IP
	UserAgent    string          `json:"user_agent" binding:"max=512"`    // User agent
	TraceID      string          `json:"trace_id" binding:"max=64"`       // Trace ID
	Message      string          `json:"message" binding:"max=1024"`      // Message
}

func (a *AuditLogForm) Validate() error {
	return nil
}

func (a *AuditLogForm) FillTo(log *AuditLog) error {
	log.TenantCode = a.TenantCode
	log.ActorUserID = a.ActorUserID
	log.ActorName = a.ActorName
	log.Action = a.Action
	log.ResourceType = a.ResourceType
	log.ResourceID = a.ResourceID
	log.ResourceName = a.ResourceName
	log.BeforeData = a.BeforeData
	log.AfterData = a.AfterData
	log.IP = a.IP
	log.UserAgent = a.UserAgent
	log.TraceID = a.TraceID
	log.Message = a.Message
	return nil
}

// AuditAction 常用审计动作常量
const (
	AuditActionCreate = "create"
	AuditActionUpdate = "update"
	AuditActionDelete = "delete"
	AuditActionEnable = "enable"
	AuditActionDisable = "disable"
	AuditActionPublish = "publish"
	AuditActionLogin = "login"
	AuditActionLogout = "logout"
)

// AuditResourceType 常用资源类型常量
const (
	AuditResourceTypeModel         = "model"
	AuditResourceTypeEndpoint      = "endpoint"
	AuditResourceTypeProvider      = "provider"
	AuditResourceTypeModelCatalog  = "model_catalog"
	AuditResourceTypePriceVersion  = "price_version"
	AuditResourceTypePolicy        = "policy"
	AuditResourceTypeTenant        = "tenant"
	AuditResourceTypeUser          = "user"
	AuditResourceTypeAPIKey        = "api_key"
	AuditResourceTypeWorkspace     = "workspace"
)
