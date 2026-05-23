import { cac } from "cac";
import { runLogin } from "./commands/login.tsx";
import { runParse } from "./commands/parse.tsx";
import { versionString } from "./version.ts";

const PROGRAM = "ck-cli";

interface GlobalFlags {
  config?: string;
  token?: string;
}

async function main(argv: string[]): Promise<number> {
  const cli = cac(PROGRAM);

  cli
    .option("-c, --config <path>", "Path to configuration file", { default: "" })
    .option("-t, --token <token>", "Authentication token for ClippingKK service", { default: "" });

  cli
    .command("login", "Authenticate with ClippingKK service")
    .option("-t, --token <token>", "API token from ClippingKK profile page")
    .action(async (cmdOpts: GlobalFlags) => {
      const code = await runLogin({
        token: cmdOpts.token || (cli.options.token as string | undefined) || "",
        config: cmdOpts.config || (cli.options.config as string | undefined) || "",
      });
      process.exit(code);
    });

  cli
    .command("parse", "Parse Kindle clippings file and output structured data")
    .option("-i, --input <path>", "Path to Kindle clippings file (default: read from stdin)", {
      default: "",
    })
    .option(
      "-o, --output <dest>",
      "Output destination: file path, 'http' for ClippingKK sync, or empty for stdout",
      { default: "" },
    )
    .action(async (cmdOpts: GlobalFlags & { input?: string; output?: string }) => {
      const controller = new AbortController();
      const onSignal = (): void => controller.abort();
      process.on("SIGINT", onSignal);
      process.on("SIGTERM", onSignal);

      try {
        const code = await runParse({
          input: cmdOpts.input ?? "",
          output: cmdOpts.output ?? "",
          token: cmdOpts.token || (cli.options.token as string | undefined) || "",
          config: cmdOpts.config || (cli.options.config as string | undefined) || "",
          signal: controller.signal,
        });
        process.exit(code);
      } finally {
        process.off("SIGINT", onSignal);
        process.off("SIGTERM", onSignal);
      }
    });

  cli.help();
  cli.version(versionString());

  try {
    cli.parse(argv);
  } catch (err) {
    process.stderr.write(`❌ ${(err as Error).message}\n`);
    return 1;
  }
  return 0;
}

main(process.argv).catch((err: unknown) => {
  process.stderr.write(`❌ ${(err as Error).message}\n`);
  process.exit(1);
});
