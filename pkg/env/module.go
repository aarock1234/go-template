package env

import "go.uber.org/fx"

// Module provides the application config to the fx container.
var Module = fx.Module("env",
	fx.Provide(New),
)
