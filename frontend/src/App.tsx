import { Link, Outlet, Route, Routes } from "react-router"

import { AppSidebar } from "@/components/app-sidebar"
import { Dashboard } from "@/components/dashboard"
import { AIChat } from "@/components/ai-chat"
import { Home } from "@/components/home"
import { Login } from "@/components/login"
import { Account } from "@/components/account"
import { Settings } from "@/components/settings"
import { Repositories } from "@/components/repositories"
import { ProjectAnalysis } from "@/components/project-analysis"
import { Button } from "@/components/ui/button"
import { useSession, type AuthState } from "@/hooks/use-session"

import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"

import { Toaster } from "@/components/ui/sonner"

function AppLayout({ auth }: { auth: AuthState }) {
  return (
    <SidebarProvider className="flex-col md:flex-row">
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:fixed focus:top-2 focus:left-2 focus:z-50 focus:rounded-md focus:bg-background focus:p-3"
      >
        Skip to content
      </a>
      <AppSidebar auth={auth} />
      <SidebarInset>
        <main
          id="main-content"
          className="flex min-h-0 flex-1 flex-col px-6 py-10 md:px-10 md:py-12"
        >
          <Outlet />
        </main>
      </SidebarInset>
    </SidebarProvider>
  )
}

function App() {
  const auth = useSession()
  return (
    <>
      <Routes>
        <Route path="/" element={<Home auth={auth} />} />
        <Route path="/login" element={<Login auth={auth} />} />
        <Route element={<AppLayout auth={auth} />}>
          <Route path="/dashboard" element={<Dashboard auth={auth} />} />
          <Route path="/repositories" element={<Repositories />} />
          <Route path="/analysis" element={<ProjectAnalysis />} />
          <Route path="/chat" element={<AIChat />} />
          <Route path="/settings" element={<Settings />} />
          <Route path="/account" element={<Account auth={auth} />} />
          <Route
            path="*"
            element={
              <div className="mx-auto w-full max-w-3xl space-y-4 py-12">
                <h1 className="text-4xl font-normal tracking-tight">
                  Page not found
                </h1>
                <p className="text-muted-foreground">
                  This page does not exist. Head back to your workspace.
                </p>
                <Button render={<Link to="/dashboard" />} nativeButton={false}>
                  Open dashboard
                </Button>
              </div>
            }
          />
        </Route>
      </Routes>
      <Toaster position="bottom-right" theme="light" closeButton />
    </>
  )
}

export default App
