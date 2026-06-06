# Feature Request: SQLite DB Path in `ads list`

## Problem Statement
The user requested the full absolute filesystem path of the SQLite session database to be printed directly within the `ads list` command. This enables users to seamlessly copy the path and open the file via external GUI database management tools (like DBeaver or DataGrip) for raw analytical querying or forensics.

## Expected Behavior
1. The `ads list` command must dynamically compute the absolute `dbPath` utilizing `config.InitAppDataDir()`.
2. The UI table must include a new column (`DB PATH`) containing this value.

## Actions Taken
- Created isolated Git branch `feature/list-db-path`.
- Modified the `printTable` closure within `cmd/ads/main.go` to inject the `appDir` string logic.
- Expanded the `tabwriter` formatting directives to successfully append the `DB PATH` column.
- Triggered minor version bump to `v4.9.0`.
