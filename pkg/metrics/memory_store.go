package metrics

import (
	"sort"
	"sync"
	"time"
)

const minuteMetricRetention = 7 * 24 * time.Hour

type RequestMetric struct {
	Time                int64   `json:"time"` // Unix timestamp
	Model               string  `json:"model"`
	Provider            string  `json:"provider,omitempty"`
	Success             bool    `json:"success"`
	InputTokens         int64   `json:"input_tokens"`
	OutputTokens        int64   `json:"output_tokens"`
	CachedTokens        int64   `json:"cached_tokens"`
	CacheCreationTokens int64   `json:"cache_creation_tokens"`
	Cost                float64 `json:"cost"`
	EndpointID          string  `json:"endpoint_id,omitempty"`
	TTFTMs              int64   `json:"ttft_ms,omitempty"`
	DurationMs          int64   `json:"duration_ms,omitempty"`
	Attempts            []struct {
		EndpointID string `json:"endpoint_id"`
		Success    bool   `json:"success"`
	} `json:"attempts"`
}

type EndpointMinutePerf struct {
	Requests     int64
	Success      int64
	Fail         int64
	InputTokens  int64
	OutputTokens int64
	Output       int64
	Cost         float64
	TTFTSum      int64
	TTFTCount    int64
	LatencySumMs int64
	LatencyCount int64
	DurationMs   int64
}

type DailyStats struct {
	ReqCount            int64   `json:"req_count"`
	InputTokens         int64   `json:"input_tokens"`
	OutputTokens        int64   `json:"output_tokens"`
	CachedTokens        int64   `json:"cached_tokens"`
	CacheCreationTokens int64   `json:"cache_creation_tokens"`
	Cost                float64 `json:"cost"`
}

type MemoryStore struct {
	mu sync.RWMutex

	globalPerf   map[int64]*EndpointMinutePerf
	modelPerf    map[string]map[int64]*EndpointMinutePerf
	providerPerf map[string]map[int64]*EndpointMinutePerf
	endpointPerf map[string]map[int64]*EndpointMinutePerf

	// date -> stats
	dailyStats        map[string]*DailyStats
	lastCleanupMinute int64

	// circuit breaker: open endpoints & services
	openEndpoints map[string]bool
	openServices  map[string]bool
}

var GlobalStore = NewMemoryStore()

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		globalPerf:    make(map[int64]*EndpointMinutePerf),
		modelPerf:     make(map[string]map[int64]*EndpointMinutePerf),
		providerPerf:  make(map[string]map[int64]*EndpointMinutePerf),
		endpointPerf:  make(map[string]map[int64]*EndpointMinutePerf),
		dailyStats:    make(map[string]*DailyStats),
		openEndpoints: make(map[string]bool),
		openServices:  make(map[string]bool),
	}
}

func getOrCreateMinute(values map[int64]*EndpointMinutePerf, minute int64) *EndpointMinutePerf {
	perf := values[minute]
	if perf == nil {
		perf = &EndpointMinutePerf{}
		values[minute] = perf
	}
	return perf
}

func getOrCreateDimension(values map[string]map[int64]*EndpointMinutePerf, key string, minute int64) *EndpointMinutePerf {
	minutes := values[key]
	if minutes == nil {
		minutes = make(map[int64]*EndpointMinutePerf)
		values[key] = minutes
	}
	return getOrCreateMinute(minutes, minute)
}

func recordRequest(perf *EndpointMinutePerf, metric RequestMetric) {
	perf.Requests++
	if metric.Success {
		perf.Success++
	} else {
		perf.Fail++
	}
	perf.InputTokens += metric.InputTokens
	perf.OutputTokens += metric.OutputTokens
	perf.Cost += metric.Cost
	if metric.TTFTMs > 0 {
		perf.TTFTSum += metric.TTFTMs
		perf.TTFTCount++
	}
	if metric.DurationMs > 0 {
		perf.LatencySumMs += metric.DurationMs
		perf.LatencyCount++
	}
	if metric.Success && metric.OutputTokens > 0 && metric.DurationMs > 0 {
		perf.Output += metric.OutputTokens
		perf.DurationMs += metric.DurationMs
	}
}

func (s *MemoryStore) Record(m RequestMetric) {
	s.mu.Lock()
	defer s.mu.Unlock()

	minute := m.Time / 60
	dateStr := time.Unix(m.Time, 0).Format("2006-01-02")

	recordRequest(getOrCreateMinute(s.globalPerf, minute), m)
	if m.Model != "" {
		recordRequest(getOrCreateDimension(s.modelPerf, m.Model, minute), m)
	}
	if m.Provider != "" {
		recordRequest(getOrCreateDimension(s.providerPerf, m.Provider, minute), m)
	}

	for _, attempt := range m.Attempts {
		if attempt.EndpointID == "" {
			continue
		}
		perf := getOrCreateDimension(s.endpointPerf, attempt.EndpointID, minute)
		if attempt.Success {
			perf.Success++
		} else {
			perf.Fail++
		}
	}

	if m.EndpointID != "" {
		perf := getOrCreateDimension(s.endpointPerf, m.EndpointID, minute)
		perf.InputTokens += m.InputTokens
		perf.OutputTokens += m.OutputTokens
		perf.Cost += m.Cost
		if m.TTFTMs > 0 {
			perf.TTFTSum += m.TTFTMs
			perf.TTFTCount++
		}
		if m.DurationMs > 0 {
			perf.LatencySumMs += m.DurationMs
			perf.LatencyCount++
		}
		if m.Success && m.OutputTokens > 0 && m.DurationMs > 0 {
			perf.Output += m.OutputTokens
			perf.DurationMs += m.DurationMs
		}
	}

	daily := s.dailyStats[dateStr]
	if daily == nil {
		daily = &DailyStats{}
		s.dailyStats[dateStr] = daily
	}
	daily.ReqCount++
	daily.InputTokens += m.InputTokens
	daily.OutputTokens += m.OutputTokens
	daily.CachedTokens += m.CachedTokens
	daily.CacheCreationTokens += m.CacheCreationTokens
	daily.Cost += m.Cost

	currentMinute := time.Now().Unix() / 60
	if currentMinute != s.lastCleanupMinute {
		cutoff := time.Now().Add(-minuteMetricRetention).Unix() / 60
		cleanupMinutes(s.globalPerf, cutoff)
		cleanupDimensions(s.modelPerf, cutoff)
		cleanupDimensions(s.providerPerf, cutoff)
		cleanupDimensions(s.endpointPerf, cutoff)
		s.lastCleanupMinute = currentMinute
	}
}

