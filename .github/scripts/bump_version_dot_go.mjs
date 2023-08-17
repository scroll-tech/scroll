import { URL } from "url";
import { readFileSync, writeFileSync } from "fs";

const versionFilePath = new URL(
  "../../common/version/version.go",
  import.meta.url
).pathname;

const versionFileContent = readFileSync(versionFilePath, { encoding: "utf-8" });

const currentVersion = versionFileContent.match(
  /var tag = "(?<version>v(?<major>\d+)\.(?<minor>\d+)\.(?<patch>\d+))"/
);

try {
  parseInt(currentVersion.groups.major);
  parseInt(currentVersion.groups.minor);
  parseInt(currentVersion.groups.patch);
} catch (err) {
  console.error(new Error("Failed to parse version in version.go file"));
  throw err;
}

// prettier-ignore
const newVersion = `v${currentVersion.groups.major}.${currentVersion.groups.minor}.${parseInt(currentVersion.groups.patch) + 1}`;

console.log(
  `Bump version from ${currentVersion.groups.version} to ${newVersion}`
);

writeFileSync(
  versionFilePath,
  versionFileContent.replace(
    `var tag = "${currentVersion.groups.version}"`,
    `var tag = "${newVersion}"`
  )
);
