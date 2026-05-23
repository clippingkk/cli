declare const CK_VERSION: string;
declare const CK_COMMIT: string;

const HAS_VERSION = typeof CK_VERSION !== "undefined";
const HAS_COMMIT = typeof CK_COMMIT !== "undefined";

export const VERSION: string = HAS_VERSION ? CK_VERSION : "dev";
export const COMMIT: string = HAS_COMMIT ? CK_COMMIT : "unknown";

export function versionString(): string {
  return `${VERSION} (${COMMIT})`;
}
