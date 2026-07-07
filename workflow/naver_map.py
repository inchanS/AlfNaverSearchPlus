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

from search_utils import make_cache_key, create_workflow, add_update_item
from naver_map_common import (
    ICON_PLACE, ICON_ADDRESS, ICON_BUS, CACHE_AGE,
    get_data, get_location,
    add_place_item, add_address_item, add_bus_item,
    add_map_search_item, add_no_results_item,
)

def create_item_for_category(wf, args, category_type, icon, title_prefix):
    """카테고리 전용 검색으로 이동하는 항목을 생성합니다."""
    it = wf.add_item(
        title=f"{title_prefix} '{args}'",
        autocomplete=args,
        arg=args,
        icon=icon,
        valid=True
    )
    it.setvar('map_type', category_type)
    return it

def main(wf):
    # 새 버전이 확인된 경우 업데이트 안내 항목 추가
    add_update_item(wf)

    use_ip = wf.args[0] == 'useIP'
    args = wf.args[1]

    # IP 기반 위치 사용 여부를 저장하고 위치 정보 결정
    wf.cache_data('use_ip', {'use': use_ip})
    locate = get_location(wf, use_ip)

    if use_ip:
        phrase = "Search Naver Map(ip) for"
        cache_key = make_cache_key('navmapip', args)
    else:
        phrase = "Search Naver Map for"
        cache_key = make_cache_key('navmap', args)

    # 검색 결과 캐싱 및 가져오기
    def wrapper():
        return get_data(locate, args)

    res_json = wf.cached_data(cache_key, wrapper, max_age=CACHE_AGE)

    # 기본 검색 항목 추가
    add_map_search_item(wf, args, phrase)

    # 결과 카테고리 목록 가져오기
    place_list = res_json.get('place')
    address_list = res_json.get('address')
    bus_list = res_json.get('bus')

    # 검색 결과가 없는 경우 메시지 표시
    if not place_list and not address_list and not bus_list:
        add_no_results_item(wf, args)

    # 카테고리별 검색 옵션 추가
    if place_list:
        create_item_for_category(wf, args, 'place', ICON_PLACE, "Search only Place for")

    if address_list:
        create_item_for_category(wf, args, 'address', ICON_ADDRESS, "Search only Address for")

    if bus_list:
        create_item_for_category(wf, args, 'bus', ICON_BUS, "Search only Bus for")

    # 검색 결과 항목 처리
    for item in res_json['all']:
        if item.get("address"):
            add_address_item(wf, item["address"])
        elif item.get("place"):
            add_place_item(wf, item["place"])
        elif item.get("bus"):
            add_bus_item(wf, item["bus"])

    wf.send_feedback()

if __name__ == '__main__':
    wf = create_workflow()
    sys.exit(wf.run(main))
