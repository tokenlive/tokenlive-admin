package systemversion

import "github.com/google/wire"

var Set = wire.NewSet(
	ProvideService,
	wire.Struct(new(SystemVersion), "*"),
)
