import json,collections,concurrent.futures,urllib.request,time,datetime,pathlib,re,hashlib
import argparse
parser=argparse.ArgumentParser(description="Read-only full market-kit online identity audit; flags are review leads, not automatic merge decisions.")
parser.add_argument('--inventory', required=True)
parser.add_argument('--registry', required=True)
parser.add_argument('--output-dir', required=True)
parser.add_argument('--api-base', default='https://market.mintoken.xyz/api/v1')
parser.add_argument('--workers', type=int, default=2, choices=[1,2])
args=parser.parse_args()
ROOT=pathlib.Path(args.output_dir);ROOT.mkdir(parents=True,exist_ok=True)
inv=json.load(open(args.inventory));reg=json.load(open(args.registry))
fingerprint=hashlib.sha256(pathlib.Path(args.inventory).read_bytes()+pathlib.Path(args.registry).read_bytes()).hexdigest()
cache=ROOT/('batches-'+fingerprint[:16]);cache.mkdir(exist_ok=True)
norm=lambda s: {'perp':'perpetual','swap':'perpetual','linear':'perpetual'}.get(s,s)
keys={}; inventories=collections.defaultdict(list)
for r in inv['payload']['items']:
 k=(r['platformId'].lower(),r['symbol'],norm(r['marketType']));inventories[k].append(r);keys[k]={'exchange':k[0],'symbol':k[1],'marketType':k[2]}
for r in reg['market_overrides']:
 k=(r['exchange'],r['raw_symbol'],r['market_type']);keys[k]={'exchange':k[0],'symbol':k[1],'marketType':k[2]}
items=sorted(keys.items());batches=[items[i:i+250] for i in range(0,len(items),250)]
def fetch(pair):
 i,batch=pair;path=cache/('batch-%03d.json'%i)
 if path.exists():return i,json.load(open(path))
 req=urllib.request.Request(args.api_base.rstrip('/')+'/resolve/batch',data=json.dumps({'items':[v for k,v in batch]}).encode(),headers={'Content-Type':'application/json','Accept-Encoding':'identity'})
 for attempt in range(2):
  try:
   with urllib.request.urlopen(req,timeout=45) as r:
    raw=r.read(2000000)
   d=json.loads(raw)
   d['checkedAt']=datetime.datetime.now(datetime.timezone.utc).isoformat()
   if len(d['results'])!=len(batch):raise ValueError('result count mismatch')
   path.write_text(json.dumps(d));return i,d
  except Exception as e:
   if attempt: return i,{'error':str(e)}
print('Inventory rows',len(inv['payload']['items']),'unique union markets',len(items),'batches',len(batches),flush=True)
responses={}
with concurrent.futures.ThreadPoolExecutor(max_workers=args.workers) as pool:
 for i,d in pool.map(fetch,enumerate(batches)):
  responses[i]=d;print('batch',i+1,'/',len(batches),'error',d.get('error',''),flush=True)
assets={x['canonical']:x for x in reg['asset_aliases']};alias=collections.defaultdict(list)
for a in reg['asset_aliases']:
 for n in [a['canonical']]+a.get('aliases',[]):alias[n].append((a['canonical'],1))
 for u in a.get('unit_aliases',[]):alias[u['alias']].append((a['canonical'],u['multiplier']))
