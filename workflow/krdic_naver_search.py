"""
Naver Korean Dictionary Search Workflow for Alfred 5
Copyright (c) 2021 Jinuk Baek

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
"""

import sys

from workflow import web, Workflow

# 상수 정의
API_URL = 'https://ac-dict.naver.com/koko/ac'
SEARCH_URL = 'https://ko.dict.naver.com/#/search?query={}'
CACHE_MAX_AGE = 600
LANGUAGE_CODE = 'koko'

def get_dictionary_data(word):
    """
    네이버 국어사전 API에서 단어 정보를 검색합니다.
    
    Args:
        word (str): 검색할 단어
        
    Returns:
        dict: 검색 결과 데이터
    """
    params = dict(
        frm='stdkrdic',
        oe='utf8',
        m=0,
        r=1,
        st=111,
        r_lt=111,
        q=word
    )

    r = web.get(API_URL, params)
    r.raise_for_status()
    return r.json()

def create_main_search_item(wf, query):
    """
    기본 검색 항목을 생성합니다.
    
    Args:
        wf (Workflow): Alfred 워크플로우 객체
        query (str): 검색 쿼리
        
    Returns:
        Item: 생성된 워크플로우 아이템
    """
    item = wf.add_item(
        title=f"Search Naver Krdic for '{query}'",
        autocomplete=query,
        arg=query,
        quicklookurl=SEARCH_URL.format(query),
        valid=True
    )
    item.setvar('lang', LANGUAGE_CODE)
    return item

def create_result_item(wf, text, query):
    """
    검색 결과 항목을 생성합니다.
    
    Args:
        wf (Workflow): Alfred 워크플로우 객체
        text (str): 검색 결과 텍스트
        query (str): 원본 검색 쿼리
        
    Returns:
        Item: 생성된 워크플로우 아이템
    """
    item = wf.add_item(
        title=f"{text}",
        subtitle=f"Search Naver Krdic for '{text}'",
        autocomplete=text,
        arg=text,
        copytext=text,
        largetext=text,
        quicklookurl=SEARCH_URL.format(text),
        valid=True
    )
    item.setvar('lang', LANGUAGE_CODE)
    return item

def process_search_results(wf, results, query):
    """
    검색 결과를 처리하여 워크플로우 아이템으로 변환합니다.
    
    Args:
        wf (Workflow): Alfred 워크플로우 객체
        results (dict): 검색 결과 데이터
        query (str): 원본 검색 쿼리
    """
    for items in results['items']:
        for ltxt in items:
            if len(ltxt) > 0:
                txt = ltxt[0][0]
                create_result_item(wf, txt, query)

def main(wf):
    """
    메인 함수입니다. Alfred 워크플로우의 진입점입니다.
    
    Args:
        wf (Workflow): Alfred 워크플로우 객체
    """
    # 검색 쿼리 가져오기
    query = wf.args[0]
    
    # 기본 검색 항목 추가
    create_main_search_item(wf, query)
    
    # 캐시 키 생성 및 결과 데이터 가져오기
    cache_key = f"kr_{query}"
    
    def wrapper():
        return get_dictionary_data(query)
    
    results = wf.cached_data(cache_key, wrapper, max_age=CACHE_MAX_AGE)
    
    # 검색 결과 처리
    process_search_results(wf, results, query)
    
    # 결과 전송
    wf.send_feedback()

if __name__ == '__main__':
    wf = Workflow()
    sys.exit(wf.run(main))