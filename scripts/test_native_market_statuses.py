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
