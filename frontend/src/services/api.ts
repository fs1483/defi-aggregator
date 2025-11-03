// API服务层
// 封装所有后端API调用，提供类型安全的接口
// 通过API网关统一访问后端服务

import axios from 'axios';
import type { AxiosInstance, AxiosResponse } from 'axios';
import { APIResponse, APIError as APIErrorType, LoginRequest, LoginResponse, User, UserPreferences, UserStats, Token, Meta, Chain, QuoteRequest, QuoteResponse, SwapRequest, SwapResponse, Transaction } from '../types';

// API基础配置
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:5176';
const API_TIMEOUT = 30000; // 30秒超时

// 创建axios实例
class APIClient {
  private client: AxiosInstance;

  constructor() {
    this.client = axios.create({
      baseURL: API_BASE_URL,
      timeout: API_TIMEOUT,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // 请求拦截器
    this.client.interceptors.request.use(
      (config) => {
        // 添加认证令牌
        const token = localStorage.getItem('access_token');
        if (token) {
          config.headers.Authorization = `Bearer ${token}`;
        }

        // 添加请求ID
        config.headers['X-Request-ID'] = `web_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

        console.log(`🔄 API请求: ${config.method?.toUpperCase()} ${config.url}`);
        return config;
      },
      (error) => {
        console.error('❌ 请求拦截器错误:', error);
        return Promise.reject(error);
      }
    );

    // 响应拦截器
    this.client.interceptors.response.use(
      (response: AxiosResponse<APIResponse>) => {
        console.log(`✅ API响应: ${response.config.method?.toUpperCase()} ${response.config.url} - ${response.status}`);
        return response;
      },
      (error) => {
        console.error('❌ API错误:', error);
        
        // 处理认证错误
        if (error.response?.status === 401) {
          // 清除本地令牌
          localStorage.removeItem('access_token');
          localStorage.removeItem('refresh_token');
          
          // 重定向到登录页面
          window.location.href = '/login';
        }

        return Promise.reject(this.handleAPIError(error));
      }
    );
  }

  // 处理API错误
  private handleAPIError(error: any): APIErrorType {
    if (error.response?.data?.error) {
      const apiError = error.response.data.error;
      return new APIErrorType(
        apiError.code || 'UNKNOWN_ERROR',
        apiError.message || '未知错误',
        error.response.status,
        apiError.details
      );
    }

    if (error.code === 'ECONNABORTED') {
      return new APIErrorType('TIMEOUT', '请求超时', 408);
    }

    if (error.code === 'NETWORK_ERROR') {
      return new APIErrorType('NETWORK_ERROR', '网络连接失败', 0);
    }

    return new APIErrorType(
      'UNKNOWN_ERROR',
      error.message || '未知错误',
      error.response?.status || 0
    );
  }

  // 通用GET请求
  async get<T>(url: string, params?: Record<string, any>): Promise<T> {
    const response = await this.client.get<APIResponse<T>>(url, { params });
    
    if (!response.data.success) {
      throw new APIErrorType(
        response.data.error?.code || 'API_ERROR',
        response.data.error?.message || 'API请求失败'
      );
    }

    return response.data.data as T;
  }

  // 通用POST请求
  async post<T>(url: string, data?: any): Promise<T> {
    const response = await this.client.post<APIResponse<T>>(url, data);
    
    if (!response.data.success) {
      throw new APIErrorType(
        response.data.error?.code || 'API_ERROR',
        response.data.error?.message || 'API请求失败'
      );
    }

    return response.data.data as T;
  }

  // 通用PUT请求
  async put<T>(url: string, data?: any): Promise<T> {
    const response = await this.client.put<APIResponse<T>>(url, data);
    
    if (!response.data.success) {
      throw new APIErrorType(
        response.data.error?.code || 'API_ERROR',
        response.data.error?.message || 'API请求失败'
      );
    }

    return response.data.data as T;
  }

  // 通用DELETE请求
  async delete<T>(url: string): Promise<T> {
    const response = await this.client.delete<APIResponse<T>>(url);
    
    if (!response.data.success) {
      throw new APIErrorType(
        response.data.error?.code || 'API_ERROR',
        response.data.error?.message || 'API请求失败'
      );
    }

    return response.data.data as T;
  }

  // 获取原始响应（包含meta等信息）
  async getRaw<T>(url: string, params?: Record<string, any>): Promise<APIResponse<T>> {
    const response = await this.client.get<APIResponse<T>>(url, { params });
    return response.data;
  }

  // POST请求获取原始响应
  async postRaw<T>(url: string, data?: any): Promise<APIResponse<T>> {
    const response = await this.client.post<APIResponse<T>>(url, data);
    return response.data;
  }
}

// 创建API客户端实例
export const apiClient = new APIClient();

// ========================================
// 具体API服务类
// ========================================

// 认证API服务
export class AuthAPI {
  // 获取登录随机数
  static async getNonce(walletAddress: string): Promise<{ nonce: string; message: string; timestamp: number }> {
    return apiClient.post('/api/v1/auth/nonce', { wallet_address: walletAddress });
  }

  // 钱包登录
  static async login(loginData: LoginRequest): Promise<LoginResponse> {
    return apiClient.post('/api/v1/auth/login', loginData);
  }

  // 刷新令牌
  static async refreshToken(refreshToken: string): Promise<{ access_token: string; expires_in: number }> {
    return apiClient.post('/api/v1/auth/refresh', { refresh_token: refreshToken });
  }

  // 用户登出
  static async logout(): Promise<void> {
    return apiClient.post('/api/v1/auth/logout');
  }
}

// 用户API服务
export class UserAPI {
  // 获取用户资料
  static async getProfile(): Promise<User> {
    return apiClient.get('/api/v1/users/profile');
  }

  // 更新用户资料
  static async updateProfile(updates: Partial<User>): Promise<void> {
    return apiClient.put('/api/v1/users/profile', updates);
  }

  // 获取用户偏好
  static async getPreferences(): Promise<UserPreferences> {
    return apiClient.get('/api/v1/users/preferences');
  }

  // 更新用户偏好
  static async updatePreferences(preferences: UserPreferences): Promise<void> {
    return apiClient.put('/api/v1/users/preferences', preferences);
  }

  // 获取用户统计
  static async getStats(): Promise<UserStats> {
    return apiClient.get('/api/v1/users/stats');
  }
}

// 代币API服务
export class TokenAPI {
  // 获取代币列表
  static async getTokens(params?: {
    page?: number;
    page_size?: number;
    chain_id?: number;
    search?: string;
    is_verified?: boolean;
  }): Promise<{ tokens: Token[]; meta: Meta }> {
    const response = await apiClient.getRaw<Token[]>('/api/v1/tokens', params);
    return {
      tokens: response.data || [],
      meta: response.meta || {}
    };
  }

  // 获取代币详情
  static async getToken(id: number): Promise<Token> {
    return apiClient.get(`/api/v1/tokens/${id}`);
  }

  // 搜索代币
  static async searchTokens(query: string): Promise<Token[]> {
    return apiClient.get('/api/v1/tokens/search', { q: query });
  }

  // 获取热门代币
  static async getPopularTokens(limit: number = 20): Promise<Token[]> {
    return apiClient.get('/api/v1/tokens/popular', { limit });
  }
}

// 区块链API服务
export class ChainAPI {
  // 获取支持的区块链
  static async getChains(): Promise<Chain[]> {
    return apiClient.get('/api/v1/chains');
  }

  // 获取活跃区块链
  static async getActiveChains(): Promise<Chain[]> {
    return apiClient.get('/api/v1/chains?type=active');
  }

  // 获取主网区块链
  static async getMainnetChains(): Promise<Chain[]> {
    return apiClient.get('/api/v1/chains?type=mainnet');
  }
}

// 报价API服务
export class QuoteAPI {
  // 获取最优报价
  static async getQuote(request: QuoteRequest): Promise<QuoteResponse> {
    return apiClient.post('/api/v1/quotes', request);
  }

  // 获取报价历史
  static async getQuoteHistory(params?: {
    page?: number;
    page_size?: number;
  }): Promise<{ quotes: QuoteResponse[]; meta: Meta }> {
    const response = await apiClient.getRaw<QuoteResponse[]>('/api/v1/quotes/history', params);
    return {
      quotes: response.data || [],
      meta: response.meta || {}
    };
  }

  // 获取报价详情
  static async getQuoteDetails(requestId: string): Promise<QuoteResponse> {
    return apiClient.get(`/api/v1/quotes/${requestId}`);
  }
}

// 交易API服务
export class SwapAPI {
  // 创建交易
  static async createSwap(request: SwapRequest): Promise<SwapResponse> {
    return apiClient.post('/api/v1/swaps', request);
  }

  // 获取交易状态
  static async getSwapStatus(txHash: string): Promise<Transaction> {
    return apiClient.get(`/api/v1/swaps/${txHash}`);
  }

  // 获取交易历史
  static async getTransactionHistory(params?: {
    page?: number;
    page_size?: number;
    status?: string;
  }): Promise<{ transactions: Transaction[]; meta: Meta }> {
    const response = await apiClient.getRaw<Transaction[]>('/api/v1/transactions', params);
    return {
      transactions: response.data || [],
      meta: response.meta || {}
    };
  }
}

// 系统API服务
export class SystemAPI {
  // 获取系统健康状态
  static async getHealth(): Promise<any> {
    return apiClient.get('/health');
  }

  // 获取系统指标
  static async getMetrics(): Promise<any> {
    return apiClient.get('/metrics');
  }
}

// 所有API服务已在上面定义时导出
