import { describe, expect, test } from "bun:test";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { serializeClipping } from "../src/models/clipping.ts";
import { parse } from "../src/parser/parser.ts";

const FIXTURE_DIR = join(import.meta.dir, "fixtures");

const fixtures = [
  "clippings_en",
  "clippings_zh",
  "clippings_other",
  "clippings_rare",
  "clippings_ric",
];

describe("fixture parity vs Go binary", () => {
  for (const name of fixtures) {
    test(`${name} matches oracle`, async () => {
      const input = await readFile(join(FIXTURE_DIR, `${name}.txt`), "utf8");
      const oracle = await readFile(join(FIXTURE_DIR, `${name}.result.json`), "utf8");

      const items = parse(input);
      const actual = `${JSON.stringify(items.map(serializeClipping), null, 2)}\n`;

      expect(actual).toBe(oracle);
    });
  }
});
