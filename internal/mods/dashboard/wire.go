package dashboard

import (
	"github.com/google/wire"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/api"
)

var Set = wire.NewSet(
	wire.Struct(new(Dashboard), "*"),
	wire.Struct(new(api.Dashboard), "*"),
)
