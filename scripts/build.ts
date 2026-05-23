#!/usr/bin/env bun
import { $ } from "bun";
import { mkdir, rm } from "node:fs/promises";
import { existsSync } from "node:fs";
import { join } from "node:path";

interface Target {
  name: string;
  bunTarget: string;
  outName: string;
  archive: "tar.gz" | "zip";
}

const TARGETS: Target[] = [
  {
    name: "linux-amd64",
    bunTarget: "bun-linux-x64",
    outName: "ck-cli-linux-amd64",
    archive: "tar.gz",
  },
  {
    name: "linux-arm64",
    bunTarget: "bun-linux-arm64",
    outName: "ck-cli-linux-arm64",
    archive: "tar.gz",
  },
  {
    name: "darwin-amd64",
    bunTarget: "bun-darwin-x64",
    outName: "ck-cli-darwin-amd64",
    archive: "tar.gz",
  },
  {
    name: "darwin-arm64",
    bunTarget: "bun-darwin-arm64",
    outName: "ck-cli-darwin-arm64",
    archive: "tar.gz",
  },
  {
    name: "windows-amd64",
    bunTarget: "bun-windows-x64",
    outName: "ck-cli-windows-amd64.exe",
    archive: "zip",
  },
];

async function detectLocalTarget(): Promise<Target> {
  const arch = process.arch;
  const platform = process.platform;
  const archStr = arch === "x64" ? "amd64" : arch === "arm64" ? "arm64" : arch;
  const platStr = platform === "darwin" ? "darwin" : platform === "win32" ? "windows" : "linux";
  const name = `${platStr}-${archStr}`;
  const found = TARGETS.find((t) => t.name === name);
  if (found) return found;
  return TARGETS[0]!;
}

async function getVersion(): Promise<{ version: string; commit: string }> {
  let version = process.env.CK_VERSION ?? "";
  let commit = process.env.CK_COMMIT ?? "";

  if (version === "") {
    try {
      const tag = (await $`git describe --tags --abbrev=0`.quiet().text()).trim();
      version = tag.startsWith("v") ? tag.slice(1) : tag;
    } catch {
      version = "dev";
    }
  }
  if (commit === "") {
    try {
      commit = (await $`git rev-parse --short HEAD`.quiet().text()).trim();
    } catch {
      commit = "unknown";
    }
  }
  return { version, commit };
}

async function buildOne(target: Target, distDir: string): Promise<string> {
  const { version, commit } = await getVersion();
  const outPath = join(distDir, target.outName);

  console.log(`[build] ${target.name} -> ${outPath}`);
  await $`bun build --compile --target=${target.bunTarget} --define CK_VERSION=${`"${version}"`} --define CK_COMMIT=${`"${commit}"`} --define process.env.DEV=${`"false"`} --minify --sourcemap=none --outfile=${outPath} ./src/main.tsx`;
  // Bun sometimes emits a stray main.js.map next to the binary; remove it.
  const strayMap = join(distDir, "main.js.map");
  if (existsSync(strayMap)) {
    await rm(strayMap, { force: true });
  }
  return outPath;
}

async function archiveOne(target: Target, binaryPath: string, distDir: string): Promise<string> {
  const stem = target.outName.replace(/\.exe$/, "");
  const archivePath = join(distDir, `${stem}.${target.archive}`);

  console.log(`[archive] ${binaryPath} -> ${archivePath}`);
  if (target.archive === "tar.gz") {
    await $`tar -czf ${archivePath} -C ${distDir} ${target.outName} -C ${process.cwd()} README.md LICENSE`;
  } else {
    await $`zip -j ${archivePath} ${binaryPath} README.md LICENSE`;
  }
  return archivePath;
}

async function shasum(filePath: string): Promise<string> {
  const text = (await $`shasum -a 256 ${filePath}`.text()).trim();
  return text.split(/\s+/)[0] ?? "";
}

async function main(): Promise<void> {
  const args = process.argv.slice(2);
  const buildAll = args.includes("--all");
  const archive = args.includes("--archive");

  const distDir = join(process.cwd(), "dist");
  if (existsSync(distDir)) await rm(distDir, { recursive: true, force: true });
  await mkdir(distDir, { recursive: true });

  const targets = buildAll ? TARGETS : [await detectLocalTarget()];
  const binaries: Array<{ target: Target; path: string }> = [];

  for (const t of targets) {
    const p = await buildOne(t, distDir);
    binaries.push({ target: t, path: p });
  }

  if (!buildAll) {
    const binPath = binaries[0]!.path;
    const localBin = join(process.cwd(), "ck-cli");
    await $`cp ${binPath} ${localBin}`.quiet();
    console.log(`[done] local binary -> ${localBin}`);
    return;
  }

  if (archive) {
    const checksums: string[] = [];
    for (const { target, path } of binaries) {
      const archivePath = await archiveOne(target, path, distDir);
      const hash = await shasum(archivePath);
      const fileName = archivePath.split("/").pop()!;
      checksums.push(`${hash}  ${fileName}`);
    }
    await Bun.write(join(distDir, "checksums.txt"), `${checksums.join("\n")}\n`);
    console.log(`[done] artifacts in ${distDir}`);
  }
}

main().catch((err: unknown) => {
  console.error(err);
  process.exit(1);
});
