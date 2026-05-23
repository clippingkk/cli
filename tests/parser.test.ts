import { describe, expect, test } from "bun:test";
import { detectLanguage, parse, parseTitle } from "../src/parser/parser.ts";

describe("parseEnglishClippings", () => {
  test("parses two english clippings", () => {
    const input = `The Great Gatsby (F. Scott Fitzgerald)
- Your Highlight on page 7 | location 100-101 | Added on Monday, April 1, 2024 2:30:45 PM

In his blue gardens men and girls came and went like moths among the whisperings and the champagne and the stars.
==========
Another Book (Author Name)
- Your Highlight on page 15 | location 200-205 | Added on Tuesday, April 2, 2024 3:45:30 PM

This is another highlight from a different book.
==========`;

    const clippings = parse(input);
    expect(clippings).toHaveLength(2);

    const first = clippings[0]!;
    expect(first.title).toBe("The Great Gatsby");
    expect(first.pageAt).toBe("#7");
    expect(first.content).toBe(
      "In his blue gardens men and girls came and went like moths among the whisperings and the champagne and the stars.",
    );
    expect(first.createdAt.toISOString()).toBe("2024-04-01T14:30:45.000Z");
  });
});

describe("parseChineseClippings", () => {
  test("parses a chinese clipping", () => {
    const input = `深度工作 (卡尔·纽波特)
- 您在位置 #42-43的标注 | 添加于 2024年4月1日星期一 下午2:30:45

专注力就像肌肉一样，使用后会疲劳。
==========`;

    const clippings = parse(input);
    expect(clippings).toHaveLength(1);
    const first = clippings[0]!;
    expect(first.title).toBe("深度工作");
    expect(first.pageAt).toBe("#42-43");
  });
});

describe("parseTitleWithParentheses", () => {
  test("stops at first paren", () => {
    const input = `Some Book (Author Name) (Series: Book 1)
- Your Highlight on page 7 | location 100-101 | Added on Monday, April 1, 2024 2:30:45 PM

Some content here.
==========`;
    const clippings = parse(input);
    expect(clippings).toHaveLength(1);
    expect(clippings[0]!.title).toBe("Some Book");
  });
});

describe("parseBOMRemoval", () => {
  test("strips leading BOM", () => {
    const input =
      "﻿The Great Gatsby (F. Scott Fitzgerald)\n- Your Highlight on page 7 | location 100-101 | Added on Monday, April 1, 2024 2:30:45 PM\n\nSome content.\n==========";
    const clippings = parse(input);
    expect(clippings).toHaveLength(1);
    expect(clippings[0]!.title).not.toContain("﻿");
  });
});

describe("parseEmptyInput", () => {
  test("empty string returns empty array", () => {
    const clippings = parse("");
    expect(clippings).toHaveLength(0);
  });
});

describe("parseInvalidInput", () => {
  test("malformed input returns empty array", () => {
    const input = `Some Title
Invalid structure`;
    const clippings = parse(input);
    expect(clippings).toHaveLength(0);
  });
});

describe("detectLanguage", () => {
  test("detects en/zh", () => {
    expect(detectLanguage("Your Highlight on page")).toBe("en");
    expect(detectLanguage("您在位置")).toBe("zh");
    expect(detectLanguage("Some other text")).toBe("zh");
  });
});

describe("parseTitle (unit)", () => {
  test.each([
    ["Simple Title", "Simple Title"],
    ["Title (Author)", "Title"],
    ["Title (Author) (Series)", "Title"],
    ["Title（作者）", "Title"],
    ["Title) with trailing paren", "Title) with trailing paren"],
  ])("parseTitle(%p) === %p", (input, expected) => {
    expect(parseTitle(input)).toBe(expected);
  });
});
