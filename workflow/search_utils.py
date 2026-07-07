"""
Shared helpers for Naver Search Workflow scripts
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

import hashlib
from urllib.parse import quote


def make_cache_key(prefix, word):
    """
    검색어를 파일명으로 안전한 캐시 키로 변환합니다.

    캐시 키는 그대로 캐시 파일명이 되므로, 검색어에 '/' 등
    경로 구분자가 포함되면 캐시 파일 생성에 실패합니다.
    검색어를 해시하여 항상 안전한 키를 만듭니다.

    Args:
        prefix (str): 캐시 키 접두사 (예: 'nav', 'navmap')
        word (str): 검색어

    Returns:
        str: 파일명으로 안전한 캐시 키
    """
    digest = hashlib.md5(word.encode('utf-8')).hexdigest()
    return f"{prefix}_{digest}"


def url_quote(text):
    """
    URL 경로나 쿼리 값에 안전하게 넣을 수 있도록 텍스트를 인코딩합니다.

    Args:
        text: 인코딩할 텍스트

    Returns:
        str: 퍼센트 인코딩된 문자열 ('/'도 인코딩됨)
    """
    return quote(str(text), safe='')
