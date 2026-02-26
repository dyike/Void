package calculator

import "github.com/google/wire"

// ProviderSet exposes the calculator service for dependency injection.
var ProviderSet = wire.NewSet(NewService)
