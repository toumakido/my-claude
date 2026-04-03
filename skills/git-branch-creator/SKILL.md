---
name: git-branch-creator
description: >
  Guides new git branch creation by checking the current branch and confirming the base branch
  when not on the default branch. Use when the user requests creating a new git branch with
  phrases like 'create a branch', 'git checkout -b', 'git switch -c', 'new branch', or similar
  branch creation requests.
---

# Git Branch Creator

## Workflow

When the user requests creating a new git branch:

1. **Check current branch and default branch** (run in parallel):
   - `git branch --show-current` — get current branch name
   - `git remote show origin 2>/dev/null | grep 'HEAD branch' | awk '{print $NF}'` — get default branch (fallback: check if `main` or `master` exists locally)

2. **Decide whether to confirm**:
   - If on the default branch → proceed to create the branch directly (no confirmation needed)
   - If on a non-default branch → ask the user where to branch from using AskUserQuestion:
     ```
     Question: "現在 <current-branch> にいます。どこからブランチを切りますか？"
     Options:
     - "<default-branch>（デフォルトブランチ）" → checkout default branch first, then create
     - "<current-branch>（現在のブランチ）" → create from current branch as-is
     ```

3. **Create the branch**:
   - From default branch: `git checkout <default-branch> && git pull && git checkout -b <new-branch>`
   - From current branch: `git checkout -b <new-branch>`
