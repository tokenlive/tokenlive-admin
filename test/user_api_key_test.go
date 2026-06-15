package test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/crypto/hash"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

func TestUserAPIKey(t *testing.T) {
	e := tester(t)
	assert := assert.New(t)

	// 1. 创建一个临时用户用于测试关联
	userFormItem := schema.UserForm{
		Username: "apikey_test_user",
		Name:     "API Key Test User",
		Password: hash.MD5String("testpwd"),
		Phone:    "12345678",
		Email:    "testkey@example.com",
		Remark:   "temp user for key test",
		Status:   schema.UserStatusActivated,
		Roles:    schema.UserRoles{},
	}

	var user schema.User
	e.POST(baseAPI + "/users").WithJSON(userFormItem).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &user})
	assert.NotEmpty(user.ID)

	// 2. 创建一个 API Key
	expiryTime := time.Now().Add(24 * time.Hour)
	keyFormItem := schema.UserAPIKeyForm{
		UserID:      user.ID,
		Name:        "production-chat-key",
		Status:      1,
		Quota:       5000000,
		ExpiresAt:   &expiryTime,
		Description: "For testing API key creation",
	}

	var createdKey schema.UserAPIKey
	e.POST(baseAPI + "/user-api-keys").WithJSON(keyFormItem).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &createdKey})

	assert.NotEmpty(createdKey.ID)
	assert.Equal(keyFormItem.UserID, createdKey.UserID)
	assert.Equal(keyFormItem.Name, createdKey.Name)
	assert.Equal(keyFormItem.Status, createdKey.Status)
	assert.Equal(keyFormItem.Quota, createdKey.Quota)
	// 验证密钥生成：必须以 sk- 开头，且总长度为 3 + 32(hex) = 35 字符
	assert.Contains(createdKey.APIKey, "sk-")
	assert.Equal(35, len(createdKey.APIKey))

	// 3. 分页查询列表，验证 API Key 列表查询的掩码脱敏
	var list schema.UserAPIKeys
	e.GET(baseAPI+"/user-api-keys").
		WithQuery("user_id", user.ID).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &list})
	assert.GreaterOrEqual(len(list), 1)
	assert.Contains(list[0].APIKey, "****")

	// 4. 根据 ID 查询单条记录，验证单条记录查询的掩码脱敏
	var fetchedKey schema.UserAPIKey
	e.GET(baseAPI + "/user-api-keys/" + createdKey.ID).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &fetchedKey})
	assert.Equal(createdKey.ID, fetchedKey.ID)
	assert.Contains(fetchedKey.APIKey, "****")

	// 5. 更新 API Key (更新名称、修改状态为禁用、下调配额)
	fetchedKey.Name = "updated-chat-key"
	fetchedKey.Status = 2
	fetchedKey.Quota = 1000
	e.PUT(baseAPI + "/user-api-keys/" + createdKey.ID).
		WithJSON(fetchedKey).
		Expect().Status(http.StatusOK)

	// 再次获取并验证更新效果
	var updatedKey schema.UserAPIKey
	e.GET(baseAPI + "/user-api-keys/" + createdKey.ID).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &updatedKey})
	assert.Equal("updated-chat-key", updatedKey.Name)
	assert.Equal(2, updatedKey.Status)
	assert.Equal(int64(1000), updatedKey.Quota)

	// 6. 删除该 API Key 并验证其不再可查
	e.DELETE(baseAPI + "/user-api-keys/" + createdKey.ID).
		Expect().Status(http.StatusOK)

	e.GET(baseAPI + "/user-api-keys/" + createdKey.ID).
		Expect().Status(http.StatusNotFound)

	// 7. 清理临时用户
	e.DELETE(baseAPI + "/users/" + user.ID).Expect().Status(http.StatusOK)
}
