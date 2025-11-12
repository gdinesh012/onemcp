#!/bin/bash

# Start mock servers and test manually
export PATH="$(pwd)/test/mock-binaries:$PATH"
export ONEMCP_CONFIG=.onemcp-test-manual.json

# Create test config
cat > .onemcp-test-manual.json << 'TESTCONFIG'
{
  "settings": {
    "searchResultLimit": 5,
    "searchProvider": "claude",
    "claudeModel": "haiku"
  },
  "mcpServers": {}
}
TESTCONFIG

# Build and run briefly to see logs
go build -o one-mcp-test ./cmd/one-mcp
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}' | timeout 2 ./one-mcp-test 2>&1 | head -20

rm -f .onemcp-test-manual.json one-mcp-test
