package hotkey

import "github.com/google/wire"

// ProviderSet exposes the hotkey service for dependency injection.
var ProviderSet = wire.NewSet(NewService)
