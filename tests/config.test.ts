import { describe, expect, test } from "bun:test";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  DEFAULT_ENDPOINT,
  hasToken,
  loadConfig,
  newConfig,
  saveConfig,
  updateToken,
} from "../src/config/config.ts";

function tmpDir(): string {
  return mkdtempSync(join(tmpdir(), "ck-cli-test-"));
}

describe("config", () => {
  test("newConfig has default endpoint", () => {
    const cfg = newConfig();
    expect(cfg.http.endpoint).toBe(DEFAULT_ENDPOINT);
    expect(cfg.http.headers).toEqual({});
  });

  test("updateToken sets X-CLI header", () => {
    const cfg = newConfig();
    updateToken(cfg, "abc123");
    expect(cfg.http.headers.Authorization).toBe("X-CLI abc123");
    expect(hasToken(cfg)).toBe(true);
  });

  test("hasToken false when missing", () => {
    expect(hasToken(newConfig())).toBe(false);
  });

  test("save then load round-trip", async () => {
    const dir = tmpDir();
    try {
      const path = join(dir, ".ck-cli.toml");
      const cfg = newConfig();
      updateToken(cfg, "tok-xyz");
      await saveConfig(cfg, path);

      const loaded = await loadConfig(path);
      expect(loaded.http.endpoint).toBe(DEFAULT_ENDPOINT);
      expect(loaded.http.headers.Authorization).toBe("X-CLI tok-xyz");
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  });

  test("load missing file creates defaults", async () => {
    const dir = tmpDir();
    try {
      const path = join(dir, "missing.toml");
      const loaded = await loadConfig(path);
      expect(loaded.http.endpoint).toBe(DEFAULT_ENDPOINT);
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  });

  test("reads a TOML written in the Go format", async () => {
    const dir = tmpDir();
    try {
      const path = join(dir, ".ck-cli.toml");
      writeFileSync(
        path,
        `[http]\nendpoint = 'https://example.com/graphql'\n\n[http.headers]\nAuthorization = 'X-CLI sometoken'\n`,
        "utf8",
      );
      const loaded = await loadConfig(path);
      expect(loaded.http.endpoint).toBe("https://example.com/graphql");
      expect(loaded.http.headers.Authorization).toBe("X-CLI sometoken");
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  });
});
