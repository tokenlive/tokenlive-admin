package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	wsWriteWait      = 10 * time.Second
	wsPongWait       = 60 * time.Second
	wsPingPeriod     = (wsPongWait * 9) / 10
	wsMaxMessageSize = 4096
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // CORS is handled by middleware
	},
}

// WSConfigMsg represents the configuration parameters sent by the client
type WSConfigMsg struct {
	TrendsTimeRange       string `json:"trends_time_range"`
	TrendsGroupBy         string `json:"trends_group_by"`
	ModelRankingSortBy    string `json:"model_ranking_sort_by"`
	ModelRankingTimeRange string `json:"model_ranking_time_range"`
}

// WSDashboardPayload represents the unified push data payload
type WSDashboardPayload struct {
	Overview     *OverviewResponse  `json:"overview"`
	Trends       *TrendsResponse    `json:"trends"`
	ModelRanking []ModelRankingItem `json:"model_ranking"`
}

// HandleWebSocket upgrades the HTTP request to WebSocket and pushes dashboard metrics
func (a *Dashboard) HandleWebSocket(c *gin.Context) {
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Default client filter configuration
	cfg := WSConfigMsg{
		TrendsTimeRange:       "1h",
		TrendsGroupBy:         "",
		ModelRankingSortBy:    "request_count",
		ModelRankingTimeRange: "1h",
	}

	var cfgMu sync.Mutex
	triggerPush := make(chan struct{}, 1)

	// Read pump: listen for filter updates from the client
	go func() {
		defer cancel()
		conn.SetReadLimit(wsMaxMessageSize)
		_ = conn.SetReadDeadline(time.Now().Add(wsPongWait))
		conn.SetPongHandler(func(string) error {
			_ = conn.SetReadDeadline(time.Now().Add(wsPongWait))
			return nil
		})
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			var msg struct {
				Type string      `json:"type"`
				Data WSConfigMsg `json:"data"`
			}
			if err := json.Unmarshal(message, &msg); err == nil && msg.Type == "config" {
				cfgMu.Lock()
				if msg.Data.TrendsTimeRange != "" {
					cfg.TrendsTimeRange = msg.Data.TrendsTimeRange
				}
				cfg.TrendsGroupBy = msg.Data.TrendsGroupBy
				if msg.Data.ModelRankingSortBy != "" {
					cfg.ModelRankingSortBy = msg.Data.ModelRankingSortBy
				}
				if msg.Data.ModelRankingTimeRange != "" {
					cfg.ModelRankingTimeRange = msg.Data.ModelRankingTimeRange
				}
				cfgMu.Unlock()

				// Trigger immediate push on config change
				select {
				case triggerPush <- struct{}{}:
				default:
				}
			}
		}
	}()

	// Write pump: calculate and push dashboard metrics periodically
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	pingTicker := time.NewTicker(wsPingPeriod)
	defer pingTicker.Stop()

	sendData := func() error {
		cfgMu.Lock()
		currentCfg := cfg
		cfgMu.Unlock()

		overview, err := a.getOverview(ctx)
		if err != nil {
			return err
		}

		trends, err := a.getTrends(ctx, currentCfg.TrendsGroupBy, currentCfg.TrendsTimeRange)
		if err != nil {
			return err
		}

		ranking, err := a.getModelRanking(ctx, currentCfg.ModelRankingSortBy, currentCfg.ModelRankingTimeRange, 10)
		if err != nil {
			return err
		}

		payload := WSDashboardPayload{
			Overview:     overview,
			Trends:       trends,
			ModelRanking: ranking,
		}

		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		_ = conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
		return conn.WriteMessage(websocket.TextMessage, data)
	}

	// Immediate push of initial data upon handshake completion
	if err := sendData(); err != nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-triggerPush:
			if err := sendData(); err != nil {
				return
			}
		case <-ticker.C:
			if err := sendData(); err != nil {
				return
			}
		case <-pingTicker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
