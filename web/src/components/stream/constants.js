export const AGENT_MESSAGE_CONFIG = {
  // 主智能体
  MAIN_AGENT: {
    EVENT_TYPE: 0,
    CONVERSATION_TYPE: '',
  },
  // 子智能体
  SUB_AGENT: {
    EVENT_TYPE: 1,
    CONVERSATION_TYPE: 'subAgent',
  },
  // 主智能体-知识库
  MAIN_KNOWLEDGE: {
    EVENT_TYPE: 2,
    CONVERSATION_TYPE: 'agentKnowledge',
  },
  // 主智能体-工具
  MAIN_TOOL: {
    EVENT_TYPE: 3,
    CONVERSATION_TYPE: 'agentTool',
  },
  // 主智能体-思考
  MAIN_THINK: {
    EVENT_TYPE: 6,
    CONVERSATION_TYPE: 'agentThink',
  },
};

export const AGENT_SSE_EVENT_TYPES = Object.fromEntries(
  Object.entries(AGENT_MESSAGE_CONFIG).map(([key, val]) => [
    key,
    val.EVENT_TYPE,
  ]),
);
