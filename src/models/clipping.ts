export interface ClippingItem {
  title: string;
  content: string;
  pageAt: string;
  createdAt: Date;
}

export interface ClippingInput {
  title: string;
  content: string;
  bookID: string;
  pageAt: string;
  createdAt: string;
  source: string;
}

export interface SerializedClipping {
  title: string;
  content: string;
  pageAt: string;
  createdAt: string;
}

export function toRFC3339(date: Date): string {
  return date.toISOString().replace(/\.\d{3}Z$/, "Z");
}

export function serializeClipping(c: ClippingItem): SerializedClipping {
  return {
    title: c.title,
    content: c.content,
    pageAt: c.pageAt,
    createdAt: toRFC3339(c.createdAt),
  };
}

export function toClippingInput(c: ClippingItem): ClippingInput {
  return {
    title: c.title,
    content: c.content,
    bookID: "0",
    pageAt: c.pageAt,
    createdAt: toRFC3339(c.createdAt),
    source: "kindle",
  };
}
