import json
import tempfile
import unittest
from pathlib import Path
from security_metadata import extended_categories


class ExtendedCategoryTests(unittest.TestCase):
    def test_official_native_and_display_variants_keep_category(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'markets.json'
            path.write_text(json.dumps({'data': [
                dict(type='PERPETUAL', name='NOW-USD', assetName='NOW', collateralAssetName='USD', category='RWA', subCategory='Equity'),
                dict(type='PERPETUAL', name='BILL-USD', assetName='BILL', collateralAssetName='USD', category='Crypto'),
                dict(type='PERPETUAL', name='LYTE-USD', assetName='LYTE', collateralAssetName='USD', category='RWA', subCategory='ETF/Index')]}))
            result = extended_categories(path)
            self.assertEqual(result['NOW/USD'], 'rwa_stock')
            self.assertEqual(result['BILL/USD'], 'crypto')
            self.assertEqual(result['LYTE/USD'], 'rwa_stock')
            self.assertEqual(result['NOW-USD'], result['NOW/USD'])
