---
description: 'Required pre-completion checks and project-specific guidelines for connect'
applyTo: "**"
---

# Pre-completion Checks

Before a task is marked as complete, run:

```bash
make verify
```

This automates: `go generate ./...`, `go build .`, `go vet ./...`, `go test -race ./...`, and `shellcheck -o all` on shell scripts. All steps must pass.

# PR Review Workflow

## Finding the PR for the Current Branch

```bash
gh pr view --json number --jq .number
gh pr view
```

## Workflow for Addressing Review Comments

1. Fetch unresolved comments for the current branch's PR
2. Analyze each comment
3. Make code changes to address the issues
4. One commit per comment or related group of issues
5. Run `make verify`
6. Push when all checks pass

## Commit Message Format for Review Fixes

```
fix: address PR {PR_NUMBER} review comments on {topic}

This commit addresses {N} unresolved review comments:

1. {Comment title} (line {X} of {file}.go)
   - {Description of fix}

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
```
