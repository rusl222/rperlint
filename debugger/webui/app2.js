const rows = new Map();
const favorites = new Set(JSON.parse(localStorage.getItem("favorites") || "[]"));

const tbody = document.getElementById("tbody");
const search = document.getElementById("search");
const typeFilter = document.getElementById("typeFilter");
const count = document.getElementById("count");
const status = document.getElementById("status");
const webport = document.getElementById("webport");

const defaultPort = 8080;
let currentPort = null;
let activeHost = null;

function initCurrentPort() {
    currentPort = getPort();
    updateDebugHosts(currentPort);
}

function saveFavorites() {
    localStorage.setItem("favorites", JSON.stringify([...favorites]));
}

function getPort() {
    const port = Number(webport.value);
    if (!Number.isInteger(port) || port < 1 || port > 65535) {
        return defaultPort;
    }

    return port;
}

function updateDebugHosts(port) {
    window.DEBUG_UI_HOSTS = [
        window.location.origin && window.location.origin !== "null" ? window.location.origin : null,
        `http://127.0.0.1:${port}`,
        `http://localhost:${port}`
    ].filter(Boolean);
}

function getCandidateHosts() {
    return [...new Set(window.DEBUG_UI_HOSTS || [])];
}

function handlePortChange() {
    const port = getPort();

    if (port === currentPort) {
        return;
    }

    currentPort = port;
    updateDebugHosts(port);
    activeHost = null;
    status.textContent = "⏳";
    refresh();
    state();
}

async function ensureHost() {
    if (activeHost) {
        return activeHost;
    }

    for (const host of getCandidateHosts()) {
        try {
            const response = await fetch(`${host}/api/state`, { cache: "no-store" });
            if (!response.ok) {
                continue;
            }

            const data = await response.json();
            if (data && typeof data.connected === "boolean") {
                activeHost = host;
                status.textContent = data.connected ? "🟢" : "🔴";
                return host;
            }
        } catch (error) {
            // Try the next candidate host.
        }
    }

    return null;
}

function favoriteCell(name) {
    const td = document.createElement("td");
    td.className = "favorite";
    td.textContent = favorites.has(name) ? "★" : "☆";

    td.onclick = () => {
        if (favorites.has(name)) {
            favorites.delete(name);
        } else {
            favorites.add(name);
        }

        saveFavorites();
        td.textContent = favorites.has(name) ? "★" : "☆";
        sortRows();
    };

    return td;
}

function makeInput(item) {
    if (item.type === "bool") {
        const checkbox = document.createElement("input");
        checkbox.type = "checkbox";
        checkbox.checked = item.value;
        checkbox.onchange = () => apply(item.name);
        return checkbox;
    }

    const input = document.createElement("input");
    input.className = "value";
    input.value = item.value;
    input.type = item.type === "int" ? "number" : "text";

    input.onkeydown = (event) => {
        if (event.key === "Enter") {
            event.preventDefault();
            input.blur();
            apply(item.name);
        }
    };

    return input;
}

function createRow(item) {
    const tr = document.createElement("tr");
    tr.dataset.name = item.name;
    tr.dataset.type = item.type;
    tr.appendChild(favoriteCell(item.name));

    let td = document.createElement("td");
    td.textContent = item.name;
    tr.appendChild(td);

    td = document.createElement("td");
    const editor = makeInput(item);
    td.appendChild(editor);
    tr.appendChild(td);

    td = document.createElement("td");
    td.textContent = item.type;
    tr.appendChild(td);

    tr.editor = editor;
    rows.set(item.name, tr);

    return tr;
}

function updateRow(item) {
    let tr = rows.get(item.name);

    if (!tr) {
        tbody.appendChild(createRow(item));
        sortRows();
        return;
    }

    if (document.activeElement !== tr.editor) {
        if (item.type === "bool") {
            tr.editor.checked = item.value;
        } else {
            tr.editor.value = item.value;
        }
    }

    tr.className = "changed";

    if (item.type === "float") {
        tr.classList.add("float");
    }

    if (item.type === "int") {
        tr.classList.add("int");
    }

    if (item.type === "bool") {
        if (item.value) {
            tr.classList.add("booltrue");
        } else {
            tr.classList.add("boolfalse");
        }
    }

    setTimeout(() => {
        tr.classList.remove("changed");
    }, 900);
}

async function apply(name) {
    const tr = rows.get(name);
    if (!tr) {
        return;
    }

    const host = await ensureHost();
    if (!host) {
        status.textContent = "🔴";
        return;
    }

    let value = tr.editor.value;

    if (tr.editor.type === "checkbox") {
        value = tr.editor.checked;
    }

    await fetch(`${host}/api/set`, {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify({
            name,
            value: String(value),
            dost: "",
            timeout: 1
        })
    });
}

function filter() {
    const searchText = search.value.toLowerCase();
    const selectedType = typeFilter.value;

    rows.forEach((row) => {
        const matchesName = row.dataset.name.toLowerCase().includes(searchText);
        const matchesType = !selectedType || row.dataset.type === selectedType;
        row.style.display = matchesName && matchesType ? "" : "none";
    });
}

search.oninput = filter;
typeFilter.onchange = filter;
webport.onchange = handlePortChange;
webport.onkeydown = (event) => {
    if (event.key === "Enter") {
        event.preventDefault();
        webport.blur();
        handlePortChange();
    }
};

initCurrentPort();

function sortRows() {
    const arr = [...rows.values()];

    arr.sort((a, b) => {
        const aFavorite = favorites.has(a.dataset.name);
        const bFavorite = favorites.has(b.dataset.name);

        if (aFavorite !== bFavorite) {
            return Number(bFavorite) - Number(aFavorite);
        }

        return a.dataset.name.localeCompare(b.dataset.name);
    });

    arr.forEach((row) => tbody.appendChild(row));
}

async function refresh() {
    const host = await ensureHost();
    if (!host) {
        status.textContent = "🔴";
        return;
    }

    try {
        const response = await fetch(`${host}/api/list`);
        if (!response.ok) {
            throw new Error("list request failed");
        }

        const list = await response.json();
        // count.textContent = list.length + " repers";
        list.forEach(updateRow);
        filter();
    } catch (error) {
        activeHost = null;
        status.textContent = "🔴";
    }
}

async function state() {
    const host = await ensureHost();
    if (!host) {
        status.textContent = "🔴";
        return;
    }

    try {
        const response = await fetch(`${host}/api/state`, { cache: "no-store" });
        if (!response.ok) {
            throw new Error("state request failed");
        }

        const data = await response.json();
        status.textContent = data.connected ? "🟢" : "🔴";
    } catch (error) {
        activeHost = null;
        status.textContent = "🔴";
    }
}

refresh();
state();

setInterval(refresh, 1000);
setInterval(state, 1000);
