package biz

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

func TestCurrentUserRootFlagComesFromVerifiedContext(t *testing.T) {
	previous := config.C
	config.C = new(config.Config)
	config.C.General.Root.ID = "non-default-root-id"
	config.C.General.Root.Username = "non-default-name"
	t.Cleanup(func() { config.C = previous })
	user, err := (&Login{}).GetUserInfo(util.NewIsRootUser(context.Background()))
	require.NoError(t, err)
	require.True(t, user.IsRoot)
	require.Equal(t, "non-default-root-id", user.ID)
	raw, err := json.Marshal(user)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"is_root":true`)
	ordinary := schema.User{ID: "root", Username: "admin"}
	require.False(t, ordinary.IsRoot, "names and IDs alone never grant Root")
}
