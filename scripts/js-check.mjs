import { readFileSync, readdirSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { dirname, extname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), "..");
const staticRoot = join(repoRoot, "web", "static");

function collectJavaScriptFiles(dir) {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const fullPath = join(dir, entry.name);
    if (entry.isDirectory()) return collectJavaScriptFiles(fullPath);
    return entry.isFile() && entry.name.endsWith(".js") ? [fullPath] : [];
  });
}

function collectUISourceFiles(dir) {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const fullPath = join(dir, entry.name);
    if (entry.isDirectory()) return collectUISourceFiles(fullPath);
    return entry.isFile() && [".css", ".html", ".js"].includes(extname(entry.name)) ? [fullPath] : [];
  });
}

const files = collectJavaScriptFiles(staticRoot).sort((a, b) => a.localeCompare(b));

for (const file of files) {
  const display = relative(repoRoot, file);
  const result = spawnSync(process.execPath, ["--check", file], { stdio: "inherit" });
  if (result.status !== 0) {
    console.error(`JavaScript syntax check failed: ${display}`);
    process.exit(result.status || 1);
  }
}

console.log(`JavaScript syntax OK (${files.length} files checked)`);

const mojibakeMarkers = ["РЎ", "Рџ", "Рў", "Рµ", "СЊ", "вЂ", "Рґ", "Рє", "РЅ", "Рѕ"];
const uiSources = collectUISourceFiles(staticRoot).sort((a, b) => a.localeCompare(b));
const corrupted = [];

for (const file of uiSources) {
  const source = readFileSync(file, "utf8");
  const markers = mojibakeMarkers.filter((marker) => source.includes(marker));
  if (markers.length) {
    corrupted.push(`${relative(repoRoot, file)} (${markers.join(", ")})`);
  }
}

if (corrupted.length) {
  console.error("Possible mojibake detected in UI source files:");
  corrupted.forEach((entry) => console.error(`- ${entry}`));
  process.exit(1);
}

console.log(`UI source encoding OK (${uiSources.length} files checked)`);