func cleanupMinutes(values map[int64]*EndpointMinutePerf, cutoff int64) {
	for minute := range values {
		if minute < cutoff {
			delete(values, minute)
		}
	}
}

func cleanupDimensions(values map[string]map[int64]*EndpointMinutePerf, cutoff int64) {
	for key, minutes := range values {
		cleanupMinutes(minutes, cutoff)
		if len(minutes) == 0 {
			delete(values, key)
		}
	}
}

func copyPerf(perf *EndpointMinutePerf) EndpointMinutePerf {
	if perf == nil {
		return EndpointMinutePerf{}
	}
	return *perf
}

func addPerf(total *EndpointMinutePerf, perf *EndpointMinutePerf) {
	if perf == nil {
		return
	}
	total.Success += perf.Success
	total.Fail += perf.Fail
	total.Requests += perf.Requests
	total.InputTokens += perf.InputTokens
	total.OutputTokens += perf.OutputTokens
	total.Output += perf.Output
	total.Cost += perf.Cost
	total.TTFTSum += perf.TTFTSum
	total.TTFTCount += perf.TTFTCount
	total.LatencySumMs += perf.LatencySumMs
	total.LatencyCount += perf.LatencyCount
	total.DurationMs += perf.DurationMs
}

func aggregateMinutes(values map[int64]*EndpointMinutePerf, startMinute, endMinute int64) EndpointMinutePerf {
	var total EndpointMinutePerf
	for minute, perf := range values {
		if minute >= startMinute && minute <= endMinute {
			addPerf(&total, perf)
		}
	}
	return total
}

func dimensionKeys(values map[string]map[int64]*EndpointMinutePerf) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *MemoryStore) UpdateCircuitBreakers(endpoints []string, services []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.openEndpoints = make(map[string]bool)
	for _, ep := range endpoints {
		s.openEndpoints[ep] = true
	}

	s.openServices = make(map[string]bool)
	for _, svc := range services {
		s.openServices[svc] = true
	}
}

func (s *MemoryStore) GetGlobalStatus(minute int64) (int64, int64) {
	perf := s.GetGlobalMinutePerf(minute)
	return perf.Success, perf.Fail
}

func (s *MemoryStore) GetGlobalMinutePerf(minute int64) EndpointMinutePerf {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return copyPerf(s.globalPerf[minute])
}

func (s *MemoryStore) AggregateGlobalMinutePerf(startMinute, endMinute int64) EndpointMinutePerf {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return aggregateMinutes(s.globalPerf, startMinute, endMinute)
}

func (s *MemoryStore) GetModelStatus(modelCode string, minute int64) (int64, int64) {
	perf := s.GetModelMinutePerf(modelCode, minute)
	return perf.Success, perf.Fail
}

func (s *MemoryStore) GetModelMinutePerf(modelCode string, minute int64) EndpointMinutePerf {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return copyPerf(s.modelPerf[modelCode][minute])
}

func (s *MemoryStore) AggregateModelMinutePerf(modelCode string, startMinute, endMinute int64) EndpointMinutePerf {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return aggregateMinutes(s.modelPerf[modelCode], startMinute, endMinute)
}

func (s *MemoryStore) GetModelCodes() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return dimensionKeys(s.modelPerf)
}

func (s *MemoryStore) GetProviderStatus(provider string, minute int64) (int64, int64) {
	perf := s.GetProviderMinutePerf(provider, minute)
	return perf.Success, perf.Fail
}

func (s *MemoryStore) GetProviderMinutePerf(provider string, minute int64) EndpointMinutePerf {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return copyPerf(s.providerPerf[provider][minute])
}

func (s *MemoryStore) GetProviderNames() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return dimensionKeys(s.providerPerf)
}

func (s *MemoryStore) GetEndpointStatus(endpointID string, minute int64) (int64, int64) {
	perf := s.GetEndpointMinutePerf(endpointID, minute)
	return perf.Success, perf.Fail
}

func (s *MemoryStore) GetEndpointMinutePerf(endpointID string, minute int64) EndpointMinutePerf {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return copyPerf(s.endpointPerf[endpointID][minute])
}

func (s *MemoryStore) GetDailyStats(dateStr string) DailyStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if d, ok := s.dailyStats[dateStr]; ok && d != nil {
		return *d
	}
	return DailyStats{}
}

func (s *MemoryStore) GetOpenEndpoints() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []string
	for ep := range s.openEndpoints {
		res = append(res, ep)
	}
	return res
}

func (s *MemoryStore) GetOpenServices() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []string
	for svc := range s.openServices {
		res = append(res, svc)
	}
	return res
}
