#!/bin/bash
make build
# Launch a test session
./bin/ads new test-history
./bin/ads run test-history << 'INNER'
sleep 0.1
echo 'test_history_command'
sleep 0.1
exit
INNER
sleep 0.5
sqlite3 ~/.local/share/ads/sessions/$(./bin/ads search "" --json | jq -r '.[0].UUID' | head -n 1).db "SELECT * FROM command_history;"
