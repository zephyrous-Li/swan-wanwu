import http

from flask import g, jsonify, request

from callback.services.ali_asr import AliASR
from callback.utils.decorators import require_bearer_auth
from callback.utils.url_util import process_audio_to_base64_with_mime
from utils.response import BizError

from . import callback_bp

asr = AliASR()


@callback_bp.route("/ali-asr/qwen3-asr-flash", methods=["POST"])
@require_bearer_auth
def ali_asr_recognize():
    """
    【工具】阿里云 ASR 语音识别
    ---
    tags:
      - Ali ASR
    summary: 语音识别
    description: 使用阿里云通义千问进行语音识别，模型固定为 qwen3-asr-flash。
    parameters:
      - name: Authorization
        in: header
        description: "Bearer Token (API Key)"
        required: true
        schema:
          type: string
          default: "Bearer "
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            required:
              - audio
            properties:
              audio:
                type: string
                description: 音频文件URL或base64
                example: "https://dashscope.oss-cn-beijing.aliyuncs.com/audios/welcome.mp3"
    responses:
      200:
        description: 识别成功
      400:
        description: 参数错误
      401:
        description: API Key 无效或缺失
      500:
        description: 服务端错误
    """
    data = request.get_json() or {}
    audio = data.get("audio")

    if not audio:
        raise BizError("Missing audio", code=http.HTTPStatus.BAD_REQUEST)

    audio_processed = process_audio_to_base64_with_mime(audio)

    result = asr.recognize(
        audio=audio_processed, api_key=g.api_key, model="qwen3-asr-flash"
    )

    return jsonify(result)
