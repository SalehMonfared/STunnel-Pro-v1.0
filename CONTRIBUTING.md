# Contributing to STunnel Pro v1.0

Thank you for your interest in contributing to STunnel Pro v1.0! This document provides guidelines and information for contributors.

**Created by [SalehMonfared](https://github.com/SalehMonfared)**

## 🚀 Getting Started

### Prerequisites

- Go 1.21 or higher
- Node.js 18 or higher
- Docker and Docker Compose
- Git

### Development Setup

1. **Fork and Clone**
   ```bash
   git clone https://github.com/SalehMonfared/stunnel-pro.git
   cd stunnel-pro
   ```

2. **Start Development Environment**
   ```bash
   docker-compose -f docker-compose.dev.yml up -d
   ```

3. **Backend Development**
   ```bash
   cd backend
   go mod download
   go run cmd/server/main.go
   ```

4. **Frontend Development**
   ```bash
   cd frontend
   npm install
   npm run dev
   ```

## 📋 Development Guidelines

### Code Style

#### Go (Backend)
- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` and `golint`
- Write meaningful commit messages
- Add tests for new features

#### TypeScript/React (Frontend)
- Follow [Airbnb JavaScript Style Guide](https://github.com/airbnb/javascript)
- Use ESLint and Prettier
- Write component tests
- Use TypeScript strictly

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
type(scope): description

feat(auth): add two-factor authentication
fix(tunnel): resolve connection timeout issue
docs(readme): update installation instructions
```

### Pull Request Process

1. **Create Feature Branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make Changes**
   - Write clean, documented code
   - Add tests for new functionality
   - Update documentation if needed

3. **Test Your Changes**
   ```bash
   # Backend tests
   cd backend && go test ./...
   
   # Frontend tests
   cd frontend && npm test
   ```

4. **Submit Pull Request**
   - Fill out the PR template
   - Link related issues
   - Request review from maintainers

## 🧪 Testing

### Backend Testing
```bash
cd backend
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Frontend Testing
```bash
cd frontend
npm test
npm run test:coverage
```

### Integration Testing
```bash
docker-compose -f docker-compose.test.yml up --abort-on-container-exit
```

## 📚 Documentation

- Update README.md for user-facing changes
- Add inline code comments for complex logic
- Update API documentation for endpoint changes
- Create/update user guides in `docs/` directory

## 🐛 Bug Reports

When reporting bugs, please include:

- STunnel Pro v1.0 version
- Operating system and version
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs and configuration
- Screenshots if applicable

## 💡 Feature Requests

For feature requests, please:

- Check existing issues first
- Describe the use case clearly
- Explain why this feature would be valuable
- Provide implementation suggestions if possible

## 🔒 Security

- Report security vulnerabilities privately to SalehMonfared via GitHub
- Do not create public issues for security problems
- Follow responsible disclosure practices

## 📄 License

By contributing, you agree that your contributions will be licensed under the MIT License.

## 🙏 Recognition

Contributors will be recognized in:
- README.md contributors section
- Release notes for significant contributions
- Annual contributor appreciation posts

## 📞 Getting Help

- 💬 [GitHub Discussions](https://github.com/SalehMonfared/stunnel-pro/discussions)
- 📧 Contact: [SalehMonfared](https://github.com/SalehMonfared)
- 🐛 Issues: [GitHub Issues](https://github.com/SalehMonfared/stunnel-pro/issues)

Thank you for contributing to STunnel Pro v1.0! 🚀

---

## 💖 **Support the Project**

<div align="center">

### ☕ **Buy Me a Coffee**

If you find STunnel Pro v1.0 useful, consider supporting its development!

<a href="https://coffeebede.com/SalehMonfared" target="_blank">
  <img src="https://img.shields.io/badge/☕_Buy_Me_A_Coffee-Support_Development-orange?style=for-the-badge&logo=buy-me-a-coffee&logoColor=white" alt="Buy Me A Coffee" />
</a>

### 📱 **Join Our Telegram Channel**

Stay updated with the latest news and connect with the community!

<a href="https://t.me/TheSalehMonfared" target="_blank">
  <img src="https://img.shields.io/badge/📱_Telegram_Channel-Join_Now-blue?style=for-the-badge&logo=telegram&logoColor=white" alt="Telegram Channel" />
</a>

---

**Your support helps keep this project alive and growing! 🌱**

</div>
