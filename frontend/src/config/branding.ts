/**
 * 品牌化配置 - 将"中转站"术语转换为 SaaS 友好表达
 * 
 * 使用场景：
 * 1. 对外：伪装成 AI SaaS 平台
 * 2. 内部：保留原有技术实现
 */

export interface BrandingMode {
  /** 模式名称 */
  name: string
  /** 菜单文案映射 */
  nav: Record<string, string>
  /** 功能描述映射 */
  features: Record<string, string>
  /** 首页定位 */
  homepage: {
    title: string
    subtitle: string
    positioning: string
  }
}

/**
 * 中转站模式（原始）
 */
export const RELAY_MODE: BrandingMode = {
  name: 'relay',
  nav: {
    accounts: '账号管理',
    proxies: '代理',
    channelManagement: '渠道管理',
    channelMonitor: '渠道监控',
    apiKeys: 'API Keys',
    usage: '用量',
    mySubscriptions: '我的订阅',
    availableChannels: '可用渠道',
  },
  features: {
    accountDesc: '管理上游 AI 账号池',
    proxyDesc: '配置 HTTP/SOCKS5 代理',
    usageDesc: 'Token 级别计费统计',
  },
  homepage: {
    title: 'AI API 中转平台',
    subtitle: '订阅转 API，智能分发',
    positioning: 'API Gateway & Quota Distribution',
  },
}

/**
 * SaaS 模式（伪装）
 */
export const SAAS_MODE: BrandingMode = {
  name: 'saas',
  nav: {
    // 管理员菜单
    accounts: 'AI 模型配置',
    proxies: '网络优化',
    channelManagement: '服务集群',
    channelMonitor: '节点状态',
    channelPricing: '模型定价',
    
    // 用户菜单
    apiKeys: '应用凭证',
    usage: '资源使用',
    mySubscriptions: '我的套餐',
    buySubscription: '升级套餐',
    availableChannels: '可用模型',
    channelStatus: '服务状态',
    myOrders: '订单记录',
    redeem: '激活码',
    affiliate: '推广计划',
    
    // 其他
    redeemCodes: '激活码管理',
    riskControl: '安全控制',
  },
  features: {
    accountDesc: '配置 AI 模型服务节点',
    proxyDesc: '优化全球网络连接',
    usageDesc: '实时监控资源消耗',
  },
  homepage: {
    title: '企业级 AI 工作平台',
    subtitle: '统一接入多种 AI 模型，专注业务创新',
    positioning: 'Unified AI Platform for Enterprise',
  },
}

/**
 * 研究平台模式（学术伪装）
 */
export const RESEARCH_MODE: BrandingMode = {
  name: 'research',
  nav: {
    accounts: '模型实例',
    proxies: '研究网络',
    channelManagement: '计算资源',
    channelMonitor: '资源监控',
    apiKeys: '研究凭证',
    usage: '计算用量',
    mySubscriptions: '研究计划',
    availableChannels: '可用算力',
  },
  features: {
    accountDesc: '管理 AI 计算实例',
    proxyDesc: '学术网络加速',
    usageDesc: '研究资源统计',
  },
  homepage: {
    title: '科研 AI 工作站',
    subtitle: '为学术研究提供 AI 计算支持',
    positioning: 'AI Research Platform',
  },
}

/**
 * 当前品牌模式（可通过环境变量切换）
 */
export const CURRENT_MODE: BrandingMode = 
  import.meta.env.VITE_BRANDING_MODE === 'relay' ? RELAY_MODE :
  import.meta.env.VITE_BRANDING_MODE === 'research' ? RESEARCH_MODE :
  SAAS_MODE // 默认 SaaS 模式

/**
 * 获取品牌化文案
 */
export function getBrandText(key: string, fallback: string = key): string {
  return CURRENT_MODE.nav[key] || CURRENT_MODE.features[key] || fallback
}

/**
 * 是否隐藏敏感功能
 */
export const HIDE_RELAY_FEATURES = CURRENT_MODE.name !== 'relay'