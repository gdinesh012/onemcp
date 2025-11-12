#!/bin/bash

# Quick test - manually start mock servers and try to connect
go test -v -tags=integration ./integration/... -run "TestHTTPIntegrationSuite/TestStreamableHTTPListTools" 2>&1 | tail -30

