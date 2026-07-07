"""
Naver Map Search common module for Alfred 5
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

import os

from workflow import web
from search_utils import url_quote

# 환경 변수 및 기본값 설정 (기본 좌표: 서울시청)
DEFAULT_LATITUDE = os.getenv('latitude') or '37.5665851'
DEFAULT_LONGITUDE = os.getenv('longitude') or '126.9782038'
CACHE_AGE = int(os.getenv('cache_age') or '30')

# URL 및 API 설정
API_LOCATION_URL = 'https://map.naver.com/p/api/location'
API_SEARCH_URL = 'https://map.naver.com/p/api/search/instant-search'
NAVER_MAP_SEARCH_URL = 'https://map.naver.com/p/search/{}'
NAVER_MAP_ENTRY_URL = 'https://map.naver.com/p/entry/{}/{},{},{}'
NAVER_MAP_SEARCH_TYPE_URL = 'https://map.naver.com/p/search/{}/{}/{}'

# HTTP 헤더 설정
HTTP_HEADERS = {
    "user-agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Safari/605.1.15",
    "Referer": "https://map.naver.com/"
}

# 아이콘 파일 이름
ICON_NO_RESULTS = 'noresults.png'
ICON_PLACE = '7FBDB33A-E342-411C-B00B-8B797AE8C19A.png'
ICON_ADDRESS = '3F6E3BB6-64CC-481E-990D-F3823D3616A8.png'
ICON_BUS = '845B46E7-61FB-43CD-A287-FCB4C075A4A6.png'

def get_ip_location():
    """
    현재 IP 기반 위치 정보를 가져옵니다.

    Returns:
        dict: {'lat': 위도, 'lng': 경도} 형태의 위치 정보
    """
    r = web.get(API_LOCATION_URL)
    r.raise_for_status()
    data = r.json()
    return data["lngLat"]

def get_location(wf, use_ip):
    """
    사용자 위치 정보를 가져옵니다.

    Args:
        wf (Workflow): Alfred 워크플로우 객체
        use_ip (bool): IP 기반 위치 사용 여부

    Returns:
        dict: {'lat': 위도, 'lng': 경도} 형태의 위치 정보
    """
    if use_ip:
        try:
            return wf.cached_data('location_data', get_ip_location, max_age=CACHE_AGE)
        except Exception as e:
            # 위치 정보를 가져오는 데 실패할 경우 기본 좌표 사용
            # (stdout은 Alfred 피드백 전용이므로 로그로만 남긴다)
            wf.logger.error(f"위치 정보를 가져오는 데 실패했습니다: {e}")
    return {'lat': DEFAULT_LATITUDE, 'lng': DEFAULT_LONGITUDE}

def get_data(locate, word):
    """
    검색어에 대한 즉시검색(instant-search) 결과를 가져옵니다.

    Args:
        locate (dict): {'lat': 위도, 'lng': 경도} 형태의 위치 정보
        word (str): 검색할 단어

    Returns:
        dict: 카테고리별(all, place, address, bus) 검색 결과
    """
    params = dict(
        query=word,
        type="all",
        coords=f"{locate['lat']},{locate['lng']}",
        lang="ko",
        caller="pcweb"
    )

    r = web.get(API_SEARCH_URL, params, headers=HTTP_HEADERS)
    r.raise_for_status()
    return r.json()

def add_place_item(wf, place, title_prefix="Search Naver Map for", icon=None):
    """
    장소 항목을 워크플로우에 추가합니다.

    Args:
        wf (Workflow): Alfred 워크플로우 객체
        place (dict): 장소 데이터 (title, type, id, 주소 필드 포함)
        title_prefix (str): 항목 제목 접두사
        icon (str): 아이콘 파일 이름 (없으면 워크플로우 기본 아이콘)
    """
    txt = place["title"]
    address = place.get("roadAddress") or place.get("jibunAddress")
    url = NAVER_MAP_SEARCH_TYPE_URL.format(url_quote(txt), place["type"], place["id"])

    wf.add_item(
        title=f"{title_prefix} '{txt}'",
        subtitle=address,
        autocomplete=txt,
        arg=url,
        copytext=txt,
        largetext=txt,
        quicklookurl=url,
        icon=icon,
        valid=True
    )

def add_address_item(wf, address, title_prefix="Search Naver Map for", icon=None):
    """
    주소 항목을 워크플로우에 추가합니다.

    Args:
        wf (Workflow): Alfred 워크플로우 객체
        address (dict): 주소 데이터 (fullAddress, title, x, y 포함)
        title_prefix (str): 항목 제목 접두사
        icon (str): 아이콘 파일 이름 (없으면 워크플로우 기본 아이콘)
    """
    txt = address["fullAddress"]
    url = NAVER_MAP_ENTRY_URL.format("address", address["y"], address["x"], url_quote(txt))

    wf.add_item(
        title=f"{title_prefix} '{txt}'",
        subtitle=address["title"],
        autocomplete=txt,
        arg=url,
        copytext=txt,
        largetext=txt,
        quicklookurl=url,
        icon=icon,
        valid=True
    )

def add_bus_item(wf, bus, title_prefix="Search Naver Map for", icon=None):
    """
    버스 항목을 워크플로우에 추가합니다.

    Args:
        wf (Workflow): Alfred 워크플로우 객체
        bus (dict): 버스 데이터 (title, cityName, id 포함)
        title_prefix (str): 항목 제목 접두사
        icon (str): 아이콘 파일 이름 (없으면 워크플로우 기본 아이콘)
    """
    txt = bus["title"]
    subtitle = bus["cityName"] + "버스 " + txt
    url = NAVER_MAP_SEARCH_TYPE_URL.format(url_quote(txt), "bus-route", bus["id"])

    wf.add_item(
        title=f"{title_prefix} '{txt}'",
        subtitle=subtitle,
        autocomplete=txt,
        arg=url,
        copytext=txt,
        largetext=txt,
        quicklookurl=url,
        icon=icon,
        valid=True
    )

def add_map_search_item(wf, args, title_prefix="Search Naver Map for"):
    """
    네이버 지도 통합 검색으로 이동하는 기본 항목을 추가합니다.

    Args:
        wf (Workflow): Alfred 워크플로우 객체
        args (str): 검색어
        title_prefix (str): 항목 제목 접두사
    """
    wf.add_item(
        title=f"{title_prefix} '{args}'",
        autocomplete=args,
        arg=NAVER_MAP_SEARCH_URL.format(url_quote(args)),
        quicklookurl=NAVER_MAP_SEARCH_URL.format(url_quote(args)),
        valid=True
    )

def add_no_results_item(wf, args):
    """
    검색 결과가 없음을 알리는 항목을 추가합니다.

    Args:
        wf (Workflow): Alfred 워크플로우 객체
        args (str): 검색어
    """
    wf.add_item(
        title=f"No search results for '{args}'",
        icon=ICON_NO_RESULTS,
        valid=False
    )
