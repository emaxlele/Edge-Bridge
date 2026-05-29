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

---

## ⚠️ Disclaimer

> **PLEASE READ CAREFULLY BEFORE USING THIS SOFTWARE**

This project was developed exclusively for **personal research, self-study, and technical experimentation** in the field of portable browser automation, browser extension development, and MCP / Native Messaging bridge integration. It is not intended for use in production, commercial, or enterprise environments without appropriate legal and compliance review.

### Permitted use

The software automates or facilitates only operations that an end user could perform **manually** through officially accessible tools, interfaces, and browser APIs. It does not introduce capabilities unavailable to an ordinary user with the appropriate technical knowledge; it simply makes such operations more efficient and repeatable.

### No liability

The author **assumes no responsibility** — direct, indirect, incidental, consequential, or otherwise — for:

-   any use of the software contrary to applicable laws, regulations, or local/international rules;
-   any violation of the **Terms of Service, Conditions of Use, or Policies** of any third-party platform, product, or service the software interacts with;
-   direct or indirect damage to data, systems, accounts, contracts, or business relationships arising from the use, misuse, or inability to use the software;
-   loss of access to services, account suspension, or contractual penalties imposed by third-party service providers;
-   any legal, disciplinary, civil, or criminal consequences arising from the use of the software in violation of applicable rules.

### Third parties

This project is an independent work and is **not affiliated with, endorsed by, sponsored by, or authorized by** Microsoft Corporation or any of its subsidiaries or products (Microsoft Edge, Edge WebDriver, etc.). All trademarks, logos, and product names mentioned are the property of their respective owners.

### User's exclusive responsibility

**Anyone who downloads, installs, modifies, or uses this software does so at their own sole risk.** It is the user's responsibility to:

1. verify that their use is compatible with the Terms of Service of the platforms on which the software runs;
2. obtain, where necessary, authorization from their organization, IT administrators, or system owners;
3. ensure that their use complies with applicable laws in their country and/or jurisdiction.

### Warranty

The software is provided **"AS IS"**, without warranty of any kind, express or implied, including — but not limited to — warranties of merchantability, fitness for a particular purpose, and non-infringement of third-party rights.
