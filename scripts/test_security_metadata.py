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
