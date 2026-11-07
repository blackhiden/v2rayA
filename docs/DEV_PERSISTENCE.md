# Local Development Persistence

This document explains how to keep your local development state (config, database,
servers, subscriptions, settings) intact when merging changes from the upstream
repository. It is written for contributors who maintain a fork of v2rayA and
regularly pull upstream changes.

---

## Why This Matters

v2rayA stores all user data **locally** — never in the cloud and never in the
repository. That data lives in a config directory that the service creates at
runtime. When you merge from upstream, git only touches files that are **tracked**
in the repository. Your local config and database are **not** tracked, so they
survive merges automatically — **as long as you keep them out of the repo**.

The risk is not the merge itself; it is accidentally committing local state into
the repository, or pointing the service at a config directory that lives inside
the repo tree where a merge could clobber it.

---

## The Golden Rule

> **Never store local runtime state inside the repository tree.**

If your config directory, database, or dev scripts live inside the repo, an
upstream merge can overwrite or delete them. Keep them outside the repo, or in a
git-ignored path.

---

## Recommended Setup

### 1. Use a Dedicated Dev Config Directory

Always run the service with an explicit config directory that is **outside** the
repo, or in a git-ignored path inside the repo.

**Option A — Outside the repo (recommended):**

```bash
# Linux / macOS
mkdir -p ~/.config/v2raya-dev
./v2raya --config ~/.config/v2raya-dev --lite

# Windows
mkdir %APPDATA%\v2raya-dev
v2raya.exe --config %APPDATA%\v2raya-dev --lite
```

**Option B — Inside the repo but git-ignored:**

The `/dev-config/` directory is already in `.gitignore`. Use it for quick local
iteration:

```bash
./v2raya --config ./dev-config --lite
```

Either way, your servers, subscriptions, settings, and routing rules persist
across upstream merges because git never touches those files.

### 2. Keep the Database Out of Git

The SQLite database (`v2raya.db`), its WAL/SHM sidecars, and any BoltDB files are
already covered by `.gitignore` patterns:

```
*.db
*.db-shm
*.db-wal
*.sqlite
*.sqlite3
*.bolt
*.boltdb
```

If you ever see a `*.db` file in `git status`, stop and add it to `.gitignore`
before committing.

### 3. Keep Dev Scripts Out of Git

Local helper scripts, notes, and environment overrides should not be committed.
The `.gitignore` already covers:

```
/dev-*.sh
/dev-*.ps1
/dev-*.md
/dev-*.json
/dev-*.yaml
/dev-*.yml
.env.local
.env.development.local
.env.test.local
.env.production.local
```

Name your local scripts with a `dev-` prefix (e.g. `dev-run.sh`, `dev-build.sh`)
so they are automatically ignored.

---

## Merge Workflow

### Safe Merge Steps

```bash
# 1. Fetch upstream changes
git fetch upstream

# 2. Stash any uncommitted local changes (if any)
git stash

# 3. Merge upstream into your branch
git merge upstream/main

# 4. Resolve any conflicts (code only — your config/data is untouched)

# 5. Restore stashed changes
git stash pop

# 6. Verify your local config is intact
ls ~/.config/v2raya-dev/   # or ./dev-config/
```

### What Survives a Merge

| Item | Survives? | Why |
|------|-----------|-----|
| Config directory (`~/.config/v2raya-dev/`) | ✅ | Outside repo, never touched by git |
| SQLite database (`v2raya.db`) | ✅ | Git-ignored |
| Servers, subscriptions, settings | ✅ | Stored in the config dir |
| Routing rules | ✅ | Stored in the config dir |
| Dev scripts (`dev-*.sh`) | ✅ | Git-ignored |
| Environment overrides (`.env.local`) | ✅ | Git-ignored |
| Code changes | ⚠️ | May conflict; resolve manually |
| Docs | ⚠️ | May conflict; resolve manually |

### What Does NOT Survive

| Item | Why |
|------|-----|
| Uncommitted code changes | Git requires them to be committed or stashed |
| Files tracked by git that you modified locally | Merge may overwrite them |
| Anything stored in a non-ignored path inside the repo | Merge may overwrite it |

---

## Recovery

If you accidentally lose local config or data:

1. **Check for backups** — the service does not auto-backup, but you may have
   copies in your shell history, editor, or a previous `git stash`.
2. **Re-import servers** — paste your subscription links or server configs again.
3. **Re-configure settings** — the web GUI makes this quick.
4. **Check `session.md`** — it may contain notes about your previous setup.

---

## Checklist Before Merging

- [ ] `git status` shows no unexpected `*.db`, `*.sqlite`, or config files
- [ ] Your config directory is outside the repo or in a git-ignored path
- [ ] Your dev scripts use the `dev-` prefix (or are otherwise git-ignored)
- [ ] You have no uncommitted changes you care about (or you've stashed them)
- [ ] You know which branch you're merging into and what upstream branch you want

---

## Related

- [Contributing](CONTRIBUTING.md) — development setup and workflow
- [Architecture](ARCHITECTURE.md) — how the service stores and loads config
- [Specification](SPECIFICATION.md) — data models and environment variables