# Edenred CLI

Small Go command-line helper that logs into the Edenred Finland API and prints the current lunch and Virike (wellness) wallet balances.

## Requirements

- Go 1.22 or newer
- (Optional) `make` for convenience targets

## Building

```bash
make build
```

This produces the binary in `bin/edenred`. Alternatively, build directly:

```bash
go build -o bin/edenred ./cmd/edenred
```

## Usage

Provide your Edenred credentials via flags:

```bash
bin/edenred --username your.user --password 'your-password'
```

Flags:

- `--format text|json` – choose output format (default `text`)
- `--base-url` – override API base URL (useful for testing)
- `--timeout` – request timeout (default `15s`)

## Testing

```bash
go test ./...
```
