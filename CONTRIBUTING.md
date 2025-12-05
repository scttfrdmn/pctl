# Contributing to pctl

Thank you for your interest in contributing to pctl! This document provides guidelines and instructions for contributing.

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment for all contributors.

## How to Contribute

### Reporting Bugs

Before creating a bug report:
- Check the [issue tracker](https://github.com/scttfrdmn/pctl/issues) to see if the issue already exists
- Collect relevant information about the bug (version, OS, steps to reproduce)

Create a bug report with:
- Clear, descriptive title
- Detailed description of the issue
- Steps to reproduce
- Expected vs actual behavior
- Environment details (OS, Go version, pctl version)
- Relevant logs or error messages

### Suggesting Enhancements

Enhancement suggestions are welcome! Please:
- Check existing issues and discussions first
- Provide clear use case and rationale
- Describe the proposed solution
- Consider alternative approaches

### Pull Requests

1. **Fork and Clone**
   ```bash
   git clone git@github.com:YOUR_USERNAME/pctl.git
   cd pctl
   ```

2. **Create a Branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make Changes**
   - Write clear, concise code
   - Follow Go best practices
   - Add tests for new functionality
   - Update documentation as needed

4. **Test Your Changes**
   ```bash
   make check  # Runs fmt, vet, lint, and tests
   ```

5. **Commit**
   - Write clear commit messages
   - Reference related issues
   ```bash
   git commit -m "Add feature X to address #123"
   ```

6. **Push and Create PR**
   ```bash
   git push origin feature/your-feature-name
   ```
   - Create PR on GitHub
   - Fill out the PR template
   - Link related issues

## Development Guidelines

### Code Style

- Follow standard Go conventions
- Run `go fmt` before committing
- Use meaningful variable and function names
- Add comments for exported functions and complex logic
- Keep functions focused and concise

### Testing

- Write unit tests for new code
- Aim for >80% code coverage
- Include both positive and negative test cases
- Use table-driven tests where appropriate

```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        // test cases
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

### Documentation

- Update README.md for user-facing changes
- Add or update godoc comments
- Update CHANGELOG.md following [Keep a Changelog](https://keepachangelog.com/)
- Add examples for new features

### Quality Standards

This project maintains an A+ rating on [Go Report Card](https://goreportcard.com/). All contributions must:
- Pass `go fmt`
- Pass `go vet`
- Pass `golangci-lint`
- Pass all existing tests
- Maintain or improve code coverage

Run all checks with:
```bash
make check
```

## Project Structure

```
pctl/
├── cmd/pctl/              # CLI entry point and commands
├── pkg/                   # Public packages
├── internal/              # Private packages
├── tests/                 # Test suites
├── docs/                  # Documentation
└── seeds/             # Example templates
```

- `cmd/`: Command-line interface code
- `pkg/`: Reusable packages (public API)
- `internal/`: Internal packages (not for external use)
- `tests/`: Integration and end-to-end tests

## Commit Message Guidelines

Format:
```
<type>: <subject>

<body>

<footer>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

Example:
```
feat: add template validation for S3 mounts

Implements validation to ensure S3 bucket names are valid and
mount points are absolute paths. Adds corresponding tests.

Closes #123
```

## Versioning

This project follows [Semantic Versioning 2.0.0](https://semver.org/):
- MAJOR: Incompatible API changes
- MINOR: Backwards-compatible functionality additions
- PATCH: Backwards-compatible bug fixes

## Release Process

1. Update CHANGELOG.md with release notes
2. Update version in appropriate files
3. Create a git tag: `git tag -a v1.0.0 -m "Release v1.0.0"`
4. Push tag: `git push origin v1.0.0`
5. GitHub Actions will automatically create the release

## Getting Help

- Check the [documentation](docs/)
- Search existing [issues](https://github.com/scttfrdmn/pctl/issues)
- Ask in [discussions](https://github.com/scttfrdmn/pctl/discussions)

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
