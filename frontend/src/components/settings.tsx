import { useState } from "react"
import { Link } from "react-router"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  isRepositoryName,
  readDefaultRepository,
  saveDefaultRepository,
} from "@/lib/preferences"

export function Settings() {
  const [repository, setRepository] = useState(readDefaultRepository)
  return (
    <div className="mx-auto w-full max-w-3xl space-y-10 py-2 md:py-6">
      <header>
        <h1 className="text-4xl leading-none font-normal tracking-[-0.04em] md:text-5xl">
          Settings
        </h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Make this workspace easier to return to.
        </p>
      </header>
      <section className="border-t pt-6">
        <h2 className="text-2xl font-normal tracking-tight">This browser</h2>
        <p className="mt-2 text-sm leading-6 text-muted-foreground">
          Preferences are saved on this device and are not synced to your GitHub
          account.
        </p>
        <form
          className="mt-6 space-y-4"
          onSubmit={(event) => {
            event.preventDefault()
            const value = repository.trim()
            if (value && !isRepositoryName(value)) {
              toast.error("Use the format owner/repository.")
              return
            }
            try {
              saveDefaultRepository(value)
              setRepository(value)
              toast.success("Preferences saved.")
            } catch {
              toast.error(
                "Could not save preferences. Browser storage may be unavailable.",
              )
            }
          }}
        >
          <div className="max-w-md space-y-2">
            <label htmlFor="default-repository" className="text-sm font-medium">
              Default repository
            </label>
            <Input
              id="default-repository"
              value={repository}
              onChange={(event) => setRepository(event.target.value)}
              placeholder="owner/repository"
              maxLength={140}
              aria-describedby="default-hint"
            />
            <p
              id="default-hint"
              className="text-sm leading-6 text-muted-foreground"
            >
              Prefills AI chat and project questions. Leave blank to start with
              an empty field.
            </p>
          </div>
          <Button type="submit">Save preferences</Button>
        </form>
      </section>
      <section className="border-t pt-6">
        <h2 className="text-2xl font-normal tracking-tight">Connections</h2>
        <dl className="mt-4 divide-y text-sm">
          <div className="flex flex-wrap justify-between gap-3 py-4">
            <dt>GitHub identity</dt>
            <dd>
              <Link to="/account" className="underline underline-offset-4">
                Manage in Account
              </Link>
            </dd>
          </div>
          <div className="flex flex-wrap justify-between gap-3 py-4">
            <dt>GitHub repository data</dt>
            <dd className="text-muted-foreground">Not connected</dd>
          </div>
          <div className="flex flex-wrap justify-between gap-3 py-4">
            <dt>SonarQube Cloud</dt>
            <dd className="text-muted-foreground">Not connected</dd>
          </div>
        </dl>
      </section>
    </div>
  )
}
