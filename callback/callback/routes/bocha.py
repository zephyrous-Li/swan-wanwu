import http
from dataclasses import asdict

from flask import g, jsonify, request

from callback.services import bocha
from callback.utils.decorators import require_bearer_auth
from utils.response import BizError

from . import callback_bp

search_client = bocha.BochaMultimodalSearch()


@callback_bp.route("/bocha/comprehensive", methods=["POST"])
@require_bearer_auth
def bocha_comprehensive_search():
    """
    【工具】全面综合搜索
    ---
    tags:
      - Bocha Search
    summary: 执行多模态综合搜索
    description: 返回网页、图片及模态卡片信息。需在 Header 中携带 Bearer Token。
    parameters:
      - name: Authorization
        in: header
        schema:
          type: string
        required: true
        description: Bearer <API_KEY>
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            required:
              - query
            properties:
              query:
                type: string
                description: 搜索关键词
                example: "DeepSeek R1 模型特点"
              max_results:
                type: integer
                description: 返回结果数量限制
                default: 10
                example: 10
              freshness:
                type: string
                description: 结果新鲜度过滤选项
                enum: [noLimit, oneMonth,oneYear]
                default: noLimit
    responses:
      200:
        description: 搜索成功
        content:
          application/json:
            schema:
              type: object
              properties:
                query:
                  type: string
                  description: 原始查询词
                conversation_id:
                  type: string
                  description: 会话ID
                answer:
                  type: string
                  description: AI生成的总结回答
                follow_ups:
                  type: array
                  items:
                    type: string
                  description: 推荐追问
                webpages:
                  type: array
                  description: 网页搜索结果列表
                  items:
                    type: object
                    properties:
                      name:
                        type: string
                      url:
                        type: string
                      snippet:
                        type: string
                      display_url:
                        type: string
                      date_last_crawled:
                        type: string
                images:
                  type: array
                  description: 图片结果列表
                  items:
                    type: object
                    properties:
                      name:
                        type: string
                      content_url:
                        type: string
                      host_page_url:
                        type: string
                      thumbnail_url:
                        type: string
                      width:
                        type: integer
                      height:
                        type: integer
                modal_cards:
                  type: array
                  description: 模态卡片列表
                  items:
                    type: object
                    properties:
                      card_type:
                        type: string
                        description: 卡片类型
                      content:
                        type: object
                        description: 结构化数据内容
    """
    data = request.json or {}
    query = data.get("query")
    max_results = data.get("max_results", 10)
    freshness = data.get("freshness", "noLimit")

    if not query:
        raise BizError("Missing Query", code=http.HTTPStatus.BAD_REQUEST)

    result = search_client.comprehensive_search(
        api_key=g.api_key, query=query, freshness=freshness, max_results=max_results
    )
    return asdict(result)


@callback_bp.route("/bocha/web-only", methods=["POST"])
@require_bearer_auth
def bocha_web_search_only():
    """
    【工具】纯网页搜索
    ---
    tags:
      - Bocha Search
    summary: 仅执行网页搜索
    parameters:
      - name: Authorization
        in: header
        schema:
          type: string
        required: true
        description: Bearer <API_KEY>
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            required:
              - query
            properties:
              query:
                type: string
                description: 搜索关键词
                example: "Python Flask 教程"
              max_results:
                type: integer
                description: 返回结果数量限制
                default: 15
                example: 15
    responses:
      200:
        description: 搜索成功
        content:
          application/json:
            schema:
              type: object
              properties:
                query:
                  type: string
                  description: 原始查询词
                conversation_id:
                  type: string
                  description: 会话ID
                answer:
                  type: string
                  description: AI生成的总结回答
                follow_ups:
                  type: array
                  items:
                    type: string
                  description: 推荐追问
                webpages:
                  type: array
                  description: 网页搜索结果列表
                  items:
                    type: object
                    properties:
                      name:
                        type: string
                      url:
                        type: string
                      snippet:
                        type: string
                      display_url:
                        type: string
                      date_last_crawled:
                        type: string
                images:
                  type: array
                  description: 图片结果列表
                  items:
                    type: object
                    properties:
                      name:
                        type: string
                      content_url:
                        type: string
                      host_page_url:
                        type: string
                      thumbnail_url:
                        type: string
                      width:
                        type: integer
                      height:
                        type: integer
                modal_cards:
                  type: array
                  description: 模态卡片列表
                  items:
                    type: object
                    properties:
                      card_type:
                        type: string
                        description: 卡片类型
                      content:
                        type: object
                        description: 结构化数据内容
    """
    data = request.json or {}
    query = data.get("query")
    max_results = data.get("max_results", 15)

    if not query:
        raise BizError("Missing query", code=http.HTTPStatus.BAD_REQUEST)

    result = search_client.web_search_only(
        api_key=g.api_key, query=query, max_results=max_results
    )
    return asdict(result)


