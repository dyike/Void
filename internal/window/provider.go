package window

import "github.com/google/wire"

// ProviderSet exposes the window manager for dependency injection.
var ProviderSet = wire.NewSet(NewManager)
