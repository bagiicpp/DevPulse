import { useState } from "react"
import axios from "axios"
import { useSearchParams, Link } from "react-router"
import { toast } from "sonner"
import type { AuthState } from "@/hooks/use-session"
import { Button } from "@/components/ui/button"
import { RepositoryPicker } from "@/components/repository-picker"
import { AnalysisResult } from "@/components/analysis-result"
import { isRepositoryName, readDefaultRepository } from "@/lib/preferences"
import { requestError, type AIResponse } from "@/lib/analysis"

const topics = [
  { label: "Overall project health", question: "Assess this project's health based on its file structure and README. Identify strengths, gaps, and practical next steps, stating what evidence is missing." },
  { label: "CI reliability", question: "Identify CI configuration from the file names and README. Explain what is documented and what CI evidence is needed to assess reliability." },
  { label: "Code quality", question: "Review this project's structure and documented testing practices. Identify possible gaps without assuming source-code quality or test coverage." },
  { label: "Pull request workflow", question: "Review this project's documented contribution and pull request workflow. Suggest improvements and identify missing evidence." },
]

export function ProjectAnalysis({ auth }: { auth: AuthState }) {
  const [params] = useSearchParams()
  const [repository, setRepository] = useState(() => params.get("repository") ?? readDefaultRepository())
  const [question, setQuestion] = useState(topics[0].question)
  const [result, setResult] = useState<AIResponse | null>(null)
  const [loading, setLoading] = useState(false)

  async function analyze() {
    if (loading) return
    if (!isRepositoryName(repository.trim())) {
      toast.error("Use the format owner/repository.")
      return
    }
    setResult(null)
    setLoading(true)
    try {
      const { data } = await axios.post<AIResponse>("/api/chat", { repository: repository.trim(), question }, { timeout: 75_000 })
      setResult(data)
    } catch (error) {
      toast.error(requestError(error))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="mx-auto w-full max-w-3xl space-y-8 py-2 md:py-6">
      <header>
        <h1 className="text-4xl leading-none font-normal tracking-[-0.04em] md:text-5xl">Project Analysis</h1>
        <p className="mt-2 text-sm leading-6 text-muted-foreground">
          Analyze your repository's structure and documentation with AI.
        </p>
      </header>
      <form noValidate className="space-y-6" onSubmit={(event) => { event.preventDefault(); void analyze() }}>
        <RepositoryPicker user={auth.user} value={repository} onChange={(value) => { setRepository(value); setResult(null) }} disabled={loading} />
        <div className="space-y-2">
          <label htmlFor="analysis-topic" className="text-sm font-medium">What would you like to understand?</label>
          <select id="analysis-topic" value={question} disabled={loading} onChange={(event) => { setQuestion(event.target.value); setResult(null) }} className="h-11 w-full rounded-full border border-input bg-card px-4 text-base outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:opacity-50 md:text-sm">
            {topics.map((topic) => <option key={topic.label} value={topic.question}>{topic.label}</option>)}
          </select>
        </div>
        <Button type="submit" disabled={loading || !repository.trim()}>{loading ? "Analyzing…" : "Analyze repository"}</Button>
      </form>
      {loading && <p role="status" className="text-sm text-muted-foreground">Reading repository context and generating analysis…</p>}
      {result && (
        <section aria-label="Analysis results" aria-live="polite" className="space-y-5 border-t pt-6">
          <h2 className="text-2xl font-normal tracking-tight wrap-anywhere">Analysis for {repository.trim()}</h2>
          <AnalysisResult response={result} />
          <Button variant="outline" render={<Link to={`/chat?repository=${encodeURIComponent(repository.trim())}`} />} nativeButton={false}>Ask a follow-up in chat</Button>
        </section>
      )}
    </div>
  )
}
