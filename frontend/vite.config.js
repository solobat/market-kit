import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

const dirname = path.dirname(fileURLToPath(import.meta.url));
const localSourcesPath = path.join(dirname, "sync-sources.local.json");
const exampleSourcesPath = path.join(dirname, "sync-sources.example.json");

function readSyncSources() {
  const target = fs.existsSync(localSourcesPath) ? localSourcesPath : exampleSourcesPath;
  if (!fs.existsSync(target)) {
    return [];
  }

  try {
    const parsed = JSON.parse(fs.readFileSync(target, "utf8"));
    const sources = Array.isArray(parsed) ? parsed : parsed.sources;
    return Array.isArray(sources) ? sources : [];
  } catch {
    return [];
  }
}

function marketKitSyncPlugin() {
  return {
    name: "market-kit-sync",
    configureServer(server) {
      server.middlewares.use("/__market-kit/sources", async (_req, res) => {
        const sources = readSyncSources().map((item) => ({
          id: item.id,
          label: item.label || item.id,
          project: item.project || "",
          url: item.url || "",
          hasHeaders: Boolean(item.headers && Object.keys(item.headers).length)
        }));
        res.setHeader("Content-Type", "application/json; charset=utf-8");
        res.end(JSON.stringify({ sources }));
      });

      server.middlewares.use("/__market-kit/sync", async (req, res) => {
        try {
          const url = new URL(req.originalUrl || req.url || "", "http://localhost");
          const sourceId = url.searchParams.get("source") || "";
          const source = readSyncSources().find((item) => item.id === sourceId);

          if (!source || !source.url) {
            res.statusCode = 404;
            res.setHeader("Content-Type", "application/json; charset=utf-8");
            res.end(JSON.stringify({ error: "sync source not found" }));
            return;
          }

          const response = await fetch(source.url, {
            headers: source.headers || {}
          });

          const text = await response.text();
          res.statusCode = response.status;
          res.setHeader("Content-Type", "application/json; charset=utf-8");

          if (!response.ok) {
            res.end(
              JSON.stringify({
                error: `remote responded ${response.status}`,
                source: source.id,
                body: text
              })
            );
            return;
          }

          const payload = JSON.parse(text);
          res.end(
            JSON.stringify({
              source: {
                id: source.id,
                label: source.label || source.id,
                project: source.project || ""
              },
              payload
            })
          );
        } catch (error) {
          res.statusCode = 500;
          res.setHeader("Content-Type", "application/json; charset=utf-8");
          res.end(
            JSON.stringify({
              error: error instanceof Error ? error.message : "sync proxy failed"
            })
          );
        }
      });
    }
  };
}

export default defineConfig({
  server: {
    host: "127.0.0.1"
  },
  plugins: [svelte(), marketKitSyncPlugin()]
});
