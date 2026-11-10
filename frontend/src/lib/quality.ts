export type FileMetric = {
  path: string
  language: string
  lines: number
  codeLines: number
  functions: number
  complexity: number
  cyclomatic: number
}
export type QualityFinding = {
  rule: string
  level: string
  message: string
  locations: { path: string; startLine: number; endLine: number }[]
}
export type QualityReport = {
  repository: string
  branch: string
  commit: string
  completedAt: string
  engineVersion: string
  durationSeconds: number
  inputFiles: number
  skippedFiles: number
  limited: boolean
  codeLines: number
  functions: number
  complexity: number
  cyclomatic: number
  files: FileMetric[]
  findings: QualityFinding[]
  totalFindings: number
  limitations: string[]
}

export function sourceLink(report: Pick<QualityReport, "repository" | "commit">, path: string, start?: number, end?: number) {
  const url = `https://github.com/${report.repository}/blob/${report.commit}/${path.split("/").map(encodeURIComponent).join("/")}`
  return start ? `${url}#L${start}${end && end !== start ? `-L${end}` : ""}` : url
}

export function ruleLabel(rule: string) {
  return rule.replace(/^qlty:/, "").replaceAll("-", " ")
}

export function findingCounts(findings: QualityFinding[]) {
  const counts = new Map<string, number>()
  for (const finding of findings) counts.set(finding.rule, (counts.get(finding.rule) ?? 0) + 1)
  return [...counts].map(([rule, count]) => ({ rule, count })).sort((a, b) => b.count - a.count || a.rule.localeCompare(b.rule))
}

export function isQualityReport(value: QualityReport): boolean {
  const number = (item: unknown) => typeof item === "number" && Number.isFinite(item) && item >= 0
  return !!value && typeof value.repository === "string" && typeof value.commit === "string" && /^[a-f0-9]{40}$/.test(value.commit)
    && typeof value.branch === "string" && typeof value.engineVersion === "string" && Number.isFinite(Date.parse(value.completedAt))
    && [value.inputFiles, value.skippedFiles, value.codeLines, value.functions, value.complexity, value.cyclomatic, value.totalFindings, value.durationSeconds].every(number)
    && typeof value.limited === "boolean" && Array.isArray(value.limitations) && value.limitations.every((item) => typeof item === "string")
    && Array.isArray(value.files) && value.files.length > 0 && value.files.every((file) => file && typeof file.path === "string" && typeof file.language === "string" && [file.lines, file.codeLines, file.functions, file.complexity, file.cyclomatic].every(number))
    && Array.isArray(value.findings) && value.findings.every((finding) => finding && typeof finding.rule === "string" && typeof finding.message === "string" && typeof finding.level === "string" && Array.isArray(finding.locations) && finding.locations.length > 0 && finding.locations.every((location) => typeof location.path === "string" && Number.isInteger(location.startLine) && location.startLine > 0 && Number.isInteger(location.endLine) && location.endLine >= location.startLine))
}
