import { Route, Routes } from "react-router"

import { AppSidebar } from "@/components/app-sidebar"
import { Dashboard } from "@/components/dashboard"
import { AIChat } from "@/components/ai-chat"

import {
  SidebarInset,
  SidebarProvider,
} from "@/components/ui/sidebar"

import { Toaster } from "@/components/ui/sonner"

function Placeholder({ title }: { title: string }) {
  return (
    <h1 className="font-serif text-3xl font-semibold">
      {title}
    </h1>
  )
}

function App() {
  return (
    <SidebarProvider>
      <AppSidebar />

      <SidebarInset>
        <main className="flex min-h-0 flex-1 flex-col p-6">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route
              path="/repositories"
              element={<Placeholder title="Repositories" />}
            />
            <Route
              path="/analysis"
              element={<Placeholder title="Project Analysis" />}
            />
            <Route path="/chat" element={<AIChat />} />
            <Route
              path="/settings"
              element={<Placeholder title="Settings" />}
            />
          </Routes>
        </main>
      </SidebarInset>

      <Toaster position="bottom-right" theme="light" />
    </SidebarProvider>
  )
}

export default App

