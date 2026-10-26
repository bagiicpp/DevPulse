import requests
import json
from bs4 import BeautifulSoup

url = "https://github.com/psf/requests"

response = requests.get(url, timeout=15)
response.raise_for_status()

soup = BeautifulSoup(response.text, "html.parser")

description = soup.select_one('meta[name="description"]')
stars = soup.select_one('a[href$="/stargazers"]')
forks = soup.select_one('a[href$="/forks"]')

data = {
    "repository": "psf/requests",
    "url": url,
    "description": description.get("content") if description else None,
    "stars": stars.get_text(" ", strip=True) if stars else None,
    "forks": forks.get_text(" ", strip=True) if forks else None,
}

with open("repository.json", "w", encoding="utf-8") as file:
    json.dump(data, file, indent=4, ensure_ascii=False)

print("Repository data saved to repository.json")

