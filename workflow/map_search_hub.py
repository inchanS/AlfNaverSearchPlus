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

# 환경 변수 및 상수 정의
DEFAULT_LATITUDE = os.getenv('latitude')
DEFAULT_LONGITUDE = os.getenv('longitude')
CACHE_AGE = int(os.getenv('cache_age'))
MAP_TYPE = os.getenv('map_type')

# URL 및 API 설정
API_LOCATION_URL = 'https://map.naver.com/p/api/location'
API_SEARCH_URL = 'https://map.naver.com/p/api/search/instant-search'
NAVER_MAP_SEARCH_URL = 'https://map.naver.com/p/search/{}'
NAVER_MAP_ENTRY_URL = 'https://map.naver.com/p/entry/{}/{},{},{}'
NAVER_MAP_SEARCH_TYPE_URL = 'https://map.naver.com/p/search/{}/{}/{}'

# HTTP 헤더 설정
HTTP_HEADERS = {
    "user-agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 12_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.4 Safari/605.1.15",
    "Referer": "https://map.naver.com/"
}

# 아이콘 파일 정의
ICON_NO_RESULTS = 'noresults.png'
ICON_PLACE = '7FBDB33A-E342-411C-B00B-8B797AE8C19A.png'
ICON_ADDRESS = '3F6E3BB6-64CC-481E-990D-F3823D3616A8.png'
ICON_BUS = '845B46E7-61FB-43CD-A287-FCB4C075A4A6.png'

# 검색 타입별 캐시 이름
CACHE_NAME_MAP = {
    'place': 'navplace',
    'address': 'navaddress',
    'bus': 'navbus'
}

def get_ip_location():
    """
    현재 IP 기반 위치 정보를 가져옵니다.
    
    Returns:
        tuple: 위도, 경도 튜플
    """
    try:
        r = web.get(API_LOCATION_URL)
        r.raise_for_status()
        data = r.json()
        return data["lngLat"]
    except Exception as e:
        # 위치 정보를 가져오는 데 실패할 경우 기본값 반환
        print(f"위치 정보를 가져오는 데 실패했습니다: {e}")
        return DEFAULT_LATITUDE, DEFAULT_LONGITUDE

def get_location_coordinates(use_ip, wf):
    """
    사용자 위치 정보를 가져옵니다.
    
    Args:
        use_ip (dict): IP 기반 위치 사용 여부 설정
        wf (Workflow): Alfred 워크플로우 객체
        
    Returns:
        tuple: 위도, 경도 튜플
    """
    if use_ip.get('use', True):
        locate = wf.cached_data('location_data', get_ip_location, max_age=CACHE_AGE)
        return locate['lat'], locate['lng']
    else:
        return DEFAULT_LATITUDE, DEFAULT_LONGITUDE

def get_data(word, use_ip):
    """
    검색어에 대한 데이터를 가져옵니다.
    
    Args:
        word (str): 검색할 단어
        use_ip (dict): IP 기반 위치 사용 여부 설정
        
    Returns:
        list: 검색 결과 목록
    """
    lat, lng = get_location_coordinates(use_ip, wf)

    params = dict(
        query=word,
        type="all",
        coords=f"{lat},{lng}",
        lang="ko",
        caller="pcweb"
    )
    
    r = web.get(API_SEARCH_URL, params, headers=HTTP_HEADERS)
    r.raise_for_status()
    return r.json().get(MAP_TYPE)

def get_cache_name(map_type, use_ip):
    """
    검색 유형과 IP 사용 여부에 따른 캐시 이름을 반환합니다.
    
    Args:
        map_type (str): 지도 검색 유형 ('place', 'address', 'bus')
        use_ip (dict): IP 기반 위치 사용 여부 설정
        
    Returns:
        tuple: 캐시 기본 이름, IP 접미사, IP 사용 여부
    """
    res_cache_name = CACHE_NAME_MAP.get(map_type, f"nav{map_type}")
    
    if use_ip.get('use', True):
        return res_cache_name, 'ip', True
    else:
        return res_cache_name, '', False

