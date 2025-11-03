// 代币数据获取Hook
// 动态获取当前环境支持的代币列表和默认代币ID

import { useState, useEffect } from 'react';
import { envConfig, getAPIBaseURL, getCurrentChainId } from '../config/environment';

// 代币接口
export interface Token {
  id: number;
  symbol: string;
  name: string;
  contract_address: string;
  chain_id: number;
  decimals: number;
  is_native: boolean;
  is_stable: boolean;
  logo_url?: string;
}

// Hook返回类型
export interface UseTokensReturn {
  tokens: Token[];
  loading: boolean;
  error: string | null;
  getTokenBySymbol: (symbol: string) => Token | undefined;
  getDefaultFromToken: () => Token | undefined;
  getDefaultToToken: () => Token | undefined;
  refresh: () => void;
}

export const useTokens = (): UseTokensReturn => {
  const [tokens, setTokens] = useState<Token[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchTokens = async () => {
    try {
      setLoading(true);
      setError(null);

      const chainId = getCurrentChainId();
      const apiBaseURL = getAPIBaseURL();
      
      console.log(`🔄 获取代币列表: chainId=${chainId}, apiURL=${apiBaseURL}`);

      // 调用代币列表API
      const response = await fetch(`${apiBaseURL}/api/v1/tokens?chain_id=${chainId}`);
      
      if (!response.ok) {
        throw new Error(`API请求失败: ${response.status} ${response.statusText}`);
      }

      const data = await response.json();
      
      if (data.success && Array.isArray(data.data)) {
        setTokens(data.data);
        console.log(`✅ 代币列表获取成功: ${data.data.length}个代币`);
      } else {
        throw new Error(data.error?.message || '代币列表响应格式错误');
      }

    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : '获取代币列表失败';
      setError(errorMessage);
      console.error('❌ 代币列表获取失败:', err);
      
      // 设置默认代币作为备用
      setTokens([]);
    } finally {
      setLoading(false);
    }
  };

  // 初始化获取代币列表
  useEffect(() => {
    fetchTokens();
  }, []);

  // 根据符号查找代币
  const getTokenBySymbol = (symbol: string): Token | undefined => {
    return tokens.find(token => 
      token.symbol.toLowerCase() === symbol.toLowerCase()
    );
  };

  // 获取默认源代币
  const getDefaultFromToken = (): Token | undefined => {
    const defaultTokens = envConfig.getDefaultTokens();
    return getTokenBySymbol(defaultTokens.from);
  };

  // 获取默认目标代币
  const getDefaultToToken = (): Token | undefined => {
    const defaultTokens = envConfig.getDefaultTokens();
    return getTokenBySymbol(defaultTokens.to);
  };

  // 刷新代币列表
  const refresh = () => {
    fetchTokens();
  };

  return {
    tokens,
    loading,
    error,
    getTokenBySymbol,
    getDefaultFromToken,
    getDefaultToToken,
    refresh
  };
};

