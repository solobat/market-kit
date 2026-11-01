import json
import tempfile
import unittest
from pathlib import Path
from security_metadata import variational_name_hints


class VariationalNameHintsTests(unittest.TestCase):
    def test_names_distinguish_etf_index_commodity_and_crypto_without_ratios(self):
        rows = [('ZEC', 'Zcash'), ('ZCSH', 'Grayscale Zcash ETF'),
                ('US100S', 'Swap on US Non-Financial 100'),
                ('USOILP', 'Swap on WTI Crude Oil'), ('AGLD', 'Adventure Gold'),
                ('TZA', 'Direxion Daily Small Cap Bear 3X ETF')]
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'stats.json'
            path.write_text(json.dumps({'listings': [dict(ticker=t, name=n) for t, n in rows]}))
            hints = variational_name_hints(path)
        self.assertIsNone(hints['ZEC']['kind'])
        self.assertIsNone(hints['AGLD']['kind'])
        self.assertEqual(hints['ZCSH']['kind'], 'stock')
        self.assertEqual(hints['US100S']['kind'], 'index')
        self.assertEqual(hints['USOILP']['kind'], 'commodity')
        self.assertEqual(hints['TZA']['kind'], 'stock')
        self.assertNotIn('multiplier', hints['TZA'])
