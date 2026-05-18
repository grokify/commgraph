# Installation

## Prerequisites

- Go 1.21 or later

## Install from Source

```bash
go install github.com/grokify/commgraph/cmd/commgraph@latest
```

This installs the `commgraph` binary to your `$GOPATH/bin` directory.

## Verify Installation

```bash
commgraph --help
```

You should see output like:

```
CommGraph - Communication graph analysis framework

Usage:
  commgraph [command]

Available Commands:
  analyze     Run graph analysis
  export      Export analysis results
  identity    Manage identity resolution
  ingest      Ingest messages from source
  pipeline    Run full ingestion and analysis pipeline
  help        Help about any command

Flags:
  -h, --help   help for commgraph

Use "commgraph [command] --help" for more information about a command.
```

## Build from Source

To build from source with the latest changes:

```bash
git clone https://github.com/grokify/commgraph.git
cd commgraph
go build -o commgraph ./cmd/commgraph
```

## Configuration

CommGraph can be configured via:

1. Command-line flags (highest priority)
2. Configuration file (`.commgraph.yaml`)
3. Environment variables

See the [Configuration Guide](../guide/configuration.md) for details.
