# Naver English-English Dictionary Search Workflow for Alfred 5
# Copyright (c) 2021 Jinuk Baek
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.


import sys

from workflow import web, Workflow
from search_utils import make_cache_key, url_quote

QUICK_LOOK_URL = 'https://dict.naver.com/enendict/#/search?query={}'

def get_dictionary_data(word):
    url = 'https://ac-dict.naver.com/enen3/ac'
    params = dict(st=11,
                  r_format='json',
                  r_lt=11,
                  q=word)

    r = web.get(url, params)
    r.raise_for_status()
    return r.json()


def main(wf):
    args = wf.args[0]

    it = wf.add_item(title=f"Search Naver Endic for '{args}'",
                autocomplete=args,
                arg=args,
                quicklookurl=QUICK_LOOK_URL.format(url_quote(args)),
                valid=True)
    it.setvar('lang', 'enen')

    def wrapper():
        return get_dictionary_data(args)

    res_json = wf.cached_data(make_cache_key('en1', args), wrapper, max_age=600)

    for ltxt in res_json['items'][0]:
        if len(ltxt) > 0:
            txt = ltxt[0][0]
            rtxt = ltxt[1][0]

            it = wf.add_item(title=f"{txt}     {rtxt}",
                        subtitle=f"Search Naver Endic for '{txt}'",
                        autocomplete=txt,
                        arg=txt,
                        copytext=rtxt,
                        largetext=txt,
                        quicklookurl=QUICK_LOOK_URL.format(url_quote(txt)),
                        valid=True)
            it.setvar('lang', 'enen')

    wf.send_feedback()


if __name__ == '__main__':
    wf = Workflow()
    sys.exit(wf.run(main))
