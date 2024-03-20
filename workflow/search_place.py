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
query_cache_age = int(os.environ.get('query_cache_age'))

def get_data(word):
    url = 'https://map.naver.com/p/api/search/instant-search'
    params = dict(query=word,
                  type="all",
                  coords= f'{default_latitude},{default_longitude}',
                  lang="ko",
                  caller="pcweb"
                  )
    headers = {"user-agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 12_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.4 Safari/605.1.15"}
    r = web.get(url, params, headers=headers)
    r.raise_for_status()
    return r.json().get("place")


def main(wf):
    args = wf.args[0]

    wf.add_item(title=f"Search Place for '{args}'",
                autocomplete=args,
                arg=f"https://map.naver.com/p/search/{args}",
                quicklookurl=f"https://map.naver.com/p/search/{args}",
                valid=True)
    
    def wrapper():
        return get_data(args)

    res_json = wf.cached_data(f"navplace_{args}", wrapper, max_age=query_cache_age)

    if not res_json:  
        wf.add_item(
                    title=f"No search results for '{args}'",
                    valid=False)

    for item in res_json:
        address_key = "roadAddress"
        txt = item["title"]
        if not item.get(address_key):
            address_key = "jibunAddress"
        address = item[address_key]
        _id = item["id"]
        type = item["type"]
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
