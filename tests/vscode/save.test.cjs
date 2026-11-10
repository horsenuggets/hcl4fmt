const assert = require("node:assert/strict");
const path = require("node:path");
const vscode = require("vscode");

async function replace(document, text) {
  const edit = new vscode.WorkspaceEdit();
  edit.replace(
    document.uri,
    new vscode.Range(
      document.positionAt(0),
      document.positionAt(document.getText().length),
    ),
    text,
  );
  assert.equal(await vscode.workspace.applyEdit(edit), true);
}

exports.run = async () => {
  const provider = vscode.extensions.getExtension(
    "jkillian.custom-local-formatters",
  );
  assert.ok(provider, "The formatter provider must be installed.");
  await provider.activate();
  await vscode.extensions.getExtension("hashicorp.terraform").activate();
  const folder = vscode.workspace.workspaceFolders[0].uri;
  const readinessUri = vscode.Uri.joinPath(folder, "save-readiness.txt");
  await vscode.workspace.fs.writeFile(readinessUri, Buffer.from(""));
  const readinessDocument =
    await vscode.workspace.openTextDocument(readinessUri);
  await vscode.window.showTextDocument(readinessDocument);
  let ready = false;
  for (let attempt = 0; attempt < 100; attempt += 1) {
    await replace(readinessDocument, `Readiness ${attempt}`);
    await vscode.commands.executeCommand("workbench.action.files.save");
    if (readinessDocument.getText().endsWith("\n")) {
      ready = true;
      break;
    }
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  assert.equal(ready, true, "Workbench save participants must be ready.");
  const nested = vscode.Uri.joinPath(folder, "nested terraform");
  await vscode.workspace.fs.createDirectory(nested);
  const cases = [
    [
      "main.tf",
      "terraform",
      "locals {\n  value=1\n}\n",
      "locals {\n    value = 1\n}\n",
    ],
    [
      "values.tfvars",
      "terraform-vars",
      "value={\n  nested=1\n}\n",
      "value = {\n    nested = 1\n}\n",
    ],
    [
      "main.tftest.hcl",
      "terraform-test",
      'run "example" {\n  command=plan\n}\n',
      'run "example" {\n    command = plan\n}\n',
    ],
    [
      "main.tfmock.hcl",
      "terraform-mock",
      'mock_resource "example" {\n  defaults={}\n}\n',
      'mock_resource "example" {\n    defaults = {}\n}\n',
    ],
    [
      "main.tfquery.hcl",
      "terraform-search",
      'list "example" "example" {\n  provider=example\n}\n',
      'list "example" "example" {\n    provider = example\n}\n',
    ],
  ];
  for (const [filename, language, input, expected] of cases) {
    const uri = vscode.Uri.joinPath(nested, filename);
    await vscode.workspace.fs.writeFile(uri, Buffer.from("# Fixture.\n"));
    const document = await vscode.workspace.openTextDocument(uri);
    assert.equal(document.languageId, language);
    await vscode.window.showTextDocument(document);
    await replace(document, input);
    assert.equal(
      vscode.workspace.getConfiguration("editor", document).get("formatOnSave"),
      true,
    );
    await vscode.commands.executeCommand("workbench.action.files.save");
    assert.equal(
      document.getText(),
      expected,
      `${filename} must format on save.`,
    );
    assert.equal(
      Buffer.from(await vscode.workspace.fs.readFile(uri)).toString(),
      expected,
    );
    await replace(document, expected);
    await vscode.commands.executeCommand("workbench.action.files.save");
    assert.equal(
      document.getText(),
      expected,
      "Repeated saves must be idempotent.",
    );
    const invalid = 'locals {\n  value = "unterminated\n}\n';
    await replace(document, invalid);
    await vscode.commands.executeCommand("workbench.action.files.save");
    assert.equal(
      document.getText(),
      invalid,
      "Invalid input must not be replaced.",
    );
    assert.equal(
      Buffer.from(await vscode.workspace.fs.readFile(uri)).toString(),
      invalid,
      "Invalid input must be saved without replacement.",
    );
    const heredoc =
      "locals {\n  value=<<-TEXT\n  literal content\n    stays indented\n  TEXT\n}\n";
    await replace(document, heredoc);
    await vscode.commands.executeCommand("workbench.action.files.save");
    assert.ok(
      document
        .getText()
        .includes("\n  literal content\n    stays indented\n  TEXT\n"),
    );
    assert.ok(document.getText().startsWith("locals {\n    value = <<-TEXT\n"));
    console.log(`Verified explicit save for "${path.basename(uri.fsPath)}".`);
  }
};
