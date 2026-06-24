#!/usr/bin/env node

import { spawn } from "node:child_process";
import {
  appendFile,
  lstat,
  mkdir,
  readdir,
  readFile,
  rename,
  rm,
  symlink,
  writeFile,
} from "node:fs/promises";
import { basename, dirname, join, resolve } from "node:path";
import process from "node:process";

const DEFAULT_TARGET_COUNT = 10_000;

function usage() {
  return `Generate one Vali map per country or territory, sequentially and resumably.

Usage:
  node scripts/generate-vali-country-batch.mjs --template <file> --output-root <dir> [options]

Required:
  --template <file>        Base Vali config JSON, e.g. datasets-config/country.json
  --output-root <dir>      Directory that will receive per-country outputs

Options:
  --source-root <dir>      Directory containing downloaded country folders (default: cwd)
  --target-count <n>       Per-country location goal (default: ${DEFAULT_TARGET_COUNT})
  --state-file <file>      Progress log path (default: <output-root>/.progress.jsonl)
  --countries <list>       Comma-separated allowlist, e.g. US,CA,PR
  --keep-staging           Keep staging directories after successful runs
  --help                   Show this help

Behavior:
  - Processes one country directory at a time in sorted order.
  - Creates an isolated staging directory for each country.
  - Symlinks the country data into staging so Vali can read it without writing into source data.
  - Writes completion state as JSONL so reruns skip successful countries.
  - If interrupted, rerun the same command to resume remaining countries.
`;
}

function parseArgs(argv) {
  const options = {
    sourceRoot: process.cwd(),
    targetCount: DEFAULT_TARGET_COUNT,
    keepStaging: false,
  };

  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === "--help" || arg === "-h") {
      options.help = true;
      continue;
    }
    if (arg === "--keep-staging") {
      options.keepStaging = true;
      continue;
    }
    if (!arg.startsWith("--")) throw new Error(`Unexpected argument: ${arg}`);
    const key = arg.slice(2).replace(/-([a-z])/g, (_, letter) => letter.toUpperCase());
    const value = argv[i + 1];
    if (!value || value.startsWith("--")) throw new Error(`Missing value for ${arg}`);
    options[key] = value;
    i += 1;
  }

  if (options.help) return options;
  if (!options.template) throw new Error("--template is required");
  if (!options.outputRoot) throw new Error("--output-root is required");

  options.template = resolve(options.template);
  options.sourceRoot = resolve(options.sourceRoot);
  options.outputRoot = resolve(options.outputRoot);
  options.targetCount = parsePositiveInteger(options.targetCount, "--target-count");
  options.stateFile = resolve(options.stateFile || join(options.outputRoot, ".progress.jsonl"));
  options.countries = parseCountries(options.countries);
  return options;
}

function parsePositiveInteger(value, name) {
  const parsed = Number(value);
  if (!Number.isInteger(parsed) || parsed < 1) throw new Error(`${name} must be a positive integer`);
  return parsed;
}

function parseCountries(value) {
  if (!value) return null;
  const countries = value
    .split(",")
    .map((entry) => entry.trim().toUpperCase())
    .filter(Boolean);
  if (countries.length === 0) throw new Error("--countries must include at least one code");
  return new Set(countries);
}

async function readJson(path) {
  return JSON.parse(await readFile(path, "utf8"));
}

async function readState(path) {
  try {
    const state = new Map();
    const lines = (await readFile(path, "utf8")).split("\n");
    for (const line of lines) {
      if (!line.trim()) continue;
      const entry = JSON.parse(line);
      if (entry && entry.country) state.set(entry.country, entry);
    }
    return state;
  } catch (error) {
    if (error.code === "ENOENT") return new Map();
    throw error;
  }
}

async function appendState(path, entry) {
  await mkdir(dirname(path), { recursive: true });
  await appendFile(path, `${JSON.stringify({ at: new Date().toISOString(), ...entry })}\n`, "utf8");
}

async function listCountryDirectories(sourceRoot, allowlist) {
  const entries = await readdir(sourceRoot, { withFileTypes: true });
  return entries
    .filter((entry) => entry.isDirectory())
    .map((entry) => entry.name)
    .filter((name) => !allowlist || allowlist.has(name.toUpperCase()))
    .sort((left, right) => left.localeCompare(right));
}

async function directoryHasArtifacts(path) {
  try {
    const entries = await readdir(path, { withFileTypes: true });
    return entries.some((entry) => !entry.name.startsWith("."));
  } catch (error) {
    if (error.code === "ENOENT") return false;
    throw error;
  }
}

function buildCountryConfig(template, country, targetCount) {
  const config = structuredClone(template);
  config.countryCodes = [country];
  config.distributionStrategy = {
    ...config.distributionStrategy,
    locationCountGoal: targetCount,
  };
  return config;
}

