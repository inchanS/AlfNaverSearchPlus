"""
Naver Finance Search Workflow for Alfred 5
Copyright (c) 2024 Inchan Song

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
import os

from workflow import web
from search_utils import make_cache_key, url_quote, create_workflow, add_update_item

# 환경 변수 및 상수 정의
CACHE_AGE = int(os.getenv('cache_age') or '30')

# API 및 URL 설정
API_SEARCH_URL = 'https://ac.stock.naver.com/ac'
FINANCE_ITEM_URL = 'https://finance.naver.com/item/main.naver?code={}'

# HTTP 헤더 설정
HTTP_HEADERS = {
    "user-agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 12_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.4 Safari/605.1.15"
}

# 아이콘 파일
ICON_NO_RESULTS = 'noresults.png'

def get_data(word):
    """
    네이버 금융 검색 API에서 주식 정보를 검색합니다.
    
    Args:
        word (str): 검색할 단어
        
    Returns:
        dict: 검색 결과 데이터
    """
    params = dict(
        q=word,
        target="index,stock,marketindicator",
        lang="ko",
        caller="pcweb"
    )
    
    r = web.get(API_SEARCH_URL, params, headers=HTTP_HEADERS)
    r.raise_for_status()
    return r.json()

def format_item_subtitle(name, code, type_code, nation_name):
    """
    항목의 부제목을 형식화합니다. (종목명-코드-시장구분-국가 순)

    Args:
        name (str): 종목 이름
        code (str): 종목 코드
        type_code (str): 시장 구분 코드
        nation_name (str): 국가 이름

    Returns:
        str: 형식화된 부제목
    """
    return f"{name}, {code}, {type_code}, {nation_name}"

def process_finance_item(wf, item):
    """
    금융 항목을 처리하여 워크플로우에 추가합니다.
    
    Args:
        wf (Workflow): Alfred 워크플로우 객체
        item (dict): 처리할 금융 항목 데이터
    """
    code = item["code"]
    type_code = item["typeCode"]
    txt = item["name"]
    type_name = item["typeName"]
    nation_code = item["nationCode"]
    nation_name = item["nationName"]
    
    subtitle = format_item_subtitle(txt, code, type_code, nation_name)
    copy_text = subtitle
    url = FINANCE_ITEM_URL.format(url_quote(code))
    
    wf.add_item(
        title=f"Search Naver Finance for '{txt}'",
        subtitle=subtitle,
        autocomplete=txt,
        arg=url,
        copytext=copy_text,
        largetext=copy_text,
        quicklookurl=url,
        valid=True
    )

def main(wf):
    """
    메인 함수입니다. Alfred 워크플로우의 진입점입니다.
    
    Args:
        wf (Workflow): Alfred 워크플로우 객체
    """
    # 새 버전이 확인된 경우 업데이트 안내 항목 추가
    add_update_item(wf)

    args = wf.args[0]
    cache_key = make_cache_key('navfinance', args)

    # 검색 결과를 캐싱하여 가져오기
    def wrapper():
        return get_data(args)

    res_json = wf.cached_data(cache_key, wrapper, max_age=CACHE_AGE)
    
    # 검색 결과가 없는 경우 처리
    if not res_json or not res_json.get("items"):
        wf.add_item(
            title=f"No search results for '{args}'",
            icon=ICON_NO_RESULTS,
            valid=False
        )
        wf.send_feedback()
        return

    # 검색 결과 항목 처리
    for item in res_json["items"]:
        process_finance_item(wf, item)

    wf.send_feedback()

if __name__ == '__main__':
    wf = create_workflow()
    sys.exit(wf.run(main))