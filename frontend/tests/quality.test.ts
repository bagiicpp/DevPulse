import assert from "node:assert/strict"
import { test } from "node:test"
import { findingCounts, isQualityReport, ruleLabel, sourceLink, type QualityReport } from "../src/lib/quality.ts"

export const report: QualityReport = {
  repository: "owner/repo", branch: "main", commit: "a".repeat(40), completedAt: "2026-11-10T12:00:00Z", engineVersion: "0.668.0", durationSeconds: 2,
  inputFiles: 2, skippedFiles: 3, limited: false, codeLines: 10, functions: 1, complexity: 3, cyclomatic: 4,
  files: [{ path: "src/main.go", language: "LANGUAGE_GO", lines: 10, codeLines: 10, functions: 1, complexity: 3, cyclomatic: 4 }],
  findings: [{ rule: "qlty:complexity", level: "warning", message: "Complex function", locations: [{ path: "src/main.go", startLine: 1, endLine: 10 }] }],
  totalFindings: 1, limitations: ["Static scan only."],
}

test("validates measured reports while preserving valid zero findings and complexity", () => {
  assert.equal(isQualityReport(report), true)
  assert.equal(isQualityReport({ ...report, findings: [], totalFindings: 0, complexity: 0 }), true)
  for (const value of [null, {}, { ...report, files: null }, { ...report, findings: null }, { ...report, files: [] }, { ...report, complexity: -1 }, { ...report, durationSeconds: Number.NaN }, { ...report, commit: "main" }, { ...report, completedAt: "yesterday" }, { ...report, findings: [{ ...report.findings[0], locations: [] }] }]) {
    assert.equal(isQualityReport(value as QualityReport), false)
  }
})

test("groups findings by exact scanner rule without inventing quality scores", () => {
  const findings = [...report.findings, ...report.findings, { ...report.findings[0], rule: "qlty:duplicate-code" }]
  assert.deepEqual(findingCounts(findings), [{ rule: "qlty:complexity", count: 2 }, { rule: "qlty:duplicate-code", count: 1 }])
  assert.deepEqual(findingCounts([]), [])
  assert.equal(ruleLabel("qlty:function-parameters"), "function parameters")
})

test("links code to the captured commit and escapes special filename characters", () => {
  assert.equal(sourceLink(report, "src/file #1?.ts", 2, 5), `https://github.com/owner/repo/blob/${report.commit}/src/file%20%231%3F.ts#L2-L5`)
  assert.equal(sourceLink(report, "main.go", 2, 2).endsWith("#L2"), true)
})
