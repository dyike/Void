package snippets

import "github.com/google/wire"

// ProviderSet exposes the snippets service for dependency injection.
var ProviderSet = wire.NewSet(NewService)
