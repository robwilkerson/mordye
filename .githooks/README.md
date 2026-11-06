# Git Hooks

This directory contains git hooks that are checked into the repository.

## Activation

To use these hooks, configure git to look here instead of `.git/hooks/`:

```bash
git config core.hooksPath .githooks
```

This only needs to be done once per clone. Or run `just bootstrap` to set this up automatically.

## Hooks

| Hook | Purpose |
|------|---------|
| `pre-commit` | Scans staged files for secrets using Kingfisher, then formats staged Go files with `gofmt` and re-stages them. |
| `post-merge` | On main branch only: prunes deleted remote branches and deletes local branches whose upstream is gone. |

## Requirements

- **Kingfisher** - Secrets scanner. Install via your package manager or from [GitHub](https://github.com/mongodb/kingfisher).

## Bypassing Hooks

In rare cases where you need to bypass a hook:

```bash
git commit --no-verify  # Skip pre-commit
```

Use sparingly and only when you understand why the hook is failing.
