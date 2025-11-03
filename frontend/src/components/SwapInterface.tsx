// 交易界面组件 - 企业级版本
// 支持动态环境配置的代币交换界面

import React, { useState, useEffect } from 'react';
import { useAccount } from 'wagmi';
import { TokenSelector, type Token } from './TokenSelector';
import { Toast, type ToastProps } from './Toast';
import { envConfig, getCurrentChainId, getDefaultSlippage, getAPIBaseURL } from '../config/environment';

export const SwapInterface: React.FC = () => {
  const { address, isConnected } = useAccount();
  
  const [fromToken, setFromToken] = useState<Token | undefined>();
  const [toToken, setToToken] = useState<Token | undefined>();
  const [fromAmount, setFromAmount] = useState('');
  const [toAmount, setToAmount] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [quoteData, setQuoteData] = useState<any>(null);
  
  // Toast通知状态
  const [toast, setToast] = useState<{
    isVisible: boolean;
    type: ToastProps['type'];
    title: string;
    message?: string;
    details?: string[];
  }>({
    isVisible: false,
    type: 'info',
    title: '',
    message: '',
    details: []
  });
  
  // 环境配置
  const chainId = getCurrentChainId();
  const defaultSlippage = getDefaultSlippage();
  const apiBaseURL = getAPIBaseURL();
  
  // 初始化时输出环境配置
  useEffect(() => {
    envConfig.logCurrentConfig();
  }, []);

  // Toast显示函数
  const showToast = (type: ToastProps['type'], title: string, message?: string, details?: string[]) => {
    setToast({
      isVisible: true,
      type,
      title,
      message,
      details
    });
  };

  const hideToast = () => {
    setToast(prev => ({ ...prev, isVisible: false }));
  };

  const handleGetQuote = async () => {
    if (!fromAmount || !isConnected || !fromToken || !toToken || isLoading) return;

    try {
      setIsLoading(true);
      setQuoteData(null);
      
      // 将用户输入的数量转换为wei单位
      const amountInWei = (parseFloat(fromAmount) * Math.pow(10, fromToken.decimals)).toString();
      
      // 构建报价请求参数
      const quoteRequest = {
        from_token_id: fromToken.id,           // 选择的源代币ID
        to_token_id: toToken.id,              // 选择的目标代币ID
        amount_in: amountInWei,               // 转换为wei单位的数量
        chain_id: chainId,                    // 当前链ID
        slippage: defaultSlippage,            // 滑点配置
        user_address: address                 // 用户钱包地址
      };
      
      console.log('🔄 发送报价请求:', {
        ...quoteRequest,
        fromToken: fromToken.symbol,
        toToken: toToken.symbol,
        environment: envConfig.getEnvironment(),
        chainName: envConfig.getCurrentChainConfig().name,
        isTestnet: envConfig.isTestnet()
      });
      
      // 调用报价API
      const response = await fetch(`${apiBaseURL}/api/v1/quotes`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(quoteRequest)
      });
      
      if (!response.ok) {
        throw new Error(`API请求失败: ${response.status} ${response.statusText}`);
      }
      
      const data = await response.json();
      
      console.log('✅ 报价API调用成功:', data);
      
      // 检查响应格式
      if (data.success && data.data) {
        setQuoteData(data.data);
        // 更新接收金额显示
        if (data.data.amount_out) {
          // 将wei转换为可读格式
          const amountOut = parseFloat(data.data.amount_out) / Math.pow(10, toToken.decimals);
          setToAmount(amountOut.toFixed(6));
        }
        
        // 显示成功Toast
        showToast(
          'success',
          '报价获取成功！',
          `通过 ${data.data.best_aggregator || 'Unknown'} 聚合器获取最优报价`,
          [
            `环境: ${envConfig.getCurrentChainConfig().name} (${envConfig.isTestnet() ? '测试网' : '主网'})`,
            `交易对: ${fromToken.symbol} → ${toToken.symbol}`,
            `预期收到: ${data.data.amount_out ? (parseFloat(data.data.amount_out) / Math.pow(10, toToken.decimals)).toFixed(6) : 'N/A'} ${toToken.symbol}`,
            `Gas估算: ${data.data.gas_estimate || 'N/A'}`
          ]
        );
      } else {
        throw new Error(data.error?.message || '报价响应格式错误');
      }
      
    } catch (error) {
      console.error('❌ 报价API调用失败:', error);
      setToAmount('');
      setQuoteData(null);
      
      // 显示错误Toast
      showToast(
        'error',
        '报价获取失败',
        error instanceof Error ? error.message : '未知错误',
        [
          '请检查以下项目：',
          '• 后端服务是否正常运行',
          '• 聚合器是否启用',
          `• 当前环境: ${envConfig.getCurrentChainConfig().name}`,
          '• 网络连接是否正常',
          '• 代币选择是否正确'
        ]
      );
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 shadow-lg p-6">
      <div className="text-center mb-6">
        <h2 className="text-2xl font-bold text-gray-900 dark:text-gray-100 mb-2">
          代币交换
        </h2>
        <p className="text-gray-600 dark:text-gray-400">
          聚合多个DEX，获取最优交易价格
        </p>
        {/* 环境指示器 */}
        <div className="mt-2 inline-flex items-center px-3 py-1 rounded-full text-xs bg-blue-100 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300">
          <span className="w-2 h-2 bg-blue-500 rounded-full mr-2"></span>
          {envConfig.getCurrentChainConfig().name} {envConfig.isTestnet() ? '测试网' : '主网'}
        </div>
      </div>

      {/* 交易界面 */}
      <div className="max-w-md mx-auto space-y-4">
        {/* 支付代币选择 */}
        <div className="text-left">
          <label className="block text-sm text-gray-600 dark:text-gray-400 mb-2">支付</label>
          <div className="space-y-3">
            <TokenSelector
              selectedToken={fromToken}
              onTokenSelect={setFromToken}
              placeholder="选择支付代币"
              className="w-full"
              showChainInfo={true}
              filter="verified"
            />
            <input
              type="number"
              placeholder="0.0"
              value={fromAmount}
              onChange={(e) => setFromAmount(e.target.value)}
              className="w-full px-4 py-3 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-lg"
              disabled={!fromToken}
            />
          </div>
        </div>

        {/* 交换按钮 */}
        <div className="flex justify-center">
          <button 
            className="p-2 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
            onClick={() => {
              // 交换代币选择
              const tempToken = fromToken;
              setFromToken(toToken);
              setToToken(tempToken);
              setFromAmount('');
              setToAmount('');
              setQuoteData(null);
            }}
            disabled={!fromToken && !toToken}
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16V4m0 0L3 8m4-4l4 4m6 0v12m0 0l4-4m-4 4l-4-4" />
            </svg>
          </button>
        </div>

        {/* 接收代币选择 */}
        <div className="text-left">
          <label className="block text-sm text-gray-600 dark:text-gray-400 mb-2">接收</label>
          <div className="space-y-3">
            <TokenSelector
              selectedToken={toToken}
              onTokenSelect={setToToken}
              placeholder="选择接收代币"
              className="w-full"
              showChainInfo={true}
              filter="verified"
            />
            <input
              type="text"
              placeholder="0.0"
              value={toAmount}
              className="w-full px-4 py-3 border border-gray-300 dark:border-gray-600 rounded-lg bg-gray-50 dark:bg-gray-800 text-lg"
              readOnly
            />
          </div>
        </div>

        {/* 获取报价按钮 */}
        <button
          onClick={handleGetQuote}
          disabled={!isConnected || !fromAmount || !fromToken || !toToken || isLoading}
          className={`w-full py-4 rounded-xl font-semibold text-lg transition-all ${
            isConnected && fromAmount && fromToken && toToken && !isLoading
              ? 'bg-blue-600 hover:bg-blue-700 text-white shadow-lg hover:shadow-xl'
              : 'bg-gray-200 dark:bg-gray-700 text-gray-500 dark:text-gray-400 cursor-not-allowed'
          }`}
        >
          {isLoading ? (
            <div className="flex items-center justify-center">
              <div className="animate-spin rounded-full h-5 w-5 border-2 border-white border-t-transparent mr-2"></div>
              获取最优报价中...
            </div>
          ) : !isConnected ? (
            '请连接钱包'
          ) : !fromToken ? (
            '请选择支付代币'
          ) : !toToken ? (
            '请选择接收代币'
          ) : !fromAmount ? (
            '请输入交易数量'
          ) : (
            '获取最优报价'
          )}
        </button>

        {/* 报价信息显示 */}
        {quoteData && (
          <div className="mt-6 p-4 bg-green-50 dark:bg-green-900/20 rounded-lg border border-green-200 dark:border-green-700">
            <h4 className="font-medium text-green-900 dark:text-green-100 mb-3 flex items-center">
              <span className="text-green-500 mr-2">💚</span>
              报价详情
            </h4>
            <div className="text-sm text-green-800 dark:text-green-200 space-y-2">
              <div className="flex justify-between">
                <span>最优聚合器:</span>
                <span className="font-medium">{quoteData.best_aggregator || 'CoW Protocol'}</span>
              </div>
              <div className="flex justify-between">
                <span>预期收到:</span>
                <span className="font-medium">{quoteData.amount_out || toAmount} {toToken?.symbol || 'N/A'}</span>
              </div>
              <div className="flex justify-between">
                <span>预估Gas:</span>
                <span className="font-medium">{quoteData.gas_estimate || 'N/A'}</span>
              </div>
              <div className="flex justify-between">
                <span>价格影响:</span>
                <span className="font-medium">{quoteData.price_impact || '< 0.1%'}</span>
              </div>
              <div className="flex justify-between">
                <span>链网络:</span>
                <span className="font-medium">{envConfig.getCurrentChainConfig().name}</span>
              </div>
            </div>
          </div>
        )}
        
      </div>

      {/* Toast通知组件 */}
      <Toast
        type={toast.type}
        title={toast.title}
        message={toast.message}
        details={toast.details}
        isVisible={toast.isVisible}
        onClose={hideToast}
        autoClose={true}
        duration={6000}
      />
    </div>
  );
};