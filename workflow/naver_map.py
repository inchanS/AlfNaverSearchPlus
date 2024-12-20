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

# 위치 기본좌표는 서울시청
default_latitude = os.getenv('latitude', '37.5665851')
default_longitude = os.getenv('longitude', '126.9782038')
cache_age = int(os.getenv('cache_age', '30'))

def get_ip_location():
    try:
        r = web.get('https://map.naver.com/p/api/location')
        r.raise_for_status()
        data = r.json()
        return data["lngLat"]
    except Exception as e:
        # 위치 정보를 가져오는 데 실패할 경우 기본값 반환
        print(f"위치 정보를 가져오는 데 실패했습니다: {e}")
        return default_latitude, default_longitude

def get_data(locate, word):
    url = 'https://map.naver.com/p/api/search/instant-search'
    params = dict(query=word,
                  type="all",
                  coords=f"{locate['lat']},{locate['lng']}",
                  lang="ko",
                  caller="pcweb"
                  )
    headers = {
                "user-agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Safari/605.1.15",
                "Referer": "https://map.naver.com/"}
    r = web.get(url, params, headers=headers)
    r.raise_for_status()
    return r.json()


def main(wf):
    use_ip = wf.args[0]
    args = wf.args[1]

    if use_ip == 'useIP':
        data_to_cache = {'use': True}
        wf.cache_data('use_ip', data_to_cache)
        locate = wf.cached_data('location_data', get_ip_location, max_age=cache_age)
        phrase = "Search Naver Map(ip) for"

        def wrapper():
            return get_data(locate, args)

        res_json = wf.cached_data(f"navmapip_{args}", wrapper, max_age=cache_age)

    else:
        data_to_cache = {'use': False}
        wf.cache_data('use_ip', data_to_cache)
        locate = {'lat': default_latitude, 'lng': default_longitude}
        phrase = "Search Naver Map for"

        def wrapper():
            return get_data(locate, args)

        res_json = wf.cached_data(f"navmap_{args}", wrapper, max_age=cache_age)

    wf.add_item(title=f"{phrase} '{args}'",
                autocomplete=args,
                arg=f"https://map.naver.com/p/search/{args}",
                quicklookurl=f"https://map.naver.com/p/search/{args}",
                valid=True)

    place_list = res_json.get('place')
    address_list = res_json.get('address')
    bus_list = res_json.get('bus')

    is_place_empty = not res_json.get('place')
    is_address_empty = not res_json.get('address')
    is_bus_empty = not res_json.get('bus')

    if is_place_empty and is_address_empty and is_bus_empty:
        wf.add_item(
                    title=f"No search results for '{args}'",
                    icon='noresults.png',
                    valid=False)

    if isinstance(place_list, list) and len(place_list) > 0:
        it = wf.add_item(title=f"Search only Place for '{args}'",
                         autocomplete=args,
                         arg=args,
                         icon='7FBDB33A-E342-411C-B00B-8B797AE8C19A.png',
                         valid=True)
        it.setvar('map_type', 'place')

    if isinstance(address_list, list) and len(address_list) > 0:
        it = wf.add_item(title=f"Search only Address for '{args}'",
                         autocomplete=args,
                         arg=args,
                         icon='3F6E3BB6-64CC-481E-990D-F3823D3616A8.png',
                         valid=True)
        it.setvar('map_type', 'address')

    if isinstance(bus_list, list) and len(bus_list) > 0:
        it = wf.add_item(title=f"Search only Bus for '{args}'",
                         autocomplete=args,
                         arg=args,
                         icon='845B46E7-61FB-43CD-A287-FCB4C075A4A6.png',
                         valid=True)
        it.setvar('map_type', 'bus')

    for item in res_json['all']:
        if item.get("address"):
            ltxt = item["address"]
            address_key = "fullAddress"
            txt = ltxt[address_key]
            address = ltxt["title"]
            type = "address"
            x = ltxt["x"]
            y = ltxt["y"]
            wf.add_item(
                title=f"Search Naver Map for \'{txt}\'",
                subtitle=address,
                autocomplete=txt,
                arg=f"https://map.naver.com/p/entry/{type}/{y},{x},{txt}",
                copytext=txt,
                largetext=txt,
                quicklookurl=f"https://map.naver.com/p/entry/{type}/{y},{x},{txt}",
                valid=True)
        elif item.get("place"):
            ltxt = item["place"]
            address_key = "roadAddress"
            txt = ltxt["title"]
            if not ltxt.get(address_key):
                address_key = "jibunAddress"
            address = ltxt[address_key]
            _id = ltxt["id"]
            type = ltxt["type"]
            wf.add_item(
                title=f"Search Naver Map for \'{txt}\'",
                subtitle=address,
                autocomplete=txt,
                arg=f"https://map.naver.com/p/search/{txt}/{type}/{_id}",
                copytext=txt,
                largetext=txt,
                quicklookurl=f"https://map.naver.com/p/search/{txt}/{type}/{_id}",
                valid=True)
        elif item.get("bus"):
            ltxt = item["bus"]
            type = "bus-route"
            address_key = ltxt["cityName"]
            txt = ltxt["title"]
            address = address_key + "버스 " + txt
            _id = ltxt["id"]
            wf.add_item(
                title=f"Search Naver Map for \'{txt}\'",
                subtitle=address,
                autocomplete=txt,
                arg=f"https://map.naver.com/p/search/{txt}/{type}/{_id}",
                copytext=txt,
                largetext=txt,
                quicklookurl=f"https://map.naver.com/p/search/{txt}/{type}/{_id}",
                valid=True)

    wf.send_feedback()

if __name__ == '__main__':
    wf = Workflow()
    sys.exit(wf.run(main))
