package tui

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/text/language"
)

// Translations holds all UI strings
type Translations struct {
	mu sync.RWMutex
	// Current locale
	locale language.Tag
	// Translation map: locale -> key -> value
	data map[string]map[string]string
	// Embedded fallback translations
	fallback embed.FS
}

// Singleton instance
var i18n *Translations
var once sync.Once

// GetI18n returns the singleton i18n instance
func GetI18n() *Translations {
	once.Do(func() {
		i18n = &Translations{
			locale: language.English,
			data:   make(map[string]map[string]string),
		}
		i18n.loadDefaults()
		i18n.detectLocale()
	})
	return i18n
}

// Translation keys
const (
	// Home screen
	KeyHomeClean        = "home.clean"
	KeyHomeUninstall    = "home.uninstall"
	KeyHomeAnalyze      = "home.analyze"
	KeyHomeStatus       = "home.status"
	KeyHomeTools        = "home.tools"
	KeyHomeProtect      = "home.protect"
	KeyHomeDoctor       = "home.doctor"

	// Common actions
	KeyActionExecute = "action.execute"
	KeyActionCancel  = "action.cancel"
	KeyActionConfirm = "action.confirm"
	KeyActionRetry   = "action.retry"
	KeyActionBack    = "action.back"
	KeyActionQuit    = "action.quit"
	KeyActionRefresh = "action.refresh"
	KeyActionAdd     = "action.add"
	KeyActionDelete  = "action.delete"

	// Status messages
	KeyStatusLoading    = "status.loading"
	KeyStatusProcessing = "status.processing"
	KeyStatusComplete   = "status.complete"
	KeyStatusError      = "status.error"

	// Protect screen
	KeyProtectTitle       = "protect.title"
	KeyProtectNoPaths     = "protect.no_paths"
	KeyProtectAddPrompt   = "protect.add_prompt"
	KeyProtectAdded       = "protect.added"
	KeyProtectRemoved     = "protect.removed"
	KeyProtectNotFound    = "protect.not_found"

	// Uninstall screen
	KeyUninstallTitle     = "uninstall.title"
	KeyUninstallScanning = "uninstall.scanning"
	KeyUninstallNoApps   = "uninstall.no_apps"

	// Analyze screen
	KeyAnalyzeTitle    = "analyze.title"
	KeyAnalyzeNoData   = "analyze.no_data"
	KeyAnalyzeSearch  = "analyze.search"

	// Errors
	KeyErrorEmptyPath     = "error.empty_path"
	KeyErrorInvalidPath   = "error.invalid_path"
	KeyErrorAccessDenied = "error.access_denied"
	KeyErrorGeneric      = "error.generic"
)

// T translates a key to the current locale
func (i *Translations) T(key string) string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	lang := i.locale.String()
	if trans, ok := i.data[lang]; ok {
		if val, ok := trans[key]; ok {
			return val
		}
	}

	// Fallback to English
	if lang != "en" {
		if trans, ok := i.data["en"]; ok {
			if val, ok := trans[key]; ok {
				return val
			}
		}
	}

	// Return key if not found
	return key
}

// TFunc returns a translate function for templates
func (i *Translations) TFunc() func(string) string {
	return i.T
}

// SetLocale changes the current locale
func (i *Translations) SetLocale(lang string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	tag, err := language.Parse(lang)
	if err != nil {
		return fmt.Errorf("invalid locale: %w", err)
	}
	i.locale = tag
	return nil
}

// GetLocale returns the current locale
func (i *Translations) GetLocale() language.Tag {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.locale
}

func (i *Translations) detectLocale() {
	// Try to detect from environment
	lang := os.Getenv("LANG")
	if lang != "" {
		tag, err := language.Parse(lang)
		if err == nil {
			i.mu.Lock()
			i.locale = tag
			i.mu.Unlock()
		}
	}
}

