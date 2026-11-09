# AGENTS.md — DevPulse

## Purpose and scope

DevPulse is a university project assessing **software-project health**, not individual developer productivity. Keep the implementation small and understandable for coursework. Prioritize the user's current request and assignment over the roadmap.

GitHub activity/CI data and SonarQube Cloud quality metrics are future integrations. Add integrations, services, scoring, authentication, deployment infrastructure, or vulnerability scanning only when requested. Developer ranking and employee productivity measurement are outside the project scope.

Historical coursework in `n0/`, `n1/`, `n2/`, and `n3/` stays untouched unless explicitly requested.

## Working agreement

1. Read the affected files before changing them. Use the actual repository as the source of truth for implemented behavior; this file records conventions and constraints. Ask about conflicts only when they block the requested change.
2. State the smallest required change and affected files. Preserve working code, configuration, and UI choices; use existing patterns and dependencies. Architecture or framework changes require a user request.
3. When asked to implement, edit the minimum required files. When asked for an explanation or example, provide code without editing files. Complete the requested change, including affected callers and types, before stopping; leave subsequent features to the user.
4. Validate external input and handle errors at API boundaries. Keep secrets and LLM calls on the Go server. Keep `backend/.env` out of version control and Docker build contexts; use placeholders when documenting credentials.
5. Run checks relevant to the change using the repository's scripts and configuration. Report files changed, checks run, and anything unverified. Past success and configuration alone are not evidence of current runtime behavior.

## Frontend conventions

- Preserve the React/TypeScript/Vite frontend, Tailwind v4, and Neutral `base-nova` shadcn components. Reuse established components before introducing custom equivalents.
- Use Axios with relative `/api` paths, React Router for navigation, and Sonner for notifications.
- When composing shadcn controls, inspect their generated APIs: `base-nova` does not imply Radix-style `asChild` support. Keep interactive elements from nesting inside other interactive elements.
- Before generating shadcn components, verify TypeScript and Vite resolve `@/*` to `src/*`. Generated UI files belong in `frontend/src/components/ui/`; a previous alias error created a literal `frontend/@/` directory.

## UI choices to preserve

- Minimalist, light/Neutral styling with Noto Serif branding.
- Text-only sidebar navigation: Dashboard, Repositories, Project Analysis, Chat with AI, Settings.
- Italic “DevPulse” branding without a logo. Keep the collapse/expand control inside the sidebar header as its only icon.
- Keep Dashboard in its own component; unimplemented routes may remain placeholders.

## Current assignment: dynamic LLM integration

Preserve the assignment flow: collect repository and question input → build a dynamic Gemini prompt → obtain a real JSON response → deserialize it into a Go struct → display summary, findings, and recommendations in React.

When changing chat behavior, read `backend/main.go`, `backend/llm.go`, and `frontend/src/components/ai-chat.tsx`. Keep request and response types aligned. Use Go's standard `net/http` and `encoding/json` for the API.

The repository name provides prompt context only; the chat does not retrieve live GitHub data. Keep the prompt explicit about this limitation and describe responses without claiming repository inspection. Add GitHub retrieval only when requested.

## Local development

When changing startup, proxying, or containers, read `docker-compose.yml`, both Dockerfiles, and `frontend/vite.config.ts` together. Keep browser requests relative to `/api`; the frontend container reaches the backend through `http://backend:8080`. Read current ports and commands from configuration rather than assuming defaults.

For local backend runs, load `backend/.env` through godotenv and read `GEMINI_API_KEY` on the server. Run Go from `backend/` so the existing relative `.env` lookup resolves correctly.
