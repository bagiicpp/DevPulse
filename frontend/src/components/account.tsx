import { ExternalLink, GitBranch, LogOut } from "lucide-react"
import { Link, useNavigate } from "react-router"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import type { AuthState } from "@/hooks/use-session"

export function Account({ auth }: { auth: AuthState }) {
  const navigate = useNavigate()
  return (
    <div className="mx-auto w-full max-w-3xl space-y-8 py-2 md:py-6">
      <header>
        <h1 className="text-4xl leading-none font-normal tracking-[-0.04em] md:text-5xl">
          Account
        </h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Your GitHub identity and current DevPulse session.
        </p>
      </header>
      {auth.loading ? (
        <p role="status">Loading your account…</p>
      ) : auth.error ? (
        <div className="space-y-4">
          <p role="alert">Could not load your account. Please try again.</p>
          <Button onClick={auth.retry}>Try again</Button>
        </div>
      ) : auth.user ? (
        <>
          <Card>
            <CardContent className="space-y-6 p-6">
              <div className="flex items-center gap-4">
                <img
                  src={`https://avatars.githubusercontent.com/u/${auth.user.id}?s=96`}
                  alt=""
                  width={48}
                  height={48}
                  className="size-12 rounded-full"
                />
                <div className="min-w-0">
                  <h2 className="truncate text-lg font-medium">
                    {auth.user.name || auth.user.login}
                  </h2>
                  <p className="text-sm text-muted-foreground">
                    @{auth.user.login}
                  </p>
                </div>
              </div>
              <dl className="divide-y border-y text-sm">
                <div className="flex justify-between gap-4 py-4">
                  <dt className="text-muted-foreground">Sign-in provider</dt>
                  <dd>GitHub</dd>
                </div>
                <div className="flex justify-between gap-4 py-4">
                  <dt className="text-muted-foreground">Session</dt>
                  <dd>Signed in</dd>
                </div>
                <div className="flex justify-between gap-4 py-4">
                  <dt className="text-muted-foreground">Repository access</dt>
                  <dd>Public and private repositories</dd>
                </div>
              </dl>
              <div className="flex flex-wrap gap-3">
                <Button
                  variant="outline"
                  render={
                    <a
                      href={`https://github.com/${auth.user.login}`}
                      target="_blank"
                      rel="noopener noreferrer"
                    />
                  }
                  nativeButton={false}
                >
                  GitHub profile <ExternalLink aria-hidden="true" />
                </Button>
                <Button
                  variant="outline"
                  disabled={auth.signingOut}
                  onClick={async () => {
                    if (await auth.signOut()) navigate("/")
                  }}
                >
                  <LogOut aria-hidden="true" />
                  {auth.signingOut ? "Signing out…" : "Sign out"}
                </Button>
              </div>
            </CardContent>
          </Card>
          <p className="text-sm leading-6 text-muted-foreground">
            Sessions last up to eight hours and end when the server restarts.
            Signing out ends this DevPulse session; it does not revoke the app
            in your GitHub settings.
          </p>
        </>
      ) : (
        <Card>
          <CardContent className="space-y-4 p-6">
            <GitBranch
              className="size-6 text-muted-foreground"
              aria-hidden="true"
            />
            <h2 className="text-lg font-medium">You are browsing as a guest</h2>
            <p className="max-w-lg text-sm leading-6 text-muted-foreground">
              Sign in with GitHub to see your profile here. You can use AI chat
              without an account.
            </p>
            <Button render={<Link to="/login" />} nativeButton={false}>
              Sign in with GitHub
            </Button>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
