package web

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/thevibeworks/ccx/internal/config"
	"github.com/thevibeworks/ccx/internal/i18n"
)

// loc is the request-scoped translator used by HTML renderers.
// A zero value speaks English so tests that call render functions
// directly keep their existing assertions.
type loc struct {
	tr *i18n.Translator
}

func defaultLoc() loc {
	return loc{tr: i18n.Default()}
}

func locFrom(r *http.Request) loc {
	// Detect from the request itself so handler-level tests (no
	// middleware) still honor ?lang=, the cookie, and Accept-Language.
	tr, _ := i18n.Detect(r, config.Locale())
	return loc{tr: tr}
}

func (l loc) T(key string, args ...any) string {
	if l.tr == nil {
		return i18n.Default().T(key, args...)
	}
	return l.tr.T(key, args...)
}

func (l loc) Code() string {
	if l.tr == nil {
		return i18n.En
	}
	return l.tr.Code()
}

// localeMiddleware resolves the UI language, persists an explicit
// ?lang= choice, and stores the translator on the request context.
func localeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tr, fromQuery := i18n.Detect(r, config.Locale())
		if fromQuery {
			http.SetCookie(w, i18n.Cookie(tr.Code()))
		}
		next.ServeHTTP(w, r.WithContext(i18n.WithLocale(r.Context(), tr)))
	})
}

// i18nScript emits window.CCX_I18N / CCX_LOCALE / t() for client JS.
func (l loc) i18nScript() string {
	code := l.Code()
	cat := i18n.Catalog(code)
	if cat == nil {
		cat = i18n.Catalog(i18n.En)
	}
	raw, err := json.Marshal(cat)
	if err != nil {
		raw = []byte("{}")
	}
	var b strings.Builder
	b.WriteString(`<script>window.CCX_LOCALE=`)
	b.WriteString(strconvQuote(code))
	b.WriteString(`;window.CCX_I18N=`)
	b.Write(raw)
	b.WriteString(`;function t(key){var d=window.CCX_I18N||{},s=d[key];if(s==null)s=key;if(arguments.length<2)return s;var a=Array.prototype.slice.call(arguments,1),i=0;return String(s).replace(/%(?:%|s|d)/g,function(m){if(m==='%%')return '%';if(i>=a.length)return m;return String(a[i++]);});}</script>`)
	return b.String()
}

func strconvQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func (l loc) langSwitcher() string {
	var b strings.Builder
	b.WriteString(`<label class="lang-switch-wrap">`)
	b.WriteString(`<span class="visually-hidden">`)
	b.WriteString(htmlText(l.T("nav.language")))
	b.WriteString(`</span>`)
	b.WriteString(`<select class="lang-switch" id="lang-switch" title="`)
	b.WriteString(htmlText(l.T("nav.language")))
	b.WriteString(`" aria-label="`)
	b.WriteString(htmlText(l.T("nav.language")))
	b.WriteString(`">`)
	for _, opt := range i18n.Locales() {
		sel := ""
		if opt.Code == l.Code() {
			sel = " selected"
		}
		// Labels stay in their own script so the switcher is recognizable
		// regardless of the active locale (English / 简体中文).
		label := opt.Label
		if opt.Code == i18n.ZhCN {
			label = l.T("nav.lang_zh")
		} else {
			label = l.T("nav.lang_en")
		}
		b.WriteString(`<option value="`)
		b.WriteString(htmlText(opt.Code))
		b.WriteString(`"`)
		b.WriteString(sel)
		b.WriteString(`>`)
		b.WriteString(htmlText(label))
		b.WriteString(`</option>`)
	}
	b.WriteString(`</select></label>`)
	return b.String()
}

func htmlText(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

// ui escapes catalog text for HTML body content. Apostrophes stay
// literal so tests and readers see "Can't" rather than "Can&#39;t".
func ui(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
