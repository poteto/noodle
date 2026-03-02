#!/usr/bin/env node

import { execFileSync } from "node:child_process";
import { readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import process from "node:process";
import { ConventionalChangelog } from "conventional-changelog";

const usage = `Usage: node scripts/release-changelog.mjs <tag>

Generates CHANGELOG.md for previous-tag..HEAD, commits it, and tags that commit.
If no prior semver tag exists, uses the repository's initial commit as the lower bound.`;

function fail(message) {
  console.error(message);
  process.exit(1);
}

function git(args) {
  try {
    return execFileSync("git", args, {
      cwd: process.cwd(),
      encoding: "utf8",
      stdio: ["ignore", "pipe", "pipe"],
    }).trim();
  } catch (error) {
    const stderr = error?.stderr?.toString().trim();
    fail(stderr || `git ${args.join(" ")} failed`);
  }
}

function gitOk(args) {
  try {
    execFileSync("git", args, {
      cwd: process.cwd(),
      encoding: "utf8",
      stdio: ["ignore", "pipe", "pipe"],
    });
    return true;
  } catch {
    return false;
  }
}

function tagExistsLocally(tag) {
  return gitOk(["rev-parse", "-q", "--verify", `refs/tags/${tag}`]);
}

function tagExistsOnOrigin(tag) {
  if (!gitOk(["remote", "get-url", "origin"])) {
    return false;
  }
  const out = git(["ls-remote", "--tags", "--refs", "origin", `refs/tags/${tag}`]);
  return out.length > 0;
}

function parseTag(argv) {
  let args = argv.slice(2);
  if (args[0] === "--") {
    args = args.slice(1);
  }
  const tag = args[0];
  if (!tag || tag === "--help" || tag === "-h") {
    console.log(usage);
    process.exit(tag ? 0 : 1);
  }
  if (args.length !== 1) {
    console.log(usage);
    process.exit(1);
  }
  if (/\s/.test(tag)) {
    fail(`invalid tag "${tag}"`);
  }
  return tag;
}

function isSemverTag(tag) {
  return /^v?\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/.test(tag);
}

function versionFromTag(tag) {
  if (/^v\d/.test(tag)) {
    return tag.slice(1);
  }
  return tag;
}

function findPreviousBoundary(targetTag) {
  const out = git(["tag", "--merged", "HEAD", "--sort=-v:refname"]);
  const tags = out
    .split("\n")
    .map((tag) => tag.trim())
    .filter((tag) => tag.length > 0 && isSemverTag(tag) && tag !== targetTag);

  if (tags.length > 0) {
    return { ref: tags[0], label: tags[0] };
  }

  const initialCommit = git(["rev-list", "--max-parents=0", "--reverse", "HEAD"])
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line.length > 0)[0];

  if (!initialCommit) {
    fail("failed to resolve repository initial commit");
  }

  return { ref: initialCommit, label: `${initialCommit} (initial commit)` };
}

async function generateReleaseSection(previousRef, targetTag) {
  const generator = new ConventionalChangelog(process.cwd())
    .loadPreset("conventionalcommits")
    .readPackage()
    .readRepository()
    .context({
      currentTag: targetTag,
      previousTag: previousRef,
      version: versionFromTag(targetTag),
    })
    .commits({ from: previousRef });

  let output = "";
  for await (const chunk of generator.write()) {
    output += chunk;
  }
  const releaseSection = output.trim();
  if (!releaseSection) {
    fail(`no changelog entries found between ${previousRef} and HEAD`);
  }
  return releaseSection;
}

async function updateChangelogFile(releaseSection, targetTag) {
  const changelogPath = path.join(process.cwd(), "CHANGELOG.md");
  let existing = "";

  try {
    existing = await readFile(changelogPath, "utf8");
  } catch (error) {
    if (error?.code !== "ENOENT") {
      throw error;
    }
  }

  const version = versionFromTag(targetTag);
  if (existing.includes(`## [${version}]`)) {
    fail(`CHANGELOG.md already contains version ${version}`);
  }

  const next = existing.trim().length > 0
    ? `${releaseSection}\n\n${existing.trimStart()}`
    : `${releaseSection}\n`;

  await writeFile(changelogPath, next, "utf8");
}

async function main() {
  const targetTag = parseTag(process.argv);

  if (tagExistsLocally(targetTag)) {
    fail(`tag already exists locally: ${targetTag}`);
  }
  if (tagExistsOnOrigin(targetTag)) {
    fail(`tag already exists on origin: ${targetTag}`);
  }

  const previous = findPreviousBoundary(targetTag);
  const releaseSection = await generateReleaseSection(previous.ref, targetTag);
  await updateChangelogFile(releaseSection, targetTag);

  git(["add", "--", "CHANGELOG.md"]);
  git([
    "commit",
    "-m",
    `docs(changelog): update changelog for ${targetTag}`,
    "--",
    "CHANGELOG.md",
  ]);
  git(["tag", "-a", targetTag, "-m", `Release ${targetTag}`]);

  console.log(`CHANGELOG.md updated for ${previous.label}..${targetTag}`);
  console.log(`Committed CHANGELOG.md and created tag ${targetTag}`);
}

main().catch((error) => {
  fail(error?.message || String(error));
});