rows=[];summary=collections.defaultdict(collections.Counter)
for i,batch in enumerate(batches):
 d=responses[i]
 for j,(k,req) in enumerate(batch):
  result=d.get('results',[None]*len(batch))[j];sources=inventories.get(k,[]);flags=[];m=(result or {}).get('market') or {};base=m.get('baseAsset','');cl=m.get('assetClass','');aid=m.get('assetId','');uid=m.get('underlyingId','')
  if result is None: flags.append('api_error')
  elif result['status']!='resolved':flags.append('resolve_'+result['status'])
  if m:
   if cl in ('','unknown'):flags.append('unknown_class')
   if not m.get('comparisonKey') or m.get('comparisonStatus')!='eligible':flags.append('comparison_not_eligible')
   prefix={'crypto':'crypto:','rwa_stock':'equity:','rwa_commodity':'commodity:','fiat_stable':'fiat:'}.get(cl)
   if prefix and aid and not aid.startswith(prefix):flags.append('asset_class_id_mismatch')
   for s in sources:
    b=s['baseAsset'].upper();meta=[s.get(n,'').lower() for n in ['assetClass','assetClassHint','underlyingCategory']]+[t.lower() for t in s.get('tags') or []]
    strongcoin=s.get('underlyingCategory','').lower()=='coin' or 'underlying-type:coin' in meta
    stock=any(t in ['stock','equity','rwa_stock','tokenized-stock','bybit-stock'] for t in meta)
    if s.get('quoteAsset') and s['quoteAsset'].upper()!=m.get('quoteAsset'):flags.append('source_quote_conflict')
    if strongcoin and cl=='rwa_stock' and not stock:flags.append('explicit_coin_class_conflict')
    if stock and cl=='crypto':flags.append('explicit_stock_class_conflict')
    targets=alias.get(b,[])
    for target,mult in targets:
     if target!=b and base==b:flags.append('registered_alias_not_normalized')
     if target==base and mult!=1 and abs(m.get('unitMultiplier',1)-mult)>1e-9:flags.append('unit_multiplier_conflict')
    if b!=base and not any(t[0]==base for t in targets):flags.append('base_changed_without_global_alias')
   if m.get('unitMultiplier',1)!=1:
    mult=m['unitMultiplier'];p=m.get('canonicalPriceMultiplier',1);q=m.get('canonicalQuantityMultiplier',1)
    if abs(p*mult-1)>1e-8 or abs(q-mult)>1e-8:flags.append('unit_conversion_inconsistent')
   if m.get('marketType')=='future' and not m.get('expiryAtMs'):flags.append('future_missing_expiry')
   symbol=k[1].upper().split(':')[-1];rawbase=base
   for s in sources:
    b=s['baseAsset'].upper()
    wrapped=re.fullmatch(r'(.+?)(ON|STOCK|X|B|G)',b)
    if wrapped and wrapped[1] in assets and assets[wrapped[1]].get('asset_class')=='rwa_stock' and base==b:flags.append('wrapper_identity_review')
   if re.match(r'^(1000|10000|1000000|1M)[A-Z]',base) and m.get('unitMultiplier',1)==1:flags.append('scaled_name_review')
  if len({str(v.get('chain')) for v in sources if v.get('chain')})>1:flags.append('multiple_chains_single_resolve_key')
  if len({(v['baseAsset'].upper(),v['quoteAsset'].upper(),v.get('assetClass') or v.get('assetClassHint') or '') for v in sources})>1:flags.append('conflicting_source_identity')
  row={'key':list(k),'checkedAt':d.get('checkedAt'),'inventoryRows':len(sources),'inventory':sources,'result':result,'flags':sorted(set(flags))};rows.append(row)
  summary[k[0]]['markets']+=1;summary[k[0]][(result or {}).get('status','api_error')]+=1
  if flags:summary[k[0]]['flagged']+=1
  for f in set(flags):summary[k[0]][f]+=1
# Cross-platform fragmentation by explicit aliases and same source base; flags are review leads, not confirmed equivalence.
families=collections.defaultdict(list)
for row in rows:
 m=(row['result'] or {}).get('market') or {}
 for b in {s['baseAsset'].upper() for s in row['inventory']}:
  families[b].append({'key':row['key'],'canonical':m.get('baseAsset'),'class':m.get('assetClass'),'assetId':m.get('assetId'),'underlyingId':m.get('underlyingId'),'comparisonKey':m.get('comparisonKey'),'multiplier':m.get('unitMultiplier',1)})
fragmented={b:v for b,v in families.items() if len({x['key'][0] for x in v})>1 and len({x['assetId'] for x in v if x['assetId']})>1}
asset_issues=[]
for a in reg['asset_aliases']:
 prefix={'crypto':'crypto:','rwa_stock':'equity:','rwa_commodity':'commodity:','fiat_stable':'fiat:'}.get(a['asset_class'])
 if prefix and a.get('asset_id') and not a['asset_id'].startswith(prefix):asset_issues.append(a)
out={'inputFingerprint':fingerprint,'startedFromInventoryAt':inv['cachedAt'],'generatedAt':datetime.datetime.now(datetime.timezone.utc).isoformat(),'inventoryRows':len(inv['payload']['items']),'marketCount':len(rows),'platformCount':len(summary),'platforms':dict(summary),'flagCounts':dict(collections.Counter(f for r in rows for f in r['flags'])),'fragmentedFamilies':fragmented,'assetClassIdMismatches':asset_issues,'rows':rows}
(ROOT/'audit.json').write_text(json.dumps(out,ensure_ascii=False,indent=2));print(json.dumps({k:v for k,v in out.items() if k not in ['rows','fragmentedFamilies','assetClassIdMismatches']},ensure_ascii=False),flush=True)
