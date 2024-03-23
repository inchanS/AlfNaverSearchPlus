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

default_latitude = os.getenv('latitude')
default_longitude = os.getenv('longitude')
cache_age = int(os.getenv('cache_age'))
map_type = os.getenv('map_type')

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

def get_data(word, use_ip):
    if use_ip.get('use', True):
        locate = wf.cached_data('location_data', get_ip_location, max_age=cache_age)
        lat = locate['lat']
        lng = locate['lng']
    else:
        lat, lng = default_latitude, default_longitude

    url = 'https://map.naver.com/p/api/search/instant-search'
    params = dict(query=word,
                  type="all",
                  coords= f"{lat},{lng}",
                  lang="ko",
                  caller="pcweb"
                  )
    headers = {"user-agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 12_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.4 Safari/605.1.15"}
    r = web.get(url, params, headers=headers)
    r.raise_for_status()
    return r.json().get(map_type)


def main(wf):
    args = wf.args[0]

    use_ip = wf.cached_data('use_ip', None)

    def wrapper():
        return get_data(args, use_ip)

    if map_type == 'place':
        res_cache_name = 'navplace'
    elif map_type == 'address':
        res_cache_name = 'navaddress'
    elif map_type == 'bus':
        res_cache_name = 'navbus'

    if use_ip.get('use', True):
        useIP = True
        res_cache_name_ip = 'ip'
    else:
        useIP = False
        res_cache_name_ip = ''

    res_json = wf.cached_data(f"{res_cache_name}{res_cache_name_ip}_{args}", wrapper, max_age=cache_age)
    
    if not res_json:  
        wf.add_item(
                    title=f"No search results for '{args}'",
                    icon='noresults.png',
                    valid=False )

    wf.add_item(title=f"Search Bus for '{args}'",
                autocomplete=args,
                arg=f"https://map.naver.com/p/search/{args}",
                quicklookurl=f"https://map.naver.com/p/search/{args}",
                valid=True)
        
    it = wf.add_item(title=f"Return... search naver map for '{args}'",
                autocomplete=args,
                arg=args,
                valid=True)
    it.setvar('useIP', useIP)
    it.setvar('return', True)

    for item in res_json:
        if map_type == 'place':
            address_key = "roadAddress"
            txt = item["title"]
            if not item.get(address_key):
                address_key = "jibunAddress"
            address = item[address_key]
            _id = item["id"]
            type = item["type"]
            wf.add_item(
                title=f"Search Place for \'{txt}\'",
                subtitle=address,
                autocomplete=txt,
                arg=f"https://map.naver.com/p/search/{txt}/{type}/{_id}",
                copytext=txt,
                largetext=txt,
                quicklookurl=f"https://map.naver.com/p/search/{txt}/{type}/{_id}",
                icon='7FBDB33A-E342-411C-B00B-8B797AE8C19A.png',
                valid=True)
        elif map_type == 'address':
            address_key = "fullAddress"
            txt = item[address_key]
            address = item["title"]
            type = "address"
            x = item["x"]
            y = item["y"]
            wf.add_item(
                title=f"Search Address for \'{txt}\'",
                subtitle=address,
                autocomplete=txt,
                arg=f"https://map.naver.com/p/entry/{type}/{y},{x},{txt}",
                copytext=txt,
                largetext=txt,
                quicklookurl=f"https://map.naver.com/p/entry/{type}/{y},{x},{txt}",
                icon='3F6E3BB6-64CC-481E-990D-F3823D3616A8.png',
                valid=True)
        elif map_type == 'bus':
            type = "bus-route"
            address_key = item["cityName"]
            txt = item["title"]
            address = address_key + "버스 " + txt
            _id = item["id"]
            wf.add_item(
                title=f"Search Bus for \'{txt}\'",
                subtitle=address,
                autocomplete=txt,
                arg=f"https://map.naver.com/p/search/{txt}/{type}/{_id}",
                copytext=txt,
                largetext=txt,
                quicklookurl=f"https://map.naver.com/p/search/{txt}/{type}/{_id}",
                icon='845B46E7-61FB-43CD-A287-FCB4C075A4A6.png',
                valid=True)
    
    wf.send_feedback()

if __name__ == '__main__':
    wf = Workflow()
    sys.exit(wf.run(main))
