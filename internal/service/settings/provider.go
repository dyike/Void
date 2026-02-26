package settings

import "github.com/google/wire"

// ProviderSet exposes the settings service for dependency injection.
var ProviderSet = wire.NewSet(NewService)