@callback_bp.route("/bocha/structured", methods=["POST"])
@require_bearer_auth
def bocha_search_structured():
    """
    【工具】结构化数据查询
    ---
    tags:
      - Bocha Search
    summary: 触发特定领域的结构化模态卡
    parameters:
      - name: Authorization
        in: header
        schema:
          type: string
        required: true
        description: Bearer <API_KEY>
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            required:
              - query
            properties:
              query:
                type: string
                description: 关键词，例如 "北京天气", "英伟达股价"
                example: "北京朝阳区天气"
    responses:
      200:
        description: 搜索成功
        content:
          application/json:
            schema:
              type: object
              properties:
                query:
                  type: string
                  description: 原始查询词
                conversation_id:
                  type: string
                  description: 会话ID
                answer:
                  type: string
                  description: AI生成的总结回答
                follow_ups:
                  type: array
                  items:
                    type: string
                  description: 推荐追问
                webpages:
                  type: array
                  description: 网页搜索结果列表
                  items:
                    type: object
                    properties:
                      name:
                        type: string
                      url:
                        type: string
                      snippet:
                        type: string
                      display_url:
                        type: string
                      date_last_crawled:
                        type: string
                images:
                  type: array
                  description: 图片结果列表
                  items:
                    type: object
                    properties:
                      name:
                        type: string
                      content_url:
                        type: string
                      host_page_url:
                        type: string
                      thumbnail_url:
                        type: string
                      width:
                        type: integer
                      height:
                        type: integer
                modal_cards:
                  type: array
                  description: 模态卡片列表
                  items:
                    type: object
                    properties:
                      card_type:
                        type: string
                        description: 卡片类型
                      content:
                        type: object
                        description: 结构化数据内容
    """
    data = request.json or {}
    query = data.get("query")

    if not query:
        raise BizError("缺少必填参数: query", code=http.HTTPStatus.BAD_REQUEST)

    result = search_client.search_for_structured_data(api_key=g.api_key, query=query)
    return asdict(result)


@callback_bp.route("/bocha/day", methods=["POST"])
@require_bearer_auth
def bocha_search_day():
    """
    【工具】搜索24小时内信息
    ---
    tags:
      - Bocha Search
    summary: 实时性搜索 (1天内)
    parameters:
      - name: Authorization
        in: header
        schema:
          type: string
        required: true
        description: Bearer <API_KEY>
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            required:
              - query
            properties:
              query:
                type: string
                description: 搜索关键词
                example: "今天的新闻热点"
    responses:
      200:
        description: 搜索成功
        content:
          application/json:
            schema:
              type: object
              properties:
                query:
                  type: string
                  description: 原始查询词
                conversation_id:
                  type: string
                  description: 会话ID
                answer:
                  type: string
                  description: AI生成的总结回答
                follow_ups:
                  type: array
                  items:
                    type: string
                  description: 推荐追问
                webpages:
                  type: array
                  description: 网页搜索结果列表
                  items:
                    type: object
                    properties:
                      name:
                        type: string
                      url:
                        type: string
                      snippet:
                        type: string
                      display_url:
                        type: string
                      date_last_crawled:
                        type: string
                images:
                  type: array
                  description: 图片结果列表
                  items:
                    type: object
                    properties:
                      name:
                        type: string
                      content_url:
                        type: string
                      host_page_url:
                        type: string
                      thumbnail_url:
                        type: string
                      width:
                        type: integer
                      height:
                        type: integer
                modal_cards:
                  type: array
                  description: 模态卡片列表
                  items:
                    type: object
                    properties:
                      card_type:
                        type: string
                        description: 卡片类型
                      content:
                        type: object
                        description: 结构化数据内容
    """
    data = request.json or {}
    query = data.get("query")

    if not query:
        raise BizError("Missing query", code=http.HTTPStatus.BAD_REQUEST)

    result = search_client.search_last_24_hours(api_key=g.api_key, query=query)
    return asdict(result)


@callback_bp.route("/bocha/week", methods=["POST"])
@require_bearer_auth
def bocha_search_last_week():
    """
    【工具】搜索本周信息
    ---
    tags:
      - Bocha Search
    summary: 近期搜索 (1周内)
    parameters:
      - name: Authorization
        in: header
        schema:
          type: string
        required: true
        description: Bearer <API_KEY>
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            required:
              - query
            properties:
              query:
                type: string
                description: 搜索关键词
                example: "本周科技圈大事"
    responses:
      200:
        description: 搜索成功
        content:
          application/json:
            schema:
              type: object
              properties:
                query:
                  type: string
                  description: 原始查询词
                conversation_id:
                  type: string
                  description: 会话ID
                answer:
                  type: string
                  description: AI生成的总结回答
                follow_ups:
                  type: array
                  items:
                    type: string
                  description: 推荐追问
                webpages:
                  type: array
                  description: 网页搜索结果列表
                  items:
                    type: object
                    properties:
                      name:
                        type: string
                      url:
                        type: string
                      snippet:
                        type: string
                      display_url:
                        type: string
                      date_last_crawled:
                        type: string
                images:
                  type: array
                  description: 图片结果列表
                  items:
                    type: object
                    properties:
                      name:
                        type: string
                      content_url:
                        type: string
                      host_page_url:
                        type: string
                      thumbnail_url:
                        type: string
                      width:
                        type: integer
                      height:
                        type: integer
                modal_cards:
                  type: array
                  description: 模态卡片列表
                  items:
                    type: object
                    properties:
                      card_type:
                        type: string
                        description: 卡片类型
                      content:
                        type: object
                        description: 结构化数据内容
    """
    data = request.json or {}
    query = data.get("query")

    if not query:
        raise BizError("Missing query", code=http.HTTPStatus.BAD_REQUEST)

    result = search_client.search_last_week(api_key=g.api_key, query=query)
    return asdict(result)
