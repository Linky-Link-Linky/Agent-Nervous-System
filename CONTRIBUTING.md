# Contributing to Agent Nervous System

## Development Setup

### Prerequisites
- Go 1.21+
- Python 3.10+ (for Python SDK)
- Node.js 18+ (for TypeScript SDK)

### Building

```bash
cd ans
go build ./...
```

### Testing

```bash
cd ans
go test ./...
```

### Running the linter

```bash
cd ans
go vet ./...
staticcheck ./...
gosec -quiet ./...
```

## Pull Request Process

1. Fork the repo and create your branch from `master`.
2. Run tests and linters before submitting.
3. Include a clear description of the change.
4. PRs require at least one maintainer review.
