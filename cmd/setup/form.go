package main

import (
	"errors"
	"fmt"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

// runForm asks which features to keep. The choices in c seed the form and
// receive the answers. It reports whether the user confirmed the changes.
func runForm(c *config) (bool, error) {
	confirmed := true

	form := huh.NewForm(
		huh.NewGroup(
			yesNo("Include HTTP Server?", "Chi router, middleware, graceful shutdown, JSON handlers", &c.server),
			yesNo("Include HTTP Client?", "TLS fingerprint, proxy rotation, HTTP/2, scraper entry point", &c.client).
				Validate(func(client bool) error {
					choice := *c
					choice.client = client

					return choice.validate()
				}),
		),

		huh.NewGroup(
			yesNo("Include Docker?", "Multi-stage Dockerfile, compose for dev, hot reload", &c.docker),
			yesNo("Include PostgreSQL?", "pgx pool, sqlc queries, goose migrations", &c.postgres),
		),

		huh.NewGroup(
			huh.NewSelect[hosting]().
				Title("PostgreSQL hosting").
				Options(
					huh.NewOption("Docker", hostingDocker),
					huh.NewOption("External", hostingExternal),
				).
				Value(&c.pgHosting),
		).WithHideFunc(func() bool {
			return !c.postgres || !c.docker
		}),

		huh.NewGroup(
			packageSelect(&c.packages),
		),

		huh.NewGroup(
			huh.NewSelect[bool]().
				Title("Apply changes?").
				Options(
					huh.NewOption("Apply", true),
					huh.NewOption("Cancel", false),
				).
				Value(&confirmed),
		),
	).WithTheme(huh.ThemeFunc(theme))

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return false, errUserAborted
		}

		return false, fmt.Errorf("run form: %w", err)
	}

	return confirmed, nil
}

// yesNo builds a Yes/No select bound to value. The current value is the
// initial choice.
func yesNo(title, description string, value *bool) *huh.Select[bool] {
	return huh.NewSelect[bool]().
		Title(title).
		Description(description).
		Options(
			huh.NewOption("Yes", true),
			huh.NewOption("No", false),
		).
		Value(value)
}

// packageSelect builds a multi-select over optionalPackages bound to names.
// The packages already in names start selected.
func packageSelect(names *[]string) *huh.MultiSelect[string] {
	options := make([]huh.Option[string], len(optionalPackages))
	for i, p := range optionalPackages {
		options[i] = huh.NewOption(p.label, p.name)
	}

	return huh.NewMultiSelect[string]().
		Title("Packages to include").
		Description("Deselect any you don't need.").
		Options(options...).
		Value(names)
}

// theme returns a minimal monotone theme with checkbox-style indicators.
func theme(isDark bool) *huh.Styles {
	styles := huh.ThemeBase(isDark)
	styles.Focused.SelectedPrefix = lipgloss.NewStyle().SetString("[x] ")
	styles.Focused.UnselectedPrefix = lipgloss.NewStyle().SetString("[ ] ")
	styles.Blurred.SelectedPrefix = styles.Focused.SelectedPrefix
	styles.Blurred.UnselectedPrefix = styles.Focused.UnselectedPrefix

	return styles
}
