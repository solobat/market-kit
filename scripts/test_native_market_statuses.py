import json
import tempfile
import unittest
from pathlib import Path
from security_metadata import native_market_statuses


class NativeStatusTests(unittest.TestCase):
    def test_inactive_evidence_stays_exact_and_contract_scoped(self):
        with tempfile.TemporaryDirectory() as directory:
            a = Path(directory) / 'aster.json'
            l = Path(directory) / 'lighter.json'
            a.write_text(json.dumps({'symbols': [
                dict(symbol='OLDUSDT', contractType='PERPETUAL', status='SETTLING'),
                dict(symbol='LIVEUSDT', contractType='PERPETUAL', status='TRADING'),
                dict(symbol='SPOT', contractType='', status='SETTLING')]}))
            l.write_text(json.dumps({'order_books': [
                dict(symbol='SKHYNIX', market_type='perp', market_id=143, status='inactive'),
                dict(symbol='COIN/USDC', market_type='spot', market_id=2048, status='active')]}))
            rows = native_market_statuses(a, l)
        self.assertTrue(rows[('aster', 'OLDUSDT', 'perpetual')]['inactive'])
        self.assertFalse(rows[('aster', 'LIVEUSDT', 'perpetual')]['inactive'])
        self.assertNotIn(('aster', 'SPOT', 'perpetual'), rows)
        self.assertTrue(rows[('lighter', 'SKHYNIX', 'perpetual')]['inactive'])
        self.assertNotIn(('lighter', 'SKHYNIXUSD', 'perpetual'), rows)
        self.assertFalse(rows[('lighter', 'COIN/USDC', 'spot')]['inactive'])

    def test_binance_nontrading_states_do_not_assume_expiry_or_aliases(self):
        with tempfile.TemporaryDirectory() as directory:
            p = Path(directory) / 'binance.json'
            p.write_text(json.dumps({'symbols': [
                dict(symbol='NEWUSDC', contractType='PERPETUAL', status='PENDING_TRADING', marginAsset='USDC', deliveryDate=4133404800000),
                dict(symbol='STOCKUSDT', contractType='TRADIFI_PERPETUAL', status='SETTLING', marginAsset='USDT'),
                dict(symbol='LIVEUSDT', contractType='PERPETUAL', status='TRADING'),
                dict(symbol='UNKNOWN', contractType='', status='SETTLING')]}))
            rows = native_market_statuses(binance_path=p)
        self.assertTrue(rows[('binance', 'NEWUSDC', 'perpetual')]['inactive'])
        self.assertTrue(rows[('binance', 'STOCKUSDT', 'perpetual')]['inactive'])
        self.assertFalse(rows[('binance', 'LIVEUSDT', 'perpetual')]['inactive'])
        self.assertNotIn('expiryAtMs', rows[('binance', 'NEWUSDC', 'perpetual')])
        self.assertNotIn(('binance', 'NEWUSDC', 'spot'), rows)
        self.assertNotIn(('binance', 'UNKNOWN', 'perpetual'), rows)
