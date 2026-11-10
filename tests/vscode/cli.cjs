const path = require("node:path");

exports.cliEntryPoint = (executable, platform, launcher) => {
  if (platform === "win32") {
    // Windows launchers can route the CLI through a versioned package directory.
    const match = launcher.match(
      /"%~dp0([^"\r\n]*\\resources\\app\\out\\cli\.js)"/i,
    );
    if (!match) {
      throw new Error(
        "Windows VS Code launcher does not identify a CLI entry point.",
      );
    }
    return path.win32.resolve(path.win32.dirname(executable), "bin", match[1]);
  }
  const resources = platform === "darwin" ? "../Resources" : "resources";
  return path.posix.join(
    path.posix.dirname(executable),
    resources,
    "app/out/cli.js",
  );
};
