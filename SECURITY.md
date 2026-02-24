# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.x     | :white_check_mark: |

## Reporting a Vulnerability

The Govern team and community take security bugs seriously. We appreciate your efforts to responsibly disclose your findings.

If you discover a security vulnerability, please send an email to the project maintainer at **security@govern.dev** (replace with actual email address).

Please include:
* A description of the vulnerability
* Steps to reproduce the vulnerability
* Affected versions of the project
* Potential impact of the vulnerability

We will:
* Acknowledge receipt of your report within 48 hours
* Provide a detailed response within 7 days
* Work with you to understand and resolve the issue
* Provide regular updates on our progress

### What Happens Next?

After receiving your report:
1. We will verify the vulnerability
2. We will determine the severity and impact
3. We will develop a fix
4. We will release a security update
5. We will announce the vulnerability and fix

### Disclosure Policy

We will coordinate disclosure of the vulnerability with you to ensure users have time to update before the vulnerability is made public.

## Security Best Practices

### Configuration Security

The `config` package follows these security practices:

**1. Credential Management**
- Never commit .env files to version control
- Add `.env` to `.gitignore`
- Use environment variables for sensitive data (API keys, passwords)
- Validate all configuration values on startup

**2. Input Validation**
- All configuration is validated using struct tags
- Required fields are enforced
- Type and range checks are performed automatically

**3. File Permissions**
- Config files should have restricted permissions (0600)
- .env files should never be world-readable

### Example Secure Configuration

```go
// .gitignore
.env
*.env.local
config.production.yaml

// Good: Use environment variables for secrets
type Config struct {
    DatabaseURL string `validate:"required,url"`
    APIKey      string `validate:"required"`
}

// Load config with .env file (not committed)
cfg, err := config.LoadWithOptions[Config](
    "config.yaml",
    config.WithEnvFile(".env"),
)

// Bad: Never hardcode secrets
type Config struct {
    APIKey string // Don't do this!
}
```

### JWT Security

When using JWT authentication:
- Use strong signing algorithms (RS256, ES256)
- Rotate keys regularly
- Set appropriate expiration times
- Validate all tokens on every request
- Use HTTPS for token transmission

### Database Security

**PostgreSQL:**
- Use SSL/TLS for database connections
- Rotate database credentials regularly
- Use least-privilege database users
- Enable connection pooling with appropriate limits

**Redis:**
- Enable AUTH for Redis connections
- Use TLS for Redis connections in production
- Rotate Redis passwords regularly

## Security Audits

This project has not yet undergone a professional security audit. We welcome security researchers to review our code and report any issues they find.

## Dependency Security

We regularly update dependencies to address known vulnerabilities:
* GitHub Dependabot automatically monitors for vulnerabilities
* Security updates are prioritized in our release process

## Security Contact Information

* **Project Maintainer**: Haipham22
* **Security Email**: security@govern.dev (replace with actual email)
* **GitHub Security Advisories**: https://github.com/haipham22/govern/security/advisories

## License

This project is licensed under the TBD license - see the LICENSE file for details.
