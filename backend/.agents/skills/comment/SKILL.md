---
name: code-comments
description: >
  Apply this skill whenever the user asks about commenting code, adding comments,
  reviewing comment quality, cleaning up comments, or making comments consistent
  across a codebase. Also trigger when the user says things like "add comments to
  this", "review my comments", "clean up comments", "too many comments", "missing
  comments", or "make comments consistent". Use this skill even if the request is
  phrased casually — e.g. "can you comment this function" or "the comments in this
  file are a mess". This skill ensures comments are consistent, meaningful, and
  non-redundant across any language or codebase.
---

# Code Comments Skill

Produce comments that are consistent, purposeful, and non-redundant. The goal is
comments that add information the code itself cannot convey — not comments that
restate it.

---

## Core Rules

### Comment WHEN:
- A function/method's purpose isn't obvious from its name alone
- There's non-trivial logic, a workaround, or a "why not the obvious approach" decision
- A magic number or constant needs context
- An external constraint, API quirk, or bug fix drives a specific implementation
- Public/exported API surface needs a doc comment
- The first occurrence of a repeated pattern (not every occurrence)

### Do NOT Comment WHEN:
- The code is self-explanatory (`i++`, `return err`, `user.Save()`)
- The comment just restates what the code does in English
- A better name would eliminate the need for a comment entirely
- The same pattern has already been explained earlier in the file

---

## Style Rules

- **Doc comments**: All exported/public symbols get doc comments in the language-native format
  - Go: `// FunctionName does X` (GoDoc style, starts with symbol name)
  - JS/TS: JSDoc (`/** ... */`) for exported functions and classes
  - Python: docstrings for public functions, classes, and modules
- **Inline comments**: Explain *why*, not *what*
- **Tags**: Use `TODO`, `FIXME`, `HACK` with a short reason — e.g. `// TODO: replace with streaming once API supports it`
- **Abstraction level**: Comments should match the abstraction of surrounding code — don't explain assembly-level steps inside a high-level business logic function
- **Tone and tense**: Match the existing file's comment style — don't mix tenses or formality levels
- **Style consistency**: Don't mix `//` and `/* */` in the same file without a reason

---

## Consistency Rules (for codebases, not single files)

- If a pattern appears more than twice, comment it once (at first occurrence or in a shared doc) — not every time
- Match the comment density of the existing codebase — don't over-comment a file that uses sparse comments, or under-comment one that documents heavily
- When reviewing a codebase, flag inconsistencies: missing doc comments on exported symbols, redundant inline comments, mixed tag formats

---

## Language-Specific Defaults

**Go**: GoDoc on all exported identifiers. Inline `//` only. No `/* */` except build tags.

**TypeScript/JavaScript**: JSDoc on exported functions/classes. Inline `//` elsewhere.

**Python**: Docstrings (triple-quoted) on all public functions/classes/modules. Inline `#` for implementation notes.

**Other languages**: Follow the dominant convention in the existing file.

---

## When Reviewing Existing Comments

1. Identify and remove comments that restate the code
2. Identify missing doc comments on public symbols
3. Flag `TODO`/`FIXME` without reasons and ask for context
4. Normalize mixed styles within a file
5. Collapse repeated pattern comments to the first occurrence

---

## Output Format

When adding or fixing comments:
- Return the modified code with changes clearly visible
- If reviewing a codebase, output a summary: what was added, what was removed, what was changed, and why
- If a comment was removed because a rename would be better, suggest the rename

---

## Quick Reference

| Situation | Action |
|---|---|
| Exported function with no comment | Add doc comment |
| Comment restates the code | Delete it |
| Magic number | Add inline comment with context |
| Workaround or hack | Add `// HACK:` or `// NOTE:` with reason |
| Pattern repeated 3+ times | Comment first instance only |
| Mixed `//` and `/* */` | Normalize to file's dominant style |
| `TODO` with no reason | Ask for context before adding |