async function collectArtifacts(stagingDir, excludedNames) {
  const entries = await readdir(stagingDir, { withFileTypes: true });
  return entries
    .filter((entry) => !excludedNames.has(entry.name))
    .map((entry) => entry.name)
    .sort((left, right) => left.localeCompare(right));
}

function runVali(configPath, cwd) {
  const child = spawn("vali", ["generate", "--file", basename(configPath)], {
    cwd,
    stdio: "inherit",
  });
  const promise = new Promise((resolveRun, rejectRun) => {
    child.on("error", rejectRun);
    child.on("exit", (code, signal) => resolveRun({ code, signal }));
  });
  return { child, promise };
}

let activeChild = null;
let stopRequested = false;
let sigintCount = 0;

process.on("SIGINT", () => {
  sigintCount += 1;
  stopRequested = true;
  if (activeChild && activeChild.exitCode === null) {
    activeChild.kill("SIGINT");
  }
  if (sigintCount > 1) process.exit(130);
});

async function main() {
  const options = parseArgs(process.argv.slice(2));
  if (options.help) {
    console.log(usage());
    return;
  }

  const template = await readJson(options.template);
  const countries = await listCountryDirectories(options.sourceRoot, options.countries);
  const state = await readState(options.stateFile);
  const stagingRoot = join(options.outputRoot, ".staging");

  await mkdir(options.outputRoot, { recursive: true });
  await mkdir(stagingRoot, { recursive: true });

  if (countries.length === 0) {
    console.log("No country directories matched the current filters.");
    return;
  }

  console.log(`Found ${countries.length} country directories under ${options.sourceRoot}`);
  console.log(`Writing outputs to ${options.outputRoot}`);
  console.log(`Progress log: ${options.stateFile}`);

  for (const country of countries) {
    if (stopRequested) break;

    const finalDir = join(options.outputRoot, country);
    const latestState = state.get(country);
    if (latestState?.status === "success" && (await directoryHasArtifacts(finalDir))) {
      console.log(`[skip] ${country} already completed`);
      continue;
    }

    const sourceDir = join(options.sourceRoot, country);
    const sourceStat = await lstat(sourceDir);
    if (!sourceStat.isDirectory()) {
      console.log(`[skip] ${country} is not a directory`);
      await appendState(options.stateFile, { country, status: "skipped", reason: "not_directory" });
      continue;
    }

    const stagingDir = join(stagingRoot, country);
    const configName = `${country.toLowerCase()}-10k-country.json`;
    const configPath = join(stagingDir, configName);
    const linkedCountryDir = join(stagingDir, country);

    await rm(stagingDir, { recursive: true, force: true });
    await mkdir(stagingDir, { recursive: true });
    await symlink(sourceDir, linkedCountryDir, "dir");
    await writeFile(
      configPath,
      `${JSON.stringify(buildCountryConfig(template, country, options.targetCount), null, 2)}\n`,
      "utf8",
    );

    console.log(`[run ] ${country}`);
    await appendState(options.stateFile, { country, status: "started", outputDir: finalDir });

    const run = runVali(configPath, stagingDir);
    activeChild = run.child;
    const result = await run.promise;
    activeChild = null;

    if (result.code !== 0) {
      const status = stopRequested || result.signal === "SIGINT" ? "interrupted" : "failed";
      await appendState(options.stateFile, {
        country,
        status,
        exitCode: result.code,
        signal: result.signal || null,
        stagingDir,
      });
      if (status === "interrupted") break;
      continue;
    }

    const artifacts = await collectArtifacts(stagingDir, new Set([country, configName]));
    if (artifacts.length === 0) {
      await appendState(options.stateFile, {
        country,
        status: "failed",
        reason: "no_artifacts_detected",
        stagingDir,
      });
      console.log(`[fail] ${country} produced no output artifacts`);
      continue;
    }

    await rm(finalDir, { recursive: true, force: true });
    await mkdir(finalDir, { recursive: true });
    for (const artifact of artifacts) {
      await rename(join(stagingDir, artifact), join(finalDir, artifact));
    }

    await appendState(options.stateFile, {
      country,
      status: "success",
      outputDir: finalDir,
      artifacts,
    });
    console.log(`[done] ${country} -> ${finalDir}`);

    if (!options.keepStaging) await rm(stagingDir, { recursive: true, force: true });
  }

  if (stopRequested) {
    console.log("Stopped. Re-run the same command to resume remaining countries.");
  } else {
    console.log("Batch generation complete.");
  }
}

main().catch((error) => {
  console.error(error.message || error);
  process.exit(1);
});