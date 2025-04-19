"""
Naver Map Search Workflow for Alfred 5
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

from workflow import web, Workflow

# 환경 변수 및 기본값 설정
DEFAULT_LATITUDE = os.getenv('latitude', '37.5665851')
DEFAULT_LONGITUDE = os.getenv('longitude', '126.9782038')
CACHE_AGE = int(os.getenv('cache_age', '30'))

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
    """현재 IP 기반 위치 정보를 가져옵니다."""
    try:
        r = web.get(API_LOCATION_URL)
        r.raise_for_status()
        data = r.json()
        return data["lngLat"]
    except Exception as e:
        # 위치 정보를 가져오는 데 실패할 경우 기본값 반환
        print(f"위치 정보를 가져오는 데 실패했습니다: {e}")
        return DEFAULT_LATITUDE, DEFAULT_LONGITUDE

def get_data(locate, word):
    """검색어에 대한 데이터를 가져옵니다."""
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

def create_item_for_category(wf, args, category_type, icon, title_prefix):
    """카테고리 항목을 생성합니다."""
    it = wf.add_item(
        title=f"{title_prefix} '{args}'",
        autocomplete=args,
        arg=args,
        icon=icon,
        valid=True
    )
    it.setvar('map_type', category_type)
    return it

def process_address_item(wf, item):
    """주소 항목을 처리합니다."""
    ltxt = item["address"]
    address_key = "fullAddress"
    txt = ltxt[address_key]
    address = ltxt["title"]
    type = "address"
    x = ltxt["x"]
    y = ltxt["y"]
    
    wf.add_item(
        title=f"Search Naver Map for '{txt}'",
        subtitle=address,
        autocomplete=txt,
        arg=NAVER_MAP_ENTRY_URL.format(type, y, x, txt),
        copytext=txt,
        largetext=txt,
        quicklookurl=NAVER_MAP_ENTRY_URL.format(type, y, x, txt),
        valid=True
    )

def process_place_item(wf, item):
    """장소 항목을 처리합니다."""
    ltxt = item["place"]
    address_key = "roadAddress"
    txt = ltxt["title"]
    if not ltxt.get(address_key):
        address_key = "jibunAddress"
    address = ltxt[address_key]
    _id = ltxt["id"]
    type = ltxt["type"]
    
    wf.add_item(
        title=f"Search Naver Map for '{txt}'",
        subtitle=address,
        autocomplete=txt,
        arg=NAVER_MAP_SEARCH_TYPE_URL.format(txt, type, _id),
        copytext=txt,
        largetext=txt,
        quicklookurl=NAVER_MAP_SEARCH_TYPE_URL.format(txt, type, _id),
        valid=True
    )

def process_bus_item(wf, item):
    """버스 항목을 처리합니다."""
    ltxt = item["bus"]
    type = "bus-route"
    address_key = ltxt["cityName"]
    txt = ltxt["title"]
    address = address_key + "버스 " + txt
    _id = ltxt["id"]
    
    wf.add_item(
        title=f"Search Naver Map for '{txt}'",
        subtitle=address,
        autocomplete=txt,
        arg=NAVER_MAP_SEARCH_TYPE_URL.format(txt, type, _id),
        copytext=txt,
        largetext=txt,
        quicklookurl=NAVER_MAP_SEARCH_TYPE_URL.format(txt, type, _id),
        valid=True
    )

def main(wf):
    use_ip = wf.args[0]
    args = wf.args[1]

    # IP 기반 위치 정보 사용 여부 설정
    if use_ip == 'useIP':
        data_to_cache = {'use': True}
        wf.cache_data('use_ip', data_to_cache)
        locate = wf.cached_data('location_data', get_ip_location, max_age=CACHE_AGE)
        phrase = "Search Naver Map(ip) for"
        cache_key = f"navmapip_{args}"
    else:
        data_to_cache = {'use': False}
        wf.cache_data('use_ip', data_to_cache)
        locate = {'lat': DEFAULT_LATITUDE, 'lng': DEFAULT_LONGITUDE}
        phrase = "Search Naver Map for"
        cache_key = f"navmap_{args}"

    # 검색 결과 캐싱 및 가져오기
    def wrapper():
        return get_data(locate, args)

    res_json = wf.cached_data(cache_key, wrapper, max_age=CACHE_AGE)

    # 기본 검색 항목 추가
    wf.add_item(
        title=f"{phrase} '{args}'",
        autocomplete=args,
        arg=NAVER_MAP_SEARCH_URL.format(args),
        quicklookurl=NAVER_MAP_SEARCH_URL.format(args),
        valid=True
    )

    # 결과 카테고리 목록 가져오기
    place_list = res_json.get('place')
    address_list = res_json.get('address')
    bus_list = res_json.get('bus')

    # 결과가 비어있는지 확인
    is_place_empty = not place_list
    is_address_empty = not address_list
    is_bus_empty = not bus_list

    # 검색 결과가 없는 경우 메시지 표시
    if is_place_empty and is_address_empty and is_bus_empty:
        wf.add_item(
            title=f"No search results for '{args}'",
            icon=ICON_NO_RESULTS,
            valid=False
        )

    # 카테고리별 검색 옵션 추가
    if isinstance(place_list, list) and len(place_list) > 0:
        create_item_for_category(wf, args, 'place', ICON_PLACE, "Search only Place for")

    if isinstance(address_list, list) and len(address_list) > 0:
        create_item_for_category(wf, args, 'address', ICON_ADDRESS, "Search only Address for")

    if isinstance(bus_list, list) and len(bus_list) > 0:
        create_item_for_category(wf, args, 'bus', ICON_BUS, "Search only Bus for")

    # 검색 결과 항목 처리
    for item in res_json['all']:
        if item.get("address"):
            process_address_item(wf, item)
        elif item.get("place"):
            process_place_item(wf, item)
        elif item.get("bus"):
            process_bus_item(wf, item)

    wf.send_feedback()

if __name__ == '__main__':
    wf = Workflow()
    sys.exit(wf.run(main))