package schema

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/crypto/hash"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

const (
	UserStatusActivated = "activated"
	UserStatusFreezed   = "freezed"
)

// User management for RBAC
type User struct {
	ID        string    `json:"id" gorm:"type:varchar(20);primaryKey;comment:ID;"`
	Username  string    `json:"username" gorm:"type:varchar(64);default:null;index:idx_user_username;comment:用户名;"`
	Name      string    `json:"name" gorm:"type:varchar(64);default:null;index:idx_user_name;comment:用户名称;"`
	Password  string    `json:"-" gorm:"type:varchar(64);default:null;comment:密码;"`
	Phone     string    `json:"phone" gorm:"type:varchar(32);default:null;comment:电话;"`
	Email     string    `json:"email" gorm:"type:varchar(128);default:null;comment:邮件;"`
	Remark    string    `json:"remark" gorm:"type:varchar(1024);default:null;comment:备注;"`
	Tenant    string    `json:"tenant" gorm:"type:varchar(255);default:null;comment:租户信息;"`
	Status    string    `json:"status" gorm:"type:varchar(20);default:null;index:idx_user_status;comment:状态;"`
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime;default:null;autoCreateTime;comment:创建时间;"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:datetime;default:null;autoUpdateTime;comment:更新时间;"`
	Roles     UserRoles `json:"roles" gorm:"-"` // Roles of user
}

func (a *User) TableName() string {
	return config.C.FormatTableName("user")
}

// Defining the query parameters for the `User` struct.
type UserQueryParam struct {
	util.PaginationParam
	LikeUsername string `form:"username"`                                    // Username for login
	LikeName     string `form:"name"`                                        // Name of user
	Tenant       string `form:"tenant"`                                      // Tenant
	Status       string `form:"status" binding:"oneof=activated freezed ''"` // Status of user (activated, freezed)
}

// Defining the query options for the `User` struct.
type UserQueryOptions struct {
	util.QueryOptions
}

// Defining the query result for the `User` struct.
type UserQueryResult struct {
	Data       Users
	PageResult *util.PaginationResult
}

// Defining the slice of `User` struct.
type Users []*User

func (a Users) ToIDs() []string {
	var ids []string
	for _, item := range a {
		ids = append(ids, item.ID)
	}
	return ids
}

// Defining the data structure for creating a `User` struct.
type UserForm struct {
	Username string    `json:"username" binding:"required,max=64"`                // Username for login
	Name     string    `json:"name" binding:"required,max=64"`                    // Name of user
	Password string    `json:"password" binding:"max=64"`                         // Password for login (md5 hash)
	Phone    string    `json:"phone" binding:"max=32"`                            // Phone number of user
	Email    string    `json:"email" binding:"max=128"`                           // Email of user
	Remark   string    `json:"remark" binding:"max=1024"`                         // Remark of user
	Tenant   string    `json:"tenant" binding:"max=255"`                          // Tenant
	Status   string    `json:"status" binding:"required,oneof=activated freezed"` // Status of user (activated, freezed)
	Roles    UserRoles `json:"roles" binding:"required"`                          // Roles of user
}

// A validation function for the `UserForm` struct.
func (a *UserForm) Validate() error {
	if a.Email != "" && validator.New().Var(a.Email, "email") != nil {
		return errors.BadRequest("", "Invalid email address")
	}
	return nil
}

// Convert `UserForm` to `User` object.
func (a *UserForm) FillTo(user *User) error {
	user.Username = a.Username
	user.Name = a.Name
	user.Phone = a.Phone
	user.Email = a.Email
	user.Remark = a.Remark
	user.Tenant = a.Tenant
	user.Status = a.Status

	if pass := a.Password; pass != "" {
		hashPass, err := hash.GeneratePassword(pass)
		if err != nil {
			return errors.BadRequest("", "Failed to generate hash password: %s", err.Error())
		}
		user.Password = hashPass
	}

	return nil
}
