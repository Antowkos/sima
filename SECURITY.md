# Security Policy

## Reporting vulnerabilities

Please do **not** report security vulnerabilities, secrets, tokens, credentials, or private configuration values in public GitHub issues.

For now, report security-sensitive problems privately to the repository owner through a trusted private channel. If you need to include logs, redact secrets as `[REDACTED]` before sharing.

## Scope

Security-sensitive reports include, but are not limited to:

- credential/token leakage;
- unsafe command execution;
- prompt-injection paths that could cause unintended actions;
- release artifact tampering or checksum mismatch;
- unsafe handling of `.sima/` evidence, logs, or config.

Public issues may be closed or edited if they contain sensitive data.
