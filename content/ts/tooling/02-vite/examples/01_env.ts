// LEARN(ts.tooling.vite): import.meta.env holds strings from .env files; parse them once, at startup.
import { z } from "zod";

// import.meta is standard JavaScript; .env on it is added by Vite. This runner
// transpiles with esbuild and has no Vite, so there is nothing there:
console.log("import.meta.env here:", typeof (import.meta as { env?: unknown }).env);

// What `vite build` substitutes for this .env file (verified in the lesson's prose):
//   VITE_API_URL=/api
//   VITE_DEBUG=false
//   VITE_PAGE_SIZE=20
// Typed the way vite/client types it: ImportMetaEnv extends Record<string, any>.
// eslint-disable-next-line @typescript-eslint/no-explicit-any -- mirrors Vite's own declaration
const env: Record<string, any> = {
  VITE_API_URL: "/api",
  VITE_DEBUG: "false",
  VITE_PAGE_SIZE: "20",
  MODE: "production",
  DEV: false,
  PROD: true,
};

// Naive reads: user variables are strings, whatever they look like.
console.log("debug?", env.VITE_DEBUG ? "on" : "off", "| page size + 1 =", env.VITE_PAGE_SIZE + 1);

// One schema at the edge turns strings into typed config, or fails loudly.
const Config = z.object({
  VITE_API_URL: z.string().default(""),
  VITE_DEBUG: z.enum(["true", "false"]).default("false").transform((v) => v === "true"),
  VITE_PAGE_SIZE: z.coerce.number().int().positive().default(20),
});
type Config = z.infer<typeof Config>; // { VITE_API_URL: string; VITE_DEBUG: boolean; ... }

const config: Config = Config.parse(env);
console.log("parsed:", config);

// A typo in .env.production surfaces at startup, not in the third request.
const broken = Config.safeParse({ ...env, VITE_PAGE_SIZE: "yirmi" });
if (!broken.success) {
  for (const issue of broken.error.issues) {
    console.log(`invalid ${issue.path.join(".")}: ${issue.message}`);
  }
}
