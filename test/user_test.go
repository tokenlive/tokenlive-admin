package test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/crypto/hash"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

func TestUser(t *testing.T) {
	e := tester(t)

	menuFormItem := schema.MenuForm{
		Code:        "user",
		Name:        "User management",
		Description: "User management",
		Sequence:    7,
		Type:        "page",
		Path:        "/system/user",
		Properties:  `{"icon":"user"}`,
		Status:      schema.MenuStatusEnabled,
	}

	var menu schema.Menu
	e.POST(baseAPI + "/menus").WithJSON(menuFormItem).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &menu})

	assert := assert.New(t)
	assert.NotEmpty(menu.ID)
	assert.Equal(menuFormItem.Code, menu.Code)
	assert.Equal(menuFormItem.Name, menu.Name)
	assert.Equal(menuFormItem.Description, menu.Description)
	assert.Equal(menuFormItem.Sequence, menu.Sequence)
	assert.Equal(menuFormItem.Type, menu.Type)
	assert.Equal(menuFormItem.Path, menu.Path)
	assert.Equal(menuFormItem.Properties, menu.Properties)
	assert.Equal(menuFormItem.Status, menu.Status)

	roleFormItem := schema.RoleForm{
		Code: "user",
		Name: "Normal",
		Menus: schema.RoleMenus{
			{MenuID: menu.ID},
		},
		Description: "Normal",
		Sequence:    8,
		Status:      schema.RoleStatusEnabled,
	}

	var role schema.Role
	e.POST(baseAPI + "/roles").WithJSON(roleFormItem).Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &role})
	assert.NotEmpty(role.ID)
	assert.Equal(roleFormItem.Code, role.Code)
	assert.Equal(roleFormItem.Name, role.Name)
	assert.Equal(roleFormItem.Description, role.Description)
	assert.Equal(roleFormItem.Sequence, role.Sequence)
	assert.Equal(roleFormItem.Status, role.Status)
	assert.Equal(len(roleFormItem.Menus), len(role.Menus))

	userFormItem := schema.UserForm{
		Username: "test",
		Name:     "Test",
		Password: hash.MD5String("test"),
		Phone:    "0720",
		Email:    "test@gmail.com",
		Remark:   "test user",
		Tenant:   "tenant-a",
		Status:   schema.UserStatusActivated,
		Roles:    schema.UserRoles{{RoleID: role.ID}},
	}

	var user schema.User
	e.POST(baseAPI + "/users").WithJSON(userFormItem).Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &user})
	assert.NotEmpty(user.ID)
	assert.Equal(userFormItem.Username, user.Username)
	assert.Equal(userFormItem.Name, user.Name)
	assert.Equal(userFormItem.Phone, user.Phone)
	assert.Equal(userFormItem.Email, user.Email)
	assert.Equal(userFormItem.Remark, user.Remark)
	assert.Equal(userFormItem.Tenant, user.Tenant)
	assert.Equal(userFormItem.Status, user.Status)
	assert.Equal(len(userFormItem.Roles), len(user.Roles))

	otherUserFormItem := schema.UserForm{
		Username: "test-other-tenant",
		Name:     "Test Other Tenant",
		Password: hash.MD5String("test"),
		Tenant:   "tenant-b",
		Status:   schema.UserStatusActivated,
		Roles:    schema.UserRoles{{RoleID: role.ID}},
	}
	var otherUser schema.User
	e.POST(baseAPI + "/users").WithJSON(otherUserFormItem).Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &otherUser})

	var users schema.Users
	e.GET(baseAPI+"/users").WithQuery("username", userFormItem.Username).Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &users})
	assert.GreaterOrEqual(len(users), 1)

	var tenantUsers schema.Users
	e.GET(baseAPI+"/users").WithQuery("tenant", "tenant-a").Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &tenantUsers})
	assert.Len(tenantUsers, 1)
	assert.Equal(user.ID, tenantUsers[0].ID)

	newName := "Test 1"
	newStatus := schema.UserStatusFreezed
	user.Name = newName
	user.Status = newStatus
	e.PUT(baseAPI + "/users/" + user.ID).WithJSON(user).Expect().Status(http.StatusOK)

	var getUser schema.User
	e.GET(baseAPI + "/users/" + user.ID).Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &getUser})
	assert.Equal(newName, getUser.Name)
	assert.Equal(newStatus, getUser.Status)

	e.DELETE(baseAPI + "/users/" + user.ID).Expect().Status(http.StatusOK)
	e.GET(baseAPI + "/users/" + user.ID).Expect().Status(http.StatusNotFound)
	e.DELETE(baseAPI + "/users/" + otherUser.ID).Expect().Status(http.StatusOK)
	e.GET(baseAPI + "/users/" + otherUser.ID).Expect().Status(http.StatusNotFound)

	e.DELETE(baseAPI + "/roles/" + role.ID).Expect().Status(http.StatusOK)
	e.GET(baseAPI + "/roles/" + role.ID).Expect().Status(http.StatusNotFound)

	e.DELETE(baseAPI + "/menus/" + menu.ID).Expect().Status(http.StatusOK)
	e.GET(baseAPI + "/menus/" + menu.ID).Expect().Status(http.StatusNotFound)
}

func TestUserQueryLimitsNonRootToCurrentTenant(t *testing.T) {
	ctx := util.NewTenant(util.NewUsername(context.Background(), "tenant-user"), "tenant-a")

	_, err := testInjector.M.RBAC.UserAPI.UserBIZ.Query(ctx, schema.UserQueryParam{Tenant: "tenant-b"})
	assert.Error(t, err)
}
