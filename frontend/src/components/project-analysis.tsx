import { useState } from "react"
import { useNavigate } from "react-router"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { isRepositoryName, readDefaultRepository } from "@/lib/preferences"

const topics = [
  {
    label: "Overall project health",
    question:
      "What evidence should I collect to assess this project's overall health?",
  },
  {
    label: "CI reliability",
    question:
      "What should I measure and investigate to improve this project's CI reliability?",
  },
  {
    label: "Code quality",
    question:
      "What should I review to understand this project's code quality and test coverage?",
  },
  {
    label: "Pull request workflow",
    question:
      "What project-level signals help identify bottlenecks in the pull request workflow?",
  },
]

export function ProjectAnalysis() {
  const [repository, setRepository] = useState(readDefaultRepository)
  const [question, setQuestion] = useState(topics[0].question)
  const navigate = useNavigate()
  return (
    <div className="mx-auto w-full max-w-3xl space-y-8 py-2 md:py-6">
      <header>
        <h1 className="text-4xl leading-none font-normal tracking-[-0.04em] md:text-5xl">
          Project Analysis
        </h1>
        <p className="mt-2 text-sm leading-6 text-muted-foreground">
          Start with a useful question, then add what you know about your
          project.
        </p>
      </header>
      <p className="rounded-xl border bg-muted/30 p-4 text-sm leading-6 text-muted-foreground">
        Live analysis is not available yet. This form prepares a question for AI
        chat; it does not scan your repository or calculate a health score.
      </p>
      <form
        noValidate
        className="space-y-6"
        onSubmit={(event) => {
          event.preventDefault()
          const value = repository.trim()
          if (!isRepositoryName(value)) {
            toast.error("Use the format owner/repository.")
            return
          }
          navigate(
            `/chat?${new URLSearchParams({ repository: value, question })}`,
          )
        }}
      >
        <div className="space-y-2">
          <label htmlFor="analysis-repository" className="text-sm font-medium">
            Repository name
          </label>
          <Input
            id="analysis-repository"
            value={repository}
            onChange={(event) => setRepository(event.target.value)}
            placeholder="owner/repository"
            required
            maxLength={140}
          />
        </div>
        <div className="space-y-2">
          <label htmlFor="analysis-topic" className="text-sm font-medium">
            What would you like to understand?
          </label>
          <select
            id="analysis-topic"
            value={question}
            onChange={(event) => setQuestion(event.target.value)}
            className="h-11 w-full rounded-full border border-input bg-card px-4 text-base outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 md:text-sm"
          >
            {topics.map((topic) => (
              <option key={topic.label} value={topic.question}>
                {topic.label}
              </option>
            ))}
          </select>
        </div>
        <div className="border-t pt-5">
          <h2 className="text-sm font-medium">Your starting question</h2>
          <p className="mt-2 text-sm leading-6 text-muted-foreground">
            {question}
          </p>
        </div>
        <Button type="submit">Continue to AI chat</Button>
      </form>
    </div>
  )
}
