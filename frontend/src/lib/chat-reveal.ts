import type { AIResponse } from "./analysis"

export function responseWordCount(response: AIResponse) {
  return [response.summary, ...response.findings, ...response.recommendations]
    .reduce((count, text) => count + (text.match(/\s*\S+\s*/gu)?.length ?? 0), 0)
}

// Reveal whole words in reading order, keeping whitespace and list structure intact.
export function revealResponse(response: AIResponse, wordLimit: number): AIResponse {
  let remaining = Math.max(0, Math.floor(wordLimit))
  function reveal(text: string) {
    const words = text.match(/\s*\S+\s*/gu) ?? []
    const visible = words.slice(0, remaining).join("")
    remaining = Math.max(0, remaining - words.length)
    return visible
  }
  return {
    summary: reveal(response.summary),
    findings: response.findings.map(reveal).filter(Boolean),
    recommendations: response.recommendations.map(reveal).filter(Boolean),
  }
}
