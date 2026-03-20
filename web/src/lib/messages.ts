export type Locale = 'en' | 'zh-CN'

export const localeStorageKey = 'golem-web-ui-locale'

export const messages = {
  en: {
    localeLabel: 'Language',
    localeEnglish: 'EN',
    localeChinese: '中文',
    landing: {
      eyebrow: 'Terminal-native. Geo-capable. Auto-evolving.',
      title: 'Golem turns your Gateway into a vivid, workspace-aware control surface.',
      copy:
        'Operate chat, tools, Geo workflows, and learned execution paths from a product-grade web interface built for people who want both signal and atmosphere.',
      enterConsole: 'Enter Console',
      seeGeo: 'See Geo + Auto-Evolution',
      heroCards: [
        {
          label: 'Gateway',
          title: 'health · version · chat',
          body: 'One front door into the existing Go backend.',
        },
        {
          label: 'Geo',
          title: 'GDAL · PostGIS · pipelines',
          body: 'Workspace-native spatial execution instead of disconnected tools.',
        },
        {
          label: 'Evolution',
          title: 'reuse · scaffold · telemetry',
          body: 'Successful workflows leave artifacts the next run can actually use.',
        },
      ],
      capability: {
        eyebrow: 'Core Surface',
        title: 'One interface for the parts that actually matter.',
        items: [
          {
            title: 'Operational Chat',
            body: 'Talk to the agent through the same Gateway your automations and integrations already use.',
          },
          {
            title: 'Workspace Context',
            body: 'Surface the system as an instrument panel instead of burying it behind raw terminal commands.',
          },
          {
            title: 'Geo Vertical',
            body: 'Promote spatial data discovery, codebooks, fabricated tools, and learned pipelines to first-class UI concepts.',
          },
        ],
      },
      geo: {
        eyebrow: 'Geo Verticalization and Auto-Evolution',
        title: 'Not a GIS button. A reusable execution loop.',
        copy:
          "Golem's Geo stack already knows how to inspect, transform, discover, and query spatial data. The WebUI turns that into a visible product surface and makes the evolution loops legible.",
        panels: [
          {
            title: 'Replay-ready reuse',
            body: 'Learned Geo pipelines can come back as prompt-time reuse hints with example args and parameter update markers.',
          },
          {
            title: 'Dry-run fabrication',
            body: 'Missing Geo capabilities can be scaffolded into manifest and script targets before implementation starts.',
          },
          {
            title: 'Skill telemetry',
            body: 'Shown, selected, success, and failure signals produce local reports that expose underperforming Geo skills first.',
          },
        ],
      },
      cta: {
        eyebrow: 'Ready',
        title: 'Go from narrative to operator mode in one click.',
        copy: 'Use the console to point at your Gateway, validate the connection, and start sending prompts immediately.',
        action: 'Open Control Console',
      },
    },
    console: {
      eyebrow: 'Console',
      title: 'Gateway Control Surface',
      subtitle: 'Connect to health, version, and chat from one operator-grade workspace.',
      refresh: 'Refresh Gateway',
      refreshing: 'Checking...',
      backToLanding: 'Back to Landing',
      connection: {
        eyebrow: 'Connection',
        title: 'Gateway Settings',
        baseUrl: 'Base URL',
        bearerToken: 'Bearer Token',
        bearerPlaceholder: 'optional',
        sessionId: 'Session ID',
        senderId: 'Sender ID',
        note:
          'These values stay local in the browser. The first release supports bearer-token based Gateway access without adding a separate auth system.',
      },
      composer: {
        placeholder: 'Send a prompt to the Gateway chat endpoint.',
        send: 'Send Prompt',
        sending: 'Sending...',
        check: 'Check Gateway',
        checking: 'Checking...',
      },
      timeline: {
        introTitle: 'Console Ready',
        introBody: 'Check Gateway connectivity, inspect version, then send a prompt through the existing /chat API.',
        introMeta: 'idle',
        gatewayCheckPassedTitle: 'Gateway Check Passed',
        gatewayCheckFailedTitle: 'Gateway Check Failed',
        requestFailedTitle: 'Request Failed',
        promptTitle: 'Prompt',
        responseTitle: 'Response',
        sendingTitle: 'Sending',
        sendingBody: 'The Gateway request is in flight.',
        sendingMeta: 'pending',
        checkMeta: 'check failed',
        chatMeta: 'chat error',
      },
    },
  },
  'zh-CN': {
    localeLabel: '语言',
    localeEnglish: 'EN',
    localeChinese: '中文',
    landing: {
      eyebrow: '终端原生。Geo 加持。持续进化。',
      title: 'Golem 把你的 Gateway 变成一个有氛围、感知工作区的控制界面。',
      copy:
        '在一个成品级 WebUI 里操作对话、工具、Geo 工作流和学习到的执行路径，兼顾密度、信号感与可用性。',
      enterConsole: '进入控制台',
      seeGeo: '查看 Geo + 自动进化',
      heroCards: [
        {
          label: 'Gateway',
          title: 'health · version · chat',
          body: '用一个前门接入现有 Go 后端。',
        },
        {
          label: 'Geo',
          title: 'GDAL · PostGIS · pipelines',
          body: '把空间数据执行面放进工作区，而不是零散工具集合。',
        },
        {
          label: 'Evolution',
          title: 'reuse · scaffold · telemetry',
          body: '成功的工作流会留下下一次真能复用的痕迹。',
        },
      ],
      capability: {
        eyebrow: '核心界面',
        title: '把真正重要的部分放进同一个操作界面。',
        items: [
          {
            title: '操作型对话',
            body: '通过和自动化、集成共用的 Gateway 与 Agent 交互。',
          },
          {
            title: '工作区上下文',
            body: '把系统从裸终端命令堆，提升成可读的仪表面板。',
          },
          {
            title: 'Geo 垂直能力',
            body: '把空间数据发现、codebook、fabricated tools 和 learned pipelines 提升成一等 UI 概念。',
          },
        ],
      },
      geo: {
        eyebrow: 'Geo 垂直化与自动进化',
        title: '不是一个 GIS 按钮，而是一条可复用的执行闭环。',
        copy:
          'Golem 的 Geo 栈已经能检查、转换、发现和查询空间数据。WebUI 把这些能力变成可见的产品表面，也让进化闭环更容易被看见和复用。',
        panels: [
          {
            title: '可回放复用',
            body: 'learned Geo pipeline 会在后续请求里回到 prompt 中，带着示例参数和参数待更新标记。',
          },
          {
            title: 'dry-run 脚手架',
            body: '缺失的 Geo 能力可以先生成 manifest 和 script 目标，再进入具体实现。',
          },
          {
            title: '技能遥测',
            body: 'shown、selected、success、failure 信号会沉淀成本地报告，把表现偏弱的 Geo skills 先暴露出来。',
          },
        ],
      },
      cta: {
        eyebrow: '准备好了',
        title: '一键从叙事页切到操作台。',
        copy: '在控制台里指向你的 Gateway，验证连接，然后立刻开始发送提示词。',
        action: '打开控制台',
      },
    },
    console: {
      eyebrow: '控制台',
      title: 'Gateway 控制台',
      subtitle: '在同一个操作台里接入 health、version 和 chat。',
      refresh: '刷新 Gateway',
      refreshing: '检查中...',
      backToLanding: '返回首页',
      connection: {
        eyebrow: '连接',
        title: 'Gateway 设置',
        baseUrl: 'Base URL',
        bearerToken: 'Bearer Token',
        bearerPlaceholder: '可选',
        sessionId: 'Session ID',
        senderId: 'Sender ID',
        note: '这些值只保存在浏览器本地。当前版本先支持基于 Bearer Token 的 Gateway 访问，不额外引入独立认证系统。',
      },
      composer: {
        placeholder: '向 Gateway 的 chat 端点发送一个提示词。',
        send: '发送提示词',
        sending: '发送中...',
        check: '检查 Gateway',
        checking: '检查中...',
      },
      timeline: {
        introTitle: '控制台已就绪',
        introBody: '先检查 Gateway 连通性和版本，再通过现有 /chat API 发送请求。',
        introMeta: '空闲',
        gatewayCheckPassedTitle: 'Gateway 检查通过',
        gatewayCheckFailedTitle: 'Gateway 检查失败',
        requestFailedTitle: '请求失败',
        promptTitle: '提示词',
        responseTitle: '响应',
        sendingTitle: '发送中',
        sendingBody: 'Gateway 请求正在处理中。',
        sendingMeta: '处理中',
        checkMeta: '检查失败',
        chatMeta: '对话错误',
      },
    },
  },
} as const

export type MessageSchema = (typeof messages)['en']
