import unittest
from security_metadata import source_identity_conflicts


class SourceIdentityConflictsTest(unittest.TestCase):
    def test_missing_fields_do_not_invent_conflicts(self):
        self.assertFalse(source_identity_conflicts([
            {'baseAsset': 'BTC', 'quoteAsset': 'USDT', 'assetClass': 'crypto'},
            {'baseAsset': '', 'quoteAsset': ''},
            {'baseAsset': 'BTC', 'assetClassHint': 'unknown'},
        ]))

    def test_known_identity_conflicts_remain_visible(self):
        for different in [
            {'baseAsset': 'ETH'}, {'quoteAsset': 'USD'},
            {'assetClassHint': 'crypto'},
        ]:
            self.assertTrue(source_identity_conflicts([
                {'baseAsset': 'BTC', 'quoteAsset': 'USDT', 'assetClass': 'rwa_stock'},
                {}, different,
            ]))

    def test_equity_labels_are_synonyms(self):
        self.assertFalse(source_identity_conflicts([
            {'assetClass': 'rwa_stock'}, {'assetClassHint': 'stock'},
            {'assetClass': 'equity'},
        ]))


if __name__ == '__main__':
    unittest.main()
