import { useCallback, useEffect, useState } from "react"
import axios from "axios"
import { Link } from "react-router"
import type { GitHubUser } from "@/hooks/use-session"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { requestError } from "@/lib/analysis"

type Repository = { full_name: string; private: boolean }

export function RepositoryPicker({ user, value, onChange, disabled = false, qualityScan = false }: {
  user: GitHubUser | null
  value: string
  onChange: (value: string) => void
  disabled?: boolean
  qualityScan?: boolean
}) {
  const [repositories, setRepositories] = useState<Repository[]>([])
  const [page, setPage] = useState(0)
  const [hasMore, setHasMore] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState("")

  const load = useCallback((page: number, signal?: AbortSignal) => {
    return axios.get<Repository[]>("/api/repositories", {
      params: { page }, signal, timeout: 15_000,
    }).then(({ data }) => {
      if (signal?.aborted) return
      setRepositories((current) => {
        const all = page === 1 ? data : [...current, ...data]
        return [...new Map(all.map((repo) => [repo.full_name, repo])).values()]
      })
      setPage(page)
      setHasMore(data.length === 100)
    }).catch((error: unknown) => {
      if (!axios.isCancel(error)) setError(requestError(error))
    }).finally(() => {
      if (!signal?.aborted) setLoading(false)
    })
  }, [])

  useEffect(() => {
    if (!user) return
    const controller = new AbortController()
    void load(1, controller.signal)
    return () => controller.abort()
  }, [user, load])

  return (
    <div className="space-y-4">
      {user ? (
        <div className="space-y-2">
          <label htmlFor="your-repositories" className="text-sm font-medium">Your GitHub repositories</label>
          <select
            id="your-repositories"
            value={repositories.some((repo) => repo.full_name === value) ? value : ""}
            onChange={(event) => onChange(event.target.value)}
            disabled={disabled || loading}
            className="h-11 w-full min-w-0 rounded-full border border-input bg-card px-4 text-base outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:opacity-50 md:text-sm"
          >
            <option value="" disabled>{loading ? "Loading repositories…" : "Select a repository"}</option>
            {repositories.map((repo) => (
              <option key={repo.full_name} value={repo.full_name}>
                {repo.full_name}{repo.private ? " (private)" : ""}
              </option>
            ))}
          </select>
          {loading && <p role="status" className="text-sm text-muted-foreground">Loading repositories…</p>}
          {!loading && !error && page > 0 && repositories.length === 0 && (
            <p className="text-sm text-muted-foreground">No repositories available. Try a public repository below or sign in again to grant access.</p>
          )}
          {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
          {(error || hasMore) && (
            <Button type="button" variant="outline" disabled={disabled || loading} onClick={() => { setLoading(true); setError(""); void load(page + 1) }}>
              {error ? "Retry loading repositories" : "Load more repositories"}
            </Button>
          )}
        </div>
      ) : (
        <p className="text-sm leading-6 text-muted-foreground">
          <Link to="/login" className="text-foreground underline underline-offset-4">Sign in with GitHub</Link> to select your public and private repositories. Guests can analyze public repositories.
        </p>
      )}
      <div className="space-y-2">
        <label htmlFor="repository" className="text-sm font-medium">GitHub Repository</label>
        <Input id="repository" placeholder="owner/repository" value={value} onChange={(event) => onChange(event.target.value)} disabled={disabled} maxLength={140} required />
      </div>
      <p className="text-sm leading-6 text-muted-foreground">
        {qualityScan ? "Qlty scans supported source files from the default branch. No SonarQube setup or AI provider is required." : "File names and README content from this repository will be sent to the AI provider for analysis. Source file contents and live CI metrics are not included."}
      </p>
    </div>
  )
}
