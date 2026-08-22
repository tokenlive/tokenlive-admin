package metrics

import (
	"sync"
	"time"
)

type RequestMetric struct {
	Time                int64   `json:"time"` // Unix timestamp
	Model               string  `json:"model"`
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
	Success    int64
	Fail       int64
	TTFTSum    int64
	TTFTCount  int64
	Output     int64
	DurationMs int64
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

	// minute -> success/failure
	globalSuccess map[int64]int64
	globalFailure map[int64]int64

	// model_code -> minute -> success/failure
	modelSuccess map[string]map[int64]int64
	modelFailure map[string]map[int64]int64
	modelTTFTSum map[string]map[int64]int64
	modelTTFTCnt map[string]map[int64]int64
	modelOutput  map[string]map[int64]int64
	modelDurMs   map[string]map[int64]int64

	// endpoint_id -> minute -> success/failure
	endpointSuccess map[string]map[int64]int64
	endpointFailure map[string]map[int64]int64
	endpointTTFTSum map[string]map[int64]int64
	endpointTTFTCnt map[string]map[int64]int64
	endpointOutput  map[string]map[int64]int64
	endpointDurMs   map[string]map[int64]int64

	// date -> stats
	dailyStats map[string]*DailyStats

	// circuit breaker: open endpoints & services
	openEndpoints map[string]bool
	openServices  map[string]bool
}

var GlobalStore = NewMemoryStore()

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		globalSuccess:   make(map[int64]int64),
		globalFailure:   make(map[int64]int64),
		modelSuccess:    make(map[string]map[int64]int64),
		modelFailure:    make(map[string]map[int64]int64),
		modelTTFTSum:    make(map[string]map[int64]int64),
		modelTTFTCnt:    make(map[string]map[int64]int64),
		modelOutput:     make(map[string]map[int64]int64),
		modelDurMs:      make(map[string]map[int64]int64),
		endpointSuccess: make(map[string]map[int64]int64),
		endpointFailure: make(map[string]map[int64]int64),
		endpointTTFTSum: make(map[string]map[int64]int64),
		endpointTTFTCnt: make(map[string]map[int64]int64),
		endpointOutput:  make(map[string]map[int64]int64),
		endpointDurMs:   make(map[string]map[int64]int64),
		dailyStats:      make(map[string]*DailyStats),
		openEndpoints:   make(map[string]bool),
		openServices:    make(map[string]bool),
	}
}

