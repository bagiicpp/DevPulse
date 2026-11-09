export function isRepositoryName(value: string) {
  return /^[a-zA-Z0-9][a-zA-Z0-9-]{0,38}\/[a-zA-Z0-9_.-]{1,100}$/.test(value)
}

export function readDefaultRepository() {
  try {
    const value = localStorage.getItem("devpulse.defaultRepository") ?? ""
    return isRepositoryName(value) ? value : ""
  } catch {
    return ""
  }
}

export function saveDefaultRepository(value: string) {
  if (value && !isRepositoryName(value))
    throw new Error("Invalid repository name")
  localStorage.setItem("devpulse.defaultRepository", value)
}
