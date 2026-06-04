# image-tools-mcp overlay — build-wrapup

## Augments the version / release-notes recording

This project's release mechanics:

- **Version source of truth** is the repo-root `VERSION` file (a bare semantic
  version, e.g. `1.2.11`). `.github/workflows/release.yml` reads it via
  `cat VERSION`. Bump this file as part of cutting a release — the Makefile's
  `git describe` is only a dev-time fallback.
- **Releases are git-tagged** `vX.Y.Z`, matching the `VERSION` contents.
- **`CHANGELOG.md`** is the `RELEASE_NOTES_DOC` and follows *Keep a Changelog*
  (`## [X.Y.Z] - YYYY-MM-DD` headings with `### Added` / `### Fixed` /
  `### Changed` subsections) and Semantic Versioning.
- **Commit-title convention**: `Release vX.Y.Z - <summary>` for feature
  releases, `Hotfix vX.Y.Z - <summary>` for patch-level fixes (see git log for
  the established pattern).
- Audience (`RELEASE_NOTES_AUDIENCE`) is developers integrating the MCP server,
  so a technical voice is appropriate (CGO linking, alpha channels, soname
  drift are fair game — no need to soften for end users).
