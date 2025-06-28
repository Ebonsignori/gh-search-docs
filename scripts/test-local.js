#!/usr/bin/env node

import { execSync } from "child_process";
import fs from "fs";
import dotenv from "dotenv";

// Load environment variables from .env file
dotenv.config();

console.log("🧪 Testing Release Analysis Locally");
console.log("=====================================");

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

// Validate token format
console.log("\n🔑 Validating GitHub Token...");
const token = process.env.GITHUB_TOKEN;
if (token.startsWith("ghp_") || token.startsWith("github_pat_")) {
  console.log("✅ Token format looks valid");
} else {
  console.log(
    "⚠️  Token format may be invalid (should start with 'ghp_' or 'github_pat_')",
  );
}

// Check if we have dependencies installed
if (!fs.existsSync("node_modules")) {
  console.log("📦 Installing dependencies...");
  try {
    execSync("npm install", { stdio: "inherit" });
  } catch (error) {
    console.error("❌ Failed to install dependencies");
    process.exit(1);
  }
}

// Get current git status
console.log("\n📊 Git Status:");
try {
  const lastTag = execSync("git describe --tags --abbrev=0", {
    encoding: "utf8",
  }).trim();
  console.log(`Last tag: ${lastTag}`);
} catch (error) {
  console.log("Last tag: None found");
}

try {
  const branch = execSync("git branch --show-current", {
    encoding: "utf8",
  }).trim();
  console.log(`Current branch: ${branch}`);
} catch (error) {
  console.log("Current branch: Unable to determine");
}

// Check for uncommitted changes
try {
  const status = execSync("git status --porcelain", {
    encoding: "utf8",
  }).trim();
  if (status) {
    console.log("⚠️  Uncommitted changes detected:");
    console.log(status);
  } else {
    console.log("✅ Working directory clean");
  }
} catch (error) {
  console.log("⚠️  Unable to check git status");
}

console.log("\n🤖 Running Release Analysis...");
console.log(
  "This will analyze changes since the last tag and use GitHub Models AI",
);

try {
  // Run the analysis script
  execSync("node analyze-release.js", { stdio: "inherit" });

  // Check if analysis file was created
  if (fs.existsSync("release-analysis.json")) {
    console.log("\n📄 Analysis Results:");
    const results = JSON.parse(
      fs.readFileSync("release-analysis.json", "utf8"),
    );

    console.log(`Current Version: ${results.current_version}`);
    console.log(`New Version: ${results.new_version || "No release needed"}`);
    console.log(`Version Bump: ${results.version_bump || "None"}`);
    console.log(`Changed Files: ${results.changed_files.length}`);

    if (results.reasoning) {
      console.log(`\nReasoning: ${results.reasoning}`);
    }

    if (results.release_notes) {
      console.log("\nRelease Notes:");
      console.log("==============");
      console.log(results.release_notes);
    }
  }

  console.log("\n✅ Test completed successfully!");
} catch (error) {
  console.error("\n❌ Test failed:", error.message);
  process.exit(1);
}
