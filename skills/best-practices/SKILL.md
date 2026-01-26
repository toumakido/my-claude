---
name: best-practices
description: Automatically enforce Claude Code best practices. Use proactively when creating skills, editing CLAUDE.md, managing context, or implementing features.
---

# Automatic Best Practices Enforcement

Apply these rules automatically during development.

## Skill Creation Workflow

When creating a new skill:

### Step 1: Validate Necessity

**Create Skill if**:
- Reusable workflow (deploy, review, release)
- Long reference docs (API conventions, style guide)
- On-demand knowledge (not needed every session)

**Use CLAUDE.md if**:
- Always-on rules ("Use pnpm", "Run tests before commit")
- Short critical info (< 500 lines total)
- Cannot be inferred from code

**Use MCP if**:
- External service connection (Slack, database)

### Step 2: Set Frontmatter

**Side-effect workflows** (deploy, commit, send-message):
```yaml
disable-model-invocation: true  # Prevent automatic execution
context: fork  # Isolate logs from main session
```

**Background knowledge** (not user-invocable):
```yaml
user-invocable: false  # Hide from menu
```

**Heavy file reading**:
```yaml
context: fork  # Isolate context
agent: Explore  # Fast read-only agent
```

**Tool restriction**:
```yaml
allowed-tools: Read, Grep, Glob  # Read-only
```

### Step 3: Write Description

Include:
- When to use clearly
- "Use proactively" for automatic invocation
- Specific trigger conditions

Example:
```yaml
description: Expert code reviewer. Use proactively after code changes to review quality, security, and maintainability.
```

### Step 4: Validate Before Completion

Run these checks:
```bash
# Check line count (must be < 500)
wc -l skills/SKILL-NAME/SKILL.md

# Verify description exists
grep "description:" skills/SKILL-NAME/SKILL.md

# Check frontmatter for side-effect workflows
grep -E "(deploy|commit|send|push|delete)" skills/SKILL-NAME/SKILL.md
```

If contains side-effects → verify `disable-model-invocation: true`

## CLAUDE.md Management

### Before Editing

Check current line count:
```bash
wc -l CLAUDE.md
```

### After Editing

1. **Verify line count < 500**:
```bash
wc -l CLAUDE.md
```

2. **If > 500 lines**: Immediately move reference content to skills
   - Long API conventions → skills/api-conventions/SKILL.md
   - Testing patterns → skills/testing-patterns/SKILL.md
   - Style guides → skills/style-guide/SKILL.md

### What to Include in CLAUDE.md

**Include**:
- Commands Claude cannot infer (`npm run build:prod`)
- Always/Never rules ("Never commit secrets")
- Project-specific decisions ("Use Zustand, not Redux")
- Test commands (`npm test -- --coverage`)

**Do NOT include**:
- Long reference docs (move to skills)
- File descriptions (delete)
- Self-evident practices (delete)
- Frequently changing info (delete)

## Implementation Workflow

### Before Implementation

**Non-trivial changes** → Use Plan mode:
- Multiple files affected
- Approach uncertain
- Unfamiliar code

**Heavy file reading** → Use subagent:
```
Use a subagent to investigate how authentication works.
Return summary only.
```

### During Implementation

**Always combine**:
- Implementation
- Verification method
- Running verification

Example:
```
Implement OAuth flow, write tests, run test suite and fix failures.
```

### After Implementation

**Include verification**:
- Test suite execution
- Lint/type checking
- Expected output validation
- Screenshot comparison (for UI)

**Never complete without**:
- Verification method
- Verification execution
- Verification passed

## Session Management

### When to /clear

**After 2 failed corrections**:
```
1st attempt failed
2nd attempt failed
→ /clear
→ Restart with better prompt including learnings
```

**Between unrelated tasks**:
```
Task A completed
→ /clear
→ Start Task B
```

**Auto-compaction triggered repeatedly**:
→ `/clear` to reset context

### Context Isolation

**Heavy file reading** → Subagent:
```
# Bad: Pollutes main context
Read all files in src/auth/ and understand sessions

# Good: Isolated context
Use a subagent to research session management in src/auth/.
Return summary with key findings.
```

