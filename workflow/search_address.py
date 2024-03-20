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

default_latitude = os.environ.get('latitude')
default_longitude = os.environ.get('longitude')
cache_age = int(os.environ.get('cache_age'))

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
    return r.json().get("address")


def main(wf):
    args = wf.args[0]

    wf.add_item(title=f"Search Bus for '{args}'",
                autocomplete=args,
                arg=f"https://map.naver.com/p/search/{args}",
                quicklookurl=f"https://map.naver.com/p/search/{args}",
                valid=True)
    
    use_ip = wf.cached_data('use_ip', None)

    def wrapper():
        return get_data(args, use_ip)

    if use_ip.get('use', True):
        res_json = wf.cached_data(f"navaddressip_{args}", wrapper, max_age=cache_age)
    else:
        res_json = wf.cached_data(f"navaddress_{args}", wrapper, max_age=cache_age)

    if not res_json:  
        wf.add_item(
                    title=f"No search results for '{args}'",
                    icon='noresults.png',
                    valid=False)

    for item in res_json:
        address_key = "fullAddress"
        txt = item[address_key]
        address = item["title"]
        type = "address"
        x = item["x"]
        y = item["y"]
        wf.add_item(
            title=f"Search Naver Map for \'{txt}\'",
            subtitle=address,
            autocomplete=txt,
            arg=f"https://map.naver.com/p/entry/{type}/{y},{x},{txt}",
            copytext=txt,
            largetext=txt,
            quicklookurl=f"https://map.naver.com/p/entry/{type}/{y},{x},{txt}",
            valid=True)
    
    wf.send_feedback()

if __name__ == '__main__':
    wf = Workflow()
    sys.exit(wf.run(main))
