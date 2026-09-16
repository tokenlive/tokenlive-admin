package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

type usageQueryFunc func(context.Context, schema.Query) (*schema.Response, error)

func (f usageQueryFunc) Query(ctx context.Context, query schema.Query) (*schema.Response, error) {
	return f(ctx, query)
}

func TestUsageAPIAuthorizesBeforeQueryAndSanitizesFailures(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, query string
		root        bool
		failure     bool
		want, calls int
	}{
		{"non-root", "", false, false, 403, 0},
		{"non-root invalid", "?limit=99", false, false, 403, 0},
		{"bad params", "?sort_by=api_key", true, false, 400, 0},
		{"source down", "", true, true, 503, 1},
		{"ready", "?limit=20&sort_by=cost&time_range=7d", true, false, 200, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			api := &APIKeyUsage{Service: usageQueryFunc(func(_ context.Context, query schema.Query) (*schema.Response, error) {
				calls++
				if tc.failure {
					return nil, errors.New("fixture-sensitive-driver-error")
				}
				require.Equal(t, schema.Query{"7d", "cost", 20}, query)
				return &schema.Response{State: "ready", DataSource: "clickhouse", Items: []schema.Item{}}, nil
			})}
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/api-key-ranking"+tc.query, nil)
			if tc.root {
				c.Request = c.Request.WithContext(util.NewIsRootUser(c.Request.Context()))
			}
			api.Query(c)
			require.Equal(t, tc.want, recorder.Code)
			require.Equal(t, tc.calls, calls)
			require.NotContains(t, recorder.Body.String(), "fixture-sensitive-driver-error")
			if tc.want == 200 {
				var body struct {
					Success bool            `json:"success"`
					Data    schema.Response `json:"data"`
				}
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
				require.True(t, body.Success)
				require.Equal(t, "ready", body.Data.State)
			}
		})
	}
}
