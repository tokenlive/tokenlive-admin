package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
)

const (
	OAuthProviderGoogle = "google"
	OAuthProviderGitHub = "github"
)

type ExternalIdentity struct {
	ID             string    `json:"id" gorm:"type:varchar(20);primaryKey;comment:ID;"`
	UserID         string    `json:"user_id" gorm:"type:varchar(20);not null;index:idx_external_identity_user_id;comment:用户ID;"`
	Provider       string    `json:"provider" gorm:"type:varchar(32);not null;uniqueIndex:uniq_external_identity_provider_user,priority:1;comment:身份提供方;"`
	ProviderUserID string    `json:"provider_user_id" gorm:"type:varchar(128);not null;uniqueIndex:uniq_external_identity_provider_user,priority:2;comment:身份提供方用户ID;"`
	Email          string    `json:"email" gorm:"type:varchar(128);default:null;index:idx_external_identity_email;comment:邮箱;"`
	EmailVerified  bool      `json:"email_verified" gorm:"type:boolean;not null;default:false;comment:邮箱是否已验证;"`
	DisplayName    string    `json:"display_name" gorm:"type:varchar(128);default:null;comment:显示名称;"`
	AvatarURL      string    `json:"avatar_url" gorm:"type:varchar(512);default:null;comment:头像地址;"`
	CreatedAt      time.Time `json:"created_at" gorm:"type:datetime;default:null;autoCreateTime;comment:创建时间;"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"type:datetime;default:null;autoUpdateTime;comment:更新时间;"`
}

func (a *ExternalIdentity) TableName() string {
	return config.C.FormatTableName("external_identity")
}

type OAuthProfile struct {
	Provider       string
	ProviderUserID string
	Email          string
	EmailVerified  bool
	DisplayName    string
	AvatarURL      string
}

type OAuthResolveResult struct {
	User     *User
	Tenant   *Tenant
	Identity *ExternalIdentity
	Created  bool
}

type OAuthProvider struct {
	Provider string `json:"provider"`
	LoginURL string `json:"login_url"`
}

type OAuthExchangeForm struct {
	Ticket string `json:"ticket" binding:"required"`
}
