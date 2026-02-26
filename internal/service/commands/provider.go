package commands

import "github.com/google/wire"

// ProviderSet exposes the commands service for dependency injection.
var ProviderSet = wire.NewSet(NewService)
