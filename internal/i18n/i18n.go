// Package i18n is a small catalog-based localizer for the ccx web UI.
//
// English is the source locale: every key lives in the English catalog
// first. Other locales must cover the same keys (tests enforce this).
// Missing translations fall back to English rather than showing a key.
package i18n

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// Supported locale codes. Aliases (zh, zh-Hans, zh_CN) resolve to ZhCN.
const (
	En   = "en"
	ZhCN = "zh-CN"
)

// CookieName is the cookie that persists an explicit language choice.
const CookieName = "ccx-lang"

// QueryParam is the URL parameter that selects a language (?lang=zh-CN).
const QueryParam = "lang"

type ctxKey struct{}

// Translator looks up catalog strings for one locale.
type Translator struct {
	code    string
	catalog map[string]string
}

// Default returns the English translator.
func Default() *Translator {
	return For(En)
}

// For returns a translator for code. Unknown or empty codes become English.
func For(code string) *Translator {
	code = Normalize(code)
	cat, ok := catalogs[code]
	if !ok {
		code = En
		cat = catalogs[En]
	}
	return &Translator{code: code, catalog: cat}
}

// Code is the canonical locale id (en, zh-CN).
func (t *Translator) Code() string {
	if t == nil {
		return En
	}
	return t.code
}

// T looks up key and formats it with args via fmt.Sprintf.
// Unknown keys fall back to English, then to the key itself.
func (t *Translator) T(key string, args ...any) string {
	if t == nil {
		return Default().T(key, args...)
	}
	s, ok := t.catalog[key]
	if !ok {
		s, ok = catalogs[En][key]
	}
	if !ok {
		s = key
	}
	if len(args) == 0 {
		return s
	}
	return fmt.Sprintf(s, args...)
}

// Keys returns the English catalog keys, sorted, for coverage tests.
func Keys() []string {
	keys := make([]string, 0, len(catalogs[En]))
	for k := range catalogs[En] {
		keys = append(keys, k)
	}
	return keys
}

// Catalog returns a copy of the catalog for code, or nil if unknown.
func Catalog(code string) map[string]string {
	src, ok := catalogs[Normalize(code)]
	if !ok {
		return nil
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

// Supported reports whether code (after Normalize) has a catalog.
func Supported(code string) bool {
	_, ok := catalogs[Normalize(code)]
	return ok && Normalize(code) != ""
}

// Locales is the UI language list, source locale first.
func Locales() []LocaleInfo {
	return []LocaleInfo{
		{Code: En, Label: catalogs[En]["nav.lang_en"]},
		{Code: ZhCN, Label: catalogs[En]["nav.lang_zh"]},
	}
}

// LocaleInfo is one entry in the language switcher.
type LocaleInfo struct {
	Code  string
	Label string
}

// Normalize maps aliases onto a catalog code. Empty / auto / unknown → "".
func Normalize(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return ""
	}
	code = strings.ReplaceAll(code, "_", "-")
	low := strings.ToLower(code)
	switch low {
	case "auto", "default":
		return ""
	case "en", "en-us", "en-gb", "en-au":
		return En
	case "zh", "zh-cn", "zh-hans", "zh-hans-cn", "zh-sg":
		return ZhCN
	}
	// zh-TW / zh-Hant are not a catalog yet — do not silently
	// serve Simplified Chinese for a Traditional request.
	return ""
}

// WithLocale stores the translator on ctx.
func WithLocale(ctx context.Context, t *Translator) context.Context {
	return context.WithValue(ctx, ctxKey{}, t)
}

// FromContext returns the translator stored on ctx, or English.
func FromContext(ctx context.Context) *Translator {
	if ctx == nil {
		return Default()
	}
	if t, ok := ctx.Value(ctxKey{}).(*Translator); ok && t != nil {
		return t
	}
	return Default()
}

// FromRequest is FromContext(r.Context()).
func FromRequest(r *http.Request) *Translator {
	if r == nil {
		return Default()
	}
	return FromContext(r.Context())
}

var catalogs = map[string]map[string]string{
	En:   en,
	ZhCN: zhCN,
}
