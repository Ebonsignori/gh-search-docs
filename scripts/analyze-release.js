#!/usr/bin/env node

import OpenAI from "openai";
import { execSync } from "child_process";
import fs from "fs";
import dotenv from "dotenv";

// Load environment variables from .env file
dotenv.config();

const token = process.env["GITHUB_TOKEN"];
const endpoint =
  process.env["GITHUB_MODELS_ENDPOINT"] || "https://models.github.ai/inference";
const modelName = process.env["GITHUB_MODELS_MODEL"] || "openai/o4-mini";

// Check if this is a tag-based release
const isTagBasedRelease = process.env["TAG_BASED_RELEASE"] === "true";
const tagVersion = process.env["TAG_VERSION"];

if (!token) {
  console.error("GITHUB_TOKEN environment variable is required");
  process.exit(1);
}

class ReleaseAnalyzer {
  constructor() {
    this.client = new OpenAI({ baseURL: endpoint, apiKey: token });
  }

  async getLastTag() {
    try {
      const lastTag = execSync("git describe --tags --abbrev=0", {
        encoding: "utf8",
      }).trim();
      console.log(`Last tag found: ${lastTag}`);
      return lastTag;
    } catch (error) {
      console.log("No previous tags found, analyzing all changes");
      return null;
    }
  }

  async getChangedFiles(lastTag) {
    try {
      const command = lastTag
        ? `git diff --name-only ${lastTag}..HEAD`
        : "git ls-files";

      const files = execSync(command, { encoding: "utf8" })
        .trim()
        .split("\n")
        .filter((file) => file.length > 0);

      console.log(`Found ${files.length} changed files`);
      return files;
    } catch (error) {
      console.error("Error getting changed files:", error.message);
      return [];
    }
  }

  async getFileDiff(file, lastTag) {
    try {
      if (!lastTag) {
        // If no previous tag, show the entire file content (limited)
        if (fs.existsSync(file)) {
          const content = fs.readFileSync(file, "utf8");
          return content.slice(0, 5000); // Limit to first 5000 chars
        }
        return "";
      }

      const diff = execSync(`git diff ${lastTag}..HEAD -- "${file}"`, {
        encoding: "utf8",
      });
      return diff;
    } catch (error) {
      console.log(`Could not get diff for ${file}:`, error.message);
      return "";
    }
  }

  async analyzeChanges(changedFiles, lastTag) {
    if (changedFiles.length === 0) {
      console.log("No changes detected");
      return null;
    }

    // Get diffs for all changed files
    const fileDiffs = {};
    for (const file of changedFiles) {
      const diff = await this.getFileDiff(file, lastTag);
      if (diff) {
        fileDiffs[file] = diff;
      }
    }

    // Create a summary of changes for the AI
    const changesSummary = Object.entries(fileDiffs)
      .map(([file, diff]) => `=== ${file} ===\n${diff}`)
      .join("\n\n")
      .slice(0, 15000); // Limit total input size

    const prompt = `You are analyzing code changes for a GitHub CLI extension called "gh-ask-docs" to determine the appropriate semantic version bump and generate release notes.

This is a Go-based CLI extension that helps users ask questions about documentation.

Changed files since last release:
${changedFiles.join(", ")}

File changes and diffs:
${changesSummary}

Based on these changes, please provide a JSON response with the following structure:
{
  "version_bump": "major|minor|patch",
  "reasoning": "Brief explanation of why this version bump was chosen",
  "release_notes": "Markdown-formatted release notes describing the changes"
}

Version bump guidelines:
- MAJOR: Breaking changes to the CLI interface, removed commands, or incompatible changes
- MINOR: New features, new commands, or backwards-compatible functionality additions
- PATCH: Bug fixes, documentation updates, dependency updates, or minor improvements

The release notes should be concise but informative, using bullet points and proper markdown formatting.

IMPORTANT: Respond only with valid JSON. Do not include any explanatory text before or after the JSON.`;

    try {
      console.log("Analyzing changes with GitHub Models...");
      const response = await this.client.chat.completions.create({
        messages: [
          {
            role: "developer",
            content:
              "You are a helpful assistant that analyzes code changes for semantic versioning and release notes generation.",
          },
          { role: "user", content: prompt },
        ],
        model: modelName,
      });

      const content = response.choices[0].message.content;
      console.log("AI Analysis received");

      // Try to parse JSON from the response
      const jsonMatch = content.match(/\{[\s\S]*\}/);
      if (jsonMatch) {
        const parsed = JSON.parse(jsonMatch[0]);

        // Validate required fields
        if (
          !parsed.version_bump ||
          !["major", "minor", "patch"].includes(parsed.version_bump)
        ) {
          throw new Error(`Invalid version_bump: ${parsed.version_bump}`);
        }
        if (!parsed.reasoning || typeof parsed.reasoning !== "string") {
          throw new Error("Missing or invalid reasoning");
        }
        if (!parsed.release_notes || typeof parsed.release_notes !== "string") {
          throw new Error("Missing or invalid release_notes");
        }

        return parsed;
      } else {
        console.error("AI Response content:", content);
        throw new Error("Could not parse JSON from AI response");
      }
    } catch (error) {
      console.error("Error analyzing changes:", error.message);
      return null;
    }
  }

