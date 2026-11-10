import { useEffect, useRef, useState } from "react"
import axios from "axios"
import { useSearchParams } from "react-router"
import { LoaderCircle } from "lucide-react"
import type { AuthState } from "@/hooks/use-session"
import { Button } from "@/components/ui/button"
import { RepositoryPicker } from "@/components/repository-picker"
import { QualityReportView } from "@/components/quality-report"
import { isRepositoryName, readDefaultRepository } from "@/lib/preferences"
import { requestError } from "@/lib/analysis"
import { isQualityReport, type QualityReport } from "@/lib/quality"

export function ProjectAnalysis({ auth }: { auth: AuthState }) {
  const [params] = useSearchParams()
  const [repository, setRepository] = useState(() => params.get("repository") ?? readDefaultRepository())
  const [result, setResult] = useState<QualityReport | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const request = useRef<AbortController | null>(null)
  const resultHeading = useRef<HTMLHeadingElement>(null)
  useEffect(() => () => request.current?.abort(), [])
  useEffect(() => { if (result) resultHeading.current?.focus({ preventScroll: true }) }, [result])

  function cancel() {
    request.current?.abort()
    setLoading(false)
    setError("Scan canceled. Your repository selection is ready to retry.")
  }

  async function analyze() {
    if (loading) return
    if (!isRepositoryName(repository.trim())) { setError("Enter a GitHub repository in the format owner/repository."); return }
    const controller = new AbortController()
    request.current = controller
    setResult(null); setError(""); setLoading(true)
    try {
      const { data } = await axios.post<QualityReport>("/api/analysis", { repository: repository.trim() }, { timeout: 150_000, signal: controller.signal })
      if (!isQualityReport(data)) throw new Error("The scanner returned an invalid report. Please retry.")
      if (!controller.signal.aborted) setResult(data)
    } catch (error) {
      if (!controller.signal.aborted) setError(axios.isAxiosError(error) ? requestError(error) : error instanceof Error ? error.message : "The scan failed. Please retry.")
    } finally { if (!controller.signal.aborted) setLoading(false) }
  }

  return (
    <div className="mx-auto w-full max-w-5xl space-y-8 py-2 md:py-6">
      <header>
        <h1 className="text-4xl leading-tight font-normal tracking-[-0.04em] md:text-5xl">Project Analysis</h1>
        <p className="mt-3 max-w-prose text-sm leading-6 text-muted-foreground">Try Qlty CLI on a GitHub repository. Explore code complexity, duplication findings, and other structural code smells from a fixed commit.</p>
      </header>
      <form noValidate className="space-y-5 rounded-2xl border bg-card px-5 py-6 sm:px-7" onSubmit={(event) => { event.preventDefault(); void analyze() }}>
        <div className="max-w-2xl"><RepositoryPicker qualityScan user={auth.user} value={repository} onChange={(value) => { setRepository(value); setResult(null); setError("") }} disabled={loading} /></div>
        <div className="flex flex-wrap items-center gap-4"><Button type="submit" disabled={loading || !repository.trim()}>{loading ? "Scanning repository…" : error ? "Retry scan" : "Scan with Qlty"}</Button>{loading && <Button type="button" variant="outline" onClick={cancel}>Cancel scan</Button>}<p className="text-xs leading-5 text-muted-foreground">Experimental · No project import · No AI key</p></div>
      </form>
      {error && <p role="alert" className="rounded-xl border border-destructive/30 px-5 py-4 text-sm leading-6 text-destructive">{error}</p>}
      {loading && <div role="status" className="flex items-start gap-3 rounded-xl border bg-card p-5 text-sm leading-6"><LoaderCircle className="mt-1 size-4 shrink-0 motion-safe:animate-spin" aria-hidden="true" /><p>Downloading the selected commit and running Qlty’s built-in code analysis. This can take up to two minutes.</p></div>}
      {result && <QualityReportView key={result.completedAt} report={result} headingRef={resultHeading} />}
      {!result && !loading && <section className="grid gap-5 border-t pt-7 sm:grid-cols-[1fr_1.4fr] sm:gap-12" aria-label="About this trial"><h2 className="text-2xl tracking-tight">See what the scanner finds.</h2><div className="space-y-3 text-sm leading-6 text-muted-foreground"><p>This trial measures supported source files and links findings to the exact scanned code. You can inspect the per-file metrics and scan exclusions before deciding whether Qlty fits your project.</p><p>It checks structure and duplication without running your project’s code or tests. Test coverage, runtime behavior, and a quality-gate verdict are not measured.</p></div></section>}
    </div>
  )
}
