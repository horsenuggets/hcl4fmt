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
]) {
  test(`Resolve the shell-free CLI entry point for ${platform}.`, () => {
    assert.equal(cliEntryPoint(executable, platform), expected);
    assert.equal(cliEntryPoint(executable, platform), expected);
  });
}

for (const directory of ["", "package-version\\", "package with spaces\\"]) {
  test(`Resolve the Windows launcher route through "${directory}".`, () => {
    const executable = "C:\\test folder\\VSCode\\Code.exe";
    const launcher = `@echo off\n"%~dp0..\\Code.exe" "%~dp0..\\${directory}resources\\app\\out\\cli.js" %*\n`;
    const expected = `C:\\test folder\\VSCode\\${directory}resources\\app\\out\\cli.js`;
    assert.equal(cliEntryPoint(executable, "win32", launcher), expected);
    assert.equal(cliEntryPoint(executable, "win32", launcher), expected);
  });
}

test("Reject Windows launchers without an identifiable CLI route.", () => {
  assert.throws(
    () => cliEntryPoint("C:\\test folder\\Code.exe", "win32", "@echo off\n"),
    /does not identify a CLI entry point/,
  );
});
