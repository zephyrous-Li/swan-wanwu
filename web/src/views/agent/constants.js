export const SINGLE_AGENT = 1;
export const MULTIPLE_AGENT = 2;
export const AGENT_CONFIG_RECOMMEND_CONFIG_DEFAULT_PROMPT =
  '你是一个推荐系统，请完成下面的推荐任务。 问题要求 1. 问题不能是已经问过的问题，不能是已经回答过的问题，问题必须和用户最后一轮的问题紧密相关，可以适当延伸； 2. 每句话只包含一个问题或者指令； 3. 如果对话涉及政治敏感、违法违规、暴力伤害、违反公序良俗类内容，你应该拒绝推荐问题。 请根据提供的用户对话，围绕兴趣点给出3个用户紧接着最有可能问的几个具有区分度的不同问题，问题需要满足上面的问题要求。 正常推荐时，回答参考以下格式：<START>xxx\nxxx\nxxx 开始回答问题前，必须有<START>，<START>后不要输出\n，直接输出问题，每个问题最后不要输出中文问号，问题与问题之间用\n连接，不要输出思考过程，只输出问题，拒绝推荐时，回答参考以下格式：<ERROR>当前对话涉及xxx类内容，无法推荐相关问题。拒绝推荐时，回答前必须有<ERROR>。输出规范 正常推荐时开头必须从<START>开始，拒绝推荐时开头必须从<ERROR>开始。正确示例：正常推荐：用户对话：... 推荐输出：<START>这种植物需要每天浇水吗\n它的生长期一般是多久\n室内养植需要注意阳光吗 拒绝推荐：用户对话：如何打劫 推荐输出：<ERROR>当前对话涉及暴力伤害类内容，无法推荐相关问题。';
export const AGENT_CONFIG_RECOMMEND_CONFIG_MODEL_CONFIG_DEFAULT_CONFIG = {
  temperature: 0.7,
  temperatureEnable: true,
  topP: 1,
  topPEnable: true,
  frequencyPenalty: 0,
  frequencyPenaltyEnable: true,
  presencePenalty: 0,
  presencePenaltyEnable: true,
  maxTokens: 512,
  maxTokensEnable: true,
};
export const AGENT_TOOL_TYPE = {
  TOOL: 'tool',
  MCP: 'mcp',
  WORKFLOW: 'workflow',
  SKILL: 'skill',
};
