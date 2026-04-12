## 1. Rename core application identity

- [ ] 1.1 Rename the Go module path, import paths, entrypoint/binary references, and other source-level `symphony` identifiers to `heimdall`.
- [ ] 1.2 Update hard-coded workflow naming defaults such as managed-service identity, branch prefixes, bootstrap namespaces, monitor-label examples, and runtime database naming to the Heimdall forms defined in the specs.
- [ ] 1.3 Update configuration parsing, validation, and defaults to use `HEIMDALL_*` keys, Heimdall-branded filesystem paths, and the renamed operator-facing service identity without keeping legacy `SYMPHONY_*` aliases.

## 2. Rename workflow command and repository surfaces

- [ ] 2.1 Update slash-command parsing, help text, and managed pull request handling to use `/heimdall status` and `/heimdall refine` on Heimdall-managed pull requests.
- [ ] 2.2 Update bootstrap generation, branch/worktree naming, and bootstrap file targeting to use `heimdall/<issue-key>-<slug>` branches and `.heimdall/bootstrap/<issue-key>.md` paths.
- [ ] 2.3 Update GitHub label reconciliation, audit/log messaging, and operator-visible workflow naming so PR monitoring and troubleshooting consistently refer to Heimdall.

## 3. Rewrite documentation and OpenSpec artifacts

- [ ] 3.1 Update maintained documentation in `docs/`, the root README, and repo guidance files to use Heimdall naming, commands, env vars, and filesystem examples.
- [ ] 3.2 Rewrite durable specs and active change artifacts under `openspec/` so current product requirements consistently use Heimdall naming and examples.
- [ ] 3.3 Rewrite archived OpenSpec change artifacts under `openspec/changes/archive/` to remove remaining Symphony-branded product references while preserving their original intent.

## 4. Update automated behavior coverage

- [ ] 4.1 Update Gherkin `.feature` files to assert the Heimdall command namespace, branch naming, bootstrap paths, PR labels, and operator-facing examples.
- [ ] 4.2 Update Go BDD step bindings, fixtures, fake clients, and helper assertions to match the renamed Heimdall workflow surfaces and configuration keys.
- [ ] 4.3 Update unit and integration tests that still assert Symphony-branded strings, paths, commands, labels, or module/import references.

## 5. Verify the rename end to end

- [ ] 5.1 Run `go test ./...` and confirm the renamed behavior and unit test suites pass.
- [ ] 5.2 Search the repository for remaining `Symphony`, `symphony`, `/symphony`, `.symphony`, and `SYMPHONY_` references, then remove or document any intentional exceptions before closing the change.
