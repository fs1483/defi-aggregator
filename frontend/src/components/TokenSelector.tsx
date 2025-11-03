// 代币选择组件 - 支持按链分组和动态加载
// 根据数据库预制数据动态显示可用代币
import React, { useState, useEffect } from 'react';
// 使用React内置的SVG图标替代heroicons
// import { ChevronDownIcon, ChevronUpIcon } from '@heroicons/react/24/outline';

export interface Token {
  id: number;
  symbol: string;
  name: string;
  contract_address: string;
  chain_id: number;
  decimals: number;
  is_native: boolean;
  is_stable: boolean;
  is_verified: boolean;
  logo_url?: string;
}

export interface Chain {
  id: number;
  chain_id: number;
  name: string;
  display_name: string;
  symbol: string;
  is_testnet: boolean;
  is_active: boolean;
}

interface TokensByChain {
  [chainId: number]: {
    chain: Chain;
    tokens: Token[];
  };
}

interface TokenSelectorProps {
  selectedToken?: Token;
  onTokenSelect: (token: Token) => void;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  showChainInfo?: boolean;
  filter?: 'all' | 'verified' | 'stable';
}

export const TokenSelector: React.FC<TokenSelectorProps> = ({
  selectedToken,
  onTokenSelect,
  placeholder = "选择代币",
  disabled = false,
  className = "",
  showChainInfo = true,
  filter = 'verified'
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [tokensByChain, setTokensByChain] = useState<TokensByChain>({});
  const [chains, setChains] = useState<Chain[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');

  // 获取链和代币数据
  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        setError(null);

        // 并行获取链信息和代币信息
        const [chainsResponse, tokensResponse] = await Promise.all([
          fetch('/api/v1/chains?type=active'),
          fetch('/api/v1/tokens?is_active=true&is_verified=true&page_size=1000')
        ]);

        if (!chainsResponse.ok || !tokensResponse.ok) {
          throw new Error('获取数据失败');
        }

        const chainsData = await chainsResponse.json();
        const tokensData = await tokensResponse.json();

        if (!chainsData.success || !tokensData.success) {
          throw new Error(chainsData.error?.message || tokensData.error?.message || '数据格式错误');
        }

        const chainsMap = new Map<number, Chain>();
        chainsData.data.forEach((chain: Chain) => {
          chainsMap.set(chain.id, chain);
        });

        setChains(chainsData.data);

        // 按链分组代币
        const grouped: TokensByChain = {};
        tokensData.data.forEach((token: Token) => {
          const chain = chainsMap.get(token.chain_id);
          if (chain) {
            if (!grouped[chain.id]) {
              grouped[chain.id] = {
                chain,
                tokens: []
              };
            }
            grouped[chain.id].tokens.push(token);
          }
        });

        setTokensByChain(grouped);

      } catch (err) {
        setError(err instanceof Error ? err.message : '未知错误');
        console.error('获取代币数据失败:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, []);

  // 过滤代币
  const getFilteredTokensByChain = (): TokensByChain => {
    const filtered: TokensByChain = {};

    Object.entries(tokensByChain).forEach(([chainId, { chain, tokens }]) => {
      let filteredTokens = tokens;

      // 应用过滤器
      if (filter === 'verified') {
        filteredTokens = filteredTokens.filter(token => token.is_verified);
      } else if (filter === 'stable') {
        filteredTokens = filteredTokens.filter(token => token.is_stable);
      }

      // 应用搜索
      if (searchQuery) {
        const query = searchQuery.toLowerCase();
        filteredTokens = filteredTokens.filter(token =>
          token.symbol.toLowerCase().includes(query) ||
          token.name.toLowerCase().includes(query)
        );
      }

      if (filteredTokens.length > 0) {
        filtered[parseInt(chainId)] = {
          chain,
          tokens: filteredTokens.sort((a, b) => {
            // 原生代币排在前面
            if (a.is_native && !b.is_native) return -1;
            if (!a.is_native && b.is_native) return 1;
            // 稳定币排在前面
            if (a.is_stable && !b.is_stable) return -1;
            if (!a.is_stable && b.is_stable) return 1;
            // 按符号排序
            return a.symbol.localeCompare(b.symbol);
          })
        };
      }
    });

    return filtered;
  };

  const filteredTokensByChain = getFilteredTokensByChain();

  // 获取代币显示文本
  const getTokenDisplayText = (token: Token) => {
    const chain = chains.find(c => c.id === token.chain_id);
    if (showChainInfo && chain) {
      return `${token.symbol} (${chain.display_name})`;
    }
    return token.symbol;
  };

  // 渲染代币选项
  const renderTokenOption = (token: Token, chain: Chain) => (
    <div
      key={`${chain.id}-${token.id}`}
      className="flex items-center justify-between p-3 hover:bg-gray-50 dark:hover:bg-gray-800 cursor-pointer"
      onClick={() => {
        onTokenSelect(token);
        setIsOpen(false);
        setSearchQuery('');
      }}
    >
      <div className="flex items-center space-x-3">
        {token.logo_url ? (
          <img
            src={token.logo_url}
            alt={token.symbol}
            className="w-6 h-6 rounded-full"
            onError={(e) => {
              e.currentTarget.style.display = 'none';
            }}
          />
        ) : (
          <div className="w-6 h-6 rounded-full bg-gray-300 dark:bg-gray-600 flex items-center justify-center text-xs font-semibold">
            {token.symbol.charAt(0)}
          </div>
        )}
        <div>
          <div className="font-medium text-gray-900 dark:text-gray-100 flex items-center space-x-2">
            <span>{token.symbol}</span>
            {token.is_native && (
              <span className="text-xs bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300 px-1.5 py-0.5 rounded">
                原生
              </span>
            )}
            {token.is_stable && (
              <span className="text-xs bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-300 px-1.5 py-0.5 rounded">
                稳定币
              </span>
            )}
          </div>
          <div className="text-sm text-gray-600 dark:text-gray-400">
            {token.name}
          </div>
        </div>
      </div>
      {showChainInfo && (
        <div className="text-xs text-gray-500 dark:text-gray-400">
          {chain.display_name}
        </div>
      )}
    </div>
  );

  if (loading) {
    return (
      <div className={`relative ${className}`}>
        <div className="w-full px-4 py-3 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-500 dark:text-gray-400">
          加载代币列表...
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={`relative ${className}`}>
        <div className="w-full px-4 py-3 border border-red-300 rounded-lg bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400">
          {error}
        </div>
      </div>
    );
  }

  return (
    <div className={`relative ${className}`}>
      {/* 选择器按钮 */}
      <button
        type="button"
        className={`w-full px-4 py-3 border rounded-lg bg-white dark:bg-gray-800 text-left flex items-center justify-between ${
          disabled
            ? 'border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900 cursor-not-allowed'
            : 'border-gray-300 dark:border-gray-600 hover:border-gray-400 dark:hover:border-gray-500 cursor-pointer'
        }`}
        onClick={() => !disabled && setIsOpen(!isOpen)}
        disabled={disabled}
      >
        <div className="flex items-center space-x-3">
          {selectedToken ? (
            <>
              {selectedToken.logo_url ? (
                <img
                  src={selectedToken.logo_url}
                  alt={selectedToken.symbol}
                  className="w-6 h-6 rounded-full"
                  onError={(e) => {
                    e.currentTarget.style.display = 'none';
                  }}
                />
              ) : (
                <div className="w-6 h-6 rounded-full bg-gray-300 dark:bg-gray-600 flex items-center justify-center text-xs font-semibold">
                  {selectedToken.symbol.charAt(0)}
                </div>
              )}
              <div>
                <div className="font-medium text-gray-900 dark:text-gray-100">
                  {getTokenDisplayText(selectedToken)}
                </div>
                <div className="text-sm text-gray-600 dark:text-gray-400">
                  {selectedToken.name}
                </div>
              </div>
            </>
          ) : (
            <span className="text-gray-500 dark:text-gray-400">{placeholder}</span>
          )}
        </div>
        {!disabled && (
          <div className="ml-2">
            {isOpen ? (
              <svg className="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 15l7-7 7 7" />
              </svg>
            ) : (
              <svg className="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
              </svg>
            )}
          </div>
        )}
      </button>

      {/* 下拉列表 */}
      {isOpen && !disabled && (
        <div className="absolute z-50 w-full mt-1 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg shadow-lg max-h-96 overflow-hidden">
          {/* 搜索框 */}
          <div className="p-3 border-b border-gray-200 dark:border-gray-700">
            <input
              type="text"
              placeholder="搜索代币..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 placeholder-gray-500 dark:placeholder-gray-400"
              autoFocus
            />
          </div>

          {/* 代币列表 */}
          <div className="max-h-80 overflow-y-auto">
            {Object.keys(filteredTokensByChain).length === 0 ? (
              <div className="p-4 text-center text-gray-500 dark:text-gray-400">
                {searchQuery ? '未找到匹配的代币' : '暂无可用代币'}
              </div>
            ) : (
              Object.entries(filteredTokensByChain).map(([chainId, { chain, tokens }]) => (
                <div key={chainId}>
                  {/* 链信息分组标题 */}
                  <div className="sticky top-0 px-3 py-2 bg-gray-100 dark:bg-gray-700 border-b border-gray-200 dark:border-gray-600">
                    <div className="flex items-center space-x-2">
                      <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                        {chain.display_name}
                      </span>
                      {chain.is_testnet && (
                        <span className="text-xs bg-yellow-100 dark:bg-yellow-900 text-yellow-700 dark:text-yellow-300 px-1.5 py-0.5 rounded">
                          测试网
                        </span>
                      )}
                      <span className="text-xs text-gray-500 dark:text-gray-400">
                        ({tokens.length} 个代币)
                      </span>
                    </div>
                  </div>
                  
                  {/* 代币列表 */}
                  {tokens.map(token => renderTokenOption(token, chain))}
                </div>
              ))
            )}
          </div>
        </div>
      )}

      {/* 点击外部关闭下拉框 */}
      {isOpen && (
        <div
          className="fixed inset-0 z-40"
          onClick={() => setIsOpen(false)}
        />
      )}
    </div>
  );
};
