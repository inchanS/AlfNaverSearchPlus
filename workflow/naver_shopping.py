"""
Naver Search Workflow for Alfred 2
Copyright (c) 2021 Jinuk Baek

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

from workflow import web, Workflow

def get_data(word):
    url = 'https://m.shopping.naver.com/api/modules/gnb/auto-complete'

    params = dict(
        keyword=word,
        personalizeYn= "n"
    )

    r = web.get(url, params)
    r.raise_for_status()
    return r.json()['result']['keywordList']

def main(wf):
    args = wf.args[0]

    wf.add_item(title='Search Naver Shopping for \'%s\'' % args,
                autocomplete=args,
                arg=args,
                quicklookurl='https://search.shopping.naver.com/ns/search?query=%s' % args,
                valid=True)

    def wrapper():
        return get_data(args)

    res_json = wf.cached_data('navs_%s' % args, wrapper, max_age=30)

    for item in res_json:
        txt = item.get('keywordName')
        if txt:
            wf.add_item(
                title='Search Naver Shopping for \'%s\'' % txt,
                autocomplete=txt,
                arg=txt,
                copytext=txt,
                largetext=txt,
                quicklookurl='https://search.shopping.naver.com/ns/search?query=%s' % txt,
                valid=True)

    wf.send_feedback()


if __name__ == '__main__':
    wf = Workflow()
    sys.exit(wf.run(main))
