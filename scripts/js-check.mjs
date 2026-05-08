import { readdirSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { dirname, join, relative } from "node:path";
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
