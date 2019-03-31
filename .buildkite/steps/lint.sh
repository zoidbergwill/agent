#!/bin/bash
set -euo pipefail

GO111MODULE=off go get github.com/golangci/golangci-lint/cmd/golangci-lint

golangci-lint run