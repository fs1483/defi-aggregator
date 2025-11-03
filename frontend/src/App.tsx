// DeFi聚合器主应用组件
// 简化版本，专注于核心功能展示

import React from 'react';
import { WagmiProvider } from 'wagmi';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
// import { ConnectKitProvider } from 'connectkit';

import { wagmiConfig } from './config/web3';
import { WalletConnect } from './components/WalletConnect';
import { SwapInterface } from './components/SwapInterface';

// 创建React Query客户端
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 60 * 1000, // 1分钟
      refetchOnWindowFocus: false,
    },
  },
});

// ========================================
// 主应用组件
// ========================================

function App() {
  return (
    <WagmiProvider config={wagmiConfig}>
      <QueryClientProvider client={queryClient}>
        <AppContent />
      </QueryClientProvider>
    </WagmiProvider>
  );
}

// ========================================
// 辅助组件
// ========================================

const TechBadge: React.FC<{
  name: string;
  color: 'blue' | 'green' | 'purple' | 'red' | 'cyan' | 'indigo';
}> = ({ name, color }) => {
  const colorClasses = {
    blue: 'bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-400',
    green: 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400',
    purple: 'bg-purple-100 text-purple-800 dark:bg-purple-900/20 dark:text-purple-400',
    red: 'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-400',
    cyan: 'bg-cyan-100 text-cyan-800 dark:bg-cyan-900/20 dark:text-cyan-400',
    indigo: 'bg-indigo-100 text-indigo-800 dark:bg-indigo-900/20 dark:text-indigo-400',
  };

  return (
    <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${colorClasses[color]}`}>
      {name}
    </span>
  );
};

// ========================================
// 应用内容组件
// ========================================

const AppContent: React.FC = () => {
  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 via-blue-50 to-indigo-100 dark:from-gray-900 dark:via-gray-800 dark:to-indigo-900">
      {/* 背景装饰 */}
      <div className="fixed inset-0 overflow-hidden pointer-events-none">
        <div className="absolute -top-40 -right-40 w-80 h-80 rounded-full bg-gradient-to-br from-blue-400/20 to-purple-600/20 blur-3xl"></div>
        <div className="absolute -bottom-40 -left-40 w-80 h-80 rounded-full bg-gradient-to-br from-green-400/20 to-blue-600/20 blur-3xl"></div>
      </div>

      {/* 头部导航 */}
      <header className="glass-effect sticky top-0 z-50 backdrop-blur-xl">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center">
              <div className="flex-shrink-0 flex items-center">
                <div className="w-8 h-8 rounded-lg defi-gradient-bg flex items-center justify-center mr-3">
                  <svg className="w-5 h-5 text-white" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M3 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z" clipRule="evenodd" />
                  </svg>
                </div>
                <div>
                  <h1 className="text-xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
                    DeFi聚合器
                  </h1>
                  <p className="text-xs text-gray-500 dark:text-gray-400">企业级Web3聚合交易平台</p>
                </div>
              </div>
              
              <nav className="hidden md:ml-8 md:flex md:space-x-8">
                <a href="#" className="text-gray-700 dark:text-gray-300 hover:text-blue-600 dark:hover:text-blue-400 px-3 py-2 text-sm font-medium transition-colors">
                  🔄 交易
                </a>
                <a href="#" className="text-gray-700 dark:text-gray-300 hover:text-blue-600 dark:hover:text-blue-400 px-3 py-2 text-sm font-medium transition-colors">
                  📊 历史
                </a>
                <a href="#" className="text-gray-700 dark:text-gray-300 hover:text-blue-600 dark:hover:text-blue-400 px-3 py-2 text-sm font-medium transition-colors">
                  📈 分析
                </a>
              </nav>
            </div>

            {/* 钱包连接区域 */}
            <div className="flex items-center space-x-4">
              <WalletConnect />
            </div>
          </div>
        </div>
      </header>

      {/* 主要内容 */}
      <main className="relative max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* 欢迎信息 */}
        <div className="text-center mb-12">
          <h2 className="text-4xl font-bold text-gray-900 dark:text-white mb-4">
            聚合多个DEX，获取最优交易价格
          </h2>
          <p className="text-xl text-gray-600 dark:text-gray-300 max-w-3xl mx-auto">
            支持以太坊和Polygon网络，通过智能路由算法为您找到最佳的代币交换路径
          </p>
        </div>

        {/* 交易界面 - 聚焦核心功能 */}
        <div className="max-w-4xl mx-auto">
          <div className="defi-card p-8">
            <div className="flex items-center mb-6">
              <div className="w-6 h-6 rounded bg-gradient-to-r from-green-400 to-blue-500 mr-3"></div>
              <h3 className="text-2xl font-bold text-gray-900 dark:text-white">代币交换</h3>
            </div>
            <SwapInterface />
          </div>
        </div>
      </main>

      {/* 页脚 */}
      <footer className="relative mt-16 glass-effect border-t border-gray-200/20 dark:border-gray-700/20">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
          <div className="text-center">
            {/* 主标题 */}
            <div className="flex justify-center items-center mb-4">
              <div className="w-6 h-6 rounded-lg defi-gradient-bg flex items-center justify-center mr-2">
                <svg className="w-4 h-4 text-white" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M3 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z" clipRule="evenodd" />
                </svg>
              </div>
              <h3 className="text-xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
                DeFi聚合器
              </h3>
            </div>
            
            <p className="text-gray-600 dark:text-gray-400 mb-2">
              © 2024 DeFi聚合器 - 企业级Web3聚合交易平台
            </p>
            <p className="text-sm text-gray-500 dark:text-gray-500 mb-6">
              采用微服务架构，为用户提供最优的DeFi交易体验
            </p>
            
            {/* 技术栈标签 */}
            <div className="flex flex-wrap justify-center gap-3 mb-6">
              <TechBadge name="React 19" color="blue" />
              <TechBadge name="TypeScript" color="blue" />
              <TechBadge name="Go 1.21" color="green" />
              <TechBadge name="PostgreSQL" color="purple" />
              <TechBadge name="Redis" color="red" />
              <TechBadge name="Tailwind v4" color="cyan" />
              <TechBadge name="Docker" color="indigo" />
            </div>
            
            {/* 特性说明 */}
            <div className="text-xs text-gray-500 dark:text-gray-500 space-y-1">
              <p>🏗️ Database First架构 • ⚡ Go微服务 • 🔒 企业级安全</p>
              <p>🌐 支持多链 • 🔄 智能路由 • 📊 实时监控</p>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
};

export default App;