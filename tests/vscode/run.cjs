const { execFileSync } = require("node:child_process");
const fs = require("node:fs/promises");
const os = require("node:os");
const path = require("node:path");
const { downloadAndUnzipVSCode, runTests } = require("@vscode/test-electron");
const { cliEntryPoint } = require("./cli.cjs");

const languages = [
  "terraform",
  "terraform-mock",
  "terraform-search",
  "terraform-test",
  "terraform-vars",
];

async function main() {
  const root = path.resolve(__dirname, "../..");
  const temporary = await fs.mkdtemp(
    path.join(os.tmpdir(), "tf4fmt editor tests "),
  );
  try {
    const vscodeExecutablePath = await downloadAndUnzipVSCode("stable");
    const extensions = path.join(temporary, "extensions");
    const userData = path.join(temporary, "profile");
    const launcher =
      process.platform === "win32"
        ? await fs.readFile(
            path.join(path.dirname(vscodeExecutablePath), "bin/code.cmd"),
            "utf8",
          )
        : undefined;
    const cli = cliEntryPoint(vscodeExecutablePath, process.platform, launcher);
    for (const extension of [
      "hashicorp.terraform",
      "jkillian.custom-local-formatters",
    ]) {
      execFileSync(
        vscodeExecutablePath,
        [
          cli,
          "--extensions-dir",
          extensions,
          "--user-data-dir",
          userData,
          "--install-extension",
          extension,
        ],
        {
          env: { ...process.env, ELECTRON_RUN_AS_NODE: "1" },
          stdio: "inherit",
        },
      );
    }
    for (const directory of ["Submodules", "submodules"]) {
      const casing = directory === "Submodules" ? "uppercase" : "lowercase";
      const workspace = path.join(temporary, `${casing} consumer`);
      await fs.mkdir(path.join(workspace, directory), { recursive: true });
      await fs.symlink(
        root,
        path.join(workspace, directory, "tf4fmt"),
        process.platform === "win32" ? "junction" : "dir",
      );
      const settings = {
        "customLocalFormatters.formatters": [
          {
            command: `go -C ${directory}/tf4fmt run ./src/cmd/tf4fmt -`,
            languages,
          },
        ],
        "files.insertFinalNewline": true,
        "terraform.languageServer.enable": false,
      };
      for (const language of languages) {
        settings[`[${language}]`] = {
          "editor.defaultFormatter": "jkillian.custom-local-formatters",
          "editor.detectIndentation": false,
          "editor.formatOnSave": true,
          "editor.formatOnSaveMode": "file",
          "editor.insertSpaces": true,
          "editor.tabSize": 4,
        };
      }
      await fs.mkdir(path.join(workspace, ".vscode"));
      await fs.writeFile(
        path.join(workspace, ".vscode/settings.json"),
        `${JSON.stringify(settings, null, 2)}\n`,
      );
      await runTests({
        vscodeExecutablePath,
        extensionDevelopmentPath: __dirname,
        extensionTestsPath: path.join(__dirname, "save.test.cjs"),
        launchArgs: [
          workspace,
          "--extensions-dir",
          extensions,
          "--user-data-dir",
          userData,
          "--disable-workspace-trust",
          "--skip-welcome",
          "--skip-release-notes",
          "--no-sandbox",
        ],
      });
    }
  } finally {
    await fs.rm(temporary, { recursive: true, force: true });
  }
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