func (i *Translations) loadDefaults() {
	// Load embedded default translations
	defaults := map[string]map[string]string{
		"en": {
			// Home
			KeyHomeClean:      "🧹 Clean",
			KeyHomeUninstall:  "🗑️ Uninstall",
			KeyHomeAnalyze:    "📊 Analyze",
			KeyHomeStatus:     "💻 Status",
			KeyHomeTools:      "🔧 Tools",
			KeyHomeProtect:    "🛡️ Protect",
			KeyHomeDoctor:     "⚕️ Doctor",

			// Actions
			KeyActionExecute: "Execute",
			KeyActionCancel:  "Cancel",
			KeyActionConfirm: "Confirm",
			KeyActionRetry:   "Retry",
			KeyActionBack:    "Back",
			KeyActionQuit:    "Quit",
			KeyActionRefresh: "Refresh",
			KeyActionAdd:     "Add",
			KeyActionDelete:  "Delete",

			// Status
			KeyStatusLoading:    "Loading...",
			KeyStatusProcessing: "Processing...",
			KeyStatusComplete:   "Complete",
			KeyStatusError:      "Error",

			// Protect
			KeyProtectTitle:     "Protect",
			KeyProtectNoPaths:   "No user-defined protected paths yet.",
			KeyProtectAddPrompt: "Type a path and press enter.",
			KeyProtectAdded:     "Protected path added: ",
			KeyProtectRemoved:   "Protected path removed: ",
			KeyProtectNotFound:  "Protected path not found: ",

			// Uninstall
			KeyUninstallTitle:     "Uninstall",
			KeyUninstallScanning:  "Scanning installed applications...",
			KeyUninstallNoApps:    "No applications found.",

			// Analyze
			KeyAnalyzeTitle:   "Analyze",
			KeyAnalyzeNoData:  "No data to display.",
			KeyAnalyzeSearch: "filter current findings",

			// Errors
			KeyErrorEmptyPath:     "Path cannot be empty.",
			KeyErrorInvalidPath:  "Invalid path format.",
			KeyErrorAccessDenied: "Access denied.",
			KeyErrorGeneric:      "An error occurred.",
		},
		"tr": {
			// Home
			KeyHomeClean:      "🧹 Temizle",
			KeyHomeUninstall:  "🗑️ Kaldır",
			KeyHomeAnalyze:    "📊 Analiz Et",
			KeyHomeStatus:     "💻 Durum",
			KeyHomeTools:      "🔧 Araçlar",
			KeyHomeProtect:    "🛡️ Koru",
			KeyHomeDoctor:     "⚕️ Doktor",

			// Actions
			KeyActionExecute: "Çalıştır",
			KeyActionCancel: "İptal",
			KeyActionConfirm: "Onayla",
			KeyActionRetry:   "Tekrar Dene",
			KeyActionBack:    "Geri",
			KeyActionQuit:    "Çık",
			KeyActionRefresh: "Yenile",
			KeyActionAdd:     "Ekle",
			KeyActionDelete:  "Sil",

			// Status
			KeyStatusLoading:    "Yükleniyor...",
			KeyStatusProcessing: "İşleniyor...",
			KeyStatusComplete:   "Tamamlandı",
			KeyStatusError:      "Hata",

			// Protect
			KeyProtectTitle:     "Koruma",
			KeyProtectNoPaths:   "Henüz korumalı yol yok.",
			KeyProtectAddPrompt: "Bir yol girip enter'a basın.",
			KeyProtectAdded:     "Korunan yol eklendi: ",
			KeyProtectRemoved:   "Korunan yol kaldırıldı: ",
			KeyProtectNotFound:  "Korunan yol bulunamadı: ",

			// Uninstall
			KeyUninstallTitle:     "Kaldır",
			KeyUninstallScanning:  "Yüklü uygulamalar taranıyor...",
			KeyUninstallNoApps:    "Uygulama bulunamadı.",

			// Analyze
			KeyAnalyzeTitle:   "Analiz",
			KeyAnalyzeNoData:  "Gösterilecek veri yok.",
			KeyAnalyzeSearch: "bulguları filtrele",

			// Errors
			KeyErrorEmptyPath:     "Yol boş olamaz.",
			KeyErrorInvalidPath:  "Geçersiz yol biçimi.",
			KeyErrorAccessDenied: "Erişim reddedildi.",
			KeyErrorGeneric:      "Bir hata oluştu.",
		},
	}

	i.mu.Lock()
	defer i.mu.Unlock()
	i.data = defaults
}

// LoadTranslationsFromFile loads translations from a JSON file
func (i *Translations) LoadTranslationsFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read translation file: %w", err)
	}

	var translations map[string]map[string]string
	if err := json.Unmarshal(data, &translations); err != nil {
		return fmt.Errorf("failed to parse translation file: %w", err)
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	for locale, strings := range translations {
		if _, ok := i.data[locale]; !ok {
			i.data[locale] = make(map[string]string)
		}
		for key, value := range strings {
			i.data[locale][key] = value
		}
	}

	return nil
}

// LoadTranslationsFromDir loads all translation files from a directory
func (i *Translations) LoadTranslationsFromDir(dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read translation directory: %w", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			path := filepath.Join(dir, file.Name())
			if err := i.LoadTranslationsFromFile(path); err != nil {
				return err
			}
		}
	}
	return nil
}