import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  buildEmbeddedUrl,
  detectTheme,
  isEmbeddedUrl,
  isMixedContentEmbeddingBlocked,
} from '../embedded-url'

describe('embedded-url', () => {
  afterEach(() => {
    document.documentElement.classList.remove('dark')
    vi.restoreAllMocks()
  })

  it('adds only non-sensitive embedded query parameters', () => {
    const result = buildEmbeddedUrl('https://pay.example.com/checkout?plan=pro', 'dark', 'zh-CN')

    const url = new URL(result)
    expect(url.searchParams.get('plan')).toBe('pro')
    expect(url.searchParams.get('theme')).toBe('dark')
    expect(url.searchParams.get('lang')).toBe('zh-CN')
    expect(url.searchParams.get('ui_mode')).toBe('embedded')
    expect(url.searchParams.has('user_id')).toBe(false)
    expect(url.searchParams.has('token')).toBe(false)
    expect(url.searchParams.has('src_host')).toBe(false)
    expect(url.searchParams.has('src_url')).toBe(false)
  })

  it('omits optional params when they are empty', () => {
    const result = buildEmbeddedUrl('https://pay.example.com/checkout', 'light')

    const url = new URL(result)
    expect(url.searchParams.get('theme')).toBe('light')
    expect(url.searchParams.get('ui_mode')).toBe('embedded')
    expect(url.searchParams.has('lang')).toBe(false)
  })

  it('allows http and https embedded urls', () => {
    const result = buildEmbeddedUrl('http://pay.example.com/checkout', 'light')
    expect(new URL(result).searchParams.get('ui_mode')).toBe('embedded')
    expect(isEmbeddedUrl('https://pay.example.com/checkout')).toBe(true)
    expect(isEmbeddedUrl('http://pay.example.com/checkout')).toBe(true)
  })

  it('detects mixed content iframe blocking', () => {
    expect(isMixedContentEmbeddingBlocked('https:', 'http://pay.example.com/checkout')).toBe(true)
    expect(isMixedContentEmbeddingBlocked('https:', 'https://pay.example.com/checkout')).toBe(false)
    expect(isMixedContentEmbeddingBlocked('http:', 'http://pay.example.com/checkout')).toBe(false)
  })

  it('returns original string for invalid url input', () => {
    expect(buildEmbeddedUrl('not a url', 'light')).toBe('not a url')
  })

  it('detects dark mode from document root class', () => {
    document.documentElement.classList.add('dark')
    expect(detectTheme()).toBe('dark')
  })
})
