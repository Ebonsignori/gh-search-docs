#!/usr/bin/env node

import fs from "fs";
import { execSync } from "child_process";

console.log("🚀 Setting up gh-search-docs Release Automation");
console.log("===========================================");

// Check if we're in the scripts directory
if (!fs.existsSync("package.json")) {
  console.error("❌ Please run this script from the scripts directory");
  console.log("cd scripts && node setup.js");
  process.exit(1);
}

// Install dependencies
console.log("\n📦 Installing dependencies...");
try {
  execSync("npm install", { stdio: "inherit" });
  console.log("✅ Dependencies installed");
} catch (error) {
  console.error("❌ Failed to install dependencies");
  process.exit(1);
}

// Create .env file if it doesn't exist
if (!fs.existsSync(".env")) {
  console.log("\n📄 Creating .env file...");
  try {
    fs.copyFileSync(".env.example", ".env");
    console.log("✅ Created .env file from .env.example");
    console.log("⚠️  Please edit .env and add your GitHub token");
  } catch (error) {
    console.error("❌ Failed to create .env file:", error.message);
  }
} else {
  console.log("\n📄 .env file already exists");
}

// Check git repository status
console.log("\n📊 Checking git repository...");
try {
  execSync("git rev-parse --is-inside-work-tree", { stdio: "ignore" });
  console.log("✅ Git repository detected");

  try {
    const lastTag = execSync("git describe --tags --abbrev=0", {
      encoding: "utf8",
    }).trim();
    console.log(`✅ Last tag found: ${lastTag}`);
  } catch (error) {
    console.log("ℹ️  No previous tags found (this will be the first release)");
  }
} catch (error) {
  console.error("❌ Not in a git repository");
  console.log(
    "Please run this from your git repository root/scripts directory",
  );
}

console.log("\n🎯 Next Steps:");
console.log("1. Edit .env file and add your GitHub token:");
console.log("   GITHUB_TOKEN=your_github_token_here");
console.log("2. Test the setup:");
console.log("   npm test");
console.log("3. The workflow will automatically trigger on pushes to main");
console.log("4. Or manually trigger via GitHub Actions → Release workflow");

console.log("\n✅ Setup complete!");
