import unittest
from security_metadata import compare_catalogs


class SecurityMetadataTests(unittest.TestCase):
    def test_changed_isin_is_not_automatically_a_split(self):
        out = compare_catalogs({'RNHPUSDT': {'isin': 'US42226B6002'}},
                               {'RNHPUSDT': {'isin': 'US42226B5012'}})
        change = out['changes']['RNHPUSDT'][0]
        self.assertEqual(change['kind'], 'security_identifier_changed')
        self.assertNotIn('ratio', change)

    def test_enrichment_status_and_absence_do_not_become_identity_events(self):
        out = compare_catalogs({'ADRUSDT': {'isin': 'US00762U2006', 'status': 'limit_open'}},
                               {'ADRUSDT': {'isin': None, 'status': 'online'}, 'OLD': {}})
        self.assertEqual([c['kind'] for c in out['changes']['ADRUSDT']],
                         ['security_identifier_enriched', 'trading_status_changed'])
        self.assertEqual(out['absentSymbols'], ['OLD'])

    def test_cosmetic_case_and_missing_values_do_not_create_drift(self):
        out = compare_catalogs({'A': {'isin': 'us00762u2006', 'status': 'ONLINE'}},
                               {'A': {'isin': 'US00762U2006', 'status': 'online', 'fullName': None}})
        self.assertEqual(out['changes'], {})


class ExternalMarketEvidenceTests(unittest.TestCase):
    def test_external_ids_are_evidence_not_symbol_aliases(self):
        import json
        import tempfile
        from pathlib import Path
        from security_metadata import native_market_statuses
        with tempfile.TemporaryDirectory() as folder:
            path = Path(folder) / 'native.json'
            path.write_text(json.dumps({'order_books': [
                {'symbol': 'OLD', 'market_id': 150, 'market_type': 'perp', 'status': 'inactive', 'market_deployer': 'lighter'},
                {'symbol': 'NEW', 'market_id': 8286, 'market_type': 'perp', 'status': 'active', 'market_deployer': 'external'}]}))
            evidence = native_market_statuses(lighter_path=path)
        self.assertEqual(len(evidence), 2)
        self.assertTrue(evidence[('lighter', 'OLD', 'perpetual')]['inactive'])
        current = evidence[('lighter', 'NEW', 'perpetual')]
        self.assertEqual(current['marketId'], 8286)
        self.assertEqual(current['marketDeployer'], 'external')
        self.assertNotIn('assetClass', current)
        self.assertNotIn('alias', current)


class ExpiryEvidenceTests(unittest.TestCase):
    def test_expiry_boundary_and_explicit_metadata_only(self):
        from security_metadata import expired_contract_review
        self.assertFalse(expired_contract_review({'rawSymbol': 'BTCUSDT-06NOV26'}, 2000))
        self.assertFalse(expired_contract_review({'expiryAtMs': 2001}, 2000))
        self.assertTrue(expired_contract_review({'expiryAtMs': 2000, 'comparisonStatus': 'ambiguous'}, 2000))
        self.assertFalse(expired_contract_review({'expiryAtMs': 1999, 'comparisonStatus': 'prohibited'}, 2000))
        self.assertFalse(expired_contract_review({'expiryAtMs': 'not-a-timestamp'}, 2000))


class NativeQuarterlyEvidenceTests(unittest.TestCase):
    def test_native_type_supplies_expiry_not_name_suffix(self):
        import json
        import tempfile
        from pathlib import Path
        from security_metadata import native_market_statuses
        with tempfile.TemporaryDirectory() as folder:
            path = Path(folder) / 'native.json'
            path.write_text(json.dumps({'symbols': [
                {'symbol': 'BTCUSDT_261225', 'contractType': 'CURRENT_QUARTER', 'deliveryDate': 1798185600000, 'marginAsset': 'USDT', 'baseAsset': 'BTC', 'quoteAsset': 'USDT'},
                {'symbol': 'ODD_261225', 'contractType': 'PERPETUAL'}]}))
            evidence = native_market_statuses(binance_path=path)
        self.assertEqual(len(evidence), 2)
        self.assertNotIn(('binance', 'ODD_261225', 'future'), evidence)
        row = evidence[('binance', 'BTCUSDT_261225', 'future')]
        self.assertEqual(row['expiryAtMs'], 1798185600000)
        self.assertEqual(row['settlementAsset'], 'USDT')
