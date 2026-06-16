package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// ModelAlias defines an alias for a model.
type ModelAlias struct {
	ID          string          `json:"id" gorm:"size:20;primaryKey;<-:create;"`                                                 // Unique ID
	SpaceCode   string          `json:"space_code" gorm:"size:255;not null;uniqueIndex:uniq_code;"`                              // Space code
	Alias       string          `json:"alias" gorm:"size:255;not null;uniqueIndex:uniq_code;"`                                   // Model alias
	ModelId     string          `json:"model_id" gorm:"size:20;not null;"`                                                       // Model ID this alias belongs to
	Description *string         `json:"description,omitempty" gorm:"size:255;"`                                                  // Description
	CreatedAt   time.Time       `json:"created_at" gorm:"type:timestamp;autoCreateTime;"`                                        // Create timestamp
	UpdatedAt   time.Time       `json:"updated_at,omitempty" gorm:"type:timestamp;autoUpdateTime;"`                              // Update timestamp
	Deleted     string          `json:"-" gorm:"uniqueIndex:uniq_code;size:20;default:0;"`                                       // Delete flag
	DeletedAt   *gorm.DeletedAt `json:"-" gorm:"type:datetime;comment:Delete time;"`                                             // Delete time
	Model       *Model          `json:"model,omitempty" gorm:"foreignKey:ModelId;references:ID"`                                 // Model association
}

func (m *ModelAlias) TableName() string {
	return config.C.FormatTableName("model_alias")
}

// ModelAliasQueryParam defines the query parameters for ModelAlias.
type ModelAliasQueryParam struct {
	util.PaginationParam
	SpaceCode string `form:"space_code"` // Space code
	Alias     string `form:"alias"`      // Model alias
	ModelId   string `form:"model_id"`   // Model ID
}

// ModelAliasQueryOptions defines the query options for ModelAlias.
type ModelAliasQueryOptions struct {
	util.QueryOptions
}

// ModelAliasQueryResult defines the query result for ModelAlias.
type ModelAliasQueryResult struct {
	Data       ModelAliases
	PageResult *util.PaginationResult
}

// ModelAliases defines a slice of ModelAlias.
type ModelAliases []*ModelAlias

// ModelAliasForm defines the form for creating/updating a ModelAlias.
type ModelAliasForm struct {
	SpaceCode   string  `json:"space_code" binding:"required,max=255"` // Space code
	Alias       string  `json:"alias" binding:"required,max=255"`      // Model alias
	ModelId     string  `json:"model_id" binding:"required,max=20"`    // Model ID
	Description *string `json:"description"`                           // Description
}

func (m *ModelAliasForm) Validate() error {
	if m.SpaceCode == "" {
		return errors.BadRequest("", "SpaceCode is required")
	}
	if m.Alias == "" {
		return errors.BadRequest("", "Alias is required")
	}
	if m.ModelId == "" {
		return errors.BadRequest("", "ModelId is required")
	}
	return nil
}

func (m *ModelAliasForm) FillTo(alias *ModelAlias) error {
	alias.SpaceCode = m.SpaceCode
	alias.Alias = m.Alias
	alias.ModelId = m.ModelId
	alias.Description = m.Description
	return nil
}
