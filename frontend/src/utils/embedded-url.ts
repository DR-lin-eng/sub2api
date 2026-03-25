/**
 * Shared URL builder for iframe-embedded pages.
 * Used by PurchaseSubscriptionView and CustomPageView to build consistent URLs
 * with only non-sensitive presentation parameters.
 */

const EMBEDDED_THEME_QUERY_KEY = 'theme'
const EMBEDDED_LANG_QUERY_KEY = 'lang'
const EMBEDDED_UI_MODE_QUERY_KEY = 'ui_mode'
const EMBEDDED_UI_MODE_VALUE = 'embedded'

export function buildEmbeddedUrl(
  baseUrl: string,
  theme: 'light' | 'dark' = 'light',
  lang?: string,
): string {
  if (!baseUrl) return baseUrl
  try {
    const url = new URL(baseUrl)
    if (url.protocol !== 'http:' && url.protocol !== 'https:') {
      return baseUrl
    }
    url.searchParams.set(EMBEDDED_THEME_QUERY_KEY, theme)
    if (lang) {
      url.searchParams.set(EMBEDDED_LANG_QUERY_KEY, lang)
    }
    url.searchParams.set(EMBEDDED_UI_MODE_QUERY_KEY, EMBEDDED_UI_MODE_VALUE)
    return url.toString()
  } catch {
    return baseUrl
  }
}

export function isEmbeddedUrl(value: string): boolean {
  if (!value) return false
  try {
    const protocol = new URL(value).protocol
    return protocol === 'http:' || protocol === 'https:'
  } catch {
    return false
  }
}

export function isMixedContentEmbeddingBlocked(
  currentPageProtocol: string,
  embeddedUrl: string,
): boolean {
  if (!embeddedUrl) return false
  if (currentPageProtocol.trim().toLowerCase() !== 'https:') {
    return false
  }
  try {
    return new URL(embeddedUrl).protocol === 'http:'
  } catch {
    return false
  }
}

export function detectTheme(): 'light' | 'dark' {
  if (typeof document === 'undefined') return 'light'
  return document.documentElement.classList.contains('dark') ? 'dark' : 'light'
}
