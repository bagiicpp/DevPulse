import { useState } from "react"
import { useNavigate } from "react-router"
import { toast } from "sonner"
import type { AuthState } from "@/hooks/use-session"
import { Button } from "@/components/ui/button"
import { RepositoryPicker } from "@/components/repository-picker"
import { isRepositoryName, readDefaultRepository } from "@/lib/preferences"

export function Repositories({ auth }: { auth: AuthState }) {
  const [repository, setRepository] = useState(readDefaultRepository)
  const navigate = useNavigate()
  return (
    <div className="mx-auto w-full max-w-3xl space-y-8 py-2 md:py-6">
      <header>
        <h1 className="text-4xl leading-none font-normal tracking-[-0.04em] md:text-5xl">Repositories</h1>
        <p className="mt-2 text-sm leading-6 text-muted-foreground">
          Select a GitHub repository to understand its structure and documentation.
        </p>
      </header>
      <form noValidate className="space-y-6" onSubmit={(event) => {
        event.preventDefault()
        const value = repository.trim()
        if (!isRepositoryName(value)) {
          toast.error("Use the format owner/repository.")
          return
        }
        navigate(`/analysis?repository=${encodeURIComponent(value)}`)
      }}>
        <RepositoryPicker user={auth.user} value={repository} onChange={setRepository} />
        <Button type="submit" disabled={!repository.trim()}>Continue to analysis</Button>
      </form>
    </div>
  )
}
