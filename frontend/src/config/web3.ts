// Web3配置 - 简化版本
// 基础的wagmi配置，支持主要的区块链网络

import { createConfig, http } from 'wagmi';
import { mainnet, polygon, sepolia, polygonMumbai } from 'wagmi/chains';
import { injected, metaMask, walletConnect } from 'wagmi/connectors';

// 根据环境变量决定支持的网络
function getSupportedChains() {
  const isDevelopment = import.meta.env.DEV;
  
  if (isDevelopment) {
    // 开发环境：支持测试网
    return [sepolia, polygonMumbai];
  } else {
    // 生产环境：支持主网
    return [mainnet, polygon];
  }
}

export const supportedChains = getSupportedChains();

// 钱包连接器配置
// 简化配置，优先支持MetaMask
function createConnectors() {
  return [
    injected(), // 通用注入式钱包（包括MetaMask）
    metaMask(), // 专门的MetaMask连接器
  ];
}

export const connectors = createConnectors();

// RPC传输配置
// 优先使用直接配置的RPC URL，更灵活
function createRpcUrl(chainId: number): string {
  // 方案1: 直接使用配置的RPC URL (推荐)
  const mainnetRpcUrl = import.meta.env.VITE_RPC_URL_MAINNET;
  const polygonRpcUrl = import.meta.env.VITE_RPC_URL_POLYGON;
  
  if (mainnetRpcUrl || polygonRpcUrl) {
    switch (chainId) {
      case mainnet.id:
        return mainnetRpcUrl || 'https://eth.llamarpc.com';
      case sepolia.id:
        return mainnetRpcUrl || 'https://sepolia.llamarpc.com';
      case polygon.id:
        return polygonRpcUrl || 'https://polygon.llamarpc.com';
      case polygonMumbai.id:
        return polygonRpcUrl || 'https://polygon-mumbai.bor.validatrium.club';
      default:
        return mainnetRpcUrl || 'https://eth.llamarpc.com';
    }
  }
  
  // 方案2: 使用项目ID/API Key (兜底方案)
  const infuraProjectId = import.meta.env.VITE_INFURA_PROJECT_ID;
  const alchemyApiKey = import.meta.env.VITE_ALCHEMY_API_KEY;
  const networkEnv = import.meta.env.VITE_NETWORK_ENV || 'mainnet'; // mainnet 或 testnet
  
  if (infuraProjectId) {
    const isTestnet = networkEnv === 'testnet';
    switch (chainId) {
      case mainnet.id:
        return `https://mainnet.infura.io/v3/${infuraProjectId}`;
      case sepolia.id:
        return `https://sepolia.infura.io/v3/${infuraProjectId}`;
      case polygon.id:
        return `https://polygon-mainnet.infura.io/v3/${infuraProjectId}`;
      case polygonMumbai.id:
        return `https://polygon-mumbai.infura.io/v3/${infuraProjectId}`;
      default:
        return isTestnet 
          ? `https://sepolia.infura.io/v3/${infuraProjectId}`
          : `https://mainnet.infura.io/v3/${infuraProjectId}`;
    }
  } else if (alchemyApiKey) {
    switch (chainId) {
      case mainnet.id:
        return `https://eth-mainnet.g.alchemy.com/v2/${alchemyApiKey}`;
      case sepolia.id:
        return `https://eth-sepolia.g.alchemy.com/v2/${alchemyApiKey}`;
      case polygon.id:
        return `https://polygon-mainnet.g.alchemy.com/v2/${alchemyApiKey}`;
      case polygonMumbai.id:
        return `https://polygon-mumbai.g.alchemy.com/v2/${alchemyApiKey}`;
      default:
        return `https://eth-mainnet.g.alchemy.com/v2/${alchemyApiKey}`;
    }
  }
  
  // 如果都没有配置，使用公共RPC (不推荐生产环境)
  console.warn('未配置RPC服务提供商，使用公共RPC节点，可能不稳定');
  switch (chainId) {
    case mainnet.id:
      return 'https://eth.llamarpc.com';
    case sepolia.id:
      return 'https://sepolia.llamarpc.com';
    case polygon.id:
      return 'https://polygon.llamarpc.com';
    case polygonMumbai.id:
      return 'https://polygon-mumbai.bor.validatrium.club';
    default:
      return 'https://eth.llamarpc.com';
  }
}

// 动态创建传输配置
function createTransports() {
  const transports: Record<number, ReturnType<typeof http>> = {};
  
  supportedChains.forEach(chain => {
    transports[chain.id] = http(createRpcUrl(chain.id));
  });
  
  return transports;
}

export const transports = createTransports();

// 创建wagmi配置
export const wagmiConfig = createConfig({
  chains: supportedChains as any,
  connectors,
  transports,
});

// 格式化地址显示
export function formatAddress(address: string, length: number = 4): string {
  if (!address) return '';
  if (address.length <= length * 2 + 2) return address;
  
  return `${address.slice(0, length + 2)}...${address.slice(-length)}`;
}

// 格式化美元金额
export function formatUSD(amount: string | number): string {
  const num = typeof amount === 'string' ? parseFloat(amount) : amount;
  
  if (num < 0.01) {
    return '< $0.01';
  }
  
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(num);
}

// 价格冲击等级
export function getPriceImpactLevel(impact: string | number) {
  const impactNum = typeof impact === 'string' ? parseFloat(impact) : impact;
  const impactPercent = impactNum * 100;

  if (impactPercent < 1) {
    return {
      level: 'low' as const,
      color: 'text-green-600',
      description: '价格冲击较小'
    };
  } else if (impactPercent < 3) {
    return {
      level: 'medium' as const, 
      color: 'text-yellow-600',
      description: '价格冲击中等'
    };
  } else {
    return {
      level: 'high' as const,
      color: 'text-red-600',
      description: '价格冲击较大'
    };
  }
}