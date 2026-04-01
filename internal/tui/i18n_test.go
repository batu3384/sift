package tui

import (
	"testing"
)

func TestI18nBasicTranslation(t *testing.T) {
	i18n := GetI18n()

	// Test English default
	if got := i18n.T(KeyHomeClean); got != "🧹 Clean" {
		t.Errorf("expected '🧹 Clean', got '%s'", got)
	}

	// Test Turkish
	if err := i18n.SetLocale("tr"); err != nil {
		t.Errorf("failed to set locale: %v", err)
	}
	if got := i18n.T(KeyHomeClean); got != "🧹 Temizle" {
		t.Errorf("expected '🧹 Temizle', got '%s'", got)
	}

	// Reset to English
	i18n.SetLocale("en")
}

func TestI18nFallback(t *testing.T) {
	i18n := GetI18n()

	// Set to unsupported locale, should fall back to English
	i18n.SetLocale("fr")
	if got := i18n.T(KeyHomeClean); got != "🧹 Clean" {
		t.Errorf("expected fallback to English '🧹 Clean', got '%s'", got)
	}

	// Reset
	i18n.SetLocale("en")
}

func TestI18nUnknownKey(t *testing.T) {
	i18n := GetI18n()

	// Unknown key should return the key itself
	if got := i18n.T("unknown.key"); got != "unknown.key" {
		t.Errorf("expected unknown key to return itself, got '%s'", got)
	}
}

func TestI18nLocale(t *testing.T) {
	i18n := GetI18n()

	// Test locale getter
	locale := i18n.GetLocale()
	if locale.String() != "en" {
		t.Errorf("expected default locale 'en', got '%s'", locale.String())
	}

	// Change locale
	i18n.SetLocale("tr")
	locale = i18n.GetLocale()
	if locale.String() != "tr" {
		t.Errorf("expected locale 'tr', got '%s'", locale.String())
	}

	// Reset
	i18n.SetLocale("en")
}