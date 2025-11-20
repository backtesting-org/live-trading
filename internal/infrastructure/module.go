package infrastructure

import (
	"go.uber.org/fx"
)

// Module provides infrastructure components (logging, lifecycle)
var Module = fx.Module("infrastructure",
	fx.Provide(NewLogger),
	fx.Invoke(RegisterLifecycle),
)
