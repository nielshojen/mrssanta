// Run from containers/admin with: node --test tests/js/*.test.js
const test = require("node:test");
const assert = require("node:assert/strict");

const PathRegex = require("../../static/js/path_regex.js");

test("builds each kind of condition", () => {
    const cases = [
        [{kind: "folder", value: "/Applications/Tools"}, "^/Applications/Tools/"],
        [{kind: "folder", value: "/Applications/Tools//"}, "^/Applications/Tools/"],
        [{kind: "folder", value: "~/bin"}, "^/Users/[^/]+/bin/"],
        [{kind: "starts", value: "/opt/homebrew/Cellar/python"}, "^/opt/homebrew/Cellar/python"],
        [{kind: "file", value: "/usr/local/bin/my.tool"}, "^/usr/local/bin/my\\.tool$"],
        [{kind: "name", value: "terraform"}, "/terraform$"],
        [{kind: "contains", value: "/node_modules/.pnpm/"}, "/node_modules/\\.pnpm/"],
        [{kind: "custom", value: "/terraform-provider-[^/]+$"}, "/terraform-provider-[^/]+$"],
        [{kind: "folder", value: "   "}, ""],
    ];
    for (const [condition, want] of cases) {
        assert.equal(PathRegex.conditionRegex(condition), want, JSON.stringify(condition));
    }
});

test("escapes every regex metacharacter", () => {
    assert.equal(PathRegex.escapeRegex("a.b*c+d?e(f)g[h]i{j}k|l^m$n\\o"),
        "a\\.b\\*c\\+d\\?e\\(f\\)g\\[h\\]i\\{j\\}k\\|l\\^m\\$n\\\\o");
});

test("joins several conditions into one group", () => {
    assert.equal(PathRegex.build([
        {kind: "folder", value: "/opt/tools"},
        {kind: "name", value: "terraform"},
        {kind: "contains", value: ""},
    ]), "(^/opt/tools/|/terraform$)");
    assert.equal(PathRegex.build([{kind: "folder", value: "/opt/tools"}]), "^/opt/tools/");
    assert.equal(PathRegex.build([]), "");
});

test("reads a built regex back into the same conditions", () => {
    const conditions = [
        {kind: "folder", value: "/Applications/Tools"},
        {kind: "folder", value: "~/bin"},
        {kind: "starts", value: "/opt/homebrew/Cellar/python"},
        {kind: "file", value: "/usr/local/bin/my.tool"},
        {kind: "file", value: "~/Applications/App (1).app/Contents/MacOS/App"},
        {kind: "name", value: "terraform"},
        {kind: "contains", value: "/node_modules/.pnpm/"},
        {kind: "custom", value: "/terraform-provider-[^/]+$"},
        {kind: "custom", value: "^/opt/(a|b)/"},
    ];
    const regex = PathRegex.build(conditions);

    assert.deepEqual(PathRegex.parse(regex), conditions);
    assert.equal(PathRegex.build(PathRegex.parse(regex)), regex);
});

test("reads the regex suggested in the README", () => {
    const readme = "(/node_modules/\\.pnpm/|/\\.pyenv/versions/|^/opt/homebrew/Cellar/python|^/usr/local/Cellar/python|^/usr/local/microsoft/powershell/7/|/terraform-provider-[^/]+$)";

    assert.deepEqual(PathRegex.parse(readme), [
        {kind: "contains", value: "/node_modules/.pnpm/"},
        {kind: "contains", value: "/.pyenv/versions/"},
        {kind: "starts", value: "/opt/homebrew/Cellar/python"},
        {kind: "starts", value: "/usr/local/Cellar/python"},
        {kind: "folder", value: "/usr/local/microsoft/powershell/7"},
        {kind: "custom", value: "/terraform-provider-[^/]+$"},
    ]);
    assert.equal(PathRegex.build(PathRegex.parse(readme)), readme);
});

test("keeps a regex it can't break apart as one custom condition", () => {
    assert.deepEqual(PathRegex.parse("^/opt/(a|b)/[^/]+$"), [{kind: "custom", value: "^/opt/(a|b)/[^/]+$"}]);
    assert.deepEqual(PathRegex.parse("(a)|(b)"), [{kind: "custom", value: "(a)"}, {kind: "custom", value: "(b)"}]);
    assert.deepEqual(PathRegex.parse("  "), []);
});

test("built regexes match the paths they describe", () => {
    const regex = new RegExp(PathRegex.build([
        {kind: "folder", value: "~/bin"},
        {kind: "file", value: "/usr/local/bin/tool"},
        {kind: "name", value: "terraform"},
    ]));

    assert.ok(regex.test("/Users/alex/bin/script"));
    assert.ok(!regex.test("/Users/alex/binaries/script"));
    assert.ok(!regex.test("/Users/alex/sub/bin/script"));
    assert.ok(regex.test("/usr/local/bin/tool"));
    assert.ok(!regex.test("/usr/local/bin/tool2"));
    assert.ok(regex.test("/opt/hashicorp/terraform"));
    assert.ok(!regex.test("/opt/hashicorp/terraform-ls"));
});

test("warns about allow conditions users could exploit", () => {
    assert.match(PathRegex.risky({kind: "folder", value: "~/bin"}), /Users can write/);
    assert.match(PathRegex.risky({kind: "folder", value: "/tmp/x"}), /Users can write/);
    assert.match(PathRegex.risky({kind: "starts", value: "/Users/Shared"}), /Users can write/);
    assert.match(PathRegex.risky({kind: "name", value: "terraform"}), /any folder/);
    assert.match(PathRegex.risky({kind: "contains", value: "/node_modules/"}), /any folder/);
    assert.equal(PathRegex.risky({kind: "folder", value: "/Applications/Tools"}), "");
    assert.equal(PathRegex.risky({kind: "folder", value: ""}), "");
});

test("reports invalid regexes", () => {
    assert.equal(PathRegex.compile("^/opt/").error, "");
    assert.notEqual(PathRegex.compile("^/opt/(").error, "");
});