**Large investigations** → Subagent:
```
Use subagents to research authentication, database, and API modules in parallel
```

## Quick Decision Matrix

| Scenario | Action |
|----------|--------|
| Creating deploy workflow | `disable-model-invocation: true`, `context: fork` |
| Creating commit workflow | `disable-model-invocation: true` |
| Creating API reference (500+ lines) | Regular skill, content in SKILL.md |
| Creating code reviewer | `allowed-tools: Read, Grep, Glob, Bash` |
| CLAUDE.md > 500 lines | Move reference content to skills immediately |
| Reading 50+ files | Use subagent, return summary |
| 2nd correction failed | `/clear`, restart with better prompt |
| Between unrelated tasks | `/clear` |
| Implementing feature | Include verification (tests, lint, output) |

## Validation Checklist

### After Skill Creation

- [ ] Description is specific and includes "when to use"
- [ ] Side-effect workflows have `disable-model-invocation: true`
- [ ] Heavy file reading has `context: fork`
- [ ] SKILL.md < 500 lines
- [ ] Tool restrictions set if needed
- [ ] `$ARGUMENTS` placed correctly if used

### After CLAUDE.md Edit

- [ ] Total lines < 500
- [ ] Only includes non-inferable rules
- [ ] No long reference docs (moved to skills)
- [ ] No file descriptions (deleted)

### After Implementation

- [ ] Verification method included
- [ ] Verification executed
- [ ] All verifications passed
- [ ] Used Plan mode for non-trivial changes
- [ ] Used subagents for heavy file reading

## Common Patterns

### Pattern 1: Deploy Workflow

```yaml
---
name: deploy
description: Deploy application to staging or production
disable-model-invocation: true
context: fork
argument-hint: [staging|production]
---

Deploy to $ARGUMENTS:
1. Run tests
2. Build
3. Deploy
4. Verify
5. Report status
```

### Pattern 2: Code Reviewer

```yaml
---
name: code-reviewer
description: Expert code reviewer. Use proactively after code changes.
allowed-tools: Read, Grep, Glob, Bash
---

1. Run git diff
2. Review changes
3. Check: quality, security, tests
4. Report findings by priority
```

### Pattern 3: Deep Research

```yaml
---
name: deep-research
description: Research codebase thoroughly
context: fork
agent: Explore
argument-hint: [topic]
---

Research $ARGUMENTS:
1. Find relevant files
2. Read and analyze
3. Map architecture
4. Return summary with file references
```

### Pattern 4: API Reference

```yaml
---
name: api-conventions
description: API design patterns. Use when implementing or reviewing API endpoints.
---

Core principles:
- RESTful naming
- Consistent errors
- Request validation

Detailed patterns:
- URL structure
- HTTP methods
- Error formats
(Include full reference directly in SKILL.md)
```

## Anti-Patterns to Avoid

### ❌ Missing verification
```
Implement the feature
```

### ✅ Include verification
```
Implement the feature, write tests, run test suite and fix failures
```

---

### ❌ No context isolation for heavy reading
```
Read all files in src/ and understand the architecture
```

### ✅ Use subagent
```
Use a subagent to research architecture in src/. Return summary.
```

---

### ❌ Allow automatic execution of side-effects
```yaml
---
name: deploy
description: Deploy to production
---
```

### ✅ Prevent automatic execution
```yaml
---
name: deploy
description: Deploy to production
disable-model-invocation: true
context: fork
---
```

---

### ❌ CLAUDE.md > 500 lines with reference docs
```markdown
# API Conventions
(500 lines of API patterns)
```

### ✅ Move to skill
```markdown
# API Conventions
See `/api-conventions` skill

# Create skills/api-conventions/SKILL.md
```

## Enforcement

These rules are automatically enforced. When:

- Creating skills → Apply creation checklist
- Editing CLAUDE.md → Verify < 500 lines
- Implementing features → Include verification
- Managing context → Use subagents for heavy reading
- Session management → `/clear` appropriately

Non-compliance results in:
- Skills without proper frontmatter
- CLAUDE.md > 500 lines (rules ignored)
- Implementations without verification (bugs shipped)
- Context pollution (performance degraded)
