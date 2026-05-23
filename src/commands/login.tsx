import { render } from "ink";
import type React from "react";
import { getConfigPath, loadConfig, saveConfig, updateToken } from "../config/config.ts";
import { Status } from "../ui/Status.tsx";

export interface LoginOptions {
  token?: string;
  config?: string;
}

function renderUI(node: React.ReactElement): void {
  const inst = render(node, { stdout: process.stderr });
  inst.unmount();
}

export async function runLogin(opts: LoginOptions): Promise<number> {
  const token = (opts.token ?? "").trim();
  if (token === "") {
    renderUI(
      <Status emoji="❌" color="red">
        Token not found
      </Status>,
    );
    process.stderr.write("\n");
    process.stderr.write("Visit https://clippingkk.annatarhe.com and login.\n");
    process.stderr.write("Navigate to your profile page and open the 'API Token' dialog.\n");
    process.stderr.write("Copy the token and run:\n");
    process.stderr.write("  ck-cli login --token YOUR_TOKEN\n\n");
    return 1;
  }

  const configPath = getConfigPath(opts.config ?? "");
  const cfg = await loadConfig(configPath);
  updateToken(cfg, token);
  await saveConfig(cfg, configPath);

  renderUI(
    <Status emoji="✅" color="green">
      Successfully logged in!
    </Status>,
  );
  process.stderr.write("\n");
  process.stderr.write("You can now synchronize your Kindle clippings by running:\n");
  process.stderr.write("  ck-cli parse --input /path/to/My\\ Clippings.txt --output http\n\n");
  return 0;
}
