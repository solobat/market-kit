"""Read-only security metadata drift checks; changes are review leads, not rebases."""
import json
from pathlib import Path

FIELDS = ('isin', 'baseCoin', 'quoteCoin', 'symbolType', 'isReality', 'fullName', 'status')


def read_catalog(path):
    payload = json.loads(Path(path).read_text())
    rows = payload.get('data') if isinstance(payload, dict) else payload
    if not isinstance(rows, list):
        raise ValueError('official security catalog must contain a data list')
    by_symbol = {}
    for row in rows:
        symbol = str(row.get('symbol', '')).strip().upper()
        if not symbol or symbol in by_symbol:
            raise ValueError('missing or duplicate official security symbol: ' + symbol)
        by_symbol[symbol] = row
    return by_symbol


def compare_catalogs(current, previous):
    changes = {}
    for symbol in sorted(current.keys() & previous.keys()):
        before, after = previous[symbol], current[symbol]
        deltas = []
        for field in FIELDS:
            old, new = before.get(field) or '', after.get(field) or ''
            if str(old).strip().casefold() == str(new).strip().casefold():
                continue
            if field == 'isin':
                kind = ('security_identifier_changed' if old and new else
                        'security_identifier_enriched' if new else 'security_identifier_missing')
            elif field == 'status':
                kind = 'trading_status_changed'
            else:
                kind = 'security_metadata_changed'
            deltas.append({'field': field, 'kind': kind, 'before': old, 'after': new})
        if deltas:
            changes[symbol] = deltas
    return {'changes': changes, 'addedSymbols': sorted(current.keys() - previous.keys()),
            'absentSymbols': sorted(previous.keys() - current.keys()),
            'note': 'Catalog changes do not prove a split, conversion ratio, delisting, or external price equivalence.'}


def extended_categories(path):
    """Index only official perpetual names and their base/quote display variant."""
    rows = json.loads(Path(path).read_text())['data']
    result = {}
    for row in rows:
        if row.get('type') != 'PERPETUAL':
            continue
        category = {'Equity': 'rwa_stock', 'ETF/Index': 'rwa_stock',
                    'Commodity': 'rwa_commodity'}.get(row.get('subCategory'))
        if category is None and row.get('category') == 'RWA' and row.get('subCategory') == 'Pre-market':
            category = 'rwa_stock'
        if category is None and row.get('category') == 'Crypto':
            category = 'crypto'
        if category:
            for symbol in [row['name'], row['assetName'] + '/' + row['collateralAssetName']]:
                if symbol in result and result[symbol] != category:
                    raise ValueError('conflicting official Extended category: ' + symbol)
                result[symbol] = category
    return result


def variational_name_hints(path):
    """Conservative name-based review leads, never trading identities or unit ratios."""
    import re
    rows = json.loads(Path(path).read_text())['listings']
    result = {}
    companies = {'CVNA': 'Carvana Co.', 'JPM': 'JPMorgan Chase & Co.',
                 'BSP': 'Bending Spoons S.p.A.', 'NOK': 'Nokia Oyj', 'NVO': 'Novo Nordisk A/S'}
    commodities = {'Swap on Brent Crude Oil', 'Swap on Cocoa', 'Swap on Gold Spot',
                   'Swap on Silver Spot', 'Swap on WTI Crude Oil', 'Swap on Copper Spot',
                   'Swap on Heating Oil', 'Swap on Gasoline', 'Copper'}
    indices = {'Swap on US Non-Financial 100', 'Swap on China Index', 'Swap on US 500',
               'Swap on Dollar Index', 'Swap on FANG+ Index', 'Swap on Taiwan Index',
               'Swap on US2K Spot', 'Swap on Volatility Index', 'Swap on US30 Spot',
               'Swap on US MidCap Spot'}
    for row in rows:
        symbol, name = row['ticker'], row['name']
        if not symbol or symbol in result:
            raise ValueError('missing/duplicate Variational ticker: ' + symbol)
        kind = None
        if name in commodities:
            kind = 'commodity'
        elif name in indices:
            kind = 'index'
        elif companies.get(symbol) == name or re.search(r'\b(Inc\.?|Corp\.?|Corporation|Holdings|ETF|PLC|Ltd\.?|Limited|Company|Incorporated)\b', name, re.I):
            kind = 'stock'
        result[symbol] = {'name': name, 'kind': kind,
                          'note': 'Name is a review lead; no issuer address, share class, ratio or cross-venue comparability inferred.'}
    return result
