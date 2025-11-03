// 企业级钱包连接组件
// 提供智能化、用户友好的Web3钱包连接体验

import React, { useState, useEffect, useRef } from 'react';
import { useAccount, useConnect, useDisconnect } from 'wagmi';
import { formatAddress } from '../config/web3';

// 钱包图标映射
const walletIcons: Record<string, string> = {
  metaMask: '🦊',
  injected: '🔌',
  walletConnect: '📱',
  coinbaseWallet: '🔵',
};

// 钱包显示名称映射
const walletDisplayNames: Record<string, string> = {
  metaMask: 'MetaMask',
  injected: '浏览器钱包',
  walletConnect: 'WalletConnect',
  coinbaseWallet: 'Coinbase',
};

export const WalletConnect: React.FC = () => {
  const { address, isConnected, connector } = useAccount();
  const { connect, connectors, isPending, error } = useConnect();
  const { disconnect } = useDisconnect();
  const [showConnectors, setShowConnectors] = useState(false);
  const [selectedConnector, setSelectedConnector] = useState<string | null>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // 重置选中状态
  useEffect(() => {
    if (isConnected || !isPending) {
      setSelectedConnector(null);
    }
  }, [isConnected, isPending]);

  // 点击外部关闭下拉菜单
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setShowConnectors(false);
      }
    };

    if (showConnectors) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [showConnectors]);

  // 智能检测推荐钱包
  const getRecommendedConnector = () => {
    // 添加调试信息
    console.log('🔍 检测钱包环境:', {
      hasWindow: typeof window !== 'undefined',
      hasEthereum: typeof window !== 'undefined' && !!window.ethereum,
      isMetaMask: typeof window !== 'undefined' && window.ethereum?.isMetaMask,
      connectors: connectors.map(c => ({ id: c.id, name: c.name }))
    });
    
    // 检查是否安装了MetaMask
    if (typeof window !== 'undefined' && window.ethereum?.isMetaMask) {
      return connectors.find(c => c.id === 'metaMask') || connectors.find(c => c.id === 'injected');
    }
    // 默认推荐注入式钱包
    return connectors.find(c => c.id === 'injected') || connectors[0];
  };

  const recommendedConnector = getRecommendedConnector();

  const handleConnect = (connector: any) => {
    console.log('🔗 尝试连接钱包:', { id: connector.id, name: connector.name });
    setSelectedConnector(connector.id);
    try {
      connect({ connector });
    } catch (error) {
      console.error('❌ 钱包连接失败:', error);
    }
  };

  const handleQuickConnect = () => {
    if (recommendedConnector) {
      handleConnect(recommendedConnector);
    }
  };

  // 已连接状态 - 显示钱包信息
  if (isConnected && address) {
    return (
      <div className="flex items-center space-x-3">
        {/* 钱包状态指示 */}
        <div className="flex items-center space-x-2 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg px-4 py-2">
          <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
          <span className="text-green-700 dark:text-green-300 text-sm font-medium">
            {walletIcons[connector?.id || 'injected']} {walletDisplayNames[connector?.id || 'injected']}
          </span>
        </div>
        
        {/* 地址显示 */}
        <div className="bg-gray-100 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg px-3 py-2">
          <span className="text-gray-700 dark:text-gray-300 text-sm font-mono">
            {formatAddress(address)}
          </span>
        </div>
        
        {/* 断开按钮 */}
        <button
          onClick={() => disconnect()}
          className="px-3 py-2 text-sm border border-red-200 dark:border-red-800 text-red-600 dark:text-red-400 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
        >
          断开
        </button>
      </div>
    );
  }

  // 未连接状态 - 紧凑的头部钱包连接界面
  return (
    <div className="relative" ref={dropdownRef}>
      {/* 错误提示 */}
      {error && (
        <div className="absolute top-full right-0 mt-2 w-80 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-3 shadow-lg z-50">
          <div className="flex items-center space-x-2">
            <span className="text-red-500">⚠️</span>
            <span className="text-red-700 dark:text-red-300 text-sm">
              连接失败: {error.message}
            </span>
          </div>
        </div>
      )}

      {/* 主连接按钮 */}
      <div className="flex items-center space-x-2">
        {/* 快速连接（推荐） */}
        {recommendedConnector && (
          <button
            onClick={handleQuickConnect}
            disabled={isPending}
            className={`flex items-center space-x-2 px-4 py-2 rounded-lg font-medium transition-all duration-200 ${
              isPending && selectedConnector === recommendedConnector.id
                ? 'bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 text-blue-600 dark:text-blue-400 cursor-not-allowed'
                : 'bg-gradient-to-r from-blue-500 to-purple-600 hover:from-blue-600 hover:to-purple-700 text-white border border-transparent hover:shadow-md'
            }`}
          >
            {isPending && selectedConnector === recommendedConnector.id ? (
              <>
                <div className="w-4 h-4 border-2 border-blue-600 border-t-transparent rounded-full animate-spin"></div>
                <span className="text-sm">连接中...</span>
              </>
            ) : (
              <>
                <span>{walletIcons[recommendedConnector.id] || '🔌'}</span>
                <span className="text-sm font-medium">
                  推荐: {walletDisplayNames[recommendedConnector.id] || recommendedConnector.name}
                </span>
                <span className="text-xs opacity-80">（已检测到）</span>
              </>
            )}
          </button>
        )}

        {/* 更多选项按钮 */}
        <button
          onClick={() => setShowConnectors(!showConnectors)}
          className="px-3 py-2 text-sm text-gray-600 dark:text-gray-400 border border-gray-200 dark:border-gray-700 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700/50 hover:text-gray-800 dark:hover:text-gray-200 transition-colors"
        >
          查看更多钱包选项
        </button>
      </div>

      {/* 下拉钱包选项 */}
      {showConnectors && (
        <div className="absolute top-full right-0 mt-2 w-80 bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 shadow-lg z-50">
          <div className="p-4">
            <div className="flex items-center justify-between mb-3">
              <h4 className="text-sm font-semibold text-gray-900 dark:text-white">选择钱包</h4>
              <button
                onClick={() => setShowConnectors(false)}
                className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
              >
                ✕
              </button>
            </div>
            
            <div className="space-y-2">
              {connectors
                .filter(connector => connector.id !== recommendedConnector?.id)
                .map((connector) => (
                <button
                  key={connector.id}
                  onClick={() => handleConnect(connector)}
                  disabled={isPending}
                  className={`w-full flex items-center justify-between px-3 py-2 rounded-lg border transition-all duration-200 ${
                    isPending && selectedConnector === connector.id
                      ? 'bg-gray-50 dark:bg-gray-700/50 border-gray-300 dark:border-gray-600 text-gray-600 dark:text-gray-400 cursor-not-allowed'
                      : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-700/50'
                  }`}
                >
                  <div className="flex items-center space-x-2">
                    <span>{walletIcons[connector.id] || '🔌'}</span>
                    <span className="text-sm font-medium text-gray-900 dark:text-white">
                      {walletDisplayNames[connector.id] || connector.name}
                    </span>
                  </div>
                  {isPending && selectedConnector === connector.id && (
                    <div className="w-3 h-3 border-2 border-gray-600 border-t-transparent rounded-full animate-spin"></div>
                  )}
                </button>
              ))}
            </div>

            {/* 提示信息 */}
            <div className="mt-3 pt-3 border-t border-gray-200 dark:border-gray-700">
              <p className="text-xs text-gray-500 dark:text-gray-400 text-center">
                💡 没有钱包？推荐安装 <a href="https://metamask.io" target="_blank" rel="noopener noreferrer" className="text-blue-500 hover:underline">MetaMask</a>
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