func (s *MemoryStore) Record(m RequestMetric) {
	s.mu.Lock()
	defer s.mu.Unlock()

	minute := m.Time / 60
	t := time.Unix(m.Time, 0)
	dateStr := t.Format("2006-01-02")

	// 1. Global
	if m.Success {
		s.globalSuccess[minute]++
	} else {
		s.globalFailure[minute]++
	}

	// 2. Model
	if m.Model != "" {
		if _, ok := s.modelSuccess[m.Model]; !ok {
			s.modelSuccess[m.Model] = make(map[int64]int64)
			s.modelFailure[m.Model] = make(map[int64]int64)
		}
		if m.Success {
			s.modelSuccess[m.Model][minute]++
		} else {
			s.modelFailure[m.Model][minute]++
		}
		if m.TTFTMs > 0 {
			if _, ok := s.modelTTFTSum[m.Model]; !ok {
				s.modelTTFTSum[m.Model] = make(map[int64]int64)
				s.modelTTFTCnt[m.Model] = make(map[int64]int64)
			}
			s.modelTTFTSum[m.Model][minute] += m.TTFTMs
			s.modelTTFTCnt[m.Model][minute]++
		}
		if m.Success && m.OutputTokens > 0 && m.DurationMs > 0 {
			if _, ok := s.modelOutput[m.Model]; !ok {
				s.modelOutput[m.Model] = make(map[int64]int64)
				s.modelDurMs[m.Model] = make(map[int64]int64)
			}
			s.modelOutput[m.Model][minute] += m.OutputTokens
			s.modelDurMs[m.Model][minute] += m.DurationMs
		}
	}

	// 3. Endpoint
	for _, att := range m.Attempts {
		if att.EndpointID != "" {
			if _, ok := s.endpointSuccess[att.EndpointID]; !ok {
				s.endpointSuccess[att.EndpointID] = make(map[int64]int64)
				s.endpointFailure[att.EndpointID] = make(map[int64]int64)
			}
			if att.Success {
				s.endpointSuccess[att.EndpointID][minute]++
			} else {
				s.endpointFailure[att.EndpointID][minute]++
			}
		}
	}

	if m.EndpointID != "" {
		if m.TTFTMs > 0 {
			if _, ok := s.endpointTTFTSum[m.EndpointID]; !ok {
				s.endpointTTFTSum[m.EndpointID] = make(map[int64]int64)
				s.endpointTTFTCnt[m.EndpointID] = make(map[int64]int64)
			}
			s.endpointTTFTSum[m.EndpointID][minute] += m.TTFTMs
			s.endpointTTFTCnt[m.EndpointID][minute]++
		}
		if m.Success && m.OutputTokens > 0 && m.DurationMs > 0 {
			if _, ok := s.endpointOutput[m.EndpointID]; !ok {
				s.endpointOutput[m.EndpointID] = make(map[int64]int64)
				s.endpointDurMs[m.EndpointID] = make(map[int64]int64)
			}
			s.endpointOutput[m.EndpointID][minute] += m.OutputTokens
			s.endpointDurMs[m.EndpointID][minute] += m.DurationMs
		}
	}

	// 4. Daily
	d, ok := s.dailyStats[dateStr]
	if !ok {
		d = &DailyStats{}
		s.dailyStats[dateStr] = d
	}
	d.ReqCount++
	d.InputTokens += m.InputTokens
	d.OutputTokens += m.OutputTokens
	d.CachedTokens += m.CachedTokens
	d.CacheCreationTokens += m.CacheCreationTokens
	d.Cost += m.Cost

	// 5. Cleanup older than 3 hours (180 minutes) to prevent memory leak
	cutoff := (time.Now().Unix() - 10800) / 60
	for min := range s.globalSuccess {
		if min < cutoff {
			delete(s.globalSuccess, min)
			delete(s.globalFailure, min)
		}
	}
	for _, mMin := range s.modelSuccess {
		for min := range mMin {
			if min < cutoff {
				delete(mMin, min)
			}
		}
	}
	for _, mMin := range s.modelFailure {
		for min := range mMin {
			if min < cutoff {
				delete(mMin, min)
			}
		}
	}
	for _, mMin := range s.modelTTFTSum {
		for min := range mMin {
			if min < cutoff {
				delete(mMin, min)
			}
		}
	}
	for _, mMin := range s.modelTTFTCnt {
		for min := range mMin {
			if min < cutoff {
				delete(mMin, min)
			}
		}
	}
	for _, mMin := range s.modelOutput {
		for min := range mMin {
			if min < cutoff {
				delete(mMin, min)
			}
		}
	}
	for _, mMin := range s.modelDurMs {
		for min := range mMin {
			if min < cutoff {
				delete(mMin, min)
			}
		}
	}
	for _, eMin := range s.endpointSuccess {
		for min := range eMin {
			if min < cutoff {
				delete(eMin, min)
			}
		}
	}
	for _, eMin := range s.endpointFailure {
		for min := range eMin {
			if min < cutoff {
				delete(eMin, min)
			}
		}
	}
	for _, eMin := range s.endpointTTFTSum {
		for min := range eMin {
			if min < cutoff {
				delete(eMin, min)
			}
		}
	}
	for _, eMin := range s.endpointTTFTCnt {
		for min := range eMin {
			if min < cutoff {
				delete(eMin, min)
			}
		}
	}
	for _, eMin := range s.endpointOutput {
		for min := range eMin {
			if min < cutoff {
				delete(eMin, min)
			}
		}
	}
	for _, eMin := range s.endpointDurMs {
		for min := range eMin {
			if min < cutoff {
				delete(eMin, min)
			}
		}
	}
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
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.globalSuccess[minute], s.globalFailure[minute]
}

func (s *MemoryStore) GetModelStatus(modelCode string, minute int64) (int64, int64) {
	perf := s.GetModelMinutePerf(modelCode, minute)
	return perf.Success, perf.Fail
}

func (s *MemoryStore) GetModelMinutePerf(modelCode string, minute int64) EndpointMinutePerf {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var perf EndpointMinutePerf
	if mMin, ok := s.modelSuccess[modelCode]; ok {
		perf.Success = mMin[minute]
	}
	if mMin, ok := s.modelFailure[modelCode]; ok {
		perf.Fail = mMin[minute]
	}
	if mMin, ok := s.modelTTFTSum[modelCode]; ok {
		perf.TTFTSum = mMin[minute]
	}
	if mMin, ok := s.modelTTFTCnt[modelCode]; ok {
		perf.TTFTCount = mMin[minute]
	}
	if mMin, ok := s.modelOutput[modelCode]; ok {
		perf.Output = mMin[minute]
	}
	if mMin, ok := s.modelDurMs[modelCode]; ok {
		perf.DurationMs = mMin[minute]
	}
	return perf
}

func (s *MemoryStore) GetEndpointStatus(endpointID string, minute int64) (int64, int64) {
	perf := s.GetEndpointMinutePerf(endpointID, minute)
	return perf.Success, perf.Fail
}

func (s *MemoryStore) GetEndpointMinutePerf(endpointID string, minute int64) EndpointMinutePerf {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var perf EndpointMinutePerf
	if eMin, ok := s.endpointSuccess[endpointID]; ok {
		perf.Success = eMin[minute]
	}
	if eMin, ok := s.endpointFailure[endpointID]; ok {
		perf.Fail = eMin[minute]
	}
	if eMin, ok := s.endpointTTFTSum[endpointID]; ok {
		perf.TTFTSum = eMin[minute]
	}
	if eMin, ok := s.endpointTTFTCnt[endpointID]; ok {
		perf.TTFTCount = eMin[minute]
	}
	if eMin, ok := s.endpointOutput[endpointID]; ok {
		perf.Output = eMin[minute]
	}
	if eMin, ok := s.endpointDurMs[endpointID]; ok {
		perf.DurationMs = eMin[minute]
	}
	return perf
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
