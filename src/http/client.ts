import type { Config } from "../config/config.ts";
import { type ClippingItem, type ClippingInput, toClippingInput } from "../models/clipping.ts";
import { withConcurrency } from "../utils/semaphore.ts";

export const CHUNK_SIZE = 20;
export const MAX_CONCURRENCY = 10;
export const REQUEST_TIMEOUT_MS = 30_000;

export const CREATE_CLIPPINGS_MUTATION = `
mutation createClippings($payload: [ClippingInput!]!, $visible: Boolean) {
	createClippings(payload: $payload, visible: $visible) {
		id
	}
}
`;

export interface GraphQLError {
  message: string;
  locations?: Array<{ line: number; column: number }>;
  path?: unknown[];
  extensions?: Record<string, unknown>;
}

export interface GraphQLResponse<T = unknown> {
  data?: T;
  errors?: GraphQLError[];
}

export interface CreateClippingsData {
  createClippings: Array<{ id: number }>;
}

export interface SyncCallbacks {
  onStart?: (totalItems: number, totalChunks: number) => void;
  onChunkDone?: (chunkIndex: number, totalChunks: number, itemCount: number) => void;
}

export interface SyncOptions {
  endpoint?: string;
  signal?: AbortSignal;
  callbacks?: SyncCallbacks;
}

export function chunk<T>(items: T[], size: number): T[][] {
  const out: T[][] = [];
  for (let i = 0; i < items.length; i += size) {
    out.push(items.slice(i, i + size));
  }
  return out;
}

function resolveEndpoint(cfg: Config, override: string | undefined): string {
  if (override && override !== "" && override !== "http") {
    return override;
  }
  return cfg.http.endpoint;
}

export async function syncToServer(
  cfg: Config,
  clippings: ClippingItem[],
  opts: SyncOptions = {},
): Promise<void> {
  const endpoint = resolveEndpoint(cfg, opts.endpoint);
  if (!endpoint || endpoint === "" || endpoint === "http") {
    throw new Error("no valid endpoint configured");
  }

  const inputs = clippings.map(toClippingInput);
  const chunks = chunk(inputs, CHUNK_SIZE);

  opts.callbacks?.onStart?.(clippings.length, chunks.length);

  const errors: Error[] = [];
  const tasks = chunks.map((chunkData, i) => async () => {
    try {
      await uploadChunk(cfg, endpoint, chunkData, opts.signal);
      opts.callbacks?.onChunkDone?.(i + 1, chunks.length, chunkData.length);
    } catch (err) {
      errors.push(new Error(`chunk ${i + 1} failed: ${(err as Error).message}`));
    }
  });

  await withConcurrency(MAX_CONCURRENCY, tasks);

  if (errors.length > 0) {
    const messages = errors.map((e) => e.message).join("; ");
    throw new Error(`upload failed with ${errors.length} errors: ${messages}`);
  }
}

async function uploadChunk(
  cfg: Config,
  endpoint: string,
  payload: ClippingInput[],
  parentSignal: AbortSignal | undefined,
): Promise<void> {
  const body = JSON.stringify({
    operationName: "createClippings",
    query: CREATE_CLIPPINGS_MUTATION,
    variables: { payload, visible: true },
  });

  const timeoutSignal = AbortSignal.timeout(REQUEST_TIMEOUT_MS);
  const signal = parentSignal ? AbortSignal.any([parentSignal, timeoutSignal]) : timeoutSignal;

  const headers: Record<string, string> = { "Content-Type": "application/json" };
  for (const [k, v] of Object.entries(cfg.http.headers)) {
    headers[k] = v;
  }

  const res = await fetch(endpoint, { method: "POST", headers, body, signal });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(`HTTP ${res.status}: ${text}`);
  }

  const json = (await res.json()) as GraphQLResponse<CreateClippingsData>;
  if (json.errors && json.errors.length > 0) {
    const joined = json.errors.map((e) => e.message).join("; ");
    throw new Error(`GraphQL error: ${joined}`);
  }
}
