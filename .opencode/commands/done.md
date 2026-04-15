---
description: Finalize an OpenSpec change by syncing specs, archiving it, and committing/pushing the branch
---

Finalize an OpenSpec change after implementation is complete.

**Input**: Optionally specify a change name (e.g., `/done add-auth`). If omitted, check if it can be inferred from conversation context. If vague or ambiguous you MUST prompt for available changes.

**Steps**

1. **Select the change**

   If a name is provided, use it. Otherwise:
   - Infer from conversation context if the user mentioned a change
   - Auto-select if only one active change exists
   - If ambiguous, run `openspec list --json` to get available changes and use the **AskUserQuestion tool** to let the user select

   Always announce: "Using change: <name>" and how to override (e.g., `/done <other>`).

2. **Check readiness**

   ```bash
   openspec status --change "<name>" --json
   ```

   Parse the JSON to understand:
   - `schemaName`: The workflow being used
   - `artifacts`: List of artifacts with their status (`done` or other)

   Read the tasks file if it exists and count complete vs incomplete tasks.

   **If any artifacts or tasks are incomplete:**
   - Display a warning listing what remains
   - Prompt the user for confirmation to continue
   - Proceed only if the user confirms

3. **Merge delta specs into the main specs**

   Check for delta specs at `openspec/changes/<name>/specs/`. If none exist, note "No delta specs" and continue.

   **If delta specs exist:**
   - Compare each delta spec with its corresponding main spec at `openspec/specs/<capability>/spec.md`
   - Apply the delta into the main spec while preserving valid spec structure
   - Carry over additions, modifications, removals, and renames from the delta spec
   - Show a concise summary of what changed in the main specs

   **Pause if:**
   - The target main spec does not exist and the destination is unclear
   - The delta implies an ambiguous removal or rewrite
   - The merge would discard manual edits that are not clearly part of the delta

4. **Archive the change**

   Create the archive directory if it doesn't exist:
   ```bash
   mkdir -p openspec/changes/archive
   ```

   Generate target name using current date: `YYYY-MM-DD-<change-name>`

   **Check if target already exists:**
   - If yes: Fail with an error and suggest renaming the existing archive or using a different date
   - If no: Move the change directory to archive

   ```bash
   mv openspec/changes/<name> openspec/changes/archive/YYYY-MM-DD-<name>
   ```

5. **Review git state and draft the commit message**

   Review the current branch state using:
   ```bash
   git status --short
   git diff --staged
   git diff
   git log --oneline -5
   ```

   Then:
   - Identify the files that belong to this change finalization
   - Stage the synced main spec files, the archived change directory, and other implementation files that clearly belong to the same change
   - Leave unrelated user changes untouched
   - Draft a concise commit message that explains why the change is being finalized, not just which files changed

   **Do not stage or commit:**
   - Files that appear to contain secrets
   - Unrelated work that is not part of the change being finalized

6. **Commit and push to the tracked remote branch**

   Create the commit after staging the relevant files.

   Determine the current branch and its upstream tracking branch.

   **If no upstream exists:**
   - Pause and ask the user where to push

   **If an upstream exists:**
   - Push to the tracked remote branch
   - Do not use force push unless the user explicitly asks for it

7. **Display summary**

   Show:
   - Change name
   - Schema
   - Main spec sync summary
   - Archive location
   - Commit hash and commit message
   - Branch and remote push target

**Output During Finalization**

```
## Finalizing: <change-name> (schema: <schema-name>)

Syncing delta specs into main specs
[...sync happening...]
✓ Specs synced

Archiving change
[...archiving happening...]
✓ Change archived

Creating commit and pushing branch
[...git happening...]
✓ Commit created and pushed
```

**Output On Completion**

```
## Finalization Complete

**Change:** <change-name>
**Schema:** <schema-name>
**Specs:** <sync summary>
**Archived to:** openspec/changes/archive/YYYY-MM-DD-<name>/
**Commit:** <sha> <message>
**Pushed to:** <remote>/<branch>

OpenSpec finalization complete.
```

**Output On Pause (Issue Encountered)**

```
## Finalization Paused

**Change:** <change-name>
**Schema:** <schema-name>
**Stage:** <spec-sync|archive|git>

### Issue Encountered
<description of the issue>

**Options:**
1. <option 1>
2. <option 2>
3. Cancel

What would you like to do?
```

**Guardrails**
- Always perform the workflow in order: sync specs, archive the change, commit, then push
- Pause rather than guessing if the spec merge is ambiguous
- If artifacts or tasks are incomplete, require explicit user confirmation before continuing
- Do not commit unrelated changes or files that may contain secrets
- Do not force-push or amend unless the user explicitly asks for it
- Keep the final commit message concise and meaningful, aligned with recent repository history
- If commit or push fails, report the error clearly and stop for guidance
