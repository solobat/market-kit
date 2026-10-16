import test from "node:test";
import assert from "node:assert/strict";
import { setTimeout as wait } from "node:timers/promises";
import { createQueryLoader } from "./query-loader.js";

test("typing coalesces and identical query does not refetch", async () => {
  const requests = [], results = [];
  const loader = createQueryLoader(async key => { requests.push(key); return key; }, { success: value => results.push(value) }, 5);
  loader.schedule("O"); loader.schedule("OP"); loader.schedule("OPENAI");
  await wait(20); loader.schedule("OPENAI"); await wait(20);
  assert.deepEqual(requests, ["OPENAI"]); assert.deepEqual(results, ["OPENAI"]);
  loader.refresh(); await wait(20); assert.equal(requests.length, 2); loader.dispose();
});
test("late responses cannot overwrite a newer page, and disposal cancels work", async () => {
  const pending = new Map(), results = [], signals = [];
  const loader = createQueryLoader((key, signal) => { signals.push(signal); return new Promise(resolve => pending.set(key, resolve)); }, { success: value => results.push(value) }, 0);
  loader.schedule("old"); await wait(10); loader.schedule("new"); await wait(10);
  assert.equal(signals[0].aborted, true);
  pending.get("new")("new"); await wait(0); pending.get("old")("old"); await wait(0);
  assert.deepEqual(results, ["new"]);
  loader.schedule("disposed"); await wait(10); loader.dispose(); pending.get("disposed")("disposed"); await wait(0);
  assert.deepEqual(results, ["new"]); assert.equal(signals.at(-1).aborted, true);
});

import { normalizeImportedCases } from "./identity.js";
test("imported cases use the supplied server result without a complete browser registry", () => {
 const server = { status: "resolved", market: { canonicalSymbol: "OPENAI/USDT", assetClass: "rwa_stock" } };
 const [item] = normalizeImportedCases([{ exchange: "binance", symbol: "OPENAIUSDT" }], () => server);
 assert.equal(item.resolution, server); assert.equal(item.status, "resolved");
});
