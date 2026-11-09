import { useEffect } from "react"
import { ArrowLeft, GitBranch } from "lucide-react"
import { Link, Navigate, useSearchParams } from "react-router"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import type { AuthState } from "@/hooks/use-session"

const errors: Record<string, string> = {
  not_configured:
    "GitHub sign-in is not available right now. You can continue as a guest.",
  invalid_state:
    "Your sign-in attempt expired or could not be verified. Please start again.",
  denied: "GitHub sign-in was cancelled. Try again or continue as a guest.",
  failed: "We could not complete GitHub sign-in. Please try again.",
}

export function Login({ auth }: { auth: AuthState }) {
  const [params] = useSearchParams()
  const code = params.get("error") ?? ""
  const error = Object.hasOwn(errors, code) ? errors[code] : undefined

  useEffect(() => {
    if (error) toast.error(error, { id: "github-sign-in-error" })
  }, [error])

  if (auth.user) return <Navigate to="/dashboard" replace />

  return (
    <main className="flex min-h-svh flex-col bg-background p-6 md:p-10">
      <Link to="/" className="w-fit font-serif text-2xl font-medium italic tracking-tight">
        DevPulse
      </Link>
      <div className="mx-auto flex w-full max-w-md flex-1 flex-col justify-center gap-6 py-12">
        <Card>
          <CardContent className="space-y-6 p-6 md:p-8">
            <div>
              <h1 className="text-4xl leading-none font-normal tracking-[-0.04em] md:text-5xl">
                Welcome to DevPulse
              </h1>
              <p className="mt-3 text-sm leading-6 text-muted-foreground">
                Sign in with GitHub to select and analyze your public and private
                repositories.
              </p>
            </div>
            {error && (
              <p role="alert" className="text-sm leading-6">
                {error}
              </p>
            )}
            {auth.loading ? (
              <p role="status" className="text-sm text-muted-foreground">
                Checking sign-in availability…
              </p>
            ) : auth.error ? (
              <div className="space-y-3">
                <p role="alert" className="text-sm">
                  Could not reach the sign-in service.
                </p>
                <Button variant="outline" onClick={auth.retry}>
                  Try again
                </Button>
              </div>
            ) : auth.githubConfigured ? (
              <Button
                className="w-full"
                size="lg"
                render={<a href="/api/auth/github" />}
                nativeButton={false}
              >
                <GitBranch aria-hidden="true" /> Continue with GitHub
              </Button>
            ) : (
              <p
                role="status"
                className="text-sm leading-6 text-muted-foreground"
              >
                GitHub sign-in is not available right now. You can continue as a
                guest.
              </p>
            )}
            <p className="border-t pt-4 text-sm leading-6 text-muted-foreground">
              GitHub will ask for repository access. DevPulse reads file names
              and README content and sends them to the AI provider when you
              request analysis. Your GitHub token stays on the server.
            </p>
          </CardContent>
        </Card>
        <Button
          variant="ghost"
          render={<Link to="/dashboard" />}
          nativeButton={false}
        >
          <ArrowLeft aria-hidden="true" /> Continue as a guest
        </Button>
      </div>
    </main>
  )
}
