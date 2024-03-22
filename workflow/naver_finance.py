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

cache_age = int(os.getenv('cache_age', '30'))

def get_data(word):
    url = 'https://ac.stock.naver.com/ac'
    params = dict(q=word,
                  target="index,stock,marketindicator",
                  lang="ko",
                  caller="pcweb"
                  )
    headers = {"user-agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 12_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.4 Safari/605.1.15"}
    r = web.get(url, params, headers=headers)
    r.raise_for_status()
    return r.json()


def main(wf):
    args = wf.args[0]

    def wrapper():
        return get_data(args)

    res_json = wf.cached_data(f"navfinance_{args}", wrapper, max_age=cache_age)
    
    if not res_json:
        wf.add_item(
                    title=f"No search results for '{args}'",
                    icon='noresults.png',
                    valid=False)

    for item in res_json["items"]:
        code = item["code"]
        type_code = item["typeCode"]
        txt = item["name"]
        type_name = item["typeName"]
        nation_code = item["nationCode"]
        nation_name = item["nationName"]
        wf.add_item(
            title=f"Search Naver Finance for \'{txt}\'",
            subtitle=f"{code}, {type_code}, {nation_name}, {txt}",
            autocomplete=txt,
            arg=f"https://finance.naver.com/item/main.naver?code={code}",
            copytext=f"{code}, {txt}",
            largetext=f"{code}, {txt}",
            quicklookurl=f"https://finance.naver.com/item/main.naver?code={code}",
            valid=True)

    wf.send_feedback()

if __name__ == '__main__':
    wf = Workflow()
    sys.exit(wf.run(main))
