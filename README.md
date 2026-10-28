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
├── n2/             # Web scraping assignment
├── n3/             # PC-Axis data visualization
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

- **Go** – planned backend microservices
- **Python** – XML processing assignment
- **GitHub REST API** – development activity and CI/CD data
- **SonarQube Cloud API** – code-quality metrics
- **XML / JSON** – structured data formats
- **HTML / CSS / JavaScript** – interactive data visualization
- **Chart.js** – statistical charts
- **PC-Axis** – official statistical data format

## Status

DevPulse is currently being developed as a university project. Each assignment implements or demonstrates a specific part of the overall concept.
