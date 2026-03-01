package template

import (
	"fmt"

	"github.com/samber/do/v2"

	"go-template/pkg/db"
)

// Provide registers the Template constructor with the injector.
func Provide(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*Template, error) {
		database := do.MustInvoke[*db.DB](i)

		t, err := New(database, nil)
		if err != nil {
			return nil, fmt.Errorf("template: %w", err)
		}

		return t, nil
	})
}
