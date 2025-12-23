# Security Policy

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

**Do not open public GitHub issues for security vulnerabilities.**

To report a vulnerability, please email the maintainer directly or use GitHub's private vulnerability reporting feature:

1. Go to the repository's Security tab
2. Click "Report a vulnerability"
3. Provide details about the issue

You should receive a response within 48 hours. If the vulnerability is confirmed, a fix will be released as soon as possible, typically within 7 days for critical issues.

## Security Practices

This project implements several security measures:

### Authentication
- **OAuth 2.1 with PKCE** for secure token exchange
- Tokens stored locally at `~/.miro/tokens.json` with restricted permissions (600)
- Automatic token refresh before expiry (5-minute buffer)
- No tokens transmitted to third parties

### Data Protection
- **Audit log redaction**: Sensitive fields (passwords, tokens, API keys) are automatically redacted
- No credentials logged in debug output
- Environment variables used for sensitive configuration

### Input Validation
- Structured type validation for all API inputs
- Board/item ID validation before API calls
- Bulk operation limits enforced (max 20 items)
- Content size checks prevent oversized payloads

### Network Security
- HTTPS-only communication with Miro API
- HTTP server mode warns when binding to external interfaces
- Configurable timeouts prevent resource exhaustion

### Dependencies
- Minimal dependency footprint (3 direct dependencies)
- Dependabot enabled for automatic security updates
- Regular dependency audits via `go mod verify`

## Security Checklist for Users

1. **Never commit tokens** to version control
2. **Use environment variables** for `MIRO_ACCESS_TOKEN`
3. **Rotate tokens periodically** via Miro's app settings
4. **Use OAuth** for production deployments (auto-refresh, revocable)
5. **Review audit logs** periodically if enabled
