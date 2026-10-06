import assert from 'node:assert/strict';
import { afterEach, test } from 'node:test';
import { loadRegistry, resolveIdentity, setRuntimeRegistry, upsertRuntimeAssetAlias, upsertRuntimeMarketOverride } from './identity.js';
import { buildCandidateGroups } from './discovery.js';

const initialRegistry = loadRegistry();
afterEach(() => setRuntimeRegistry(initialRegistry));
const asset = (canonical, aliases = [], unit_aliases = []) => ({ canonical, asset_class: 'crypto', aliases, unit_aliases });
const override = (market_type, canonical_symbol = 'NG/USDT', exchange = 'binance') => ({ exchange, raw_symbol: 'NGUSDT', market_type, canonical_symbol });
const resolve = (symbol = 'NGUSDT', marketTypeHint = 'perpetual', exchange = 'binance') => resolveIdentity({ exchange, symbol, marketTypeHint });

test('indexed overrides preserve exchange aliases, case folding and market-type ambiguity', () => {
  setRuntimeRegistry({ exchange_aliases: { bn: 'binance' }, asset_aliases: [asset('NG')], market_overrides: [override('spot', 'NG/USDT', 'bn'), override('perpetual')] });
  assert.equal(resolve(' ngusdt ', 'swap', 'BN').market.marketType, 'perpetual');
  assert.equal(resolve('NGUSDT', 'spot').market.marketType, 'spot');
  assert.equal(resolve('NGUSDT', '').status, 'ambiguous');
});

test('aliases retain collisions, per-asset deduplication, and unit conversions', () => {
  setRuntimeRegistry({ asset_aliases: [asset('NG', ['GAS'], [{ alias: 'GAS', multiplier: 10 }]), asset('OTHER', ['SHARED']), asset('THIRD', ['SHARED'])] });
  const market = resolve('GASUSDT').market;
  assert.equal(market.canonicalSymbol, 'NG/USDT');
  assert.equal(market.canonicalPriceMultiplier, 0.1);
  assert.equal(market.canonicalQuantityMultiplier, 10);
  assert.equal(resolve('SHAREDUSDT').status, 'ambiguous');
  assert.equal(resolve('UNKNOWNUSDT').market.assetClass, 'unknown');
});

test('both incremental edits and registry replacement invalidate indexes', () => {
  setRuntimeRegistry({ asset_aliases: [asset('NG')], market_overrides: [override('perpetual')] });
  assert.equal(resolve().market.assetClass, 'crypto');
  upsertRuntimeAssetAlias({ canonical: 'NG', asset_class: 'rwa_commodity', aliases: ['GAS'] });
  assert.equal(resolve().market.assetClass, 'rwa_commodity');
  assert.equal(resolve('GASUSDT').market.canonicalSymbol, 'NG/USDT');
  upsertRuntimeMarketOverride({ ...override('perpetual', 'OTHER/USDT'), unit_multiplier: 100 });
  assert.equal(resolve().market.canonicalSymbol, 'OTHER/USDT');
  assert.equal(resolve().market.canonicalQuantityMultiplier, 100);
  setRuntimeRegistry({ asset_aliases: [asset('NEW', ['GAS'])] });
  assert.equal(resolve().reason, 'resolved using exchange-specific market inference');
  assert.equal(resolve('GASUSDT').market.canonicalSymbol, 'NEW/USDT');
});

test('full discovery resolution does not rescan registry arrays per market', () => {
  const count = 20000;
  const registry = setRuntimeRegistry({
    asset_aliases: Array.from({ length: count }, (_, i) => asset(`COIN${i}X`)),
    market_overrides: Array.from({ length: count }, (_, i) => ({ exchange: 'binance', raw_symbol: `COIN${i}XUSDT`, market_type: 'perpetual', canonical_symbol: `COIN${i}X/USDT` }))
  });
  // Count rule visits independently of wall-clock speed; building indexes is linear.
  let visits = 0;
  for (const items of [registry.asset_aliases, registry.market_overrides]) {
    for (let i = 0; i < items.length; i++) {
      const item = items[i];
      Object.defineProperty(items, i, { get() { visits++; return item; }, configurable: true });
    }
  }
  const groups = buildCandidateGroups({ items: Array.from({ length: count }, (_, i) => ({ platform: 'binance', symbol: `COIN${i}XUSDT`, marketType: 'perpetual' })) });
  assert.equal(groups.length, count);
  assert.ok(groups.every((group) => !group.needsReview));
  assert.ok(visits <= count * 2, `expected one index build, got ${visits} rule visits`);
});
