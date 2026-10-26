# Changelog

## Unreleased

### Added

- Added persisted active proxy-group selection. The node section now provides a **PROXY GROUP** table with radio-button selection, member counts, and active status. New default proxy connections use the selected group, while explicit named routing targets remain unchanged.
- Added local development persistence guidance. New `docs/DEV_PERSISTENCE.md` documents how to keep local config, database, and dev scripts intact during upstream merges. `.gitignore` now covers `/data/`, `/dev-config/`, database files, env overrides, and `dev-*` scripts.