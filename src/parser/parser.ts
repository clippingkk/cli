import type { ClippingItem } from "../models/clipping.ts";

export type Language = "en" | "zh";

const SEPARATOR = "========";
const BOM = "﻿";

const englishLocationPattern = /\d+(?:-?\d+)?/;
const chineseLocationPattern = /#?\d+(?:-?\d+)?/;
const cjkPattern = /[一-鿿　-〿]/gu;
const multiDashPattern = /-{2,}/g;

const MONTHS = [
  "January",
  "February",
  "March",
  "April",
  "May",
  "June",
  "July",
  "August",
  "September",
  "October",
  "November",
  "December",
];

export interface ParseOptions {
  removeBOM?: boolean;
}

export function parse(input: string, opts: ParseOptions = {}): ClippingItem[] {
  const { removeBOM = true } = opts;

  let text = input;
  if (removeBOM) {
    text = text.replaceAll(BOM, "");
  }

  text = text.trim();
  if (text === "") {
    return [];
  }

  const language = detectLanguage(text);
  const groups = splitIntoGroups(text);

  const result: ClippingItem[] = [];
  for (const group of groups) {
    const item = parseGroup(group, language);
    if (item !== null) {
      result.push(item);
    }
  }
  return result;
}

export function detectLanguage(input: string): Language {
  return input.includes("Your Highlight on") ? "en" : "zh";
}

export function splitIntoGroups(input: string): string[][] {
  const lines = input.split("\n");
  const groups: string[][] = [];
  let current: string[] = [];

  for (const line of lines) {
    if (line.includes(SEPARATOR)) {
      if (current.length > 0) {
        groups.push(current);
        current = [];
      }
    } else {
      current.push(line);
    }
  }
  if (current.length > 0) {
    groups.push(current);
  }
  return groups;
}

function parseGroup(group: string[], language: Language): ClippingItem | null {
  if (group.length < 4) {
    return null;
  }

  const titleLine = (group[0] ?? "").replaceAll(BOM, "");
  const title = parseTitle(titleLine);
  if (title === "") {
    return null;
  }

  const infoLine = group[1] ?? "";
  const info = parseInfo(infoLine, language);
  if (info === null) {
    return null;
  }

  const content = (group[3] ?? "").trim();
  if (content === "") {
    return null;
  }

  return {
    title,
    content,
    pageAt: info.location,
    createdAt: info.createdAt,
  };
}

export function parseTitle(line: string): string {
  let title = line.trim();
  for (const stop of ["(", "（"]) {
    const idx = title.indexOf(stop);
    if (idx !== -1) {
      title = title.slice(0, idx);
    }
  }
  if (title.endsWith(")")) title = title.slice(0, -1);
  if (title.endsWith("）")) title = title.slice(0, -1);
  return title.trim();
}

interface InfoParts {
  location: string;
  createdAt: Date;
}

function parseInfo(line: string, language: Language): InfoParts | null {
  const parts = line.split("|");
  if (parts.length < 2) {
    return null;
  }

  const locationSection = (parts[0] ?? "").trim();
  const pattern = language === "en" ? englishLocationPattern : chineseLocationPattern;
  const match = pattern.exec(locationSection);

  let location = "";
  if (match) {
    let pageAt = match[0];
    if (!pageAt.startsWith("#")) pageAt = `#${pageAt}`;
    location = pageAt;
  }

  let dateSection = (parts[parts.length - 1] ?? "").trim();
  dateSection = dateSection.replace("Added on ", "");
  dateSection = dateSection.replace("添加于 ", "");

  let createdAt: Date;
  try {
    createdAt = language === "en" ? parseEnglishDate(dateSection) : parseChineseDate(dateSection);
  } catch {
    createdAt = new Date(0);
  }

  return { location, createdAt };
}

function parseEnglishDate(dateStr: string): Date {
  const s = dateStr.trim();
  const commaIdx = s.indexOf(", ");
  if (commaIdx === -1) throw new Error("invalid english date");
  const afterDay = s.slice(commaIdx + 2);

  const firstSpace = afterDay.indexOf(" ");
  const monthName = afterDay.slice(0, firstSpace);
  const month = MONTHS.indexOf(monthName);
  if (month === -1) throw new Error(`invalid month: ${monthName}`);

  const rest = afterDay.slice(firstSpace + 1);
  const dayCommaIdx = rest.indexOf(", ");
  if (dayCommaIdx === -1) throw new Error("invalid english date day");
  const day = Number.parseInt(rest.slice(0, dayCommaIdx), 10);

  const afterYearStart = rest.slice(dayCommaIdx + 2);
  const yearSpaceIdx = afterYearStart.indexOf(" ");
  if (yearSpaceIdx === -1) throw new Error("invalid english date year");
  const year = Number.parseInt(afterYearStart.slice(0, yearSpaceIdx), 10);

  const timeAndAmpm = afterYearStart.slice(yearSpaceIdx + 1).trim();
  return parseTimeAndAmpm(timeAndAmpm, year, month, day);
}

function parseChineseDate(dateStr: string): Date {
  let s = dateStr;
  const ampm = s.includes("上午") ? "AM" : "PM";
  s = s.replace(cjkPattern, "-");
  s = s.replace(multiDashPattern, "");
  s = s.trim();
  s = `${s} ${ampm}`;
  return parseShortDateAmpm(s);
}

function parseShortDateAmpm(s: string): Date {
  const spaceIdx = s.indexOf(" ");
  if (spaceIdx === -1) throw new Error("invalid short date");
  const datePart = s.slice(0, spaceIdx);
  const rest = s.slice(spaceIdx + 1).trim();

  const dateBits = datePart.split("-");
  if (dateBits.length !== 3) throw new Error(`invalid date part: ${datePart}`);
  const year = Number.parseInt(dateBits[0] ?? "", 10);
  const month = Number.parseInt(dateBits[1] ?? "", 10) - 1;
  const day = Number.parseInt(dateBits[2] ?? "", 10);

  return parseTimeAndAmpm(rest, year, month, day);
}

function parseTimeAndAmpm(s: string, year: number, month: number, day: number): Date {
  const spaceIdx = s.lastIndexOf(" ");
  if (spaceIdx === -1) throw new Error(`invalid time/ampm: ${s}`);
  const timePart = s.slice(0, spaceIdx);
  const ampm = s.slice(spaceIdx + 1).toUpperCase();

  const timeBits = timePart.split(":");
  if (timeBits.length !== 3) throw new Error(`invalid time: ${timePart}`);
  let hour = Number.parseInt(timeBits[0] ?? "", 10);
  const minute = Number.parseInt(timeBits[1] ?? "", 10);
  const second = Number.parseInt(timeBits[2] ?? "", 10);

  if (
    Number.isNaN(year) ||
    Number.isNaN(month) ||
    Number.isNaN(day) ||
    Number.isNaN(hour) ||
    Number.isNaN(minute) ||
    Number.isNaN(second)
  ) {
    throw new Error("date contains NaN");
  }

  if (ampm === "PM" && hour < 12) hour += 12;
  else if (ampm === "AM" && hour === 12) hour = 0;

  return new Date(Date.UTC(year, month, day, hour, minute, second));
}
