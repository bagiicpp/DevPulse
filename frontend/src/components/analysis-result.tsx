import type { AIResponse } from "@/lib/analysis"

export function AnalysisResult({ response }: { response: AIResponse }) {
  return (
    <div className="space-y-4 text-sm leading-6 wrap-anywhere">
      <p>{response.summary}</p>
      {([
        ["Findings", response.findings],
        ["Recommendations", response.recommendations],
      ] as const).map(([title, items]) => items.length > 0 && (
        <div key={title}>
          <h3 className="mb-2 font-semibold">{title}</h3>
          <ul className="list-disc space-y-1 pl-5">
            {items.map((item, index) => <li key={index}>{item}</li>)}
          </ul>
        </div>
      ))}
    </div>
  )
}
