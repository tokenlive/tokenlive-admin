package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/monitor"
	"github.com/tokenlive/tokenlive-admin/pkg/metrics"
)

func snapshotDashboard() *Dashboard {
	now := time.Now().Truncate(time.Minute).Add(-time.Second)
	return &Dashboard{Monitor: monitor.New(monitor.Dependencies{Memory: metrics.NewMemoryStoreAt(now.Add(-48 * time.Hour)), Clock: func() time.Time { return now }})}
}

func TestSnapshotHTTPStrictQueryAndCanonicalShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		body   string
		status int
	}{
		{`{"traffic":{"range":"1h"},"models":{}}`, 200},
		{`{"traffic":{"range":"invalid"}}`, 400},
		{`{"timezone":"unknown","overview":true}`, 400},
		{`{"api_key_ranking":{}}`, 400},
		{`{"traffic":{},"unknown":true}`, 400},
		{`{"traffic":{}} {"traffic":{}}`, 400},
		{`{}`, 400},
		{`null`, 400},
		{`{"traffic":{},"timezone":"` + strings.Repeat(" ", 65536) + `"}`, 400},
	} {
		t.Run(tc.body[:min(len(tc.body), 50)], func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/dashboard/snapshot", strings.NewReader(tc.body))
			snapshotDashboard().QuerySnapshot(c)
			require.Equal(t, tc.status, w.Code, w.Body.String())
			if tc.status == 200 {
				var response struct {
					Data monitor.Snapshot `json:"data"`
				}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
				require.NotZero(t, response.Data.Cutoff)
				require.Equal(t, "ok", response.Data.Traffic.State)
				require.Equal(t, "error", response.Data.Models.State)
				require.Equal(t, "source_unavailable", response.Data.Models.Error)
				require.Len(t, response.Data.Traffic.Buckets, 60)
			}
		})
	}
}

func TestSnapshotWebSocketUsesSameReadingContractAndKeepsLocalFailures(t *testing.T) {
	a := snapshotDashboard()
	engine := gin.New()
	engine.GET("/ws", a.HandleWebSocket)
	server := httptest.NewServer(engine)
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/ws", nil)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(3*time.Second)))
	var s monitor.Snapshot
	require.NoError(t, conn.ReadJSON(&s))
	require.NotNil(t, s.Overview)
	require.Equal(t, "ok", s.Traffic.State)
	require.Equal(t, "error", s.Models.State)
	require.Equal(t, s.Traffic.Window.End, s.Models.Window.End)
	require.NoError(t, conn.WriteJSON(map[string]any{"type": "config", "data": WSConfigMsg{TrendsTimeRange: "6h", ModelRankingTimeRange: "today", Timezone: "UTC"}}))
	require.NoError(t, conn.ReadJSON(&s))
	require.Equal(t, 6*time.Hour, s.Traffic.Window.End.Sub(s.Traffic.Window.Start))
	require.Equal(t, "UTC", s.Traffic.Window.Timezone)
}
