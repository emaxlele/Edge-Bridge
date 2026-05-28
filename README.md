# EdgeBridge

Portable Microsoft Edge launcher with embedded extensions and local native messaging bridge.

Features

-   Isolated Edge profile
-   Auto-loaded extensions
-   MCP / Native Messaging bridge (bridge.exe)
-   App mode + Browser mode
-   Portable build artifacts

Architecture
Edge → Extension → Native Messaging → bridge.exe (Go)

Build & quick start

1. On Windows, run:

```
scripts\build.bat
```

2. Artifacts: `build\Wikipedia\...`, `build\Edge\...`, and `build\bridge.exe`.

See `docs/USAGE.md` for detailed usage and testing instructions.

Security

-   Local execution only
-   Allowed paths for native messaging are computed from extension path
-   No credentials stored by default

## Sensitive data scan

I scanned the repository for common sensitive artifacts (private keys, .env files, .pem/.pfx files and similar). No obvious private key files or typical credential file extensions were found in the repository tree. There are references to an `extension-key.txt` (the launcher generates it at runtime) and build artifacts (.exe) under build/ which are excluded by .gitignore.

Notes:

-   Keep any generated `extension-key.txt` or other runtime secret files out of version control. They are created at first run and must not be committed to a public repo.
-   If you have service account keys, tokens, or other secrets elsewhere, move them outside the repo and read them via environment variables or secure vaults at runtime.
