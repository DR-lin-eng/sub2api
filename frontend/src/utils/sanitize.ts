import DOMPurify from 'dompurify'
import { marked } from 'marked'

marked.setOptions({
  breaks: true,
  gfm: true,
})

export function sanitizeSvg(svg: string): string {
  if (!svg) return ''
  return DOMPurify.sanitize(svg, { USE_PROFILES: { svg: true, svgFilters: true } })
}

export function sanitizeHtml(html: string): string {
  if (!html) return ''
  return DOMPurify.sanitize(html)
}

export function renderRichContent(content: string): string {
  if (!content) return ''
  const html = marked.parse(content) as string
  return sanitizeHtml(html)
}
