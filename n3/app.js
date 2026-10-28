let dataset;
let chart;

const dimensions = [
  "LETO",
  "ZAPOSLOVANJE IN IZOBRAŽEVANJE",
  "KOHEZIJSKA REGIJA",
];

function parsePx(text) {
  const values = {};

  for (const dimension of dimensions) {
    const metadata = [
      ...text.matchAll(/VALUES\("([^"]+)"\)\s*=\s*([\s\S]*?);/g),
    ];

    const entry = metadata.find((match) => match[1] === dimension);

    if (!entry) {
      throw new Error(`Manjka dimenzija: ${dimension}`);
    }

    values[dimension] = [...entry[2].matchAll(/"([^"]*)"/g)].map(
      (match) => match[1],
    );
  }

  const match = text.match(/\bDATA\s*=\s*([\s\S]*?);/);

  if (!match) {
    throw new Error("Podatki DATA niso najdeni");
  }

  const data = match[1]
    .trim()
    .split(/\s+/)
    .map((value) => {
      if ([":", ".", "..", "...", "-"].includes(value)) {
        return null;
      }

      const number = Number(value.replace(",", "."));
      return Number.isFinite(number) ? number : null;
    });

  const expected = dimensions.reduce(
    (count, dimension) => count * values[dimension].length,
    1,
  );

  if (data.length !== expected) {
    throw new Error(`Nepravilno število vrednosti: ${data.length}/${expected}`);
  }

  return { values, data };
}

function getValue(year, indicator, region) {
  const selections = [year, indicator, region];
  let index = 0;

  dimensions.forEach((dimension, i) => {
    const options = dataset.values[dimension];
    const position = options.indexOf(selections[i]);

    if (position === -1) {
      throw new Error(`Neznana vrednost: ${selections[i]}`);
    }

    index = index * options.length + position;
  });

  return dataset.data[index];
}

function createFilter(id, dimension) {
  const select = document.getElementById(id);

  dataset.values[dimension].forEach((value) => {
    const option = document.createElement("option");
    option.value = value;
    option.textContent = value;
    select.appendChild(option);
  });

  select.addEventListener("change", updateChart);
}

function updateChart() {
  const indicator = document.getElementById("indicator").value;
  const region = document.getElementById("region").value;

  const years = dataset.values["LETO"];
  const values = years.map((year) => getValue(year, indicator, region));

  if (chart) {
    chart.destroy();
  }

  chart = new Chart(document.getElementById("statisticsChart"), {
    type: "bar",
    data: {
      labels: years,
      datasets: [
        {
          label: indicator,
          data: values,
          backgroundColor: "#6366f1",
          borderRadius: 6,
        },
      ],
    },
    options: {
      responsive: true,
      plugins: {
        legend: { display: false },
      },
      scales: {
        y: { beginAtZero: true },
      },
    },
  });

  const first = values[0];
  const last = values.at(-1);
  const difference = document.getElementById("difference");

  if (first === null || last === null) {
    difference.textContent = "Ni dovolj podatkov za primerjavo.";
    return;
  }

  const change = last - first;

  if (first === 0) {
    difference.textContent = `Absolutna sprememba: ${change.toLocaleString("sl-SI")}`;
    return;
  }

  const percentage = (change / first) * 100;

  difference.textContent =
    `${change > 0 ? "+" : ""}${change.toLocaleString("sl-SI")} ` +
    `(${percentage > 0 ? "+" : ""}${percentage.toFixed(1)} %)`;
}

async function loadData() {
  const status = document.getElementById("status");

  try {
    const response = await fetch("./data/ict.px");

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }

    const buffer = await response.arrayBuffer();
    const text = new TextDecoder("windows-1250").decode(buffer);

    dataset = parsePx(text);

    createFilter("indicator", "ZAPOSLOVANJE IN IZOBRAŽEVANJE");
    createFilter("region", "KOHEZIJSKA REGIJA");

    updateChart();

    status.textContent = "Podatki uspešno naloženi.";
  } catch (error) {
    console.error(error);
    status.textContent = `Napaka: ${error.message}`;
  }
}

loadData();
