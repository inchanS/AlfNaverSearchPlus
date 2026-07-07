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

from search_utils import make_cache_key, create_workflow, add_update_item
from naver_map_common import (
    ICON_PLACE, ICON_ADDRESS, ICON_BUS, CACHE_AGE,
    get_data, get_location,
    add_place_item, add_address_item, add_bus_item,
    add_map_search_item, add_no_results_item,
)

# 카테고리 전용 검색 유형 ('place', 'address', 'bus')
MAP_TYPE = os.getenv('map_type')

# 검색 타입별 캐시 이름
CACHE_NAME_MAP = {
    'place': 'navplace',
    'address': 'navaddress',
    'bus': 'navbus'
}

# 검색 타입별 항목 생성 설정 (title 접두사, 아이콘, 생성 함수)
ITEM_BUILDER_MAP = {
    'place': ("Search Place for", ICON_PLACE, add_place_item),
    'address': ("Search Address for", ICON_ADDRESS, add_address_item),
    'bus': ("Search Bus for", ICON_BUS, add_bus_item),
}

def get_cache_key(map_type, use_ip, args):
    """
    검색 유형과 IP 사용 여부에 따른 캐시 키를 반환합니다.

    Args:
        map_type (str): 지도 검색 유형 ('place', 'address', 'bus')
        use_ip (bool): IP 기반 위치 사용 여부
        args (str): 검색어

    Returns:
        str: 캐시 키
    """
    cache_name = CACHE_NAME_MAP.get(map_type, f"nav{map_type}")
    suffix = 'ip' if use_ip else ''
    return make_cache_key(f"{cache_name}{suffix}", args)

def main(wf):
    """
    메인 함수입니다. Alfred 워크플로우의 진입점입니다.

    Args:
        wf (Workflow): Alfred 워크플로우 객체
    """
    # 새 버전이 확인된 경우 업데이트 안내 항목 추가
    add_update_item(wf)

    args = wf.args[0]
    # max_age=0: 이전 단계(naver_map)에서 저장한 설정을 시간 제한 없이 읽는다.
    # 캐시가 아예 없으면 기본 좌표를 사용하는 설정으로 동작한다.
    use_ip_data = wf.cached_data('use_ip', None, max_age=0) or {'use': False}
    use_ip = use_ip_data.get('use', True)

    # 검색 결과 캐싱 및 가져오기
    def wrapper():
        locate = get_location(wf, use_ip)
        return get_data(locate, args).get(MAP_TYPE)

    cache_key = get_cache_key(MAP_TYPE, use_ip, args)
    res_json = wf.cached_data(cache_key, wrapper, max_age=CACHE_AGE)

    # 검색 결과가 없는 경우 처리
    if not res_json:
        add_no_results_item(wf, args)

    # 기본 검색 항목 추가
    add_map_search_item(wf, args)

    # 돌아가기 옵션 추가
    it = wf.add_item(
        title=f"Return... search naver map for '{args}'",
        autocomplete=args,
        arg=args,
        valid=True
    )
    it.setvar('useIP', use_ip)
    it.setvar('return', True)

    # 검색 결과 처리
    if res_json and MAP_TYPE in ITEM_BUILDER_MAP:
        title_prefix, icon, add_item = ITEM_BUILDER_MAP[MAP_TYPE]
        for item in res_json:
            add_item(wf, item, title_prefix, icon)

    wf.send_feedback()

if __name__ == '__main__':
    wf = create_workflow()
    sys.exit(wf.run(main))
