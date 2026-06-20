import { readdir, readFile } from "node:fs/promises";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

const projectRoot = fileURLToPath(new URL("..", import.meta.url));
const sourceRoot = join(projectRoot, "apps", "web", "src");
const checkedExtensions = new Set([".ts", ".tsx", ".css"]);
const failures = [];

async function walk(dir) {
  const entries = await readdir(dir, { withFileTypes: true });
  for (const entry of entries) {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) {
      await walk(path);
      continue;
    }
    if ([...checkedExtensions].some((extension) => entry.name.endsWith(extension))) {
      await lintFile(path);
    }
  }
}

async function lintFile(path) {
  const source = await readFile(path, "utf8");
  const lines = source.split(/\r?\n/);
  lines.forEach((line, index) => {
    const lineNumber = index + 1;
    if (/\bdebugger\b/.test(line)) {
      failures.push(`${path}:${lineNumber} contains debugger statement`);
    }
    if (/\bconsole\.log\s*\(/.test(line)) {
      failures.push(`${path}:${lineNumber} contains console.log`);
    }
  });
}

await walk(sourceRoot);

if (failures.length > 0) {
  console.error("Web lint failed:");
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log("Web lint passed");
