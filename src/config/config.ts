import { mkdir, readFile, stat, writeFile } from "node:fs/promises";
import { homedir } from "node:os";
import { dirname, join } from "node:path";
import { parse as parseToml, stringify as stringifyToml } from "smol-toml";

export const DEFAULT_ENDPOINT = "https://clippingkk-api.annatarhe.com/api/v2/graphql";
export const CONFIG_FILE_NAME = ".ck-cli.toml";

export interface HTTPConfig {
  endpoint: string;
  headers: Record<string, string>;
}

export interface Config {
  http: HTTPConfig;
}

export function newConfig(): Config {
  return {
    http: {
      endpoint: DEFAULT_ENDPOINT,
      headers: {},
    },
  };
}

export function updateToken(cfg: Config, token: string): void {
  cfg.http.headers.Authorization = `X-CLI ${token}`;
}

export function hasToken(cfg: Config): boolean {
  return (
    typeof cfg.http.headers?.Authorization === "string" && cfg.http.headers.Authorization !== ""
  );
}

function expandHome(path: string): string {
  if (path.length > 0 && path.startsWith("~")) {
    return join(homedir(), path.slice(1));
  }
  return path;
}

export function getConfigPath(customPath: string): string {
  if (customPath !== "") {
    return expandHome(customPath);
  }
  return join(homedir(), CONFIG_FILE_NAME);
}

export async function saveConfig(cfg: Config, path: string): Promise<void> {
  const dir = dirname(path);
  await mkdir(dir, { recursive: true, mode: 0o755 });
  const data = stringifyToml(cfg as unknown as Record<string, unknown>);
  await writeFile(path, data, { mode: 0o644, encoding: "utf8" });
}

async function fileExists(path: string): Promise<boolean> {
  try {
    await stat(path);
    return true;
  } catch {
    return false;
  }
}

export async function loadConfig(path: string): Promise<Config> {
  const resolvedPath = path === "" ? join(homedir(), CONFIG_FILE_NAME) : expandHome(path);

  if (!(await fileExists(resolvedPath))) {
    const cfg = newConfig();
    await saveConfig(cfg, resolvedPath);
    return cfg;
  }

  const data = await readFile(resolvedPath, "utf8");
  const parsed = parseToml(data) as Partial<Config>;

  const cfg: Config = {
    http: {
      endpoint:
        parsed.http?.endpoint && parsed.http.endpoint !== ""
          ? parsed.http.endpoint
          : DEFAULT_ENDPOINT,
      headers: parsed.http?.headers ?? {},
    },
  };

  return cfg;
}
