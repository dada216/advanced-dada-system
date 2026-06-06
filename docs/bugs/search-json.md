# Feature Request: JSON Output Interface for CLI Search

## Problem Statement
The user requested a `--json` output flag for the CLI `ads search <keyword>` command to allow for programmatic parsing of search results, matching the capability of standard modern CLI developer tools. Currently, the search command only outputs a human-readable text block.

## Expected Behavior
1. `ads search <keyword> --json` should intercept the struct data returned from the SQLite/FTS5 search engine and serialize it directly into formatted JSON, bypassing the human-readable `fmt.Printf` blocks.

## Actions Taken
- Created isolated Git branch `feature/search-json-output`.
- Added the `--json` flag to `searchCmd`.
- Implemented `json.MarshalIndent` pipeline directly dumping the serialized `plugin.SearchResult` array.
- Triggered minor version bump to `v4.5.0`.
