package scripts

import "github.com/google/wire"

// ProviderSet exposes the scripts service for dependency injection.
var ProviderSet = wire.NewSet(NewService)
