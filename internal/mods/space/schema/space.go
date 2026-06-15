package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Space management for microservice spaces
type Space struct {
	ID          string     `json:"id" gorm:"size:20;primarykey;"`              // Unique ID
	Code        string     `json:"code" gorm:"size:255;uniqueIndex:uniq_code"` // Code (unique)
	Name        string     `json:"name" gorm:"size:255"`                       // Name
	Tenant      string     `json:"tenant" gorm:"size:255"`                     // Tenant
	Creator     string     `json:"creator" gorm:"size:255"`                    // Creator
	Description string     `json:"description" gorm:"size:255"`                // Description
	Metadata    *string    `json:"metadata,omitempty" gorm:"type:json"`        // Metadata (JSON)
	CreatedAt   time.Time  `json:"created_at" gorm:"index;"`                   // Create time
	UpdatedAt   time.Time  `json:"updated_at" gorm:"index;"`                   // Update time
	Deleted     string     `json:"-" gorm:"size:20;default:0"`                 // Logical delete flag
	DeletedAt   *time.Time `json:"-"`                                          // Delete time
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
