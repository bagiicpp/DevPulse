import { ArrowDown, ArrowUpRight, GitBranch } from "lucide-react"
import { Link } from "react-router"
import { Button } from "@/components/ui/button"
import type { AuthState } from "@/hooks/use-session"

export function Home({ auth }: { auth: AuthState }) {
  return (
    <main className="landing">
      <header className="landing-nav">
        <Link
          to="/"
          className="font-serif text-2xl font-medium italic tracking-tight"
          aria-label="DevPulse home"
        >
          DevPulse
        </Link>
        <nav
          aria-label="Main navigation"
          className="flex items-center gap-4 sm:gap-8"
        >
          <a
            href="#how-it-works"
            className="hidden text-sm text-muted-foreground transition-colors hover:text-foreground sm:block"
          >
            How it works
          </a>
          <Button
            variant="outline"
            render={<Link to={auth.user ? "/account" : "/login"} />}
            nativeButton={false}
          >
            {auth.user ? "Your account" : "Log in"}
          </Button>
        </nav>
      </header>
      <section className="landing-hero" aria-labelledby="home-heading">
        <div className="hero-copy">
          <h1 id="home-heading" className="hero-heading">
            <span>Room to see</span>
            <span>the bigger picture.</span>
          </h1>
          <p className="hero-description">
            A calmer way to think about project health.
            <br className="hidden sm:block" /> Bring your questions. Find your
            next step.
          </p>
          <Button
            size="lg"
            className="hero-action"
            render={
              auth.user ? (
                <Link to="/dashboard" />
              ) : (
                <a href="/api/auth/github" />
              )
            }
            nativeButton={false}
          >
            {auth.user ? "Open workspace" : "Sign up with GitHub"}
            <ArrowUpRight aria-hidden="true" />
          </Button>
          <p className="mt-4 text-xs text-muted-foreground">
            Your GitHub identity. Your project context.
          </p>
        </div>
        <div className="hero-notes" aria-hidden="true">
          <div className="project-note note-repository">
            <GitBranch className="mb-6 size-5" />
            <p className="text-xs text-muted-foreground">A place to start</p>
            <p className="mt-2 font-mono text-sm">owner / project</p>
            <div className="mt-6 space-y-2">
              <span className="block h-px w-full bg-border" />
              <span className="block h-px w-3/4 bg-border" />
            </div>
          </div>
          <div className="project-note note-question">
            <p className="mb-5 text-xs text-muted-foreground">
              A better question
            </p>
            <p className="text-xl leading-tight tracking-tight">
              What would make our CI more reliable?
            </p>
            <ArrowUpRight className="mt-5 ml-auto size-4" />
          </div>
        </div>
        <div className="landing-wordmark" aria-hidden="true">
          DevPulse
        </div>
        <a
          href="#how-it-works"
          className="hero-scroll"
          aria-label="Learn how DevPulse works"
        >
          <ArrowDown className="size-4" aria-hidden="true" />
        </a>
      </section>
      <section
        id="how-it-works"
        className="landing-about"
        aria-labelledby="about-heading"
      >
        <h2 id="about-heading">
          From a question
          <br />
          to a way forward.
        </h2>
        <div className="max-w-md space-y-5">
          <p className="leading-7">
            Give DevPulse a repository name and the context that matters. Get an
            AI-generated summary, findings, and practical recommendations to
            help you think through what comes next.
          </p>
          <p className="text-sm leading-6 text-muted-foreground">
            DevPulse retrieves file names and README content from GitHub to
            inform its guidance. Source code and live CI metrics are not analyzed.
          </p>
          <Link
            to="/chat"
            className="inline-flex min-h-11 items-center gap-2 text-sm underline underline-offset-4"
          >
            Try a question as a guest{" "}
            <ArrowUpRight className="size-4" aria-hidden="true" />
          </Link>
        </div>
      </section>
      <footer className="landing-footer">
        <span>Software-project health, thoughtfully.</span>
        <span>A university project.</span>
      </footer>
    </main>
  )
}
