## 1. Library Research And Configuration Contract

- [x] 1.1 Do internet research and select a Go library that can load environment variables from a dotenv file and process environment variables for Heimdall.
- [x] 1.2 Define the canonical environment-variable key map for server, storage, polling, routing, authorization, and secret-related settings.
- [x] 1.3 Update the design and setup docs to replace `config.yaml` with a project-root `.env` file and document the YAML-to-dotenv migration path.

## 2. Configuration Files And Ignore Rules

- [x] 2.1 Create a committed `dist.env` file that documents every supported setting with example or placeholder values.
- [x] 2.2 Ensure `.env` is listed in `.gitignore` so local runtime configuration is not committed.

## 3. Runtime Configuration Loading

- [x] 3.1 Replace YAML-based runtime configuration loading with environment-variable loading from the process environment or the project-root `.env` file.
- [x] 3.2 Create a typed config struct populated from either environment variables or the project-root `.env` file by using the library selected in task 1.1.
- [x] 3.3 Implement typed parsing for scalar values, comma-separated lists, named repository blocks, and file-path-backed secret settings in the environment-variable schema.
- [x] 3.4 Add startup validation that rejects missing or malformed required keys, inconsistent routing configuration, and legacy YAML-only deployments.

## 4. Codebase Cleanup

- [x] 4.1 Go through the entire codebase and remove or update remaining `config.yaml` references so runtime configuration comes from environment variables and the project-root `.env` file.
- [x] 4.2 Verify the checked-in docs, examples, and setup guidance consistently reference environment-variable configuration and `dist.env` instead of YAML configuration.

## 5. Behavior Test Coverage

- [x] 5.1 Write Gherkin behavior tests for valid dotenv startup, legacy YAML rejection, multi-repository dotenv routing, environment-variable override behavior, and dotenv validation failures.
- [x] 5.2 Implement or update the step bindings, fixtures, and test runner integration needed to execute the dotenv configuration behavior tests.

## 6. Verification

- [x] 6.1 Run the relevant automated test suite and verify the dotenv configuration scenarios pass.
- [x] 6.2 Verify no `config.yaml`-based runtime configuration references remain in the codebase and that local `.env` files stay untracked while `dist.env` remains committed.
