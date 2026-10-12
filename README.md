# DevPulse

## Project overview

DevPulse is a school project for monitoring the health of software projects by combining data from existing development services.

The core idea is not to build another simple dashboard, but to combine and interpret data from multiple sources so that a user can quickly understand the overall state of a software project and why certain areas may need attention.

The system will be implemented as a set of microservices written in Go.

## Main goal

DevPulse should answer a simple question:

> How healthy is this software project, and why?

It will combine information about:

- development activity,
- pull requests,
- CI/CD reliability,
- test coverage,
- code quality,
- code duplication,
- detected code issues.

The system should provide an explainable project-health assessment instead of only displaying raw metrics.

## Main users

The main users are:

- developers, who want to quickly see problems affecting a repository,
- team or project leads, who want an overview of the state of a project.

DevPulse evaluates the health of the project itself. It is not intended to rank individual developers or measure employee productivity.

## External services

### GitHub

DevPulse will use the GitHub REST API to obtain information about software development activity.

Examples:

- repository information,
- commits,
- pull requests,
- branches,
- contributors,
- GitHub Actions workflow runs,
- CI/CD success or failure status.

Communication will use HTTPS and JSON.

### SonarQube Cloud

DevPulse will use the SonarQube Cloud API to obtain code-quality metrics.

Examples:

- Quality Gate status,
- test coverage,
- duplicated code,
- code-quality issues,
- maintainability-related metrics.

Communication will also use HTTPS and JSON.

## Core idea / innovation

DevPulse is not globally unique; similar engineering analytics products already exist.

The value of the project is the way it combines data from different development services into one explainable project-health model.

Instead of only showing separate metrics such as:

- 15 open pull requests,
- 78% test coverage,
- 20% CI failure rate,
- failed SonarQube Quality Gate,

DevPulse should interpret them together and produce findings such as:

> Development activity is high, but project health is declining because test coverage has dropped and CI failures have increased.

or:

> Code quality is stable, but several long-lived pull requests indicate a bottleneck in the review process.

The important part of the project is therefore cross-source analysis and interpretation.

## Planned architecture

The system will use Go microservices.

Initial structure:

```text
                     User
                       |
                       v
                 API Gateway
                       |
          +------------+------------+
          |                         |
          v                         v
   GitHub Service            Sonar Service
          |                         |
          v                         v
     GitHub API             SonarQube Cloud
          |                         |
          +------------+------------+
                       |
                       v
                 Health Service
                       |
                       v
              Project assessment
                       |
                       v
                  Dashboard
```

### API Gateway

Provides one entry point for the frontend and routes requests to the appropriate services.

### GitHub Service

Responsible for communication with GitHub and for converting GitHub data into DevPulse's internal format.

### Sonar Service

Responsible for communication with SonarQube Cloud and normalization of code-quality metrics.

### Health Service

Combines normalized information from the other services and calculates project-health indicators.

The exact scoring formula should remain flexible until the available API metrics are confirmed.

## Example internal data

A simplified internal project snapshot could look like this:

```json
{
  "repository": "owner/project",
  "developmentActivity": {
    "commits": 27,
    "openPullRequests": 4
  },
  "ci": {
    "successRate": 92
  },
  "quality": {
    "qualityGate": "OK",
    "coverage": 82.5,
    "duplicatedLinesDensity": 2.1,
    "issues": 14
  }
}
```

A health assessment could look like:

```json
{
  "repository": "owner/project",
  "healthScore": 84,
  "status": "healthy",
  "findings": [
    "CI reliability is high.",
    "Three pull requests have been open for more than seven days.",
    "Test coverage decreased compared with the previous analysis."
  ]
}
```

These formats are conceptual and may change during implementation.

## Scope

### In scope

- GitHub integration,
- GitHub Actions / CI data,
- SonarQube Cloud integration,
- normalization of data from multiple APIs,
- project-health analysis,
- explainable findings,
- dashboard or simple frontend,
- Go microservices,
- REST/JSON communication.

### Out of scope

DevPulse should not focus on:

- CVE aggregation,
- dependency vulnerability scanning,
- security vulnerability scoring,
- ranking individual developers,
- employee productivity monitoring.

The vulnerability-related features are intentionally excluded so the project remains focused on general project health and does not overlap with a classmate's vulnerability-analysis project.

## University form summary

### 1. Bistvo ideje in glavne sestavine

DevPulse je sistem za spremljanje zdravja programskih projektov, ki združuje podatke iz različnih razvojnih storitev. Prikazuje podatke o aktivnosti razvoja, pull requestih, CI/CD procesih in kakovosti kode. Rešitev bo zasnovana kot več mikrostoritev, implementiranih v jeziku Go.

### 2. Interakcija z obstoječimi storitvami

DevPulse bo pridobival podatke iz GitHub API-ja in SonarQube Cloud API-ja. GitHub bo zagotavljal podatke o repozitorijih, commitih, pull requestih in CI/CD procesih, SonarQube pa metrike kakovosti kode. Podatki se bodo prenašali preko REST API-jev predvsem v obliki JSON.

### 3. Inovativnost ideje

Inovativnost DevPulse je v združevanju in skupni interpretaciji podatkov iz različnih razvojnih storitev. Sistem ne prikazuje zgolj posameznih metrik iz GitHuba in SonarQube Cloud, temveč jih medsebojno poveže ter na njihovi podlagi oceni zdravje projekta in pojasni vzroke za zaznane težave. Uporabnik tako na enem mestu vidi, kako aktivnost razvoja, CI/CD procesi in kakovost kode vplivajo drug na drugega ter katera področja projekta zahtevajo pozornost.

### 4. Obstoječe rešitve in storitve

Uporabljena bosta GitHub REST API in SonarQube Cloud API. GitHub vrača podatke o projektih, commitih, pull requestih in workflowih, SonarQube pa metrike kakovosti kode, kot so pokritost s testi, Quality Gate in zaznani problemi. DevPulse bo podatke obeh storitev pretvoril v skupen interni format.

### 5. Podatki in dokumenti

Projekt bo uporabljal predvsem strukturirane podatke v obliki JSON. Ti bodo vsebovali informacije o repozitorijih, commitih, pull requestih, CI/CD rezultatih ter metrikah kakovosti kode. Uporabljena bo tudi uradna dokumentacija GitHub REST API in SonarQube Cloud API.

### 6. Akterji

Glavni akter je razvijalec oziroma vodja razvojne ekipe, ki uporablja DevPulse za spremljanje stanja projekta. GitHub in SonarQube Cloud nastopata kot zunanja ponudnika podatkov. DevPulse podatke pridobi, obdela, združi in predstavi uporabniku.

### 7. Interakcija med rešitvijo in storitvami

Uporabnik izbere GitHub repozitorij, ki ga želi spremljati. DevPulse iz GitHuba pridobi podatke o aktivnosti in CI/CD procesih, iz SonarQube Cloud pa podatke o kakovosti kode. Mikrostoritve podatke združijo in rezultat prikažejo uporabniku v enotnem pregledu.

## Current technical decisions

- Project name: DevPulse
- Backend architecture: microservices
- Backend language: Go
- External integrations: GitHub and SonarQube Cloud
- External communication: REST over HTTPS
- Main data format: JSON
- Core value: explainable software-project health analysis
- Exact health-score formula: not yet fixed
- Frontend framework: not yet decided
- Database: not yet decided
- Deployment platform: not yet decided
- Authentication approach: not yet decided
- Polling vs. webhook/event-based updates: not yet decided
