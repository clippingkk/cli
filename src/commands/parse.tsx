import { readFile, writeFile } from "node:fs/promises";
import { render } from "ink";
import type React from "react";
import { getConfigPath, hasToken, loadConfig, saveConfig, updateToken } from "../config/config.ts";
import { syncToServer } from "../http/client.ts";
import { type ClippingItem, serializeClipping } from "../models/clipping.ts";
import { parse as parseClippings } from "../parser/parser.ts";
import { Status } from "../ui/Status.tsx";
import { SyncProgress } from "../ui/SyncProgress.tsx";

export interface ParseOptions {
  input?: string;
  output?: string;
  token?: string;
  config?: string;
  signal?: AbortSignal;
}

function renderUI(node: React.ReactElement): void {
  const inst = render(node, { stdout: process.stderr });
  inst.unmount();
}

async function readInput(inputPath: string): Promise<string> {
  if (inputPath === "") {
    if (process.stdin.isTTY) {
      throw new TTYError(
        "No --input file provided and stdin is a TTY. Pipe data in or pass --input PATH.",
      );
    }
    return await Bun.stdin.text();
  }
  return await readFile(inputPath, "utf8");
}

export class TTYError extends Error {
  constructor(msg: string) {
    super(msg);
    this.name = "TTYError";
  }
}

function jsonOutput(clippings: ClippingItem[]): string {
  const serialized = clippings.map(serializeClipping);
  return `${JSON.stringify(serialized, null, 2)}\n`;
}

export async function runParse(opts: ParseOptions): Promise<number> {
  const configPath = getConfigPath(opts.config ?? "");
  const cfg = await loadConfig(configPath);

  if (opts.token && opts.token !== "") {
    updateToken(cfg, opts.token);
    await saveConfig(cfg, configPath);
  }

  let inputData: string;
  try {
    inputData = await readInput(opts.input ?? "");
  } catch (err) {
    if (err instanceof TTYError) {
      renderUI(
        <Status emoji="❌" color="red">
          {err.message}
        </Status>,
      );
      process.stderr.write("\n");
      process.stderr.write("Usage:\n");
      process.stderr.write("  ck-cli parse --input PATH/TO/My\\ Clippings.txt\n");
      process.stderr.write('  cat "My Clippings.txt" | ck-cli parse\n');
      return 1;
    }
    renderUI(
      <Status emoji="❌" color="red">
        Failed to read input: {(err as Error).message}
      </Status>,
    );
    return 1;
  }

  let clippings: ClippingItem[];
  try {
    clippings = parseClippings(inputData);
  } catch (err) {
    renderUI(
      <Status emoji="❌" color="red">
        Parsing failed: {(err as Error).message}
      </Status>,
    );
    return 1;
  }

  if (clippings.length === 0) {
    renderUI(
      <Status emoji="⚠️" color="yellow">
        No clippings found in input
      </Status>,
    );
    return 0;
  }

  renderUI(<Status emoji="📚">Parsed {clippings.length} clippings successfully</Status>);

  const output = opts.output ?? "";

  if (output === "") {
    process.stdout.write(jsonOutput(clippings));
    return 0;
  }

  if (output === "http" || output.startsWith("http")) {
    if (!hasToken(cfg)) {
      renderUI(
        <Status emoji="❌" color="red">
          No authentication token found
        </Status>,
      );
      process.stderr.write("Please login first: ck-cli login --token YOUR_TOKEN\n");
      return 1;
    }
    return await runSync(cfg, clippings, output, opts.signal);
  }

  await writeFile(output, jsonOutput(clippings), "utf8");
  renderUI(
    <Status emoji="💾">
      Saved {clippings.length} clippings to {output}
    </Status>,
  );
  return 0;
}

async function runSync(
  cfg: import("../config/config.ts").Config,
  clippings: ClippingItem[],
  endpoint: string,
  signal: AbortSignal | undefined,
): Promise<number> {
  renderUI(<Status emoji="🚀">Starting sync to ClippingKK service...</Status>);

  let totalChunks = 0;
  let completedChunks = 0;
  let failed = false;
  let errorMessage = "";

  try {
    await syncToServer(cfg, clippings, {
      endpoint,
      signal,
      callbacks: {
        onStart: (_items, chunks) => {
          totalChunks = chunks;
          process.stderr.write(`Uploading ${clippings.length} clippings in ${chunks} chunks...\n`);
        },
        onChunkDone: (idx, total, items) => {
          completedChunks = idx;
          process.stderr.write(`✅ Chunk ${idx}/${total} completed: ${items} items\n`);
        },
      },
    });
  } catch (err) {
    failed = true;
    errorMessage = (err as Error).message;
  }

  renderUI(
    <SyncProgress
      totalItems={clippings.length}
      totalChunks={totalChunks}
      completedChunks={completedChunks}
      done={!failed}
      error={failed ? errorMessage : undefined}
    />,
  );
  return failed ? 1 : 0;
}
