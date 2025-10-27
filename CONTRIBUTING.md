# Contributing to Egg

Thank you for your interest in contributing to Egg! This document provides guidelines and information for contributors.

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you agree to uphold this code.

## Getting Started

### Prerequisites

- Go 1.21 or later
- Git
- Make (optional, for using Makefile targets)

### Development Setup

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/your-username/egg.git
   cd egg
   ```
3. Set up the workspace:
   ```bash
   go work use ./core ./runtimex ./connectx ./obsx ./k8sx ./storex
   ```
4. Install development tools:
   ```bash
   make tools
   ```

## Development Guidelines

### Code Style

- Follow Go's standard formatting with `gofmt`
- Use `golangci-lint` for code analysis
- Write comprehensive tests for new functionality
- Document all exported functions and types

### Commit Messages

Use conventional commit format:

```
type(scope): description

[optional body]

[optional footer(s)]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes
- `refactor`: Code refactoring
- `test`: Test changes
- `chore`: Build process or auxiliary tool changes

Examples:
```
feat(core): add structured error handling
fix(connectx): resolve interceptor ordering issue
docs(readme): update installation instructions
```

### Testing

- Write unit tests for all new functionality
- Maintain or improve test coverage
- Use table-driven tests where appropriate
- Test error conditions and edge cases

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...
```

### Documentation

- Update README.md for user-facing changes
- Add or update package documentation
- Include usage examples for new features
- Update CHANGELOG.md for significant changes

## Pull Request Process

### Before Submitting

1. Ensure your code builds without errors:
   ```bash
   go build ./...
   ```
2. Run the linter:
   ```bash
   golangci-lint run
   ```
3. Run all tests:
   ```bash
   go test ./...
   ```
4. Update documentation as needed

### Submitting a Pull Request

1. Create a feature branch from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```
2. Make your changes and commit them:
   ```bash
   git add .
   git commit -m "feat(scope): your feature description"
   ```
3. Push your branch:
   ```bash
   git push origin feature/your-feature-name
   ```
4. Create a Pull Request on GitHub

### Pull Request Guidelines

- Provide a clear description of changes
- Reference any related issues
- Include screenshots for UI changes
- Ensure CI checks pass
- Request review from maintainers

## Module Structure

When adding new modules:

1. Create the module directory
2. Initialize the module:
   ```bash
   cd newmodule
   go mod init go.eggybyte.com/egg/newmodule
   ```
3. Add to workspace:
   ```bash
   cd ..
   go work use ./newmodule
   ```
4. Follow the established patterns:
   - Keep interfaces in `core/` minimal and stable
   - Implement functionality in satellite modules
   - Use dependency injection for testability

## Release Process

Releases are managed by maintainers:

1. Update version numbers in go.mod files
2. Update CHANGELOG.md
3. Create a git tag
4. Publish to GitHub releases

## Getting Help

- Open an issue for bug reports or feature requests
- Join discussions in GitHub Discussions
- Contact maintainers for private concerns

## License

By contributing to Egg, you agree that your contributions will be licensed under the MIT License.

## Recognition

Contributors will be recognized in:
- CHANGELOG.md for significant contributions
- README.md for major contributors
- Release notes for specific contributions

Thank you for contributing to Egg!
