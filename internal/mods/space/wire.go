package space

import (
	"github.com/google/wire"
	"github.com/tokenlive/tokenlive-admin/internal/mods/space/api"
	"github.com/tokenlive/tokenlive-admin/internal/mods/space/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/space/dal"
)

// Collection of wire providers
var Set = wire.NewSet(
	wire.Struct(new(Space), "*"),
	wire.Struct(new(dal.Space), "*"),
	wire.Struct(new(biz.Space), "*"),
	wire.Struct(new(api.Space), "*"),
)
