# Repository guidance

- Read [STYLE.md](STYLE.md), the local [go-style skill](.agents/skills/go-style/SKILL.md), and relevant existing files before editing. Keep detailed language rules in the skill's owning topic references.
- Optimize for a human reading the whole operation. Expand every nonempty struct declaration, struct literal, and keyed map literal, including one-field options, nested arguments, and returns. Use keyed struct fields and preserve gofmt alignment.
- Use purposeful whitespace and small focused functions. Separate I/O, decoding and validation, domain rules, orchestration, and presentation for real ownership reasons, not to create forwarding layers or micro-files.
- Keep domain contracts with their capability and interfaces with their consumers. Avoid unrelated `types`, `helpers`, or `utils` packages.
- Preserve the toolchain in `go.mod`, the `pkg/` layout, setup feature markers, and resource/error contracts. Do not hand-reformat generated code.
- Never add, write, run, propose, or plan tests unless explicitly requested. Formatting and static analysis are separate; inspect Makefile targets before running aggregate commands.
- Review the actual code after gofmt; formatting alone does not establish readability. Preserve unrelated work and keep log messages lowercase except proper nouns.
