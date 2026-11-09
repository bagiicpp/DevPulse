import axios from "axios"

export type AIResponse = {
  summary: string
  findings: string[]
  recommendations: string[]
}

export function requestError(error: unknown) {
  if (axios.isAxiosError(error) && error.response) {
    if (typeof error.response.data === "string" && error.response.data.trim())
      return error.response.data.trim()
    return "The server could not complete your request. Please try again."
  }
  return "Could not reach DevPulse. Check your connection and try again."
}
