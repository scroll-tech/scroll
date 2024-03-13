import { URL } from "url";
import { readFileSync, writeFileSync } from "fs";

const versionFilePath = new URL(
  "../../common/version/version.go",
  import.meta.url
).pathname;

try {
  const versionFileContent = readFileSync(versionFilePath, { encoding: "utf-8" });

  const currentVersion = versionFileContent.match(
    /var tag = "(?<version>v(?<major>\d+)\.(?<minor>\d+)\.(?<patch>\d+))"/
  );

  if (!currentVersion) {
    throw new Error("Failed to parse version in version.go file");
  }

  const major = parseInt(currentVersion.groups.major, 10);
  const minor = parseInt(currentVersion.groups.minor, 10);
  const patch = parseInt(currentVersion.groups.patch, 10);

  // Increment the patch version
  const newPatch = patch + 1;
  const newVersion = `v${major}.${minor}.${newPatch}`;

  console.log(`Bump version from ${currentVersion.groups.version} to ${newVersion}`);

  const updatedContent = versionFileContent.replace(
    `var tag = "${currentVersion.groups.version}"`,
    `var tag = "${newVersion}"`
  );

  writeFileSync(versionFilePath, updatedContent);
} catch (err) {
  console.error(err.message);
  process.exit(1);
}