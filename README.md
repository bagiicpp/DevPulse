# DevPulse

DevPulse is a university project for monitoring the health of software projects by combining development activity from **GitHub** with code-quality metrics from **SonarQube Cloud**.

The goal is to provide an explainable assessment of software-project health rather than displaying isolated metrics.

## Project Overview

DevPulse analyzes:

- Development activity and pull requests
- CI/CD reliability
- Test coverage and code quality
- Code duplication and detected issues

The planned backend architecture consists of **Go microservices** communicating through REST APIs using JSON.

## Repository Structure

```text
.
├── n0/             # Project proposal and architecture diagram
├── n1/             # XML processing assignment
└── README.md
```

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

## Technologies

- **Go** – planned backend microservices
- **Python** – XML processing assignment
- **GitHub REST API** – development activity and CI/CD data
- **SonarQube Cloud API** – code-quality metrics
- **XML / JSON** – structured data formats

## Status

DevPulse is currently being developed as a university project. Each assignment implements or demonstrates a specific part of the overall concept.
