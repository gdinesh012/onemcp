#!/bin/bash
set -x

# Create simple config
cat > .onemcp-debug.json << 'CONFIG'
{
  "settings": {
    "searchResultLimit": 5,
    "searchProvider": "claude",
    "claudeModel": "haiku"
  },
  "mcpServers": {}
}
CONFIG

# Build and run with mock PATH
export PATH="$(pwd)/test/mock-binaries:$PATH"
export ONEMCP_CONFIG=.onemcp-debug.json

go build -o one-mcp-debug ./cmd/one-mcp

# Send initialize and check response
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}' | ./one-mcp-debug 2>&1 | head -30

rm -f .onemcp-debug.json one-mcp-debug
