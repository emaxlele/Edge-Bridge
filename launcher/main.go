package main

import (
	"crypto/sha256"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed all:assets
var assets embed.FS

// Set via ldflags: -X main.debugMode=true -X main.appMode=false
var debugMode = "false"
var appMode = "true"
var buildResourceHash = "" // set to syso hash at build time to force linker cache invalidation

// ──────────────────────────────────────────────
// Main
// ──────────────────────────────────────────────

func main() {
	isDebug := debugMode == "true"
	isApp := appMode == "true"

	// Default URLs (Step 3 & 4)
	const defaultAppURL = "https://www.wikipedia.org/"
	const defaultBrowserURL = "https://www.google.com/"
	const fallbackBrowserURL = "https://www.google.com/"

	exePath, err := os.Executable()
	if err != nil {
		showErr("Cannot determine exe path: " + err.Error())
		return
	}
	exeDir := filepath.Dir(exePath)

	// ── Banner ──
	switch {
	case isApp && isDebug:
		fmt.Println("EdgeBridge [APP MODE - DEBUG]")
	case isApp:
		fmt.Println("EdgeBridge [APP MODE]")
	case isDebug:
		fmt.Println("EdgeBridge [DEBUG]")
	default:
		fmt.Println("EdgeBridge")
	}

	// ── Step 1: Determine directories ──
	// Use separate AppData dirs for App vs Browser mode
	appDirName := "EdgeBridge"
	tempName := "edgebridge"
	if isApp {
		appDirName = "WikipediaBridge"
		tempName = "wikipediabridge"
	}

	// Persistent dir (user-data-dir + extensions)
	appDir := filepath.Join(os.Getenv("APPDATA"), appDirName)
	_ = os.MkdirAll(appDir, 0755)
	if resolved, err := filepath.EvalSymlinks(appDir); err == nil {
		appDir = resolved
	}
	fixedExtDir := filepath.Join(appDir, "extensions")

	// Runtime dir (bridge.exe, config.json, native manifest)
	var runtimeDir string
	if isDebug {
		runtimeDir = filepath.Join(exeDir, "temp")
		_ = os.MkdirAll(runtimeDir, 0755)
	} else {
		runtimeDir = filepath.Join(os.TempDir(), tempName)
		_ = os.MkdirAll(runtimeDir, 0755)
		if resolved, err := filepath.EvalSymlinks(runtimeDir); err == nil {
			runtimeDir = resolved
		}
	}

	// ── Step 2: Extract assets ──
	fmt.Println("  Extracting assets...")

	// 2a: Extract runtime files to runtimeDir
	if err := extractRuntime(runtimeDir); err != nil {
		showErr("Runtime extract failed: " + err.Error())
		return
	}
	fmt.Println("  OK runtime files extracted")

	// 2b: Sync extensions to fixedExtDir
	if err := syncExtensions(fixedExtDir); err != nil {
		showErr("Extension sync failed: " + err.Error())
		return
	}
	fmt.Println("  OK extensions synced to AppData")

	if isDebug {
		fmt.Println("  [DEBUG] Runtime dir: " + runtimeDir)
		fmt.Println("  [DEBUG] Extensions dir: " + fixedExtDir)
	}

	// ── Step 3: Find Edge ──
	edgeExe := findEdge()
	if edgeExe == "" {
		showErr("Microsoft Edge not found!")
		return
	}

	// ── Step 4: Find extensions (from fixed dir) ──
	extPaths := findAllExtensions(fixedExtDir)
	if len(extPaths) == 0 {
		// Allow launching even if there are no extensions. The debug EXE binaries
		// (built with debugMode=true) will print a warning; production EXEs remain silent.
		if isDebug {
			fmt.Println("  WARNING: No extensions found in: " + fixedExtDir + " (continuing without --load-extension)")
		}
	}

	// Identify MCP extension path (one that declares nativeMessaging)
	mcpExtPath := ""
	for _, p := range extPaths {
		manifest := filepath.Join(p, "manifest.json")
		data, err := os.ReadFile(manifest)
		if err == nil && strings.Contains(string(data), "nativeMessaging") {
			mcpExtPath = p
			break
		}
	}

	if mcpExtPath == "" {
		fmt.Println("  WARNING: No extension with nativeMessaging found")
	}

	fmt.Printf("  Found %d extension(s)\n", len(extPaths))
	for _, p := range extPaths {
		fmt.Println("    - " + filepath.Base(p))
	}

	// ── Step 5: Build allowed_origins robustly ──
	extIDs := []string{"none"}
	if mcpExtPath != "" {
		extIDs = computeAllPossibleIDs(mcpExtPath)
		fmt.Printf("  OK MCP allowed_origins candidates: %d\n", len(extIDs))
		for _, id := range extIDs {
			fmt.Println("    - " + id)
		}
	}

	// ── Step 6: Generate native messaging manifest ──
	bridgePath := filepath.Join(runtimeDir, "bridge.exe")
	manifestPath := filepath.Join(runtimeDir, "com.edgebridge.host.json")
	writeManifest(manifestPath, bridgePath, extIDs)
	fmt.Println("  OK manifest generated")

	// ── Step 7: Register in Windows Registry ──
	registerHost(manifestPath)
	fmt.Println("  OK registry updated")

	// ── Step 7.5: Force Copilot preferences ──
	if err := seedCopilotPreferences(appDir, isDebug); err != nil {
		fmt.Println("  WARN: Could not seed Copilot preferences: " + err.Error())
	} else {
		fmt.Println("  OK Copilot preferences forced")
	}

	// ── Step 8: Launch Edge ──
	// (Optional) remove enterprise force-installed extensions from this profile
	nukeForceInstalledExtensions(appDir)

	var loadExt string
	if len(extPaths) > 0 {
		loadExt = strings.Join(extPaths, ",")
	}

	args := []string{
		"--user-data-dir=" + appDir,
		"--disable-component-extensions-with-background-pages",
		"--disable-background-networking",
		"--no-first-run",
		"--disable-default-apps",
	}
	if loadExt != "" {
		// ensure load-extension and disable-extensions-except are applied before other args
		args = append([]string{"--load-extension=" + loadExt, "--disable-extensions-except=" + loadExt}, args...)
	}

	if isApp {
		url := defaultAppURL
		args = append([]string{"--app=" + url}, args...)
	} else {
		url := defaultBrowserURL
		args = append(args, url)
		// Force-enable Copilot-related features via flag (kept as original behavior)
		args = append(args, "--enable-features=msEdgeCopilotMode,msEdgeNTPComposer,msEdgeSidebarCopilot")
		// Fallback URL is available as fallbackBrowserURL if needed by future logic
		_ = fallbackBrowserURL
	}

	fmt.Println("  Launching Edge...")
	cmd := exec.Command(edgeExe, args...)
	if err := cmd.Start(); err != nil {
		showErr("Failed to launch Edge: " + err.Error())
		return
	}

	fmt.Println("  OK! Ctrl+Shift+P to open panel")

	if isDebug {
		fmt.Println()
		fmt.Println("  [DEBUG] Press Enter to exit...")
		fmt.Scanln()
	} else {
		cmd.Wait()
	}
}

// ──────────────────────────────────────────────
// Asset extraction (runtime vs extensions)
// ──────────────────────────────────────────────

// extractRuntime extracts non-extension files (bridge.exe, config.json, etc.)
func extractRuntime(dest string) error {
	_ = os.MkdirAll(dest, 0755)

	return fs.WalkDir(assets, "assets", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "assets" {
			return nil
		}
		relPath := strings.TrimPrefix(path, "assets/")

		// Skip extensions — handled by syncExtensions
		if strings.HasPrefix(relPath, "extensions/") || strings.HasPrefix(relPath, "extensions\\") {
			return nil
		}

		target := filepath.Join(dest, relPath)

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		data, err := assets.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		// Tolerant overwrite (bridge.exe might be locked by an older instance)
		if err := os.WriteFile(target, data, 0644); err != nil {
			fmt.Printf("  WARN: cannot overwrite %s (in use?), skipping\n", filepath.Base(target))
		}
		return nil
	})
}

