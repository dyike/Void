package clipboard

import "github.com/google/wire"

// ProviderSet exposes the clipboard service for dependency injection.
var ProviderSet = wire.NewSet(NewService)
