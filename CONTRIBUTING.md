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

## Contributor License Agreement

By contributing to this project, you agree to the following terms:

### Grant of Rights
You grant Linky-Link-Linky a perpetual, worldwide, non-exclusive, royalty-free, irrevocable license to use, reproduce, distribute, prepare derivative works of, publicly perform, and publicly display your contributions, under the terms of the Apache 2.0 License.

### Representations
You represent that:
- Each contribution is your original work, or you have the right to submit it
- Your contribution does not violate any third-party rights or agreements

### Signature
When you submit your first pull request, the CLA Assistant bot will prompt you to sign this agreement. Signing is required once per GitHub account before contributions can be merged.

The full CLA text is available at:
https://github.com/Linky-Link-Linky/Agent-Nervous-System/blob/master/CONTRIBUTING.md
