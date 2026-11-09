import { useState } from "react"
import { FolderGit2 } from "lucide-react"
import { useNavigate } from "react-router"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { isRepositoryName, readDefaultRepository } from "@/lib/preferences"

export function Repositories() {
  const [repository, setRepository] = useState(readDefaultRepository)
  const navigate = useNavigate()
  return (
    <div className="mx-auto w-full max-w-5xl space-y-10 py-2 md:py-6">
      <header>
        <h1 className="text-4xl leading-none font-normal tracking-[-0.04em] md:text-5xl">
          Repositories
        </h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Bring a repository name into your project questions.
        </p>
      </header>
      <section className="rounded-xl border p-6 md:p-8">
        <FolderGit2
          className="size-6 text-muted-foreground"
          aria-hidden="true"
        />
        <h2 className="mt-4 text-lg font-medium">No repositories connected</h2>
        <p className="mt-2 max-w-xl text-sm leading-6 text-muted-foreground">
          GitHub sign-in verifies your identity. Repository sync is not
          available yet, so this workspace does not list or inspect your
          repositories.
        </p>
      </section>
      <section className="max-w-xl">
        <h2 className="text-2xl font-normal tracking-tight">
          Ask about a repository
        </h2>
        <p className="mt-2 text-sm leading-6 text-muted-foreground">
          Use a name as context for AI guidance. No repository data will be
          retrieved.
        </p>
        <form
          noValidate
          className="mt-5 space-y-4"
          onSubmit={(event) => {
            event.preventDefault()
            const value = repository.trim()
            if (!isRepositoryName(value)) {
              toast.error("Use the format owner/repository.")
              return
            }
            navigate(`/chat?repository=${encodeURIComponent(value)}`)
          }}
        >
          <div className="space-y-2">
            <label htmlFor="repository-name" className="text-sm font-medium">
              Repository name
            </label>
            <Input
              id="repository-name"
              value={repository}
              onChange={(event) => setRepository(event.target.value)}
              placeholder="owner/repository"
              maxLength={140}
              required
            />
          </div>
          <Button type="submit">Continue to AI chat</Button>
        </form>
      </section>
    </div>
  )
}
