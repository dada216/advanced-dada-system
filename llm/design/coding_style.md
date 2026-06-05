# Advanced Dada System (ADS) - Coding Style

This document defines the strict coding conventions for the ADS project. It implements the guidelines from [Effective Go](https://go.dev/doc/effective_go), which has been securely fetched and archived locally in `llm/design/effective_go.md`.

## 1. Formatting
- **gofmt**: All code MUST be formatted with `gofmt` (or `goimports`). No exceptions. This is enforced by our `Makefile` and `golangci-lint`.

## 2. Naming Conventions
- **Package Names**: Short, concise, lowercase. Avoid broad names like `util`, `common`, or `misc`.
- **Getters/Setters**: Do not use `Get` as a prefix for getters. A method to get `Owner` should be named `Owner()`, not `GetOwner()`. A setter should be `SetOwner()`.
- **Interfaces**: One-method interfaces should be named by the method name plus an `-er` suffix (e.g., `Reader`, `Writer`, `Formatter`).
- **MixedCaps**: Use `MixedCaps` or `mixedCaps` rather than underscores to write multi-word names.

## 3. Control Structures & Error Handling
- **Early Returns**: Handle errors immediately and return early. Keep the "happy path" at the leftmost indentation level to reduce nesting.
- **Example**:
  ```go
  // CORRECT
  f, err := os.Open(name)
  if err != nil {
      return err
  }
  // happy path continues here un-nested
  ```

## 4. Concurrency
- **Share Memory by Communicating**: Do not communicate by sharing memory; instead, share memory by communicating (using channels).
- Use `sync.Mutex` only for simple state protection, prefer channels for orchestration.

## 5. Defer
- Use `defer` heavily for resource cleanup close to the allocation (e.g., `defer f.Close()`, `defer db.Close()`, `defer mu.Unlock()`).

## 6. Data Allocation
- Use `new(T)` to allocate memory for a pointer to a zero-valued `T`.
- Use `make(T, args)` to initialize slices, maps, and channels.

## 7. Comments
- Every exported (capitalized) name in a program should have a doc comment.
- The comment must begin with the name of the exported element it describes and end with a period.

## Enforcement
These rules are unconditionally binding for all AI agents and human contributors interacting with the codebase. Before finalizing any task, agents MUST run `make lint` to mechanically verify adherence.
