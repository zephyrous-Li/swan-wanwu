import http
from functools import wraps
from typing import Callable, Optional

from flask import g, request

from utils.response import BizError


def extract_bearer_token() -> Optional[str]:
    """
    从 Request Header 中提取 Bearer Token
    格式: Authorization: Bearer <API_KEY>
    """
    auth_header = request.headers.get("Authorization")
    if not auth_header:
        return None

    parts = auth_header.split(None, 1)
    if len(parts) == 2 and parts[0].lower() == "bearer":
        return parts[1]
    return None


def require_bearer_auth(f: Callable) -> Callable:
    """
    通用 Bearer Token 鉴权装饰器
    从 Authorization Header 中提取 Bearer Token 并挂载到 g.api_key
    """

    @wraps(f)
    def decorated_function(*args, **kwargs):
        api_key = extract_bearer_token()
        if not api_key:
            raise BizError(
                "Unauthorized: Missing or invalid Bearer token",
                code=http.HTTPStatus.UNAUTHORIZED,
            )
        g.api_key = api_key
        return f(*args, **kwargs)

    return decorated_function


def require_api_key(f: Callable) -> Callable:
    """
    鉴权装饰器（Tavily 专用）：
    1. 校验 Header 中的 API Key。
    2. 实例化 TavilyNewsAgency 并挂载到 g.agency。
    """
    from callback.services.tavily_news import TavilyNewsAgency

    @wraps(f)
    def decorated_function(*args, **kwargs):
        api_key = extract_bearer_token()
        if not api_key:
            raise BizError(
                "Authentication required: Please provide tavily api key.",
                code=http.HTTPStatus.UNAUTHORIZED,
            )
        g.agency = TavilyNewsAgency(api_key=api_key)
        return f(*args, **kwargs)

    return decorated_function