// syncExtensions copies embedded extensions to persistent AppData extensions dir
func syncExtensions(dest string) error {
	_ = os.MkdirAll(dest, 0755)
	// If there are no embedded extensions, don't treat it as an error.
	if _, err := assets.ReadDir("assets/extensions"); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// nothing to sync
			return nil
		}
		// For other errors, return nil as well to be tolerant at runtime
		return nil
	}

	return fs.WalkDir(assets, "assets/extensions", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "assets/extensions" {
			return nil
		}
		relPath := strings.TrimPrefix(path, "assets/extensions/")
		target := filepath.Join(dest, relPath)

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		data, err := assets.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

// ──────────────────────────────────────────────
// Robust allowed_origins computation
// ──────────────────────────────────────────────

// computeAllPossibleIDs returns a deduplicated list of likely Chromium extension IDs
// derived from different path normalizations and encodings.
// This is robust across Windows path canonicalization differences.
func computeAllPossibleIDs(extPath string) []string {
	abs, _ := filepath.Abs(extPath)
	clean := filepath.Clean(abs)

	// Try to canonicalize via EvalSymlinks too (helps avoiding 8.3, junctions)
	if resolved, err := filepath.EvalSymlinks(clean); err == nil {
		clean = resolved
	}

	// Normalize separators (backslash for Windows)
	cleanBack := strings.ReplaceAll(clean, "/", `\`)
	cleanFwd := filepath.ToSlash(cleanBack)

	// Chromium may include or ignore \\?\ prefix depending on canonicalization path
	withPrefix := `\\?\` + cleanBack

	// Candidate strings (both original + lowercased)
	candidates := []string{
		cleanBack,
		strings.ToLower(cleanBack),

		cleanFwd,
		strings.ToLower(cleanFwd),

		withPrefix,
		strings.ToLower(withPrefix),
	}

	seen := map[string]bool{}
	var ids []string

	for _, c := range candidates {
		// UTF-8 bytes
		addID(&ids, seen, hashToID([]byte(c)))
		// UTF-16LE bytes
		addID(&ids, seen, hashToID(toUTF16LE(c)))
	}

	return ids
}

func addID(ids *[]string, seen map[string]bool, id string) {
	if id == "" || seen[id] {
		return
	}
	seen[id] = true
	*ids = append(*ids, id)
}

func hashToID(data []byte) string {
	hash := sha256.Sum256(data)
	var id strings.Builder
	for i := 0; i < 16; i++ {
		id.WriteByte('a' + (hash[i]>>4)&0x0f)
		id.WriteByte('a' + hash[i]&0x0f)
	}
	return id.String()
}

func toUTF16LE(s string) []byte {
	runes := []rune(s)
	buf := make([]byte, len(runes)*2)
	for i, r := range runes {
		buf[i*2] = byte(r)
		buf[i*2+1] = byte(r >> 8)
	}
	return buf
}

// ──────────────────────────────────────────────
// Native Messaging manifest + registry
// ──────────────────────────────────────────────

func writeManifest(manifestPath, bridgePath string, extIDs []string) {
	escaped := strings.ReplaceAll(bridgePath, `\`, `\\`)

	origins := make([]string, 0, len(extIDs))
	for _, id := range extIDs {
		if len(id) == 32 {
			origins = append(origins, fmt.Sprintf(`"chrome-extension://%s/"`, id))
		}
	}
	if len(origins) == 0 {
		origins = []string{`"chrome-extension://none/"`}
	}

	j := fmt.Sprintf(`{
  "name": "com.edgebridge.host",
  "description": "EdgeBridge Native Messaging Host",
  "path": "%s",
  "type": "stdio",
  "allowed_origins": [%s]
}`, escaped, strings.Join(origins, ", "))

	_ = os.WriteFile(manifestPath, []byte(j), 0644)
}

func registerHost(manifestPath string) {
	_ = exec.Command("reg", "add",
		`HKCU\Software\Microsoft\Edge\NativeMessagingHosts\com.edgebridge.host`,
		"/ve", "/d", manifestPath, "/f",
	).Run()
}

// ──────────────────────────────────────────────
// Find Edge
// ──────────────────────────────────────────────

func findEdge() string {
	paths := []string{
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Edge", "Application", "msedge.exe"),
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func showErr(msg string) {
	fmt.Println("  ERROR: " + msg)
	fmt.Println("  Press Enter to exit...")
	fmt.Scanln()
}

// ──────────────────────────────────────────────
// Find all extensions
// ──────────────────────────────────────────────

func findAllExtensions(baseDir string) []string {
	var paths []string
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return paths
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifest := filepath.Join(baseDir, entry.Name(), "manifest.json")
		if _, err := os.Stat(manifest); err == nil {
			paths = append(paths, filepath.Join(baseDir, entry.Name()))
		}
	}
	return paths
}

// ──────────────────────────────────────────────
// Nuke force-installed extensions (optional)
// ──────────────────────────────────────────────

func nukeForceInstalledExtensions(appDir string) {
	targets := []string{
		filepath.Join(appDir, "Default", "Extensions"),
		filepath.Join(appDir, "Default", "External Extensions"),
	}
	for _, t := range targets {
		if _, err := os.Stat(t); err == nil {
			_ = os.RemoveAll(t)
			fmt.Println("  OK nuked: " + filepath.Base(t))
		}
	}
}

// ──────────────────────────────────────────────
// Pre-seed Copilot settings in Preferences
// ──────────────────────────────────────────────

// seedCopilotPreferences legge il file Preferences esistente (se c'è),
// fa deep-merge delle impostazioni Copilot, e riscrive il file.
// In debug mode stampa tutto il contenuto prima e dopo il merge.
func seedCopilotPreferences(appDir string, debug bool) error {
	prefsDir := filepath.Join(appDir, "Default")
	_ = os.MkdirAll(prefsDir, 0755)
	prefsPath := filepath.Join(prefsDir, "Preferences")

	// ── 1. Leggi Preferences esistenti (o parti da oggetto vuoto) ──
	existing := make(map[string]interface{})
	if data, err := os.ReadFile(prefsPath); err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &existing); err != nil {
			_ = os.WriteFile(prefsPath+".bak", data, 0644)
			fmt.Println("  WARN: Preferences corrupted, backed up and recreated")
			existing = make(map[string]interface{})
		}
	}

	// ── 2. DEBUG: Dump completo delle Preferences PRIMA del merge ──
	if debug {
		fmt.Println()
		fmt.Println("  ┌──────────────────────────────────────────────")
		fmt.Println("  │ [DEBUG] Preferences BEFORE merge")
		fmt.Println("  │ File: " + prefsPath)
		fmt.Println("  ├──────────────────────────────────────────────")
		if len(existing) == 0 {
			fmt.Println("  │ (empty — file not found or no content)")
		} else {
			dumpMap(existing, "  │ ", "")
		}
		fmt.Println("  └──────────────────────────────────────────────")
		fmt.Println()
	}

	// ── 3. Definisci le chiavi Copilot da forzare ──
	copilotOverrides := map[string]interface{}{
		"edge": map[string]interface{}{
			"services": map[string]interface{}{
				"copilot": map[string]interface{}{
					"toolbar_pin_enabled":      true,
					"page_context_enabled":     true,
					"video_transcript_enabled": true,
					"ntp_copilot_enabled":      true,
				},
			},
			// quick_search UI flags (da file edge_quick_search)
			"quick_search": map[string]interface{}{
				"show_toolbar_edge_generic_sidebar_button": true,
				"show_toolbar_bookmarks_button":            true,            // opzionale
				"show_toolbar_collections_button":          true,            // opzionale
				"show_toolbar_history_button":              true,            // puoi lasciare true
				"show_toolbar_share_button":                true,            // opzionale
				"customized_disabled_sites":                []interface{}{}, // rimuove i siti bloccati come copilotstudio.microsoft.com
			},

			// metadata commerciale / eligibility (simula i server flags)
			"commercial_copilot": map[string]interface{}{
				"eligibility_info": map[string]interface{}{
					"isCopilotEligible":          true,
					"isOptedOutByAdmin":          false,
					"isCodexEnabledRegion":       true,
					"edge_copilot_theme_enabled": true,
					"featureSet": map[string]interface{}{
						"uxFeatures": []interface{}{
							"CodeInterpreter",
							"Designer",
							"GPTV",
							"Threads",
							"WebGroundingControls",
							"Pages",
							"FileUpload",
						},
						"serverFeatures": []interface{}{}, // se sai quali serverFeatures servono, aggiungile qui
					},
				},
			},

			// legacy top-level flags (alcune implementazioni leggono anche qui)
			"ntp_copilot_enabled":        true,
			"edge_copilot_theme_enabled": true,
		},

		"browser": map[string]interface{}{
			"enabled_labs_experiments": []interface{}{
				"edge-copilot-mode@1",
				"edge-ntp-composer@1",
				// aggiungi altri esperimenti se li conosci
			},
		},
	}

	// ── 4. Deep-merge ──
	deepMerge(existing, copilotOverrides)

	// ── 5. DEBUG: Dump completo delle Preferences DOPO il merge ──
	if debug {
		fmt.Println("  ┌──────────────────────────────────────────────")
		fmt.Println("  │ [DEBUG] Preferences AFTER merge")
		fmt.Println("  ├──────────────────────────────────────────────")
		dumpMap(existing, "  │ ", "")
		fmt.Println("  └──────────────────────────────────────────────")
		fmt.Println()
	}

	// ── 6. Scrivi il risultato ──
	out, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal failed: %w", err)
	}

	return os.WriteFile(prefsPath, out, 0644)
}

// ──────────────────────────────────────────────
// Debug: stampa ricorsiva di tutte le chiavi
// ──────────────────────────────────────────────

// dumpMap stampa ricorsivamente tutte le chiavi e valori di una mappa,
// con indentazione a livelli per facilitare la lettura.
func dumpMap(m map[string]interface{}, linePrefix, indent string) {
	keys := sortedKeys(m)
	for _, key := range keys {
		val := m[key]
		fullKey := indent + key

		switch v := val.(type) {
		case map[string]interface{}:
			fmt.Printf("%s📁 %s\n", linePrefix, fullKey)
			dumpMap(v, linePrefix, indent+"  ")

		case []interface{}:
			fmt.Printf("%s📋 %s = [\n", linePrefix, fullKey)
			for i, item := range v {
				fmt.Printf("%s     [%d] %v\n", linePrefix, i, item)
			}
			fmt.Printf("%s   ]\n", linePrefix)

		case string:
			// Tronca stringhe molto lunghe per leggibilità
			display := v
			if len(display) > 120 {
				display = display[:120] + "…"
			}
			fmt.Printf("%s🔹 %s = \"%s\"\n", linePrefix, fullKey, display)

		case bool:
			fmt.Printf("%s🔹 %s = %t\n", linePrefix, fullKey, v)

		case float64:
			// JSON unmarshal mette tutti i numeri come float64
			if v == float64(int64(v)) {
				fmt.Printf("%s🔹 %s = %d\n", linePrefix, fullKey, int64(v))
			} else {
				fmt.Printf("%s🔹 %s = %g\n", linePrefix, fullKey, v)
			}

		default:
			fmt.Printf("%s🔹 %s = %v (%T)\n", linePrefix, fullKey, val, val)
		}
	}
}

// sortedKeys ritorna le chiavi di una mappa ordinate alfabeticamente
func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ──────────────────────────────────────────────
// Deep-merge helper
// ──────────────────────────────────────────────

// deepMerge sovrascrive ricorsivamente le chiavi di dst con quelle di src.
// - Se entrambi i valori sono map[string]interface{}, merge ricorsivo.
// - Se src ha una slice, fa union con dedup.
// - Altrimenti src vince (forza il valore).
func deepMerge(dst, src map[string]interface{}) {
	for key, srcVal := range src {
		dstVal, exists := dst[key]

		switch srcTyped := srcVal.(type) {
		case map[string]interface{}:
			if dstMap, ok := dstVal.(map[string]interface{}); ok && exists {
				deepMerge(dstMap, srcTyped)
			} else {
				dst[key] = srcVal
			}

		case []interface{}:
			if dstSlice, ok := dstVal.([]interface{}); ok && exists {
				dst[key] = mergeSlices(dstSlice, srcTyped)
			} else {
				dst[key] = srcVal
			}

		default:
			dst[key] = srcVal
		}
	}
}

// mergeSlices unisce due slice mantenendo l'ordine e rimuovendo duplicati.
func mergeSlices(dst, src []interface{}) []interface{} {
	seen := make(map[string]bool)
	for _, v := range dst {
		seen[fmt.Sprintf("%v", v)] = true
	}

	result := make([]interface{}, len(dst))
	copy(result, dst)

	for _, v := range src {
		key := fmt.Sprintf("%v", v)
		if !seen[key] {
			result = append(result, v)
			seen[key] = true
		}
	}
	return result
}
