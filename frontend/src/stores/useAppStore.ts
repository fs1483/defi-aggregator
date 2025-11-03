// 应用全局状态管理
// 使用Zustand管理应用状态，包括用户、代币、交易等
// 提供类型安全的状态管理和持久化

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { User, Token, Chain, AppState } from '../types';

// ========================================
// 应用状态接口
// ========================================

interface AppStore extends AppState {
  // 用户相关
  setUser: (user: User | null) => void;
  setAuthenticated: (isAuthenticated: boolean) => void;
  logout: () => void;
  
  // 代币相关
  setTokens: (tokens: Token[]) => void;
  addToken: (token: Token) => void;
  updateToken: (id: number, updates: Partial<Token>) => void;
  getTokenById: (id: number) => Token | undefined;
  getTokenByAddress: (chainId: number, address: string) => Token | undefined;
  
  // 区块链相关
  setChains: (chains: Chain[]) => void;
  setSelectedChain: (chain: Chain | null) => void;
  getChainById: (chainId: number) => Chain | undefined;
  
  // UI状态
  setTheme: (theme: 'light' | 'dark') => void;
  setLoading: (isLoading: boolean) => void;
  
  // 初始化
  initialize: () => Promise<void>;
}

// ========================================
// 创建应用状态store
// ========================================

export const useAppStore = create<AppStore>()(
  persist(
    (set, get) => ({
      // 初始状态
      user: null,
      isAuthenticated: false,
      tokens: [],
      chains: [],
      selectedChain: null,
      theme: 'light',
      isLoading: false,

      // ========================================
      // 用户相关操作
      // ========================================

      setUser: (user) => {
        set({ user });
      },

      setAuthenticated: (isAuthenticated) => {
        set({ isAuthenticated });
      },

      logout: () => {
        // 清除本地存储的认证信息
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
        
        set({
          user: null,
          isAuthenticated: false,
        });
      },

      // ========================================
      // 代币相关操作
      // ========================================

      setTokens: (tokens) => {
        set({ tokens });
      },

      addToken: (token) => {
        const { tokens } = get();
        const existingIndex = tokens.findIndex(t => t.id === token.id);
        
        if (existingIndex >= 0) {
          // 更新现有代币
          const updatedTokens = [...tokens];
          updatedTokens[existingIndex] = token;
          set({ tokens: updatedTokens });
        } else {
          // 添加新代币
          set({ tokens: [...tokens, token] });
        }
      },

      updateToken: (id, updates) => {
        const { tokens } = get();
        const updatedTokens = tokens.map(token =>
          token.id === id ? { ...token, ...updates } : token
        );
        set({ tokens: updatedTokens });
      },

      getTokenById: (id) => {
        const { tokens } = get();
        return tokens.find(token => token.id === id);
      },

      getTokenByAddress: (chainId, address) => {
        const { tokens } = get();
        return tokens.find(token => 
          token.chain_id === chainId && 
          token.contract_address.toLowerCase() === address.toLowerCase()
        );
      },

      // ========================================
      // 区块链相关操作
      // ========================================

      setChains: (chains) => {
        set({ chains });
      },

      setSelectedChain: (chain) => {
        set({ selectedChain: chain });
      },

      getChainById: (chainId) => {
        const { chains } = get();
        return chains.find(chain => chain.chain_id === chainId);
      },

      // ========================================
      // UI状态操作
      // ========================================

      setTheme: (theme) => {
        set({ theme });
        
        // 更新HTML类名以支持dark mode
        if (theme === 'dark') {
          document.documentElement.classList.add('dark');
        } else {
          document.documentElement.classList.remove('dark');
        }
      },

      setLoading: (isLoading) => {
        set({ isLoading });
      },

      // ========================================
      // 初始化
      // ========================================

      initialize: async () => {
        try {
          set({ isLoading: true });

          // 检查本地存储的认证状态
          const accessToken = localStorage.getItem('access_token');
          if (accessToken) {
            set({ isAuthenticated: true });
            
            // TODO: 验证令牌有效性并获取用户信息
            // const user = await UserAPI.getProfile();
            // set({ user });
          }

          // 初始化主题
          const savedTheme = localStorage.getItem('theme') as 'light' | 'dark' || 'light';
          get().setTheme(savedTheme);

        } catch (error) {
          console.error('应用初始化失败:', error);
        } finally {
          set({ isLoading: false });
        }
      },
    }),
    {
      name: 'defi-aggregator-app-store',
      partialize: (state) => ({
        // 只持久化这些字段
        theme: state.theme,
        selectedChain: state.selectedChain,
      }),
    }
  )
);

