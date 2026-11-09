# DevPulse

DevPulse is a university project assessing **software-project health**. Its long-term goal is to combine development activity from GitHub with code-quality metrics from SonarQube Cloud into an explainable assessment.

The current application demonstrates a dynamic LLM integration: enter a repository name and a question, send them to the Go server, and display a structured response containing a summary, findings, and recommendations.

## Current Application

- React, TypeScript, and Vite frontend with Tailwind v4 and Neutral shadcn components.
- A Go HTTP server exposing `POST /api/chat`.
- A dynamic prompt and JSON response schema sent to the configured AI provider. The server deserializes and validates the response before returning it to React.
- Loading feedback, Sonner notifications for errors and confirmations, and preserved input for retry after failure.
- Lucide sidebar icons, accessible labels, and a header toggle on desktop and mobile.
- A standalone public homepage at `/`, with GitHub sign-up leading into the app at `/dashboard`.
- Warm paper colors, black pill controls, and spacious layouts following `DESIGN.md`, inspired by [Cosmos](https://www.cosmos.so/). The existing Geist font carries interface text; Noto Serif is reserved for DevPulse branding.
- Real GitHub OAuth sign-in, an Account page showing the verified profile, and sign-out.
- Settings with a browser-local default repository. Repository selection includes public and private repositories accessible through GitHub sign-in; Project Analysis displays a real structured AI assessment.

**DevPulse retrieves file paths and README text from the repository’s default branch and injects them into each AI prompt.** Analysis covers structure and documentation; source-file contents, CI results, and quality metrics are not retrieved. Repository content is treated as untrusted data in the prompt. Missing READMEs, empty repositories, and truncated context are identified for the model. Each request is independent; earlier messages are displayed locally but are not sent to the configured AI provider. Messages are lost when leaving the chat page or refreshing it.

GitHub activity/CI integration, SonarQube Cloud metrics, and the original microservice architecture are future work. The current application uses one Go server. Developer ranking and employee productivity measurement are outside the project scope.

## Repository Structure

```text
.
├── backend/        # Go API, LLM integration, and tests
├── frontend/       # React/TypeScript application
├── n0/             # Project proposal and architecture diagram
├── n1/             # XML processing assignment
├── n2/             # Web scraping assignment
├── n3/             # PC-Axis data visualization
├── docker-compose.yml
├── AGENTS.md       # Scope and implementation conventions
└── README.md
```

## Pages

| Route | Purpose |
| --- | --- |
| `/` | Public landing page, GitHub sign-up, login, and a short product introduction |
| `/dashboard` | Workspace overview and current connection status |
| `/login` | GitHub sign-in, cancellation/error handling, and guest access |
| `/account` | Verified GitHub profile, session details, and sign-out |
| `/settings` | Save or clear the default repository for this browser |
| `/repositories` | Select a public/private repository or enter a public repository name |
| `/analysis` | Choose a topic, run analysis, and view summary, findings, and recommendations |
| `/chat` | Submit repository context and a question to the configured AI provider |

The default repository stays in this browser's local storage, including after sign-out. It is not synced to an account. Query parameters supplied by Repositories or Project Analysis take precedence over this default; preparing a question does not submit it automatically.

The public homepage has no app sidebar. Its sign-up action starts the existing GitHub OAuth flow directly; after authorization, the server redirects to Dashboard. Signed-in visitors see **Open workspace** instead. The homepage uses a short CSS entrance sequence with stationary content for reduced-motion preferences. Guest access remains available through login and the homepage's AI chat link.

## Run with Docker

For a new setup, create `backend/.env` from the supplied template and fill in the OpenRouter and GitHub credentials below. If `.env` already exists, add the missing values without overwriting it:

```bash
cp backend/.env.example backend/.env
```

```dotenv
LLM_PROVIDER=openrouter
OPENROUTER_API_KEY=your_openrouter_api_key
OPENROUTER_MODEL=nvidia/nemotron-3.5-super-vl:free
GITHUB_CLIENT_ID=your_github_client_id
GITHUB_CLIENT_SECRET=your_github_client_secret
APP_URL=http://localhost:5173
```

Keep the key on the server. `backend/.env` is excluded from Git and the backend Docker build context; Compose supplies it at runtime.

From the repository root:

```bash
docker compose up --build -d
```

Open **http://localhost:5173/chat**. The backend is available at `http://localhost:8080`. Browser requests use relative `/api` paths; the frontend container proxies them to `http://backend:8080`.

Vite uses port `5173` and exits if it is occupied. Free that port before starting the app. Frontend source changes reload automatically; rebuild the backend after changing Go code.

To stop the containers:

```bash
docker compose down
```

## Run without Docker

Use Node.js 24 with npm and Go 1.26.8 or later. Configure `backend/.env` as above.

In one terminal, run Go from `backend/` so godotenv finds `.env`:

```bash
cd backend
go run .
```

In a second terminal:

```bash
cd frontend
npm ci
npm run dev
```

Open `http://localhost:5173/chat`. Vite proxies `/api` to `http://localhost:8080` for local development.

## AI Provider

OpenRouter is the default provider. Add `OPENROUTER_API_KEY` to `backend/.env`; create the key in [OpenRouter settings](https://openrouter.ai/settings/keys). The default model is `nvidia/nemotron-3.5-super-vl:free`, configurable through `OPENROUTER_MODEL`. Keep the key on the server.

Use public or nonconfidential repositories for this free-model trial. [NVIDIA's free-endpoint notice](https://openrouter.ai/nvidia/nemotron-3.5-super-vl:free) says sessions are logged for security and product improvement and asks users not to submit confidential or personal information. Private repository content needs a model/provider with suitable data-handling terms.

OpenRouter requests use [strict JSON Schema output](https://openrouter.ai/docs/guides/features/structured-outputs) and `provider.require_parameters=true`, so requests only use endpoints supporting the required parameters. Free-model availability and limits can change; an unavailable endpoint returns an error. There is no automatic switch to a paid model or another AI provider.

OpenRouter generation disables optional reasoning and limits output to 2,048 tokens for concise responses within the existing request timeout. The server rejects incomplete or invalid results instead of displaying them.

Gemini remains available for the original assignment flow. To use it, set `LLM_PROVIDER=gemini` and configure `GEMINI_API_KEY`. Supported provider values are `openrouter` and `gemini`; a missing key or unknown provider is a configuration error.

After changing environment variables, restart a local Go server or run `docker compose up -d --build backend`. Restarting the backend clears in-memory GitHub sessions. Both providers receive the same file-name/README context and return the same validated summary, findings, and recommendations.

## GitHub Sign-In

Create a [GitHub OAuth app](https://github.com/settings/applications/new) with:

- **Homepage URL:** `http://localhost:5173`
- **Authorization callback URL:** `http://localhost:5173/api/auth/github/callback`
- **Wildcard matching:** off
- **Device flow:** off
- **Expire user access tokens:** on

Copy the app's client ID and generate a client secret. Put both in `backend/.env` as shown above; keep the existing AI provider key. See [GitHub's OAuth app registration guide](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/creating-an-oauth-app).

`APP_URL` defaults to `http://localhost:5173` and defines the callback origin. `PORT` is optional and defaults to `8080`; keep that default for the supplied Compose configuration. If environment variables change, recreate the backend with `docker compose up -d --force-recreate backend` (or restart a manually running Go server).

Open `http://localhost:5173/login` and choose **Continue with GitHub**. DevPulse uses GitHub's [web application flow](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps) with one-use state and PKCE. It requests the `repo` scope to read public and private repositories. This OAuth scope also grants write permissions; DevPulse only makes GET requests to the repository API. The access token stays in the server’s in-memory session and is never returned to the browser or included in AI prompts. Refresh tokens are not retained; when access expires, sign in again. **Existing users must sign out and sign in again to grant repository access.**

The browser receives an opaque, HttpOnly, SameSite=Lax session cookie. Sessions last up to eight hours (or until the GitHub token expires) and are kept in server memory, so restarting the backend signs users out. Cookies are Secure for HTTPS origins; HTTP is allowed only for local development. This implementation supports one backend instance and has no persistent account database. Browser preferences remain local to the device. AI chat remains available to guests.

Authentication endpoints:

- `GET /api/auth/session`: verified user or `null`, plus sign-in availability; responses are not cached.
- `GET /api/auth/github`: start sign-in.
- `GET /api/auth/github/callback`: validate state, exchange the code, verify identity, and return to Dashboard.
- `POST /api/auth/logout`: invalidate this session and clear its cookie. Requires an `Origin` header matching `APP_URL`; the browser supplies it automatically.

`GET /api/repositories?page=1` lists up to 100 accessible repositories per page, using GitHub’s [repository listing API](https://docs.github.com/en/rest/repos/repos#list-repositories-for-the-authenticated-user). It requires sign-in; the picker supports loading more pages. Private repositories may require organization approval. Guests can analyze public repositories using GitHub’s unauthenticated API limits. Signing out deletes the session and its token; it does not revoke the OAuth app’s authorization in GitHub.

## Chat API

The request requires an `owner/repository` name and a nonblank `question` string. The server trims both fields and limits the entire JSON body to **16 KiB**.

```bash
curl http://localhost:8080/api/chat \
  -H 'Content-Type: application/json' \
  -d '{"repository":"owner/repository","question":"What should I check to improve CI reliability?"}'
```

Example response shape (illustrative guidance):

```json
{
  "summary": "CI reliability needs build history to assess.",
  "findings": ["No live CI data was provided."],
  "recommendations": ["Review recent failed workflow runs for recurring causes."]
}
```

A successful response always has a nonblank `summary` and `findings` and `recommendations` arrays of nonblank strings. Arrays may be empty. AI requests time out after 30 seconds, and provider responses are limited to 1 MiB. GitHub requests time out after 10 seconds each. The browser request times out after 75 seconds to allow for repository retrieval and generation.

Context uses GitHub’s [recursive tree API](https://docs.github.com/en/rest/git/trees#get-a-tree) and [README API](https://docs.github.com/en/rest/repos/contents#get-a-repository-readme). Prompts include up to 1,000 file paths, at most 64 KiB of path text, and 64 KiB of README text. Truncation is passed to the model; it must state limitations. GitHub JSON responses are capped at 8 MiB. Retrieval failures stop generation rather than falling back to invented repository findings.

Errors return plain text: `400` for invalid JSON or missing input, `401` for expired GitHub access, `403` for an invalid request origin, `404` for inaccessible/missing repositories, `405` for unsupported methods, `413` for an oversized request, `503` for GitHub permission/rate-limit errors, and `502` for retrieval/provider failures, missing server credentials, or invalid AI output. Provider details stay on the server. Errors preserve the question so it can be edited and resent.

## Checks and Demo

From `backend/`:

```bash
go test ./...
```

Tests mock GitHub and both AI providers’ HTTP transport and need no real key or paid requests. They cover input validation, private repository selection and pagination, recursive file paths and README injection, missing/oversized context, retrieval failures, the dynamic prompt and response schema, structured output, malformed provider responses, timeouts, and provider failures.

OAuth tests also mock GitHub and cover PKCE, state mismatch and replay, denial, provider failures, identity validation, session rotation/expiry, cookie flags, and cross-site logout rejection. They need no GitHub credentials.

From `frontend/`:

```bash
npm test
npm run lint
npm run build
```

Frontend tests use Node.js's built-in test runner and cover repository-name validation and saved preferences, including blocked or invalid browser storage.

For a live assignment demo, open `/chat`, enter a repository name and a question, and send it. Confirm the page renders a real AI summary and any findings and recommendations. Use a real public repository such as `octocat/Hello-World` and confirm the response uses its file names and README. Sign in again with repository access to test private repositories through `/repositories` → `/analysis`. Test a failed request and verify the question remains available to retry. Live requests require a valid server-side key and provider access and may incur charges when using a paid model.

## Assignments

### N0 – Project Proposal

Contains the initial DevPulse project description and architecture diagram.

**Files:**

- `Blagoja Vasilev.docx`
- `DevPulseDiagram.png`

### N1 – XML Data Processing

A standalone Python application that demonstrates:

- Reading and deserializing XML documents
- Connecting projects, CI runs, and quality reports through IDs
- Interactive filtering by test coverage and CI status
- Exporting filtered results to JSON and XML

The application uses simulated data based on the planned GitHub and SonarQube Cloud integrations.

**Run the application:**

```bash
cd n1
python main.py
```

The program displays an interactive menu for filtering projects. Results are printed to the console and automatically exported to `filtrirano.json` and `filtrirano.xml`.

### N2 – Web Scraping

A standalone Python parser that extracts repository information from a public GitHub webpage using **Requests** and **BeautifulSoup**.

The parser extracts:

- Repository name and URL
- Repository description
- Star count
- Fork count

The extracted data is saved to `repository.json` for potential use in the final DevPulse application.

**Run the parser:**

```bash
cd n2
uv venv
uv pip install requests beautifulsoup4
uv run python parser.py
```

### N3 – Open Data Visualization (PC-Axis)

A standalone web application that visualizes official Slovenian ICT employment and training statistics using **PC-Axis data from the Statistical Office of Slovenia (SURS)**.

The application demonstrates:

- Dynamically loading and parsing PC-Axis (`.px`) data
- Decoding Windows-1250 encoded statistical files
- Filtering statistics by ICT indicator and cohesion region
- Visualizing data from 2022 and 2024 using **Chart.js**
- Calculating absolute and percentage changes between years
- Updating charts and calculations dynamically when filters change

The dataset is obtained from the [Slovenian Open Data Portal (OPSI)](https://podatki.gov.si/dataset/surs2971015s).

**Technologies:** HTML, CSS, JavaScript, Chart.js

**Run the application:**

```bash
cd n3
python -m http.server 8000
```

Open `http://localhost:8000` in your browser.

The application loads the local `data/ict.px` file on startup and displays interactive ICT statistics.

## Technologies

- **Go** – current HTTP API and LLM integration
- **React / TypeScript / Vite** – current application frontend
- **Tailwind CSS / shadcn** – current UI styling and components
- **OpenRouter / Gemini API** – structured AI guidance
- **Python** – XML processing assignment
- **GitHub REST API** – repository selection, file paths, and README retrieval; activity and CI/CD data remain planned
- **SonarQube Cloud API** – planned code-quality metrics
- **XML / JSON** – structured data formats
- **HTML / CSS / JavaScript** – interactive data visualization
- **Chart.js** – statistical charts
- **PC-Axis** – official statistical data format

## Status

DevPulse is currently being developed as a university project. Each assignment implements or demonstrates a specific part of the overall concept.
