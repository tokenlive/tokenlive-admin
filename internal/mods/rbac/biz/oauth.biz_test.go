package biz

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

func setupOAuthTest(t *testing.T) (*OAuth, *gorm.DB) {
	t.Helper()
	config.C = new(config.Config)
	config.C.Storage.DB.TablePrefix = ""
	config.C.OAuth.Enabled = true
	config.C.OAuth.AllowSignup = true
	config.C.OAuth.AllowedDomains = []string{"company.com"}

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", t.Name())), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		new(schema.ExternalIdentity),
		new(schema.User),
		new(schema.UserRole),
		new(schema.Tenant),
		new(schema.TenantModel),
		new(schema.Role),
	))

	return &OAuth{
		Trans:               &util.Trans{DB: db},
		ExternalIdentityDAL: &dal.ExternalIdentity{DB: db},
		UserDAL:             &dal.User{DB: db},
		UserRoleDAL:         &dal.UserRole{DB: db},
		TenantDAL:           &dal.Tenant{DB: db},
		TenantModelDAL:      &dal.TenantModel{DB: db},
		RoleDAL:             &dal.Role{DB: db},
	}, db
}

func TestOAuthResolveProfileRejectsSignupWhenEmailDomainNotAllowed(t *testing.T) {
	oauth, db := setupOAuthTest(t)

	_, err := oauth.ResolveProfile(context.Background(), schema.OAuthProfile{
		Provider:       schema.OAuthProviderGoogle,
		ProviderUserID: "google-user-1",
		Email:          "alice@example.com",
		EmailVerified:  true,
		DisplayName:    "Alice",
	})

	require.Error(t, err)

	var users int64
	require.NoError(t, db.Model(new(schema.User)).Count(&users).Error)
	require.Equal(t, int64(0), users)
}

func TestOAuthResolveProfileRejectsSignupWhenAllowlistIsEmpty(t *testing.T) {
	oauth, db := setupOAuthTest(t)
	config.C.OAuth.AllowedDomains = nil
	config.C.OAuth.AllowedEmails = nil

	_, err := oauth.ResolveProfile(context.Background(), schema.OAuthProfile{
		Provider:       schema.OAuthProviderGoogle,
		ProviderUserID: "google-user-1",
		Email:          "alice@company.com",
		EmailVerified:  true,
		DisplayName:    "Alice",
	})

	require.Error(t, err)

	var users int64
	require.NoError(t, db.Model(new(schema.User)).Count(&users).Error)
	require.Equal(t, int64(0), users)
}

func TestOAuthResolveProfileCreatesUserTenantAndExternalIdentity(t *testing.T) {
	oauth, db := setupOAuthTest(t)

	result, err := oauth.ResolveProfile(context.Background(), schema.OAuthProfile{
		Provider:       schema.OAuthProviderGitHub,
		ProviderUserID: "github-user-1",
		Email:          "alice@company.com",
		EmailVerified:  true,
		DisplayName:    "Alice Dev",
		AvatarURL:      "https://avatars.example/alice.png",
	})

	require.NoError(t, err)
	require.False(t, result.User.CreatedAt.After(time.Now()))
	require.Equal(t, "alice@company.com", result.User.Username)
	require.Equal(t, "alice@company.com", result.User.Email)
	require.Equal(t, "Alice Dev", result.User.Name)
	require.Empty(t, result.User.Password)
	require.Equal(t, schema.UserStatusActivated, result.User.Status)
	require.Equal(t, "u-"+result.User.ID, result.Tenant.Code)
	require.Equal(t, result.Tenant.Code, result.User.Tenant)

	var identity schema.ExternalIdentity
	require.NoError(t, db.Where("provider = ? AND provider_user_id = ?", schema.OAuthProviderGitHub, "github-user-1").First(&identity).Error)
	require.Equal(t, result.User.ID, identity.UserID)
	require.Equal(t, "alice@company.com", identity.Email)
	require.True(t, identity.EmailVerified)

	again, err := oauth.ResolveProfile(context.Background(), schema.OAuthProfile{
		Provider:       schema.OAuthProviderGitHub,
		ProviderUserID: "github-user-1",
		Email:          "alice@company.com",
		EmailVerified:  true,
		DisplayName:    "Alice Dev",
	})
	require.NoError(t, err)
	require.Equal(t, result.User.ID, again.User.ID)

	var users int64
	require.NoError(t, db.Model(new(schema.User)).Count(&users).Error)
	require.Equal(t, int64(1), users)
}

func TestBuildTicketRedirectURLSupportsHashRouter(t *testing.T) {
	redirectURL, err := buildTicketRedirectURL("http://localhost:8040/#/login", "ticket-1", "/dashboard")

	require.NoError(t, err)
	require.Equal(t, "http://localhost:8040/#/login?oauth_ticket=ticket-1&redirect=%2Fdashboard", redirectURL)
}
