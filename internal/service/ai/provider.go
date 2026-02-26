package ai

import "github.com/google/wire"

// ProviderSet exposes the AI service for dependency injection.
var ProviderSet = wire.NewSet(NewService)
