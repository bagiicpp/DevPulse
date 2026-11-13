import assert from "node:assert/strict"
import { test } from "node:test"
import { responseWordCount, revealResponse } from "../src/lib/chat-reveal.ts"

const response = {
  summary: "A small project.\n",
  findings: ["Uses React.", "Tests are missing."],
  recommendations: ["Add tests 👩🏽‍💻."],
}

test("reveals summary, findings, then recommendations without empty list items", () => {
  assert.deepEqual(revealResponse(response, 0), { summary: "", findings: [], recommendations: [] })
  assert.deepEqual(revealResponse(response, 4), {
    summary: response.summary, findings: ["Uses "], recommendations: [],
  })
  assert.deepEqual(revealResponse(response, 9), {
    summary: response.summary, findings: response.findings, recommendations: ["Add "],
  })
})

test("completion preserves exact whitespace, Unicode, and all structured content", () => {
  const total = responseWordCount(response)
  assert.equal(total, 11)
  assert.deepEqual(revealResponse(response, total), response)
  assert.deepEqual(revealResponse(response, Infinity), response)
})

test("handles empty lists and text without spaces", () => {
  const short = { summary: "こんにちは👋", findings: [], recommendations: [] }
  assert.equal(responseWordCount(short), 1)
  assert.deepEqual(revealResponse(short, 1), short)
})

test("clamps negative word limits and rounds down fractional limits", () => {
  assert.deepEqual(revealResponse(response, -1), revealResponse(response, 0))
  assert.deepEqual(revealResponse(response, 1.9), revealResponse(response, 1))
})
