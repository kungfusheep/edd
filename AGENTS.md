# Repository Guidelines

## Project Structure & Module Organization
`main.go` hosts the CLI entrypoint for the ASCII diagram renderer. Core packages live in dedicated directories: `diagram` parses JSON layouts, `layout` and `pathfinding` compute placement and routing, `render` turns layouts into glyphs, `terminal` manages the TUI, `editor` handles interactive editing, and `validation` checks output invariants. High-level scenarios and fixtures live under `tests/`, while `demo/` implements the playback helpers used by `replay.sh` and similar scripted demos.

## Build, Test, and Development Commands
- `go build -o edd ./...` compiles the CLI binary with every package.
- `go run . -i test1.json` launches the interactive editor against a sample diagram.
- `go test ./...` executes unit and integration tests across the repo.
- `./test_jump_labels.sh` reproduces the regression suite for jump-label rendering.

## Coding Style & Naming Conventions
Follow standard Go formatting: run `gofmt -w` (or `go fmt ./...`) before sending patches, and keep imports ordered via `goimports` if available. Use tabs for indentation and wrap lines under 120 characters. Exported types, functions, and interfaces use PascalCase; package-private helpers stick to mixedCase. Prefer descriptive package-level constants over magic numbers when tuning layout heuristics.

## Testing Guidelines
Add or update `_test.go` files alongside the package under test; larger workflow checks belong in `tests/`. Mimic existing table-driven patterns for layout and routing assertions. Run `go test ./tests -run Name` to focus on a scenario, and keep coverage from dipping—new features should ship with targeted assertions or golden outputs. When adding demo transcripts, pair them with validations to guard against regressions.

## Commit & Pull Request Guidelines
Commits in this project are short, action-focused statements (e.g., "sticky headers when scrolling"). Follow the same style: present tense, under 60 characters when possible, and group related edits together. Pull requests should summarize user-impacting changes, note the commands run for validation, and link any relevant issues or recorded demos. Screenshots or terminal capture snippets help reviewers confirm TUI regressions.
