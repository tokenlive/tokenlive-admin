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
	Attempts            []struct {
		EndpointID string `json:"endpoint_id"`
		Success    bool   `json:"success"`
	} `json:"attempts"`
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

	// endpoint_id -> minute -> success/failure
	endpointSuccess map[string]map[int64]int64
	endpointFailure map[string]map[int64]int64

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
		endpointSuccess: make(map[string]map[int64]int64),
		endpointFailure: make(map[string]map[int64]int64),
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
	s.mu.RLock()
	defer s.mu.RUnlock()

	var succ, fail int64
	if mMin, ok := s.modelSuccess[modelCode]; ok {
		succ = mMin[minute]
	}
	if mMin, ok := s.modelFailure[modelCode]; ok {
		fail = mMin[minute]
	}
	return succ, fail
}

func (s *MemoryStore) GetEndpointStatus(endpointID string, minute int64) (int64, int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var succ, fail int64
	if eMin, ok := s.endpointSuccess[endpointID]; ok {
		succ = eMin[minute]
	}
	if eMin, ok := s.endpointFailure[endpointID]; ok {
		fail = eMin[minute]
	}
	return succ, fail
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
