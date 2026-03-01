# Security Policy

## Supported Versions

tui-cardman is currently in alpha (v0.1.x). Only the latest release receives security fixes.

| Version | Supported |
|---------|-----------|
| 0.1.x   | Yes       |
| < 0.1   | No        |

## Reporting a Vulnerability

Please **do not** open a public GitHub issue for security vulnerabilities.

Instead, report them privately via [GitHub Security Advisories](https://github.com/laiambryant/tui-cardman/security/advisories/new).

Include:
- A description of the vulnerability
- Steps to reproduce
- Potential impact
- Any suggested mitigations (optional)

You will receive a response within 7 days. If the vulnerability is confirmed, a fix will be released as soon as possible and credited to the reporter (unless you prefer to remain anonymous).

## Security Considerations

- **SSH server mode**: when running `serve-ssh`, ensure the host key file has restricted permissions (`chmod 600 host_key`) and the port is firewalled appropriately
- **API keys**: never commit `.env` files containing `API_KEY` or other secrets; use `.env.example` as a template only
- **SQLite database**: the database file contains user data including password hashes (bcrypt); ensure the file is not world-readable
- **Dependencies**: this project uses CGO for SQLite (`mattn/go-sqlite3`); keep the C toolchain and system libraries up to date
