// 环境配置管理
// 动态适配开发/生产环境的区块链和代币配置

// 环境类型定义
export type Environment = 'development' | 'production';

// 链配置接口
export interface ChainConfig {
  id: number;
  name: string;
  isTestnet: boolean;
  defaultTokens: {
    from: string;
    to: string;
  };
}

// 支持的链配置 - 动态从环境变量读取代币对
export const CHAIN_CONFIGS: Record<number, ChainConfig> = {
  // 以太坊主网
  1: {
    id: 1,
    name: 'Ethereum',
    isTestnet: false,
    defaultTokens: {
      from: import.meta.env.VITE_MAINNET_DEFAULT_FROM_TOKEN || 'ETH',
      to: import.meta.env.VITE_MAINNET_DEFAULT_TO_TOKEN || 'USDC'
    }
  },
  // Sepolia测试网
  11155111: {
    id: 11155111,
    name: 'Sepolia',
    isTestnet: true,
    defaultTokens: {
      from: import.meta.env.VITE_TESTNET_DEFAULT_FROM_TOKEN || 'SepoliaETH',
      to: import.meta.env.VITE_TESTNET_DEFAULT_TO_TOKEN || 'USDC'
    }
  },
  // Polygon主网
  137: {
    id: 137,
    name: 'Polygon',
    isTestnet: false,
    defaultTokens: {
      from: import.meta.env.VITE_MAINNET_DEFAULT_FROM_TOKEN || 'MATIC',
      to: import.meta.env.VITE_MAINNET_DEFAULT_TO_TOKEN || 'USDC'
    }
  },
  // Mumbai测试网
  80001: {
    id: 80001,
    name: 'Mumbai',
    isTestnet: true,
    defaultTokens: {
      from: import.meta.env.VITE_TESTNET_DEFAULT_FROM_TOKEN || 'MATIC',
      to: import.meta.env.VITE_TESTNET_DEFAULT_TO_TOKEN || 'USDC'
    }
  }
};

// 环境配置类
export class EnvironmentConfig {
  private static instance: EnvironmentConfig;
  private environment: Environment;
  private defaultChainId: number;

  private constructor() {
    // 从环境变量读取配置
    this.environment = (import.meta.env.VITE_APP_ENV || 'development') as Environment;
    this.defaultChainId = parseInt(import.meta.env.VITE_DEFAULT_CHAIN_ID || '1');
  }

  public static getInstance(): EnvironmentConfig {
    if (!EnvironmentConfig.instance) {
      EnvironmentConfig.instance = new EnvironmentConfig();
    }
    return EnvironmentConfig.instance;
  }

  // 获取当前环境
  public getEnvironment(): Environment {
    return this.environment;
  }

  // 获取默认链ID
  public getDefaultChainId(): number {
    return this.defaultChainId;
  }

  // 获取当前链配置
  public getCurrentChainConfig(): ChainConfig {
    return CHAIN_CONFIGS[this.defaultChainId] || CHAIN_CONFIGS[11155111];
  }

  // 是否为测试环境
  public isTestnet(): boolean {
    return this.getCurrentChainConfig().isTestnet;
  }

  // 获取默认代币对
  public getDefaultTokens(): { from: string; to: string } {
    return this.getCurrentChainConfig().defaultTokens;
  }

  // 获取默认滑点
  public getDefaultSlippage(): string {
    return import.meta.env.VITE_DEFAULT_SLIPPAGE || '0.5';
  }

  // 获取API基础URL
  public getAPIBaseURL(): string {
    return import.meta.env.VITE_API_URL || 'http://localhost:5176';
  }

  // 获取支持的链列表
  public getSupportedChainIds(): number[] {
    const chainIds = import.meta.env.VITE_SUPPORTED_CHAIN_IDS || '1,11155111,137,80001';
    return chainIds.split(',').map((id: string) => parseInt(id.trim()));
  }

  // 打印当前配置（调试用）
  public logCurrentConfig(): void {
    const config = this.getCurrentChainConfig();
    console.log('🔧 当前环境配置:', {
      environment: this.environment,
      chainId: this.defaultChainId,
      chainName: config.name,
      isTestnet: config.isTestnet,
      defaultTokens: config.defaultTokens,
      slippage: this.getDefaultSlippage(),
      apiBaseURL: this.getAPIBaseURL()
    });
  }
}

// 导出单例实例
export const envConfig = EnvironmentConfig.getInstance();

// 便捷函数
export const getCurrentChainId = () => envConfig.getDefaultChainId();
export const isTestnetEnvironment = () => envConfig.isTestnet();
export const getDefaultTokenPair = () => envConfig.getDefaultTokens();
export const getDefaultSlippage = () => envConfig.getDefaultSlippage();
export const getAPIBaseURL = () => envConfig.getAPIBaseURL();
