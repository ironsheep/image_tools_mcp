# image-tools-mcp overlay — baseline-health

## Augments Step 1 — Clean build, zero warnings

The slot `BUILD_COMMAND` (`go build ./... && go vet ./...`) surfaces compiler
and `go vet` warnings, but the project's *CI* warnings gate is
**golangci-lint** — the `lint` job in `.github/workflows/ci.yml` runs
golangci-lint at `version: latest` with default linters and no repo
`.golangci.*` config. To match CI, also run `make lint` (it installs and runs
golangci-lint) and drive its findings to zero alongside the `go vet` output.

Formatting counts as a warning here: Go 1.25 `gofmt` must be clean —
`gofmt -l <packages>` must print nothing — or golangci-lint/CI will flag it.
After a toolchain bump, expect `gofmt -w` to touch files (e.g. trailing-comment
alignment); that is part of getting to a clean baseline, not optional.

## Augments Step 3 — Never allow skips (known cause)

On-disk fixture tests must reference fixtures by a **repo-relative path**
(`../../testdata/...` from a package directory — Go runs tests with CWD set to
the package dir), never an absolute container path. A hardcoded absolute path
(e.g. an old `/workspaces/...` mount) makes the test `t.Skip("fixture not
present")` silently when the container layout changes — a hidden skip, not an
acceptable steady state. Treat any such skip as a path regression to fix, not a
condition to accept.
