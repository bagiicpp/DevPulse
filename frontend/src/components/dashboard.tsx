import { ArrowRight, FolderGit2, MessageSquare } from "lucide-react"
import { Link } from "react-router"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import type { AuthState } from "@/hooks/use-session"

export function Dashboard({ auth }: { auth: AuthState }) {
  return (
    <div className="mx-auto w-full max-w-5xl space-y-10 py-2 md:py-6">
      <header className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-4xl leading-none font-normal tracking-[-0.04em] md:text-5xl">
            Dashboard
          </h1>
          <p className="mt-2 text-sm leading-6 text-muted-foreground">
            {auth.user
              ? `Welcome, ${auth.user.name || auth.user.login}. Here is your workspace.`
              : "Your starting point for understanding project health."}
          </p>
        </div>
        <Button render={<Link to="/chat" />} nativeButton={false}>
          <MessageSquare aria-hidden="true" /> Ask AI
        </Button>
      </header>

      <section>
        <div className="mb-4 flex items-center justify-between gap-4">
          <h2 className="text-2xl font-normal tracking-tight">
            Project overview
          </h2>
          <Link
            to="/repositories"
            className="text-sm underline-offset-4 hover:underline"
          >
            Repositories
          </Link>
        </div>
        <Card>
          <CardContent className="flex flex-col items-start gap-4 p-6 md:p-8">
            <FolderGit2
              className="size-6 text-muted-foreground"
              aria-hidden="true"
            />
            <div>
              <h3 className="text-base font-medium">
                No repository data connected
              </h3>
              <p className="mt-2 max-w-xl text-sm leading-6 text-muted-foreground">
                Project health metrics will appear when repository integrations
                are available. For now, bring your own context to AI chat or
                choose a question in Project Analysis.
              </p>
            </div>
            <Button
              variant="outline"
              render={<Link to="/analysis" />}
              nativeButton={false}
            >
              Explore project questions <ArrowRight aria-hidden="true" />
            </Button>
          </CardContent>
        </Card>
      </section>

      <section>
        <h2 className="text-2xl font-normal tracking-tight">
          Workspace status
        </h2>
        <dl className="mt-4 divide-y border-y text-sm">
          <div className="flex flex-wrap justify-between gap-2 py-4">
            <dt className="text-muted-foreground">GitHub sign-in</dt>
            <dd>
              {auth.loading
                ? "Checking session…"
                : auth.error
                  ? "Could not check session"
                  : auth.user
                    ? `Signed in as @${auth.user.login}`
                    : "Browsing as a guest"}
            </dd>
          </div>
          <div className="flex flex-wrap justify-between gap-2 py-4">
            <dt className="text-muted-foreground">AI responses</dt>
            <dd>Generated from your supplied context</dd>
          </div>
          <div className="flex flex-wrap justify-between gap-2 py-4">
            <dt className="text-muted-foreground">Live repository metrics</dt>
            <dd>Not connected</dd>
          </div>
        </dl>
        {!auth.user && !auth.loading && !auth.error && (
          <p className="mt-4 text-sm text-muted-foreground">
            <Link
              to="/login"
              className="text-foreground underline underline-offset-4"
            >
              Sign in with GitHub
            </Link>{" "}
            to view your account. AI chat is also available to guests.
          </p>
        )}
      </section>
    </div>
  )
}
