const assert = require("node:assert/strict");
const { test } = require("node:test");
const { cliEntryPoint } = require("./cli.cjs");

for (const [platform, executable, expected] of [
  [
    "darwin",
    "/test folder/Visual Studio Code.app/Contents/MacOS/Electron",
    "/test folder/Visual Studio Code.app/Contents/Resources/app/out/cli.js",
  ],
  [
    "linux",
    "/test folder/VSCode/code",
    "/test folder/VSCode/resources/app/out/cli.js",
  ],
  [
    "win32",
    "C:\\test folder\\VSCode\\Code.exe",
    "C:\\test folder\\VSCode\\resources\\app\\out\\cli.js",
  ],
]) {
  test(`Resolve the shell-free CLI entry point for ${platform}.`, () => {
    assert.equal(cliEntryPoint(executable, platform), expected);
    assert.equal(cliEntryPoint(executable, platform), expected);
  });
}
