import argparse
import json
import xml.etree.ElementTree as ET
from pathlib import Path

BASE_DIR = Path(__file__).resolve().parent
DATA_DIR = BASE_DIR / "data"


def read_xml(filename):
    """Read and parse an XML document."""
    tree = ET.parse(DATA_DIR / filename)
    return tree.getroot()


def read_projects():
    """Convert project XML elements into dictionaries."""
    root = read_xml("projekti.xml")

    return [
        {
            "id": p.get("id"),
            "ime": p.findtext("ime"),
            "repozitorij": p.findtext("repozitorij"),
            "jezik": p.findtext("jezik"),
            "datum": p.findtext("datumZadnjeSpremembe"),
        }
        for p in root.findall("projekt")
    ]


def read_ci():
    """Convert CI runs into dictionaries indexed by project ID."""
    root = read_xml("ci.xml")

    return {
        ci.get("projektId"): {
            "status": ci.findtext("status"),
            "trajanje": int(ci.findtext("trajanjeSekund")),
            "datum": ci.findtext("datumZakljucka") or None,
        }
        for ci in root.findall("zagon")
    }


def read_quality():
    """Convert quality reports into dictionaries indexed by project ID."""
    root = read_xml("kakovost.xml")
    reports = {}

    for report in root.findall("porocilo"):
        coverage = report.findtext("pokritost")

        reports[report.get("projektId")] = {
            "pokritost": float(coverage) if coverage else None,
            "napake": int(report.findtext("napake")),
            "qualityGate": report.findtext("qualityGate"),
        }

    return reports


def combine_data():
    """Join data from all three XML files using project IDs."""
    projects = read_projects()
    ci_runs = read_ci()
    quality = read_quality()

    combined = []

    for project in projects:
        project_id = project["id"]

        if project_id not in ci_runs or project_id not in quality:
            raise ValueError(f"Missing data for project {project_id}")

        combined.append({
            **project,
            "ci": ci_runs[project_id],
            "kakovost": quality[project_id],
        })

    return combined


def filter_projects(projects, coverage_below=None, ci_status=None):
    """Filter projects using optional coverage and CI status conditions."""
    results = []

    for project in projects:
        coverage = project["kakovost"]["pokritost"]

        if coverage_below is not None:
            if coverage is None or coverage >= coverage_below:
                continue

        if ci_status is not None:
            if project["ci"]["status"] != ci_status:
                continue

        results.append(project)

    return results


def print_results(projects):
    """Display results as a readable console table."""
    print(f"{'Project':<18} {'Coverage':<12} {'CI':<10} {'Issues':<8}")
    print("-" * 50)

    for project in projects:
        quality = project["kakovost"]
        coverage = quality["pokritost"]
        coverage_text = f"{coverage:g}%" if coverage is not None else "N/A"

        print(
            f"{project['ime']:<18} "
            f"{coverage_text:<12} "
            f"{project['ci']['status']:<10} "
            f"{quality['napake']:<8}"
        )

    print(f"\nFound {len(projects)} projects.")


def export_json(projects):
    """Save filtered records in JSON format."""
    output = BASE_DIR / "filtrirano.json"

    with output.open("w", encoding="utf-8") as file:
        json.dump(projects, file, indent=4, ensure_ascii=False)


def export_xml(projects):
    """Save the same filtered records in XML format."""
    root = ET.Element("projekti")

    for project in projects:
        element = ET.SubElement(root, "projekt", id=project["id"])

        for field in ("ime", "repozitorij", "jezik", "datum"):
            ET.SubElement(element, field).text = project[field]

        ci = ET.SubElement(element, "ci")

        for field, value in project["ci"].items():
            ET.SubElement(ci, field).text = (
                str(value) if value is not None else ""
            )

        quality = ET.SubElement(element, "kakovost")

        for field, value in project["kakovost"].items():
            ET.SubElement(quality, field).text = (
                str(value) if value is not None else ""
            )

    tree = ET.ElementTree(root)
    ET.indent(tree, space="    ")

    tree.write(
        BASE_DIR / "filtrirano.xml",
        encoding="utf-8",
        xml_declaration=True,
    )


def main():
    projects = combine_data()

    while True:
        print("\n===== DevPulse =====")
        print("1. Show all projects")
        print("2. Filter by test coverage")
        print("3. Filter by CI status")
        print("4. Filter by coverage and CI status")
        print("0. Exit")

        choice = input("\nChoose an option: ").strip()

        if choice == "0":
            print("Goodbye!")
            break

        elif choice == "1":
            results = projects

        elif choice == "2":
            try:
                coverage = float(input("Coverage below (%): "))
            except ValueError:
                print("Invalid number.")
                continue

            if not 0 <= coverage <= 100:
                print("Coverage must be between 0 and 100.")
                continue

            results = filter_projects(
                projects, coverage_below=coverage
            )

        elif choice == "3":
            status = input(
                "CI status (success/failure/pending): "
            ).strip().lower()

            if status not in ("success", "failure", "pending"):
                print("Invalid CI status.")
                continue

            results = filter_projects(
                projects, ci_status=status
            )

        elif choice == "4":
            try:
                coverage = float(input("Coverage below (%): "))
            except ValueError:
                print("Invalid number.")
                continue

            if not 0 <= coverage <= 100:
                print("Coverage must be between 0 and 100.")
                continue

            status = input(
                "CI status (success/failure/pending): "
            ).strip().lower()

            if status not in ("success", "failure", "pending"):
                print("Invalid CI status.")
                continue

            results = filter_projects(
                projects,
                coverage_below=coverage,
                ci_status=status,
            )

        else:
            print("Invalid option. Try again.")
            continue

        print()
        print_results(results)

        export_json(results)
        export_xml(results)

        print("\nExported: filtrirano.json, filtrirano.xml")


if __name__ == "__main__":
    main()
