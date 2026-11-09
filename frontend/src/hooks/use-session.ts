import { useCallback, useEffect, useState } from "react"
import axios from "axios"
import { toast } from "sonner"

export type GitHubUser = { id: number; login: string; name: string }
type Session = { user: GitHubUser | null; githubConfigured: boolean }

export function useSession() {
  const [session, setSession] = useState<Session | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)
  const [signingOut, setSigningOut] = useState(false)

  const load = useCallback((signal?: AbortSignal) => {
    return axios
      .get<Session>("/api/auth/session", {
        signal,
        timeout: 10_000,
      })
      .then(({ data }) => {
        setSession(data)
        setError(false)
      })
      .catch((error: unknown) => {
        if (!axios.isCancel(error)) setError(true)
      })
      .finally(() => {
        if (!signal?.aborted) setLoading(false)
      })
  }, [])

  useEffect(() => {
    const controller = new AbortController()
    void load(controller.signal)
    return () => controller.abort()
  }, [load])

  async function signOut() {
    setSigningOut(true)
    try {
      await axios.post("/api/auth/logout", null, { timeout: 10_000 })
      setSession((current) => current && { ...current, user: null })
      toast.success("You have been signed out.")
      return true
    } catch {
      toast.error("Could not sign out. Please try again.")
      return false
    } finally {
      setSigningOut(false)
    }
  }

  return {
    user: session?.user ?? null,
    githubConfigured: session?.githubConfigured ?? false,
    loading,
    error,
    signingOut,
    signOut,
    retry: () => {
      setLoading(true)
      void load()
    },
  }
}

export type AuthState = ReturnType<typeof useSession>
