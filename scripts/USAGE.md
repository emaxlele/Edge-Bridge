Ecco i passi esatti, in ordine:

---

## Step 1 — Verifica struttura cartelle (Project layout)

Assicurati che il progetto sia così:

```
EdgeBridge-Project/
├── Extension/
│   ├── manifest.json
│   ├── background.js
│   ├── background-mcp.js
│   ├── content.js
│   ├── injected.js
│   ├── popup.html
│   ├── popup.js
│   ├── styles.css
│   ├── lib/
│   │   ├── mcp-bridge.js
│   │   └── prompt-enricher.js
│   └── icons/
├── mcp-server/
│   ├── main.go
│   ├── server.go
│   ├── tools.go
│   ├── protocol.go
│   ├── config.go
│   ├── config.json
│   ├── go.mod
│   └── go.sum
└── scripts/
    ├── install.bat
    ├── launch.bat
    ├── uninstall.bat
    └── update-extension-id.bat
```

Se ti manca `mcp-server/` (i file Go), dimmelo — quello era lo Step 1 del progetto.

---

## Step 2 — Esegui install.bat

Apri **CMD** nella cartella `scripts/`:

```
.\install.bat
```

Replace `C:\Users\<your-user>\...` paths in the examples with your local project path before running commands.
.\install.bat

```

Lo script fa tutto automaticamente:

1. ✅ Verifica che Go sia installato
2. ✅ Compila `bridge.exe` dalla cartella `mcp-server/`
3. ✅ Copia `config.json` nella root
4. ⏸️ **Ti chiede l'Extension ID** — per ora premi Enter (lo aggiorni dopo)
5. ✅ Crea il manifest Native Messaging (`com.edgebridge.host.json`)
6. ✅ Registra la chiave nel Registry di Windows

---

## Step 3 — Carica l'estensione in Edge

1. Apri Edge → `edge://extensions`
2. Attiva **Developer Mode** (in alto a destra)
3. Clicca **Load unpacked** → seleziona la cartella `Extension/`
4. **Copia l'ID** che appare sotto il nome dell'estensione (es. `abcdef1234567890...`)

---

## Step 4 — Aggiorna l'Extension ID

```

.\update-extension-id.bat

```

Incolla l'ID copiato al punto 3. Lo script aggiorna il manifest Native Messaging e il Registry.

---

## Step 5 — Testa

```

.\launch.bat

```

Oppure apri direttamente `https://www.google.com/` in Edge (browser mode). Per l'App mode usa `WikipediaBridge.exe` che apre `https://www.wikipedia.org/`.

Poi:

1. **Ctrl+Shift+P** → apre il panel Model Forcer
2. Vai al tab **🔧 MCP**
3. Digita `/connect` → dovrebbe dire "Connected!" con la lista dei tools
4. Prova `/ls C:\Users\%USERNAME%\Projects` → lista le cartelle
5. Prova `/read C:\qualche\file.txt` → legge un file locale

---

## Se qualcosa non va

| Problema                       | Soluzione                                                           |
| ------------------------------ | ------------------------------------------------------------------- |
| `go` non trovato               | Installa Go da <https://go.dev/dl/>                                 |
| Build fallisce                 | Verifica che `mcp-server/` abbia tutti i file .go                   |
| MCP dice "error" al `/connect` | Controlla che l'Extension ID sia giusto (`update-extension-id.bat`) |
| Nessuna risposta da `/connect` | Verifica che `bridge.exe` esista nella root del progetto            |
| Registry error                 | Esegui PowerShell come admin                                        |

Vai! 🚀

Risolviamo tutto — l'ID viene **generato automaticamente** dal launcher. Zero intervento manuale.

Il trucco: aggiungiamo un `"key"` nel `manifest.json` dell'estensione → Edge assegna sempre lo stesso ID → il launcher lo calcola da solo.

---

(Remaining code snippets omitted in this user-facing copy; see launcher/main.go in the repo for implementation details.)

---

Quick test notes (from former `scripts/come rpovare.txt`)

1) Portabilità

```

First run on a new machine:
WikipediaBridge.exe -> creates %APPDATA%\WikipediaBridge (empty profile), registers bridge.exe in registry and launches Edge

```

2) Enterprise extensions

- If your environment enforces extensions via GPO/Intune (ExtensionInstallForcelist), those forced extensions will still appear in the isolated profile.

3) Quick MCP test

 - Run `WikipediaBridge-debug.exe` to see console logs.
 - Press Ctrl+Shift+P in Edge → go to MCP tab → `/connect` → you should see "Connected!" and available tools.
 - Example MCP commands:
```

/status
/ls %APPDATA%\Projects
/run echo hello

```

Notes:
- Replace `%APPDATA%\Projects` with your actual test paths. Do NOT include sensitive file paths in public docs.
```
