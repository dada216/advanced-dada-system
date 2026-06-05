# Bug Report: Federated Search Missing Databases

## Description
When executing `ads search` (often triggered via the Tmux interactive prompt `Prefix + s`), the federated search mechanism does not always search across every other session's database. Some session databases are missing from the search results scope.

## Steps to Reproduce (Expected)
1. Create multiple sessions using `ads new` and run them.
2. Generate distinct output in each session.
3. Use `ads search <query>` (or the Tmux `Prefix + s` shortcut) from one session.
4. Notice that hits from some other sessions might not be displayed.

## Investigation Notes
- Needs an automated test to create multiple sessions, write distinct data, and search to verify all DBs are attached.
- Look into `cmd/ads/search.go` or the underlying `search` logic to ensure all session files in `~/.local/share/ads/sessions/` are being properly attached via `ATTACH DATABASE`.
