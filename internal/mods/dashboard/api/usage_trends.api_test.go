package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

type trendsQueryFunc func(context.Context, schema.UsageTrendsQuery) (*schema.UsageTrendsResponse, error)

func (f trendsQueryFunc) Query(ctx context.Context, q schema.UsageTrendsQuery) (*schema.UsageTrendsResponse, error) {
	return f(ctx, q)
}

func TestUsageTrendsAPIAllowsAuthenticatedNonRootAndSanitizesErrors(t *testing.T) {
	for _, tc := range []struct {
		query, user string
		fail        bool
		status      int
	}{
		{"", "", false, 401},
		{"?time_range=1h", "ordinary", false, 200},
		{"?time_range=invalid", "ordinary", false, 400},
		{"?time_range=1h&time_range=7d", "ordinary", false, 400},
		{"", "ordinary", true, 503},
	} {
		t.Run(tc.query+tc.user, func(t *testing.T) {
			calls := 0
			a := &UsageTrends{Service: trendsQueryFunc(func(_ context.Context, q schema.UsageTrendsQuery) (*schema.UsageTrendsResponse, error) {
				calls++
				if tc.fail {
					return nil, errors.New("sensitive-driver-error")
				}
				require.Equal(t, "1h", q.TimeRange)
				return &schema.UsageTrendsResponse{State: "disabled"}, nil
			})}
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/usage-trends"+tc.query, nil)
			c.Request = c.Request.WithContext(util.NewUserID(c.Request.Context(), tc.user))
			a.Query(c)
			require.Equal(t, tc.status, w.Code)
			require.NotContains(t, w.Body.String(), "sensitive-driver-error")
			if tc.status == 401 || tc.status == 400 {
				require.Zero(t, calls)
			}
		})
	}
}