def process_place_item(wf, item, txt):
    """
    장소 항목을 처리합니다.
    
    Args:
        wf (Workflow): Alfred 워크플로우 객체
        item (dict): 처리할 장소 항목 데이터
        txt (str): 검색어
    """
    address_key = "roadAddress"
    txt = item["title"]
    if not item.get(address_key):
        address_key = "jibunAddress"
    address = item[address_key]
    _id = item["id"]
    type = item["type"]
    
    wf.add_item(
        title=f"Search Place for '{txt}'",
        subtitle=address,
        autocomplete=txt,
        arg=NAVER_MAP_SEARCH_TYPE_URL.format(txt, type, _id),
        copytext=txt,
        largetext=txt,
        quicklookurl=NAVER_MAP_SEARCH_TYPE_URL.format(txt, type, _id),
        icon=ICON_PLACE,
        valid=True
    )

def process_address_item(wf, item, txt):
    """
    주소 항목을 처리합니다.
    
    Args:
        wf (Workflow): Alfred 워크플로우 객체
        item (dict): 처리할 주소 항목 데이터
        txt (str): 검색어
    """
    address_key = "fullAddress"
    txt = item[address_key]
    address = item["title"]
    type = "address"
    x = item["x"]
    y = item["y"]
    
    wf.add_item(
        title=f"Search Address for '{txt}'",
        subtitle=address,
        autocomplete=txt,
        arg=NAVER_MAP_ENTRY_URL.format(type, y, x, txt),
        copytext=txt,
        largetext=txt,
        quicklookurl=NAVER_MAP_ENTRY_URL.format(type, y, x, txt),
        icon=ICON_ADDRESS,
        valid=True
    )

def process_bus_item(wf, item, txt):
    """
    버스 항목을 처리합니다.
    
    Args:
        wf (Workflow): Alfred 워크플로우 객체
        item (dict): 처리할 버스 항목 데이터
        txt (str): 검색어
    """
    type = "bus-route"
    address_key = item["cityName"]
    txt = item["title"]
    address = address_key + "버스 " + txt
    _id = item["id"]
    
    wf.add_item(
        title=f"Search Bus for '{txt}'",
        subtitle=address,
        autocomplete=txt,
        arg=NAVER_MAP_SEARCH_TYPE_URL.format(txt, type, _id),
        copytext=txt,
        largetext=txt,
        quicklookurl=NAVER_MAP_SEARCH_TYPE_URL.format(txt, type, _id),
        icon=ICON_BUS,
        valid=True
    )

def main(wf):
    """
    메인 함수입니다. Alfred 워크플로우의 진입점입니다.
    
    Args:
        wf (Workflow): Alfred 워크플로우 객체
    """
    args = wf.args[0]
    use_ip = wf.cached_data('use_ip', None)

    # 캐시 이름 생성
    res_cache_name, res_cache_name_ip, useIP = get_cache_name(MAP_TYPE, use_ip)
    cache_key = f"{res_cache_name}{res_cache_name_ip}_{args}"

    # 검색 결과 캐싱 및 가져오기
    def wrapper():
        return get_data(args, use_ip)

    res_json = wf.cached_data(cache_key, wrapper, max_age=CACHE_AGE)

    # 검색 결과가 없는 경우 처리
    if not res_json:
        wf.add_item(
            title=f"No search results for '{args}'",
            icon=ICON_NO_RESULTS,
            valid=False
        )

    # 기본 검색 항목 추가
    wf.add_item(
        title=f"Search Naver Map for '{args}'",
        autocomplete=args,
        arg=NAVER_MAP_SEARCH_URL.format(args),
        quicklookurl=NAVER_MAP_SEARCH_URL.format(args),
        valid=True
    )

    # 돌아가기 옵션 추가
    it = wf.add_item(
        title=f"Return... search naver map for '{args}'",
        autocomplete=args,
        arg=args,
        valid=True
    )
    it.setvar('useIP', useIP)
    it.setvar('return', True)

    # 검색 결과 처리
    if res_json:
        for item in res_json:
            if MAP_TYPE == 'place':
                process_place_item(wf, item, args)
            elif MAP_TYPE == 'address':
                process_address_item(wf, item, args)
            elif MAP_TYPE == 'bus':
                process_bus_item(wf, item, args)

    wf.send_feedback()

if __name__ == '__main__':
    wf = Workflow()
    sys.exit(wf.run(main))