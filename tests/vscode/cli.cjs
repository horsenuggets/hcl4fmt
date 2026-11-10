const path = require("node:path");

exports.cliEntryPoint = (executable, platform) => {
  const paths = platform === "win32" ? path.win32 : path.posix;
  const resources = platform === "darwin" ? "../Resources" : "resources";
  return paths.join(paths.dirname(executable), resources, "app/out/cli.js");
};
