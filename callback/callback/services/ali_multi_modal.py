# -*- coding: utf-8 -*-
from typing import List, Optional

import dashscope
from dashscope import ImageSynthesis, MultiModalConversation, VideoSynthesis
from dashscope.aigc.image_generation import ImageGeneration
from dashscope.api_entities.dashscope_response import Message

from utils.log import logger


class AliGenAI:
    def __init__(self, region: str = "cn"):
        """
        初始化阿里云生图客户端
        :param region: 地域 ('cn' 为北京, 'intl' 为新加坡)
        """
        self._region = region
        self._setup_region(region)

    def _setup_region(self, region: str):
        if region == "intl":
            dashscope.base_http_api_url = "https://dashscope-intl.aliyuncs.com/api/v1"
        else:
            dashscope.base_http_api_url = "https://dashscope.aliyuncs.com/api/v1"

    def image_generate_legacy(
        self,
        api_key: str,
        prompt: str,
        images: Optional[List[str]] = None,
        model: str = "wan2.5-t2i-preview",
        negative_prompt: str = "",
        size: str = "1280*1280",
        n: int = 1,
    ):
        """
        调用 ImageSynthesis（旧版接口）
        适用模型：wan2.5及以下版本、qwen-image-plus、qwen-image
        """
        try:
            rsp = ImageSynthesis.call(
                api_key=api_key,
                model=model,
                prompt=prompt,
                images=images,
                negative_prompt=negative_prompt,
                n=n,
                size=size,
                prompt_extend=True,
                watermark=True,
            )
            return rsp
        except Exception as e:
            logger.exception(f"ImageSynthesis 调用失败: {e}")
            return None

    def image_to_image_generate(
        self,
        api_key: str,
        prompt: str,
        images: Optional[List[str]] = None,
        model: str = "qwen-image-plus",
        negative_prompt: str = "",
        size: str = "1280*1280",
        n: int = 1,
        prompt_extend: bool = True,
        watermark: bool = True,
    ):
        """
        调用 ImageGeneration（新版接口）
        支持文生图和图生图/编辑
        """
        content_list = []

        if images:
            for img in images[:3]:
                content_list.append({"image": img})

        content_list.append({"text": prompt})
        logger.info(f"image_generate content_list: {content_list}")

        message = Message(role="user", content=content_list)

        try:
            rsp = ImageGeneration.call(
                model=model,
                api_key=api_key,
                messages=[message],
                negative_prompt=negative_prompt,
                prompt_extend=prompt_extend,
                watermark=watermark,
                images=images,
                n=n,
                size=size,
            )
            return rsp
        except Exception as e:
            logger.exception(f"ImageGeneration 调用失败: {e}")
            return None
    
    def text_to_image_generate(
        self,
        api_key: str,
        prompt: str,
        model: str = "wan2.6-t2i",
        negative_prompt: str = "",
        size: str = "1280*1280",
        n: int = 1,
        prompt_extend: bool = True,
        watermark: bool = True,
    ):
        """
        调用 ImageGeneration（新版接口）
        支持文生图和图生图/编辑
        """
        content_list = []

        content_list.append({"text": prompt})
        logger.info(f"image_generate content_list: {content_list}")

        message = Message(role="user", content=content_list)

        try:
            rsp = ImageGeneration.call(
                model=model,
                api_key=api_key,
                messages=[message],
                negative_prompt=negative_prompt,
                prompt_extend=prompt_extend,
                watermark=watermark,
                n=n,
                size=size,
            )
            return rsp
        except Exception as e:
            logger.exception(f"ImageGeneration 调用失败: {e}")
            return None

    def qwen_text_to_image(
        self,
        api_key: str,
        prompt: str,
        model: str = "qwen-image-max",
        negative_prompt: str = "",
        size: str = "1280*1280",
        n: int = 1,
    ):
        """
        调用多模态对话接口
        """
        content_list = []

        content_list.append({"text": prompt})
        messages = [{"role": "user", "content": content_list}]

        try:
            rsp = MultiModalConversation.call(
                api_key=api_key,
                model=model,
                messages=messages,
                negative_prompt=negative_prompt,
                n=n,
                size=size,
                prompt_extend=True,
                watermark=True,
            )
            return rsp
        except Exception as e:
            logger.exception(f"MultiModalConversation 调用失败: {e}")
            return None

    def image_to_video_generate(
        self,
        api_key: str,
        img_url: str,
        prompt: Optional[str] = None,
        model: str = "wan2.6-i2v-flash",
        audio_url: Optional[str] = None,
        resolution: str = "720P",
        duration: int = 5,
        negative_prompt: str = "",
        shot_type: Optional[str] = None,
        template: Optional[str] = None,
    ):
        """
        图片生成视频
        """
        try:
            resp = VideoSynthesis.call(
                api_key=api_key,
                model=model,
                prompt=prompt,
                img_url=img_url,
                audio_url=audio_url,
                resolution=resolution,
                duration=duration,
                prompt_extend=True,
                watermark=True,
                negative_prompt=negative_prompt,
                shot_type=shot_type,
                template=template,
            )
            return resp
        except Exception as e:
            logger.exception(f"ImageToVideo 调用失败: {e}")
            return None

    def first_and_last_image_to_video(
        self,
        api_key: str,
        first_frame_url: str,
        prompt: Optional[str] = None,
        model: str = "wan2.2-kf2v-flash",
        last_frame_url: Optional[str] = None,
        resolution: str = "720P",
        duration: int = 5,
        negative_prompt: str = "",
        template: Optional[str] = None,
    ):
        """
        首尾帧生成视频
        """
        try:
            resp = VideoSynthesis.call(
                api_key=api_key,
                model=model,
                prompt=prompt,
                first_frame_url=first_frame_url,
                last_frame_url=last_frame_url,
                resolution=resolution,
                duration=duration,
                prompt_extend=True,
                watermark=True,
                negative_prompt=negative_prompt,
                template=template,
            )
            return resp
        except Exception as e:
            logger.exception(f"FirstLastFrameVideo 调用失败: {e}")
            return None

    def text_to_video_generate(
        self,
        api_key: str,
        prompt: str,
        model: str = "wan2.6-t2v",
        audio_url: Optional[str] = None,
        size: str = "1280*720",
        duration: int = 5,
        negative_prompt: str = "",
        shot_type: Optional[str] = None,
    ):
        """
        文本生成视频
        """
        try:
            rsp = VideoSynthesis.call(
                api_key=api_key,
                model=model,
                prompt=prompt,
                audio_url=audio_url,
                size=size,
                duration=duration,
                negative_prompt=negative_prompt,
                prompt_extend=True,
                watermark=True,
                shot_type=shot_type,
            )
            logger.info(f"text_to_video_generate response: {rsp}")
            return rsp
        except Exception as e:
            logger.exception(f"TextToVideo 调用失败: {e}")
            return None
