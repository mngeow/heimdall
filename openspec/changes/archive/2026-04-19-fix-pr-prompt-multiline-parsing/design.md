## Context

The PR comment parser (`internal/slashcmd/handler.go`) currently extracts the prompt tail using mutually exclusive branches:

1. If the command line contains ` -- `, it captures only the inline text after that separator.
2. Else if the command line ends with ` --`, it captures subsequent lines.

This means a comment like:
```
/heimdall refine --agent X -- Perfect, now add for me:
1. first point
2. second point
```
loses the numbered list entirely because branch 1 matches and branch 2 is skipped.

Additionally, GitHub comment bodies may use CRLF (`\r\n`) line endings, which pollutes the multiline prompt with stray `\r` characters. Users also want an explicit delimiter (triple backticks) to wrap complex prompts unambiguously.

## Goals / Non-Goals

**Goals:**
- Combine inline prompt text with subsequent comment lines when both are present.
- Support triple-backtick delimiters around the prompt tail.
- Normalize CRLF line endings to LF during multiline reassembly.
- Update durable specs and tests to codify the new behavior.

**Non-Goals:**
- Changing the single-line ` -- ` separator grammar.
- Supporting nested or mismatched backtick delimiters.
- Modifying the authorization, worker dispatch, or opencode execution paths.

## Decisions

**Decision 1: Append subsequent lines when inline separator is present**
- Rationale: The user intent is clear—if they write ` -- ` on the command line and then add more lines, those lines are part of the prompt. The current silent truncation is a bug.
- Implementation: In `Parse()`, after extracting the inline prompt tail, check if there are lines after the command line. If so, append them separated by `\n`.

**Decision 2: Triple-backtick stripping as a post-processing step**
- Rationale: Backticks are a natural markdown convention. Stripping them after prompt assembly keeps the parser simple and avoids complicating the line-scanning logic.
- Implementation: After the prompt tail is fully assembled (inline + multiline), if the result starts and ends with `\n```\n` or ```, trim the outer backticks and surrounding whitespace.

**Decision 3: CRLF normalization per-line**
- Rationale: `strings.Split(comment, "\n")` on a `\r\n` body leaves `\r` on each line. `strings.TrimSpace` only strips the first and last lines. Normalizing per-line prevents carriage returns from reaching opencode.
- Implementation: Strip `\r` from each line before joining multiline prompts.

## Risks / Trade-offs

- **[Risk]** A user may accidentally include a signature or unrelated text after the prompt in the same comment.  
  **→ Mitigation:** This is acceptable because GitHub PR comments are typically focused on a single command. Users who need isolation can use triple backticks to explicitly bound the prompt.
- **[Risk]** Triple-backtick stripping could accidentally remove backticks that were intended to be part of the prompt.  
  **→ Mitigation:** Only strip when the prompt is wrapped in a complete triple-backtick block (leading and trailing ``` on their own lines or as the exact string boundary). A single inline ` `` ` mid-prompt is preserved.
