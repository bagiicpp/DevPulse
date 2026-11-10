# DevPulse

DevPulse is a university project assessing **software-project health**. Its long-term goal is to combine development activity from GitHub with code-quality metrics from SonarQube Cloud into an explainable assessment.

This branch evaluates **Qlty CLI for on-demand code quality**: select a GitHub repository, capture its default-branch commit, scan supported source files on the backend, and inspect measured complexity and structural/duplication findings. No Qlty Cloud account, SonarQube project import, or AI key is needed for this scan. The separate chat retains the dynamic LLM assignment flow.

## Current Application

- React, TypeScript, and Vite frontend with Tailwind v4 and Neutral shadcn components.
- A Go HTTP server exposing `POST /api/chat` and the Qlty trial endpoint `POST /api/analysis`.
- A dynamic prompt and JSON response schema sent to the configured AI provider. The server deserializes and validates the response before returning it to React.
- Loading feedback, Sonner notifications for errors and confirmations, and preserved input for retry after failure.
- Lucide sidebar icons, accessible labels, and a header toggle on desktop and mobile.
- A standalone public homepage at `/`, with GitHub sign-up leading into the app at `/dashboard`.
- Warm paper colors, black pill controls, and spacious layouts following `DESIGN.md`, inspired by [Cosmos](https://www.cosmos.so/). The existing Geist font carries interface text; Noto Serif is reserved for DevPulse branding.
- Real GitHub OAuth sign-in, an Account page showing the verified profile, and sign-out.
- Settings with a browser-local default repository. Repository selection includes public and private repositories accessible through GitHub sign-in.
- Project Analysis runs pinned Qlty CLI 0.668.0, shows file metrics and source-linked code smells, supports filtering, cancellation/retry, and discloses scan exclusions and limits.

**Project Analysis downloads actual source at a fixed commit and runs Qlty locally in the backend container.** The trial uses built-in metrics and code-smell analysis only. It does not run repository build scripts, tests, linter plugins, or dependency installers. It does not measure coverage, issue a quality-gate verdict, or infer overall project health from a single score. Source is not sent to an AI or cloud analysis service.

Chat retrieves file paths and README text from the default branch and injects them into each AI prompt. Chat does not receive scan results automatically or retrieve source-file contents, CI results, or quality metrics. Repository content is treated as untrusted data. Each request is independent; earlier messages are displayed locally but are not sent to the provider. Messages and reports are lost when leaving their page or refreshing it.

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
| `/analysis` | Run the Qlty trial and inspect measured file complexity, code smells, duplication findings, and source links |
| `/chat` | Submit repository context and a question to the configured AI provider |

The default repository stays in this browser's local storage, including after sign-out. It is not synced to an account. Query parameters supplied by Repositories or Project Analysis take precedence over this default; preparing a question does not submit it automatically.

The public homepage has no app sidebar. Its sign-up action starts the existing GitHub OAuth flow directly; after authorization, the server redirects to Dashboard. Signed-in visitors see **Open workspace** instead. The homepage uses a short CSS entrance sequence with stationary content for reduced-motion preferences. Guest access remains available through login and the homepage's AI chat link.

## Run with Docker

For a new setup, create `backend/.env` from the supplied template and configure GitHub sign-in below. AI credentials are needed only for chat. If `.env` already exists, add missing values without overwriting it:

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

Open **http://localhost:5173/analysis**. The backend image installs checksum-verified Qlty CLI 0.668.0 for Linux amd64/arm64 automatically; you do not run CLI commands for each repository. The backend runs as a non-root user with a 1 GiB memory limit, two CPUs, and a 128-process limit. The backend is available at `http://localhost:8080`. Browser requests use relative `/api` paths; the frontend container proxies them to `http://backend:8080`.

Vite uses port `5173` and exits if it is occupied. Free that port before starting the app. Frontend source changes reload automatically; rebuild the backend after changing Go code.

To stop the containers:

```bash
docker compose down
```

## Run without Docker

Use Node.js 24 with npm and Go 1.26.8 or later. Configure `backend/.env` as above.

For the quality trial outside Docker, install Git and the [Qlty CLI 0.668.0 release](https://github.com/qltysh/qlty/releases/tag/v0.668.0). Keep `qlty` on `PATH`, or set `QLTY_BIN=/absolute/path/to/qlty` in `backend/.env`. The server checks the exact CLI version because this trial consumes Qlty's experimental JSON metric format. Docker is the recommended way to test the integration.

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

Open `http://localhost:5173/analysis`. Vite proxies `/api` to `http://localhost:8080` for local development.

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

## Qlty trial API

`POST /api/analysis` accepts `{"repository":"owner/repository"}` (maximum 4 KiB). It verifies GitHub access with the existing server-side session, resolves the default branch to a commit SHA, downloads that commit's archive, extracts eligible source files into a temporary workspace, and runs Qlty. It scans GitHub's committed snapshot, not your local working tree. Public repositories work as a guest within GitHub's unauthenticated limits; sign in to avoid those lower limits and access private repositories.

The response is a typed `QualityReport` containing the repository/branch/SHA, Qlty version, completion time, elapsed download/scan time, files supplied/excluded, per-file measurements, normalized findings with commit-pinned source links, and limitations. Findings include all related source locations, so duplication can be inspected on both sides. Totals sum file measurements only; Qlty's directory rollups are excluded to avoid double-counting.

The backend runs these built-in commands using its own minimal `.qlty/qlty.toml`:

```text
qlty metrics --all --json --quiet --no-upgrade-check
qlty smells --all --include-tests --sarif --no-snippets --quiet --no-upgrade-check
```

Qlty's metrics JSON option is experimental, so this trial pins CLI 0.668.0 and verifies its version. Metrics include code lines, functions, Qlty complexity, and cyclomatic complexity. The UI ranks files by Qlty complexity and lets you filter code smells by the actual rule IDs. These are static indicators, not a health score or proof of correctness. Rule support varies by language; unsupported/ignored/unparseable files are omitted from metric totals and the supplied/analyzed file counts remain visible. Tests are included as source, but are not executed. Coverage and quality-gate status are not measured.

The scanner receives no backend credentials. Telemetry and update checks are disabled. Repository Qlty configuration, ignore files, Git metadata, symlinks/hardlinks, secret-like paths, generated/dependency directories, unsupported formats, and minified JavaScript are not used as scan input. No project builds, scripts, dependencies, linter plugins, or tests are run. The temporary snapshot is removed when the request finishes or is canceled. The backend's Git initialization uses no templates, global configuration, or system configuration. This is a bounded local trial, not a general-purpose hosted execution sandbox.

Limits are 32 MiB compressed archive, 128 MiB expanded archive, 20,000 archive entries, 512 KiB per source file, 2,000 source files, and 24 MiB of retained source. Reaching the retained-source/file-count limits produces an explicitly partial report. Oversized archives are rejected. Scanner stdout is capped at 4 MiB per command; stderr at 64 KiB. All findings are validated, but at most 200 are displayed; totals and truncation remain visible. No duplication percentage is inferred from finding counts because overlapping duplicated regions can otherwise be counted twice.

One scan runs at a time; another receives `429` with a retry message. The full request has a two-minute deadline, Qlty execution has a 60-second deadline, archive HTTP requests have 30-second timeouts, and the browser allows 150 seconds. Cancellation aborts the request and scanner. Input/origin errors use `400`/`403`, missing/inaccessible GitHub repositories `404`, missing scanner or GitHub rate limits `503`, unsupported/empty source sets `422`, retrieval/scanner/validation failures `502`, and scanner deadline expiry `504`. Errors preserve the repository selection for retry and do not fall back to AI-generated findings.

### Evaluation and licensing

This branch is `trial/qlty-quality`, based on pre-SonarQube `main` commit `deafb16`. The earlier SonarQube implementation and existing work were preserved in the named Git stash `pre-qlty: SonarQube implementation and existing WIP` (`15c41e02f9a6fb8726161f680d6029fbf6fc343d`). No adoption decision, commit, merge, or push is implied by this trial.

Qlty CLI uses the [Business Source License 1.1 with an additional use grant](https://github.com/qltysh/qlty/blob/v0.668.0/LICENSE.md). Non-production evaluation is permitted, but the grant restricts using it for a commercial code-quality service or AI coding service. Confirm licensing before turning this experiment into a commercial hosted service. The container includes the CLI's license at `/opt/qlty/LICENSE.md`.

## Checks and Demo

From `backend/`:

```bash
go test ./...
go vet ./...
```

Tests mock GitHub and both AI providers’ HTTP transport and need no real key or paid requests. They cover input validation, private repository selection and pagination, recursive file paths and README injection, missing/oversized context, retrieval failures, the dynamic prompt and response schema, structured output, malformed provider responses, timeouts, and provider failures.

OAuth tests also mock GitHub and cover PKCE, state mismatch and replay, denial, provider failures, identity validation, session rotation/expiry, cookie flags, and cross-site logout rejection. They need no GitHub credentials.

Quality tests cover archive traversal/duplicate paths, links/config/secret exclusions, GitHub's PAX metadata, file-count limits, archive redirects and token isolation, metric rollups, zero findings, invalid source references, exclusive SARIF end positions, related duplication locations, subprocess credential isolation, output limits, and cancellation. Set `QLTY_TEST_BIN=/absolute/path/to/qlty` when running `go test` to include the real CLI/HTTP-handler integration check; this mocks GitHub and requires no network or AI key. The optional `TestQualityLiveRepositories` additionally requires explicitly supplied `QLTY_LIVE_REPOSITORIES` and `QLTY_LIVE_GITHUB_TOKEN`; do not put tokens in command history, logs, or Git.

From `frontend/`:

```bash
npm test
npm run lint
npm run build
```

Frontend tests use Node.js's built-in test runner and cover repository-name validation, saved preferences, report validation, missing/zero values, rule grouping, and commit-pinned source links with escaped filenames.

To test the quality trial, open `/analysis`, sign in if needed, enter a repository, and choose **Scan with Qlty**. Inspect a finding's source links, compare file metrics, try rule filters, and check the scope/exclusions. Test cancellation/retry, a repository without eligible source files, and a failed request. Analysis does not require either AI key or a SonarQube mapping.

Authenticated live checks on 2026-11-10 used the real Qlty 0.668.0 binary and GitHub snapshots, with both AI keys disabled: `bagiicpp/Risbo` at `1aec3f3` supplied 70 source files, returned metrics for 69 files / 12,111 code lines / 426 functions, and reported 60 findings. `bagiicpp/DevPulse` at `deafb16` returned metrics for 41 files / 4,175 code lines / 157 functions and reported 25 findings. Neither scan hit the retained-source/file-count limits. These are observations from those commits under this scan policy, not guarantees about later versions or independently verified bugs.

Desktop and mobile browser checks rendered those recorded live reports and verified source links for both duplication copies, rule filters, the file table, zero/partial/invalid reports, failure/retry, cancellation, result focus, and horizontal overflow. The real CLI integration also passed inside the non-root backend container. Browser response fixtures and authenticated live backend checks were run separately; a full browser OAuth-to-live-scan flow was not tested.

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
