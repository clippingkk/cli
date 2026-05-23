import { afterEach, beforeEach, describe, expect, mock, test } from "bun:test";
import { newConfig, updateToken } from "../src/config/config.ts";
import { CHUNK_SIZE, chunk, syncToServer } from "../src/http/client.ts";
import type { ClippingItem } from "../src/models/clipping.ts";

function makeItems(n: number): ClippingItem[] {
  return Array.from({ length: n }, (_, i) => ({
    title: `book-${i}`,
    content: `content-${i}`,
    pageAt: `#${i}`,
    createdAt: new Date(Date.UTC(2024, 0, 1, 0, 0, i)),
  }));
}

describe("chunk()", () => {
  test("splits into chunks of given size", () => {
    expect(chunk([1, 2, 3, 4, 5], 2)).toEqual([[1, 2], [3, 4], [5]]);
    expect(chunk([], 3)).toEqual([]);
    expect(chunk([1, 2, 3], 10)).toEqual([[1, 2, 3]]);
  });

  test("default CHUNK_SIZE is 20", () => {
    expect(CHUNK_SIZE).toBe(20);
  });
});

describe("syncToServer()", () => {
  let originalFetch: typeof globalThis.fetch;
  beforeEach(() => {
    originalFetch = globalThis.fetch;
  });
  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  test("chunks 25 items into 2 requests and sends proper headers", async () => {
    const calls: Array<{ url: string; init: RequestInit }> = [];
    globalThis.fetch = mock(async (url: string, init: RequestInit) => {
      calls.push({ url, init });
      return new Response(JSON.stringify({ data: { createClippings: [{ id: 1 }] } }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }) as unknown as typeof fetch;

    const cfg = newConfig();
    updateToken(cfg, "tok");
    await syncToServer(cfg, makeItems(25));

    expect(calls).toHaveLength(2);
    const first = calls[0]!;
    expect(first.url).toBe(cfg.http.endpoint);
    expect((first.init.headers as Record<string, string>)["Content-Type"]).toBe("application/json");
    expect((first.init.headers as Record<string, string>).Authorization).toBe("X-CLI tok");

    const body = JSON.parse(first.init.body as string);
    expect(body.operationName).toBe("createClippings");
    expect(body.variables.visible).toBe(true);
    expect(body.variables.payload).toHaveLength(20);
    expect(body.variables.payload[0]).toMatchObject({
      title: "book-0",
      bookID: "0",
      source: "kindle",
    });
  });

  test("joins all GraphQL errors", async () => {
    globalThis.fetch = mock(async () => {
      return new Response(
        JSON.stringify({
          errors: [{ message: "first" }, { message: "second" }],
        }),
        { status: 200 },
      );
    }) as unknown as typeof fetch;

    const cfg = newConfig();
    await expect(syncToServer(cfg, makeItems(1))).rejects.toThrow(/first; second/);
  });

  test("uses override endpoint when not 'http'", async () => {
    const calls: string[] = [];
    globalThis.fetch = mock(async (url: string) => {
      calls.push(url);
      return new Response(JSON.stringify({ data: {} }), { status: 200 });
    }) as unknown as typeof fetch;

    const cfg = newConfig();
    await syncToServer(cfg, makeItems(1), { endpoint: "https://override.example/gql" });
    expect(calls[0]).toBe("https://override.example/gql");
  });

  test("uses config endpoint when override is 'http'", async () => {
    const calls: string[] = [];
    globalThis.fetch = mock(async (url: string) => {
      calls.push(url);
      return new Response(JSON.stringify({ data: {} }), { status: 200 });
    }) as unknown as typeof fetch;

    const cfg = newConfig();
    await syncToServer(cfg, makeItems(1), { endpoint: "http" });
    expect(calls[0]).toBe(cfg.http.endpoint);
  });
});
