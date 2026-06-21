# Contributing to Agent Nervous System

## CLA Requirement

All contributors must sign a [Contributor License Agreement](CLA.md) before their pull requests can be merged. This ensures the project retains the rights needed to distribute your work and potentially relicense in the context of an acquisition.

The CLA check runs automatically on every pull request.

### For Individual Contributors

Read [CLA.md](CLA.md), then comment on your PR with:

> I have read the CLA and I hereby sign it.

A bot will record your signature and update the PR status.

### For Corporate Contributors

If you are contributing on behalf of your employer, your company must sign a [Corporate CLA](CLA-CORPORATE.md) first. Email **cla@agent-nervous-system.com** to initiate the process.

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
4. Sign the CLA (see above).
5. PRs require at least one maintainer review.
