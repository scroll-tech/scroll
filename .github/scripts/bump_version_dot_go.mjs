import { URL } from "url";
import { readFileSync, writeFileSync } from "fs";

const versionFilePath = new URL(
  "../../common/version/version.go",
  import.meta.url
).pathname;

const versionFileContent = readFileSync(versionFilePath, { encoding: "utf-8" });

const currentVersionMatch = versionFileContent.match(
  /var tag = "(?<version>v(\d+)\.(\d+)\.(\d+))"/
);

if (!currentVersionMatch) {
  throw new Error("Failed to parse version in version.go file");
}

const [, major, minor, patch] = currentVersionMatch;

const newPatch = parseInt(patch) + 1;

const newVersion = `v${major}.${minor}.${newPatch}`;

console.log(
  `Bump version from ${currentVersionMatch[1]} to ${newVersion}`
);

const newVersionFileContent = versionFileContent.replace(
  currentVersionMatch[0],
  `var tag = "${newVersion}"`
);

writeFileSync(versionFilePath, newVersionFileContent);
