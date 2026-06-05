# Bug/Feature Report: ads run Auto-Init

## Description
Currently, when a user executes `ads run <session-name>` and the session does not exist, the orchestrator instantly exits with an abrupt error ("session not found").
The user requested an improvement: if the session doesn't exist, `ads run` should intercept the error, display a warning, and interactively prompt the user to automatically create a default local session matching that name.

## Expected Behavior
- Execute `ads run <session-name>`.
- If the session does not exist, intercept the "not found" error.
- Prompt the user: "Session does not exist. Do you want to create a new local session? (y/N)"
- If the user types 'y' or 'yes', invoke the `db.CreateLocalSession()` primitives to initialize it natively, and then immediately continue to `orchestrator.Run()` to attach to it.
- If the user types 'n', exit gracefully with a warning.
- This represents a significant user experience improvement and should bump the semantic version to `v1.0.0`.
