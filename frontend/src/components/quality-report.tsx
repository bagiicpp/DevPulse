import { useState, type RefObject } from "react"
import { ChevronDownIcon } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { findingCounts, ruleLabel, sourceLink, type QualityReport } from "@/lib/quality"

function count(value: number) { return value.toLocaleString() }

export function QualityReportView({ report, headingRef }: { report: QualityReport; headingRef: RefObject<HTMLHeadingElement | null> }) {
  const [rule, setRule] = useState("")
  const counts = findingCounts(report.findings)
  const filtered = report.findings.filter((finding) => !rule || finding.rule === rule)
  const complexFiles = report.files.slice(0, 10)
  const maximum = Math.max(1, ...complexFiles.map((file) => file.complexity))
  const completed = new Date(report.completedAt).toLocaleString()

  return (
    <section aria-label="Qlty analysis results" className="overflow-hidden rounded-2xl border bg-card">
      <header className="space-y-3 px-5 py-7 sm:px-8">
        <h2 ref={headingRef} tabIndex={-1} className="text-2xl tracking-tight outline-none sm:text-3xl">Your code, measured by Qlty.</h2>
        <p className="text-sm text-muted-foreground wrap-anywhere">{report.repository} · {report.branch} · <a className="font-mono underline underline-offset-4" href={`https://github.com/${report.repository}/commit/${report.commit}`} target="_blank" rel="noreferrer">{report.commit.slice(0, 7)}</a></p>
        <p className="max-w-prose text-sm leading-6 text-muted-foreground">Static code metrics and code-smell findings from Qlty CLI {report.engineVersion}. This scan uses built-in rules and includes test files. Source is scanned without uploading it to an AI or cloud analysis service.</p>
        {report.limited && <p className="text-sm leading-6 text-destructive">This scan reached its file-count or source-size limit. Results describe a partial snapshot; see scan scope below.</p>}
      </header>

      <section aria-labelledby="quality-measurements" className="border-t px-5 py-6 sm:px-8">
        <h3 id="quality-measurements" className="text-lg font-medium">Code measurements</h3>
        <dl className="mt-4 divide-y text-sm">
          {([ ["Files with metrics", `${count(report.files.length)} / ${count(report.inputFiles)} supplied`], ["Lines of code", count(report.codeLines)], ["Functions", count(report.functions)], ["Qlty complexity", count(report.complexity)], ["Cyclomatic complexity", count(report.cyclomatic)], ["Code-smell findings", count(report.totalFindings)] ] as const).map(([label, value]) => <div className="flex items-baseline justify-between gap-4 py-3" key={label}><dt className="text-muted-foreground">{label}</dt><dd className="text-right tabular-nums">{value}</dd></div>)}
        </dl>
        <p className="mt-4 text-xs leading-5 text-muted-foreground">Complexity values are sums of Qlty’s file measurements, not quality scores. Cyclomatic complexity describes branching paths. Neither measurement establishes correctness.</p>
      </section>

      <figure className="border-t px-5 py-6 sm:px-8">
        <figcaption><h3 className="text-lg font-medium">Where complexity is concentrated</h3><p className="mt-1 text-xs leading-5 text-muted-foreground">Up to ten files, ranked by Qlty complexity. Follow a path to inspect the scanned commit.</p></figcaption>
        <div className="mt-5 space-y-2">
          {complexFiles.map((file) => <div className="grid grid-cols-[minmax(0,1.2fr)_1fr_2.5rem] items-center gap-3 py-1 text-xs" key={file.path}>
            <a className="wrap-anywhere underline underline-offset-4" href={sourceLink(report, file.path)} target="_blank" rel="noreferrer">{file.path}</a>
            <span className="h-2 overflow-hidden rounded-sm bg-muted" aria-hidden="true"><span className="block h-full bg-foreground" style={{ width: `${file.complexity / maximum * 100}%` }} /></span>
            <span className="text-right tabular-nums">{count(file.complexity)}</span>
          </div>)}
        </div>
      </figure>

      <section aria-labelledby="quality-findings" className="border-t px-5 py-6 sm:px-8">
        <div className="flex flex-wrap items-center justify-between gap-3"><h3 id="quality-findings" className="text-lg font-medium">Code-smell findings</h3>{rule && <Button variant="ghost" onClick={() => setRule("")}>Clear filter</Button>}</div>
        <p className="mt-2 text-xs leading-5 text-muted-foreground">Duplication, complex code, deep nesting, and other structural rules supported by each language. These are scanner findings, not verified bugs.</p>
        {counts.length > 0 && <div className="mt-4 flex flex-wrap gap-2" aria-label="Filter findings by rule">{counts.map((item) => <Button key={item.rule} variant={rule === item.rule ? "default" : "outline"} aria-pressed={rule === item.rule} onClick={() => setRule(rule === item.rule ? "" : item.rule)}>{ruleLabel(item.rule)} ({item.count})</Button>)}</div>}
        <p role="status" className="mt-4 text-xs text-muted-foreground">Showing {filtered.length} of {report.totalFindings} findings{report.findings.length < report.totalFindings ? " (display limited to 200)" : ""}</p>
        {report.findings.length < report.totalFindings && <p className="mt-2 text-xs text-muted-foreground">Filter counts refer to the displayed findings.</p>}
        <div className="mt-3 divide-y">{filtered.map((finding, index) => <article className="py-5" key={`${finding.rule}:${index}`}>
          <p className="text-xs text-muted-foreground">{ruleLabel(finding.rule)} · {finding.level || "notice"}</p>
          <h4 className="mt-2 text-sm font-medium leading-6 wrap-anywhere">{finding.message}</h4>
          <ul className="mt-3 space-y-2 text-xs leading-5">{finding.locations.map((location, i) => <li className="wrap-anywhere" key={i}><a className="underline underline-offset-4" href={sourceLink(report, location.path, location.startLine, location.endLine)} target="_blank" rel="noreferrer">{location.path}:{location.startLine}{location.endLine !== location.startLine ? `–${location.endLine}` : ""}</a></li>)}</ul>
        </article>)}</div>
        {report.totalFindings === 0 && <p className="mt-4 text-sm leading-6 text-muted-foreground">Qlty reported no code smells under this scan policy. This does not establish that the project is bug-free or fully tested.</p>}
      </section>

      <Collapsible className="border-t px-5 py-5 sm:px-8">
        <CollapsibleTrigger render={<Button type="button" variant="ghost" className="group w-full justify-between px-0 hover:bg-transparent" />}>
          File measurements ({report.files.length})
          <ChevronDownIcon aria-hidden="true" className="size-4 transition-transform duration-200 group-data-panel-open:rotate-180 motion-reduce:transition-none" />
        </CollapsibleTrigger>
        <CollapsibleContent className="h-(--collapsible-panel-height) overflow-hidden transition-[height,opacity] duration-200 ease-out data-ending-style:h-0 data-ending-style:opacity-0 data-starting-style:h-0 data-starting-style:opacity-0 motion-reduce:transition-none">
        <div className="mt-4 max-h-96 overflow-auto rounded border"><table className="w-full text-left text-xs"><caption className="sr-only">Qlty measurements for every analyzed file</caption><thead className="bg-muted"><tr>{["File", "Language", "Code lines", "Functions", "Qlty complexity", "Cyclomatic"].map((label) => <th key={label} className="px-3 py-3 font-medium">{label}</th>)}</tr></thead><tbody>{report.files.map((file) => <tr className="border-t" key={file.path}><td className="px-3 py-3 wrap-anywhere"><a className="underline underline-offset-4" href={sourceLink(report, file.path)} target="_blank" rel="noreferrer">{file.path}</a></td><td className="px-3">{file.language.replace("LANGUAGE_", "").toLowerCase()}</td>{[file.codeLines, file.functions, file.complexity, file.cyclomatic].map((value, i) => <td className="px-3 tabular-nums" key={i}>{count(value)}</td>)}</tr>)}</tbody></table></div>
        </CollapsibleContent>
      </Collapsible>

      <section className="border-t px-5 py-6 sm:px-8" aria-labelledby="quality-scope"><h3 id="quality-scope" className="text-lg font-medium">Scan scope & limitations</h3>
        <p className="mt-3 text-sm text-muted-foreground">{report.inputFiles} source files supplied · {report.skippedFiles} archive files excluded · {report.durationSeconds.toFixed(1)} seconds</p>
        <ul className="mt-4 list-disc space-y-3 pl-4 text-xs leading-6 text-muted-foreground">{report.limitations.map((item, index) => <li key={index}>{item}</li>)}</ul>
      </section>
      <footer className="border-t px-5 py-4 text-xs text-muted-foreground sm:px-8">Qlty CLI {report.engineVersion} · Completed {completed}</footer>
    </section>
  )
}
