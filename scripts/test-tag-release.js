#!/usr/bin/env node

import { execSync } from "child_process";
import fs from "fs";
import dotenv from "dotenv";

// Load environment variables from .env file
dotenv.config();

console.log("🏷️ Testing Tag-Based Release Analysis");
console.log("====================================");

// Check if we're in a git repository
try {
  execSync("git rev-parse --is-inside-work-tree", { stdio: "ignore" });
} catch (error) {
  console.error("❌ Not in a git repository");
  process.exit(1);
}

// Check if GITHUB_TOKEN is set
if (!process.env.GITHUB_TOKEN) {
  console.error("❌ GITHUB_TOKEN environment variable is required");
  console.log("Set it by:");
  console.log("1. Copy .env.example to .env: cp .env.example .env");
  console.log("2. Edit .env and add your GitHub token");
  console.log("3. Or set it with: export GITHUB_TOKEN='your_github_token'");
  process.exit(1);
}

// Check if dependencies are installed
if (!fs.existsSync("node_modules")) {
  console.log("📦 Installing dependencies...");
  try {
    execSync("npm install", { stdio: "inherit" });
  } catch (error) {
    console.error("❌ Failed to install dependencies");
    process.exit(1);
  }
}

// Get the test version from command line args
const args = process.argv.slice(2);
let testVersion = args[0];

if (!testVersion) {
  // Try to get current version and suggest next patch
  try {
    const lastTag = execSync("git describe --tags --abbrev=0", { encoding: "utf8" }).trim();
    const currentVersion = lastTag.replace(/^v/, "");
    const [major, minor, patch] = currentVersion.split(".").map(Number);
    testVersion = `${major}.${minor}.${patch + 1}`;
    console.log(`💡 No version specified, suggesting patch bump: ${testVersion}`);
  } catch (error) {
    testVersion = "1.0.0";
    console.log(`💡 No version specified and no tags found, using: ${testVersion}`);
  }
}

// Validate version format
if (!/^\d+\.\d+\.\d+$/.test(testVersion)) {
  console.error(`❌ Invalid version format: ${testVersion}`);
  console.log("Use format: X.Y.Z (e.g., 1.2.3)");
  process.exit(1);
}

console.log(`\n🎯 Testing tag-based release for version: ${testVersion}`);

// Get current git status
console.log("\n📊 Git Status:");
try {
  const lastTag = execSync("git describe --tags --abbrev=0", { encoding: "utf8" }).trim();
  console.log(`Last tag: ${lastTag}`);
} catch (error) {
  console.log("Last tag: None found");
}

try {
  const branch = execSync("git branch --show-current", { encoding: "utf8" }).trim();
  console.log(`Current branch: ${branch}`);
} catch (error) {
  console.log("Current branch: Unable to determine");
}

// Check for uncommitted changes
try {
  const status = execSync("git status --porcelain", { encoding: "utf8" }).trim();
  if (status) {
    console.log("⚠️  Uncommitted changes detected:");
    console.log(status);
    console.log("Note: Tag-based releases analyze committed changes only");
  } else {
    console.log("✅ Working directory clean");
  }
} catch (error) {
  console.log("⚠️  Unable to check git status");
}

console.log("\n🤖 Simulating Tag-Based Release Analysis...");
console.log("This will analyze changes and generate release notes for the specified version");

try {
  // Set environment variables to simulate tag-based release
  process.env.TAG_BASED_RELEASE = "true";
  process.env.TAG_VERSION = testVersion;

  // Run the analysis script
  execSync("node analyze-release.js", { stdio: "inherit" });

  // Check if analysis file was created
  if (fs.existsSync("release-analysis.json")) {
    console.log("\n📄 Tag-Based Release Results:");
    const results = JSON.parse(fs.readFileSync("release-analysis.json", "utf8"));

    console.log(`Previous Version: ${results.current_version}`);
    console.log(`Tagged Version: ${results.new_version}`);
    console.log(`Version Bump Type: ${results.version_bump}`);
    console.log(`Changed Files: ${results.changed_files.length}`);
    console.log(`Reasoning: ${results.reasoning}`);

    if (results.release_notes) {
      console.log("\nGenerated Release Notes:");
      console.log("========================");
      console.log(results.release_notes);
    }

    console.log("\n💡 This simulates what would happen if you:");
    console.log(`   git tag v${testVersion}`);
    console.log(`   git push origin v${testVersion}`);
    console.log("\nThe workflow would then create a GitHub release with these notes.");
  }

  console.log("\n✅ Tag-based release test completed successfully!");

} catch (error) {
  console.error("\n❌ Test failed:", error.message);
  process.exit(1);
} finally {
  // Clean up environment variables
  delete process.env.TAG_BASED_RELEASE;
  delete process.env.TAG_VERSION;
}