// ========================================
// 交易状态管理
// ========================================

interface SwapStore {
  // 交易状态
  fromToken: Token | null;
  toToken: Token | null;
  fromAmount: string;
  toAmount: string;
  slippage: string;
  isLoading: boolean;
  quote: any | null; // QuoteResponse
  error: string | null;
  lastQuoteTime: number | null;
  
  // 操作方法
  setFromToken: (token: Token | null) => void;
  setToToken: (token: Token | null) => void;
  setFromAmount: (amount: string) => void;
  setToAmount: (amount: string) => void;
  setSlippage: (slippage: string) => void;
  setLoading: (isLoading: boolean) => void;
  setQuote: (quote: any | null) => void;
  setError: (error: string | null) => void;
  swapTokens: () => void;
  reset: () => void;
}

export const useSwapStore = create<SwapStore>((set, get) => ({
  // 初始状态
  fromToken: null,
  toToken: null,
  fromAmount: '',
  toAmount: '',
  slippage: '0.5', // 默认0.5%滑点
  isLoading: false,
  quote: null,
  error: null,
  lastQuoteTime: null,

  // ========================================
  // 状态更新方法
  // ========================================

  setFromToken: (token) => {
    const { toToken } = get();
    
    // 如果选择的代币和目标代币相同，交换它们
    if (token && toToken && token.id === toToken.id) {
      set({
        fromToken: token,
        toToken: null,
        quote: null,
        error: null,
      });
    } else {
      set({
        fromToken: token,
        quote: null,
        error: null,
      });
    }
  },

  setToToken: (token) => {
    const { fromToken } = get();
    
    // 如果选择的代币和源代币相同，交换它们
    if (token && fromToken && token.id === fromToken.id) {
      set({
        toToken: token,
        fromToken: null,
        quote: null,
        error: null,
      });
    } else {
      set({
        toToken: token,
        quote: null,
        error: null,
      });
    }
  },

  setFromAmount: (amount) => {
    set({
      fromAmount: amount,
      quote: null,
      error: null,
    });
  },

  setToAmount: (amount) => {
    set({ toAmount: amount });
  },

  setSlippage: (slippage) => {
    set({
      slippage,
      quote: null, // 滑点变化时清除报价
    });
  },

  setLoading: (isLoading) => {
    set({ isLoading });
  },

  setQuote: (quote) => {
    set({
      quote,
      lastQuoteTime: quote ? Date.now() : null,
      error: null,
    });
  },

  setError: (error) => {
    set({
      error,
      quote: null,
      isLoading: false,
    });
  },

  // ========================================
  // 交易操作
  // ========================================

  swapTokens: () => {
    const { fromToken, toToken, fromAmount, toAmount } = get();
    
    set({
      fromToken: toToken,
      toToken: fromToken,
      fromAmount: toAmount,
      toAmount: fromAmount,
      quote: null,
      error: null,
    });
  },

  reset: () => {
    set({
      fromToken: null,
      toToken: null,
      fromAmount: '',
      toAmount: '',
      quote: null,
      error: null,
      isLoading: false,
      lastQuoteTime: null,
    });
  },
}));

// ========================================
// 钱包状态管理
// ========================================

interface WalletStore {
  // 连接状态
  isConnecting: boolean;
  error: string | null;
  
  // 操作方法
  setConnecting: (isConnecting: boolean) => void;
  setError: (error: string | null) => void;
  clearError: () => void;
}

export const useWalletStore = create<WalletStore>((set) => ({
  // 初始状态
  isConnecting: false,
  error: null,

  // 状态更新方法
  setConnecting: (isConnecting) => {
    set({ isConnecting });
  },

  setError: (error) => {
    set({ error, isConnecting: false });
  },

  clearError: () => {
    set({ error: null });
  },
}));
