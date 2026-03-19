export const generateChatConfig = params => {
  const config = params
    ? params.modelDetail
      ? params.modelDetail.config || {}
      : {}
    : {};
  const thinkingEnableObj =
    config.thinkingSupport === 'support' ? { thinkingEnable: true } : {};
  return {
    pending: true, // 模型信息初始化状态
    model: '', // 模型名称
    modelId: '', // 模型id
    title: '', // 对话title
    modelType: '', // 模型类型
    sessionStatus: -1, // 非会话状态
    sessionId: '', // 会话id
    modelExperienceId: 0, // 对话历史列表的id
    modelDetail: {}, // 模型详情
    modelSetting: {
      // 模型配置
      temperature: 0.7,
      topP: 1,
      frequencyPenalty: 0,
      presencePenalty: 0,
      maxTokens: 512,
      temperatureEnable: false,
      topPEnable: false,
      presencePenaltyEnable: false,
      maxTokensEnable: false,
      frequencyPenaltyEnable: false,
      thinkingSupport: config.thinkingSupport,
      ...thinkingEnableObj,
    },
    ...params,
  };
};
