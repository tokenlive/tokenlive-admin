package dashboard

import (
	"github.com/google/wire"
)

var Set = wire.NewSet(
	wire.Struct(new(Dashboard), "*"),
	ProvideMonitor,
	ProvideDashboard,
	ProvideUsageReader,
	ProvideUsageResolver,
	ProvideUsageService,
	ProvideUsageAPI,
	ProvideUsageTrendsService,
	ProvideUsageTrendsAPI,
)
