package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Space management for microservice spaces
type Space struct {
	ID          string     `json:"id" gorm:"type:varchar(20);primaryKey;comment:ID;"`
	Code        string     `json:"code" gorm:"type:varchar(255);default:null;uniqueIndex:uniq_code;comment:空间编码;"`
	Name        string     `json:"name" gorm:"type:varchar(255);default:null;comment:空间名称;"`
	Tenant      string     `json:"tenant" gorm:"type:varchar(255);default:null;comment:租户信息;"`
	Creator     string     `json:"creator" gorm:"type:varchar(255);default:null;comment:创建人;"`
	Description string     `json:"description" gorm:"type:varchar(255);default:null;comment:描述;"`
	Metadata    *string    `json:"metadata,omitempty" gorm:"type:json;default:null;comment:元数据;"`
	CreatedAt   time.Time  `json:"created_at" gorm:"type:datetime(3);default:null;autoCreateTime;comment:创建时间;"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"type:datetime(3);default:null;autoUpdateTime;comment:更新时间;"`
	Deleted     string     `json:"-" gorm:"type:varchar(20);default:'0';comment:逻辑删除标识;"`
	DeletedAt   *time.Time `json:"-" gorm:"type:datetime(3);default:null;comment:逻辑删除时间;"`
}

func (a *Space) TableName() string {
	return config.C.FormatTableName("space")
}

// SpaceQueryParam defines the query parameters for Space.
type SpaceQueryParam struct {
	util.PaginationParam
	LikeName string `form:"name"`   // Name (like)
	LikeCode string `form:"code"`   // Code (like)
	Tenant   string `form:"tenant"` // Tenant
}

// SpaceQueryOptions defines the query options for Space.
type SpaceQueryOptions struct {
	util.QueryOptions
}

// SpaceQueryResult defines the query result for Space.
type SpaceQueryResult struct {
	Data       Spaces
	PageResult *util.PaginationResult
}

// Spaces defines a slice of Space.
type Spaces []*Space

// SpaceForm defines the form for creating/updating a Space.
type SpaceForm struct {
	Code        string  `json:"code" binding:"required,max=255"` // Code (unique)
	Name        string  `json:"name" binding:"required,max=255"` // Name
	Description string  `json:"description"`                     // Description
	Metadata    *string `json:"metadata"`                        // Metadata (JSON)
}

func (a *SpaceForm) Validate() error {
	return nil
}

func (a *SpaceForm) FillTo(space *Space) error {
	space.Code = a.Code
	space.Name = a.Name
	space.Description = a.Description
	if a.Metadata != nil && *a.Metadata == "" {
		space.Metadata = nil
	} else {
		space.Metadata = a.Metadata
	}
	return nil
}