  async getCurrentVersion() {
    try {
      const lastTag = await this.getLastTag();
      if (lastTag) {
        // Remove 'v' prefix if present
        return lastTag.replace(/^v/, "");
      }
      return "0.0.0";
    } catch (error) {
      return "0.0.0";
    }
  }

  calculateNewVersion(currentVersion, versionBump) {
    // Validate version format
    if (!/^\d+\.\d+\.\d+$/.test(currentVersion)) {
      throw new Error(`Invalid current version format: ${currentVersion}`);
    }

    const [major, minor, patch] = currentVersion.split(".").map(Number);

    // Validate parsed numbers
    if (isNaN(major) || isNaN(minor) || isNaN(patch)) {
      throw new Error(
        `Failed to parse version numbers from: ${currentVersion}`,
      );
    }

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

  async run() {
    try {
      console.log("Starting release analysis...");

      // Handle tag-based releases
      if (isTagBasedRelease && tagVersion) {
        console.log(`🏷️ Tag-based release detected: v${tagVersion}`);
        return await this.handleTagBasedRelease(tagVersion);
      }

      const lastTag = await this.getLastTag();
      const changedFiles = await this.getChangedFiles(lastTag);

      if (changedFiles.length === 0) {
        console.log("No changes detected since last release");

        // Write empty result file
        const emptyResult = {
          current_version: await this.getCurrentVersion(),
          new_version: null,
          version_bump: null,
          reasoning: "No changes detected",
          release_notes: null,
          changed_files: [],
        };
        fs.writeFileSync(
          "release-analysis.json",
          JSON.stringify(emptyResult, null, 2),
        );

        process.exit(0);
      }

      const analysis = await this.analyzeChanges(changedFiles, lastTag);

      if (!analysis) {
        console.error("Failed to analyze changes");
        process.exit(1);
      }

      const currentVersion = await this.getCurrentVersion();
      const newVersion = this.calculateNewVersion(
        currentVersion,
        analysis.version_bump,
      );

      const result = {
        current_version: currentVersion,
        new_version: newVersion,
        version_bump: analysis.version_bump,
        reasoning: analysis.reasoning,
        release_notes: analysis.release_notes,
        changed_files: changedFiles,
      };

      // Output results
      console.log("\n=== Release Analysis Results ===");
      console.log(`Current Version: ${currentVersion}`);
      console.log(`Recommended Version: ${newVersion}`);
      console.log(`Version Bump: ${analysis.version_bump}`);
      console.log(`Reasoning: ${analysis.reasoning}`);
      console.log("\nRelease Notes:");
      console.log(analysis.release_notes);

      // Write results to file for GitHub Actions to consume
      fs.writeFileSync(
        "release-analysis.json",
        JSON.stringify(result, null, 2),
      );

      // Set GitHub Actions outputs
      if (process.env.GITHUB_OUTPUT) {
        const outputs = [
          `new-version=${newVersion}`,
          `version-bump=${analysis.version_bump}`,
          `release-notes<<EOF\n${analysis.release_notes}\nEOF`,
        ];

        try {
          fs.appendFileSync(
            process.env.GITHUB_OUTPUT,
            outputs.join("\n") + "\n",
          );
          console.log("GitHub Actions outputs written");
        } catch (error) {
          console.error(
            "Failed to write GitHub Actions outputs:",
            error.message,
          );
        }
      }

      console.log(
        "\nAnalysis complete! Results written to release-analysis.json",
      );
    } catch (error) {
      console.error("Error during release analysis:", error.message);
      process.exit(1);
    }
  }

  async handleTagBasedRelease(newVersion) {
    try {
      const currentVersion = await this.getCurrentVersion();
      const versionBump = this.determineVersionBump(currentVersion, newVersion);

      console.log(`Generating release notes for tag v${newVersion}`);
      console.log(
        `Version bump: ${currentVersion} -> ${newVersion} (${versionBump})`,
      );

      const lastTag = await this.getLastTag();
      const changedFiles = await this.getChangedFiles(lastTag);

      if (changedFiles.length === 0) {
        console.log("No changes detected, creating minimal release notes");
        const result = {
          current_version: currentVersion,
          new_version: newVersion,
          version_bump: versionBump,
          reasoning: `Tag-based ${versionBump} release`,
          release_notes: `## Release v${newVersion}\n\nTag-based release with version ${newVersion}.`,
          changed_files: [],
        };

        this.writeResults(result);
        return;
      }

      // Generate AI release notes for the tag
      const releaseNotes = await this.generateReleaseNotesOnly(
        changedFiles,
        lastTag,
        newVersion,
        versionBump,
      );

      const result = {
        current_version: currentVersion,
        new_version: newVersion,
        version_bump: versionBump,
        reasoning: `Tag-based ${versionBump} release`,
        release_notes: releaseNotes,
        changed_files: changedFiles,
      };

      this.writeResults(result);

      console.log("\n=== Tag-Based Release Results ===");
      console.log(`Previous Version: ${currentVersion}`);
      console.log(`Tagged Version: ${newVersion}`);
      console.log(`Version Bump: ${versionBump}`);
      console.log("\nRelease Notes:");
      console.log(releaseNotes);
    } catch (error) {
      console.error("Error handling tag-based release:", error.message);
      process.exit(1);
    }
  }

  determineVersionBump(currentVersion, newVersion) {
    const [currentMajor, currentMinor, currentPatch] = currentVersion
      .split(".")
      .map(Number);
    const [newMajor, newMinor, newPatch] = newVersion.split(".").map(Number);

    if (newMajor > currentMajor) return "major";
    if (newMinor > currentMinor) return "minor";
    if (newPatch > currentPatch) return "patch";

    // If versions are equal or new is lower, still default to patch
    return "patch";
  }

  async generateReleaseNotesOnly(
    changedFiles,
    lastTag,
    newVersion,
    versionBump,
  ) {
    const fileDiffs = {};
    for (const file of changedFiles) {
      const diff = await this.getFileDiff(file, lastTag);
      if (diff) {
        fileDiffs[file] = diff;
      }
    }

    const changesSummary = Object.entries(fileDiffs)
      .map(([file, diff]) => `=== ${file} ===\n${diff}`)
      .join("\n\n")
      .slice(0, 15000);

    const prompt = `You are generating release notes for a GitHub CLI extension called "gh-ask-docs" version ${newVersion}.

This is a ${versionBump} release from the previous version. The version was manually tagged as v${newVersion}.

Changed files since last release:
${changedFiles.join(", ")}

File changes and diffs:
${changesSummary}

Generate comprehensive, well-formatted release notes in Markdown. Include:
- A brief summary of what changed
- Bullet points for key changes
- Any breaking changes (if major version)
- Bug fixes and improvements

Focus only on user-facing changes. Use proper Markdown formatting with headings, bullet points, etc.

Return only the release notes content, no JSON or other formatting.`;

    try {
      const response = await this.client.chat.completions.create({
        messages: [
          {
            role: "developer",
            content:
              "You are a helpful assistant that generates release notes for software releases.",
          },
          { role: "user", content: prompt },
        ],
        model: modelName,
      });

      return response.choices[0].message.content.trim();
    } catch (error) {
      console.error("Error generating release notes:", error.message);
      return `## Release v${newVersion}\n\n${versionBump.charAt(0).toUpperCase() + versionBump.slice(1)} release with ${changedFiles.length} changed files.`;
    }
  }

  writeResults(result) {
    // Write results to file for GitHub Actions to consume
    fs.writeFileSync("release-analysis.json", JSON.stringify(result, null, 2));

    // Set GitHub Actions outputs
    if (process.env.GITHUB_OUTPUT) {
      const outputs = [
        `new-version=${result.new_version}`,
        `version-bump=${result.version_bump}`,
        `release-notes<<EOF\n${result.release_notes}\nEOF`,
      ];

      try {
        fs.appendFileSync(process.env.GITHUB_OUTPUT, outputs.join("\n") + "\n");
        console.log("GitHub Actions outputs written");
      } catch (error) {
        console.error("Failed to write GitHub Actions outputs:", error.message);
      }
    }

    console.log(
      "\nAnalysis complete! Results written to release-analysis.json",
    );
  }
}

// Run the analyzer
const analyzer = new ReleaseAnalyzer();
analyzer.run();
