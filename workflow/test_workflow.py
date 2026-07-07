import os
import unittest

import naver_search
import naver_shopping
import naver_terms
import naver_map
import naver_map_common
import map_search_hub
import krdic_naver_search
import hanja_naver_search
import endic_naver_search
import enendic_naver_search
import common_naver_search
import naver_finance


class MyTestCase(unittest.TestCase):
    def test_naver_search(self):
        res = naver_search.get_data('한글')

        self.assertTrue(len(res['items']) > 0)

    # FIXME 250224 - naver shopping API 변경 이후,
    #  로컬에서는 테스트가 통과하지만 github action 서버에서는 API에 정상적인 접근이 안되고 있다.
    @unittest.skipIf(os.getenv('CI'), 'GitHub Actions에서는 네이버 쇼핑 API 접근이 차단됨')
    def test_shopping(self):
        res = naver_shopping.get_data('한글')

        self.assertTrue(len(res) > 0)

    def test_terms(self):
        res = naver_terms.get_data('한글')

        self.assertTrue(len(res['items']) > 0)

    def test_naver_map(self):
        locate = {'lat': '37.5665', 'lng': '126.9780'}
        res = naver_map_common.get_data(locate, '서울')

        self.assertTrue(len(res) > 0)
        # 카테고리 전용 검색(map_search_hub)이 사용하는 카테고리 키 확인
        self.assertIn('all', res)
        self.assertIn('place', res)

    def test_krdic(self):
        res = krdic_naver_search.get_dictionary_data('한글')

        self.assertTrue(len(res['items']) > 0)

    def test_hanjadic(self):
        res = hanja_naver_search.get_dictionary_data('한글')

        self.assertTrue(len(res['items']) > 0)

    def test_endic(self):
        res = endic_naver_search.get_dictionary_data('한글')

        self.assertTrue(len(res['items']) > 0)

    def test_enendic(self):
        res = enendic_naver_search.get_dictionary_data('한글')

        self.assertTrue(len(res['items']) > 0)

    def test_common_naver_dic(self):
        langs = ['ja', 'zh', 'vi', 'th', 'uz', 'de', 'fr', 'it', 'es', 'ru', 'id', 'ne', 'mn', 'my', 'sw', 'ar',
                 'km', 'fa', 'hi', 'nl', 'sv', 'uk', 'ka', 'cs', 'hr', 'tr', 'pt', 'pl', 'fi', 'hu', 'sq', 'ro',
                 'la', 'el']

        for lang in langs:
            res = common_naver_search.get_dictionary_data(lang + 'ko', '한글')

            self.assertTrue(len(res['items']) > 0)

    def test_naver_finance(self):
        res = naver_finance.get_data('삼성')

        self.assertTrue(len(res['items']) > 0)


if __name__ == '__main__':
    unittest.main()
