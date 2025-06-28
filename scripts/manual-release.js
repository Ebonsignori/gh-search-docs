#!/usr/bin/env node

import { execSync } from "child_process";
import fs from "fs";

const args = process.argv.slice(2);

if (args.length < 2) {
  console.log("Usage: node manual-release.js <version_bump> <current_version>");
  console.log("Example: node manual-release.js minor 1.2.3");
  process.exit(1);
}

const versionBump = args[0];
const currentVersion = args[1];

function calculateNewVersion(currentVersion, versionBump) {
  const [major, minor, patch] = currentVersion.split(".").map(Number);

  switch (versionBump) {
    case "major":
      return `${major + 1}.0.0`;
    case "minor":
      return `${major}.${minor + 1}.0`;
    case "patch":
      return `${major}.${minor}.${patch + 1}`;
    default:
      throw new Error(`Invalid version bump: ${versionBump}`);
  }
}

try {
  const newVersion = calculateNewVersion(currentVersion, versionBump);

  const result = {
    current_version: currentVersion,
    new_version: newVersion,
    version_bump: versionBump,
    reasoning: `Manual ${versionBump} version bump`,
    release_notes: `## Release v${newVersion}\n\nManual release triggered with ${versionBump} version bump.`,
    changed_files: []
  };

  // Write results to file for GitHub Actions to consume
  fs.writeFileSync("release-analysis.json", JSON.stringify(result, null, 2));

  // Set GitHub Actions outputs if running in CI
  if (process.env.GITHUB_OUTPUT) {
    const outputs = [
      `new-version=${newVersion}`,
      `version-bump=${versionBump}`,
      `release-notes<<EOF\n${result.release_notes}\nEOF`
    ];
    fs.appendFileSync(process.env.GITHUB_OUTPUT, outputs.join("\n") + "\n");
  }

  console.log(`Manual release configured: ${currentVersion} -> ${newVersion} (${versionBump})`);
} catch (error) {
  console.error("Error:", error.message);
  process.exit(1);
}
