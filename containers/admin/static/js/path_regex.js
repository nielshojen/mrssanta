// Builder for the "Allowed path regex" and "Blocked path regex" settings.
//
// Santa matches these ICU regexes against the full path of each executable
// that no rule covers. The builder turns a list of simple conditions into
// one regex, and reads a regex back into conditions so it can be edited.
// Parts it can't express become "Custom regex" conditions, kept as written.
//
// Fields rendered with data-regex-builder get a "Build" button that opens
// the builder below the field.

(function (root) {
    "use strict";

    const KINDS = {
        folder: {label: "Inside folder", placeholder: "/Applications/Tools or ~/bin"},
        starts: {label: "Path starts with", placeholder: "/opt/homebrew/Cellar/python"},
        file: {label: "Exact path", placeholder: "/usr/local/bin/tool"},
        name: {label: "File name is", placeholder: "terraform"},
        contains: {label: "Path contains", placeholder: "/node_modules/.pnpm/"},
        custom: {label: "Custom regex", placeholder: "/terraform-provider-[^/]+$"},
    };

    // "~" stands for any user's home folder.
    const HOME_REGEX = "^/Users/[^/]+";

    // Folders where any local user can create files.
    const USER_WRITABLE = /^(~|\/Users\/|\/tmp\/|\/private\/|\/var\/folders\/|\/Volumes\/)/;

    function escapeRegex(text) {
        return text.replace(/[\\^$.|?*+()[\]{}]/g, "\\$&");
    }

    // unescapeRegex returns the literal text a regex matches, or null if it
    // contains anything other than literal characters and escaped
    // metacharacters.
    function unescapeRegex(regex) {
        let text = "";
        for (let i = 0; i < regex.length; i++) {
            const ch = regex[i];
            if (ch === "\\") {
                const next = regex[i + 1];
                if (next === undefined || !/[\\^$.|?*+()[\]{}/]/.test(next)) return null;
                text += next;
                i++;
            } else if (/[\\^$.|?*+()[\]{}]/.test(ch)) {
                return null;
            } else {
                text += ch;
            }
        }
        return text;
    }

    // anchoredPath turns a path that may start with "~/" into an anchored
    // regex prefix.
    function anchoredPath(path) {
        if (path === "~" || path.startsWith("~/")) {
            return HOME_REGEX + escapeRegex(path.slice(1));
        }
        return "^" + escapeRegex(path);
    }

    function conditionRegex(condition) {
        const value = condition.value.trim();
        if (!value) return "";

        switch (condition.kind) {
            case "folder":
                return anchoredPath(value.replace(/\/+$/, "")) + "/";
            case "starts":
                return anchoredPath(value);
            case "file":
                return anchoredPath(value) + "$";
            case "name":
                return "/" + escapeRegex(value.replace(/\//g, "")) + "$";
            case "contains":
                return escapeRegex(value);
            default:
                return value;
        }
    }

    function build(conditions) {
        const parts = conditions.map(conditionRegex).filter(Boolean);
        if (parts.length <= 1) return parts.join("");
        return "(" + parts.join("|") + ")";
    }

    // splitAlternatives splits a regex on its top-level "|".
    function splitAlternatives(regex) {
        const parts = [];
        let depth = 0;
        let inClass = false;
        let current = "";
        for (let i = 0; i < regex.length; i++) {
            const ch = regex[i];
            if (ch === "\\") {
                current += ch + (regex[i + 1] || "");
                i++;
                continue;
            }
            if (inClass) {
                if (ch === "]") inClass = false;
            } else if (ch === "[") {
                inClass = true;
            } else if (ch === "(") {
                depth++;
            } else if (ch === ")") {
                depth--;
            } else if (ch === "|" && depth === 0) {
                parts.push(current);
                current = "";
                continue;
            }
            current += ch;
        }
        parts.push(current);
        return parts;
    }

    // stripOuterGroup removes parentheses that wrap the whole regex.
    function stripOuterGroup(regex) {
        if (!regex.startsWith("(") || !regex.endsWith(")") || regex.startsWith("(?")) return regex;
        const inner = regex.slice(1, -1);
        let depth = 0;
        for (let i = 0; i < inner.length; i++) {
            const ch = inner[i];
            if (ch === "\\") { i++; continue; }
            if (ch === "(") depth++;
            if (ch === ")" && --depth < 0) return regex;
        }
        return depth === 0 ? inner : regex;
    }

    function parseAlternative(regex) {
        let rest = null;
        let prefix = "";
        if (regex.startsWith(HOME_REGEX)) {
            rest = regex.slice(HOME_REGEX.length);
            prefix = "~";
        } else if (regex.startsWith("^")) {
            rest = regex.slice(1);
        }

        if (rest !== null && (prefix === "" || rest.startsWith("/"))) {
            if (rest.endsWith("$")) {
                const path = unescapeRegex(rest.slice(0, -1));
                if (path !== null && (prefix + path)) return {kind: "file", value: prefix + path};
            } else if (rest.endsWith("/") && rest.length > 1) {
                const path = unescapeRegex(rest.slice(0, -1));
                if (path !== null) return {kind: "folder", value: prefix + path};
            } else {
                const path = unescapeRegex(rest);
                if (path !== null && (prefix + path)) return {kind: "starts", value: prefix + path};
            }
        }

        if (regex.startsWith("/") && regex.endsWith("$")) {
            const name = unescapeRegex(regex.slice(1, -1));
            if (name && !name.includes("/")) return {kind: "name", value: name};
        }

        const literal = unescapeRegex(regex);
        if (literal) return {kind: "contains", value: literal};

        return {kind: "custom", value: regex};
    }

    function parse(regex) {
        regex = (regex || "").trim();
        if (!regex) return [];
        return splitAlternatives(stripOuterGroup(regex)).filter(Boolean).map(parseAlternative);
    }

    // risky reports why an allow condition could let users run anything, or
    // "" if it looks safe.
    function risky(condition) {
        const value = condition.value.trim();
        if (!value) return "";
        if (condition.kind === "contains" || condition.kind === "name") {
            return "Matches in any folder, including ones users can write to.";
        }
        if (condition.kind === "custom") return "";
        if (USER_WRITABLE.test(value)) {
            return "Users can write to this folder, so anyone could place a binary there.";
        }
        return "";
    }

    function compile(regex) {
        try {
            return {regex: new RegExp(regex), error: ""};
        } catch (e) {
            return {regex: null, error: e.message};
        }
    }

    const PathRegex = {KINDS, build, parse, risky, compile, escapeRegex, conditionRegex};

    if (typeof module !== "undefined" && module.exports) {
        module.exports = PathRegex;
        return;
    }
    root.PathRegex = PathRegex;

    // UI

    function el(tag, attrs, children) {
        const node = document.createElement(tag);
        Object.entries(attrs || {}).forEach(([key, value]) => {
            if (key === "class") node.className = value;
            else if (key === "text") node.textContent = value;
            else node.setAttribute(key, value);
        });
        (children || []).forEach((child) => node.appendChild(child));
        return node;
    }

    function openBuilder(input, panel) {
        const isAllow = input.dataset.regexBuilder === "allow";
        let conditions = parse(input.value);
        if (conditions.length === 0) conditions = [{kind: "folder", value: ""}];

        panel.replaceChildren();
        const rows = el("div", {class: "regex-builder-rows"});
        const preview = el("code", {class: "regex-builder-preview"});
        const status = el("div", {class: "form-text"});
        const testInput = el("input", {type: "text", class: "form-control form-control-sm font-monospace", placeholder: "Test a path, e.g. /Users/alex/bin/tool"});
        const testResult = el("div", {class: "form-text"});

        function current() {
            return conditions.map((c) => ({kind: c.kind, value: c.value}));
        }

        function refresh() {
            const regex = build(current());
            preview.textContent = regex || "(empty: setting not set)";
            const compiled = compile(regex);
            status.textContent = compiled.error ? "Invalid regex: " + compiled.error : "";
            status.classList.toggle("text-danger", Boolean(compiled.error));

            const path = testInput.value.trim();
            if (!path || compiled.error || !regex) {
                testResult.textContent = "";
                return;
            }
            const matched = compiled.regex.test(path);
            testResult.textContent = matched
                ? (isAllow ? "Matches: allowed when no rule covers it." : "Matches: blocked when no rule covers it.")
                : "Doesn't match.";
            testResult.className = "form-text " + (matched ? "fw-semibold" : "");
        }

        function renderRows() {
            rows.replaceChildren();
            conditions.forEach((condition, index) => {
                const kind = el("select", {class: "form-select form-select-sm", "aria-label": "Condition"});
                Object.entries(KINDS).forEach(([value, info]) => {
                    const option = el("option", {value, text: info.label});
                    if (value === condition.kind) option.selected = true;
                    kind.appendChild(option);
                });
                const value = el("input", {
                    type: "text",
                    class: "form-control form-control-sm font-monospace",
                    placeholder: KINDS[condition.kind].placeholder,
                    "aria-label": "Value",
                });
                value.value = condition.value;
                const remove = el("button", {type: "button", class: "btn btn-sm btn-outline-secondary", "aria-label": "Remove condition", text: "×"});
                const warning = el("div", {class: "form-text text-warning-emphasis"});

                function updateWarning() {
                    warning.textContent = isAllow ? risky(condition) : "";
                }

                kind.addEventListener("change", () => {
                    condition.kind = kind.value;
                    value.placeholder = KINDS[condition.kind].placeholder;
                    updateWarning();
                    refresh();
                });
                value.addEventListener("input", () => {
                    condition.value = value.value;
                    updateWarning();
                    refresh();
                });
                remove.addEventListener("click", () => {
                    conditions.splice(index, 1);
                    renderRows();
                    refresh();
                });
                updateWarning();

                rows.appendChild(el("div", {class: "regex-builder-row"}, [
                    el("div", {class: "input-group input-group-sm"}, [kind, value, remove]),
                    warning,
                ]));
            });
        }

        const add = el("button", {type: "button", class: "btn btn-sm btn-outline-secondary", text: "Add condition"});
        add.addEventListener("click", () => {
            conditions.push({kind: "folder", value: ""});
            renderRows();
            refresh();
            rows.lastChild.querySelector("input").focus();
        });

        const apply = el("button", {type: "button", class: "btn btn-sm btn-santa", text: "Use this regex"});
        apply.addEventListener("click", () => {
            const regex = build(current());
            if (compile(regex).error) return;
            input.value = regex;
            input.dispatchEvent(new Event("input", {bubbles: true}));
            panel.hidden = true;
        });
        const cancel = el("button", {type: "button", class: "btn btn-sm btn-outline-secondary", text: "Cancel"});
        cancel.addEventListener("click", () => { panel.hidden = true; });

        testInput.addEventListener("input", refresh);

        panel.append(
            el("p", {class: "form-text mt-0", text: "A path matches if any condition matches. Start a path with ~ for any user's home folder."}),
            rows,
            el("div", {class: "mb-3"}, [add]),
            el("div", {class: "mb-2"}, [el("span", {class: "form-text", text: "Regex: "}), preview, status]),
            el("div", {class: "mb-3"}, [testInput, testResult]),
            el("div", {class: "d-flex gap-2 justify-content-end"}, [cancel, apply]),
        );

        renderRows();
        refresh();
        panel.hidden = false;
    }

    function attach(input) {
        if (input.dataset.regexBuilderReady) return;
        input.dataset.regexBuilderReady = "true";

        const button = el("button", {type: "button", class: "btn btn-outline-secondary", text: "Build…"});
        const panel = el("div", {class: "regex-builder", hidden: ""});

        const group = el("div", {class: "input-group"});
        input.replaceWith(group);
        group.append(input, button);
        group.after(panel);

        button.addEventListener("click", () => {
            if (panel.hidden) openBuilder(input, panel);
            else panel.hidden = true;
        });
    }

    // closeRegexBuilders hides open builders, e.g. when the form is filled
    // with another device's settings.
    root.closeRegexBuilders = function (container) {
        container.querySelectorAll(".regex-builder").forEach((panel) => { panel.hidden = true; });
    };

    document.addEventListener("DOMContentLoaded", () => {
        document.querySelectorAll("[data-regex-builder]").forEach(attach);
    });
})(typeof window !== "undefined" ? window : globalThis);
