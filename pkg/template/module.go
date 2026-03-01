package template

import "go.uber.org/fx"

// Module provides the Template service to the fx container.
var Module = fx.Module("template",
	fx.Provide(fx.Annotate(
		New,
		fx.ParamTags(``, `optional:"true"`),
	)),
)
