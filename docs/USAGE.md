Ecco i passi essenziali per usare EdgeBridge (estratto e riorganizzato).

Prerequisiti (Windows)

-   Microsoft Edge installato
-   Go 1.21+ (solo per build)
-   goversioninfo (lo script installa automaticamente se mancante durante la build)

Quick layout

```
EdgeBridge-Project/
├── Extension/             # estensione Chrome/Edge (manifest.json + source)
├── mcp-server/            # server Go -> bridge.exe
├── launcher/              # codice launcher Go (genera EXE con assets)
└── scripts/               # .bat helper per build/install/launch
```

Build

1. Apri CMD nella root del repository
2. Esegui: `scripts\build.bat`

Output (build/) contiene:

-   build\Wikipedia\WikipediaBridge.exe (App mode)
-   build\Wikipedia\WikipediaBridge-debug.exe (App mode debug)
-   build\Edge\EdgeBridge.exe (Browser mode)
-   build\Edge\EdgeBridge-debug.exe (Browser mode debug)
-   build\bridge.exe (standalone MCP server)

Installazione (native messaging manifest)

1. Apri CMD nella cartella `scripts/`
2. Esegui: `install.bat`
    - Compila `bridge.exe` e genera il manifest `com.edgebridge.host.json`
    - Registra la chiave nel registro HKCU\Software\Microsoft\Edge\NativeMessagingHosts\com.edgebridge.host

Esecuzione

-   App mode (isolated app profile): usa `WikipediaBridge.exe` (apre https://www.wikipedia.org/)
-   Browser mode (clean browser profile): usa `EdgeBridge.exe` (apre https://www.google.com/)
-   Debug exe (`*-debug.exe`) stampa la console e messaggi diagnostici.

Avvio rapido

1. Lancia `build\Wikipedia\WikipediaBridge-debug.exe` per vedere la console e i messaggi.
2. Controlla `edge://extensions` per verificare che l'estensione sia caricata.

Testing MCP

1. Apri il pannello (Ctrl+Shift+P) → tab 🔧 MCP → `/connect`.
2. Comandi di esempio:
    - `/status`
    - `/ls %APPDATA%\Projects`
    - `/read <path>`

Security & privacy

-   Non committare mai `extension-key.txt` né altri file contenenti chiavi/token.
-   Nessun file .pem/.pfx/.env è incluso nel repo. Se hai segreti, usali tramite variabili d'ambiente o vault.

Contribuire

-   Apri una issue o un pull request. Segui CONTRIBUTING.md per linee guida.
