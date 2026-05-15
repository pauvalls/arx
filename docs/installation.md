# Installation

## Via Go Install

```bash
go install github.com/pauvalls/arx/cmd/arx@latest

# Verify
arx --version
```

## Build from Source

```bash
git clone https://github.com/pauvalls/arx.git
cd arx
go build ./cmd/arx
./arx --help
```

## Requirements

- **Go 1.21+** for building from source
- **Git** for `arx diff` command
