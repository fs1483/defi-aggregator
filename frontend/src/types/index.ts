// DeFi聚合器前端类型定义
// 定义与后端API对应的TypeScript类型，确保类型安全
// 包含用户、代币、报价、交易等所有业务实体类型

// ========================================
// 通用类型定义
// ========================================

export interface APIResponse<T = any> {
  success: boolean;
  data?: T;
  error?: APIError;
  meta?: Meta;
  message?: string;
  timestamp: number;
  request_id: string;
}

export interface APIError {
  code: string;
  message: string;
  details?: Record<string, any>;
}

export interface Meta {
  page?: number;
  page_size?: number;
  total?: number;
  total_pages?: number;
}

// ========================================
// 用户相关类型
// ========================================

export interface User {
  id: number;
  wallet_address: string;
  username?: string;
  email?: string;
  avatar_url?: string;
  preferred_language: string;
  timezone: string;
  is_active: boolean;
  created_at: string;
  last_login_at?: string;
}

export interface UserPreferences {
  default_slippage: string;
  preferred_gas_speed: 'slow' | 'standard' | 'fast';
  auto_approve_tokens: boolean;
  show_test_tokens: boolean;
  notification_email: boolean;
  notification_browser: boolean;
  privacy_analytics: boolean;
}

export interface UserStats {
  total_transactions: number;
  successful_transactions: number;
  total_volume_usd: string;
  total_gas_fee_usd: string;
  avg_price_impact: string;
  last_transaction_at?: string;
}

export interface LoginRequest {
  wallet_address: string;
  signature: string;
  message: string;
  nonce: string;
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user: User;
}

// ========================================
// 代币相关类型
// ========================================

export interface Token {
  id: number;
  chain_id: number;
  contract_address: string;
  symbol: string;
  name: string;
  decimals: number;
  logo_url?: string;
  is_native: boolean;
  is_stable: boolean;
  is_verified: boolean;
  is_active: boolean;
  price_usd?: string;
  daily_volume_usd?: string;
  market_cap_usd?: string;
  price_updated_at?: string;
}

export interface Chain {
  id: number;
  chain_id: number;
  name: string;
  display_name: string;
  symbol: string;
  is_testnet: boolean;
  is_active: boolean;
  gas_price_gwei: number;
  block_time_sec: number;
}

// ========================================
// 报价相关类型
// ========================================

export interface QuoteRequest {
  from_token_id: number;
  to_token_id: number;
  amount_in: string;
  chain_id: number;
  slippage: string;
  user_address?: string;
}

export interface QuoteResponse {
  request_id: string;
  from_token: Token;
  to_token: Token;
  amount_in: string;
  amount_out: string;
  best_aggregator: string;
  gas_estimate: number;
  price_impact: string;
  exchange_rate: string;
  route?: RouteStep[];
  valid_until: string;
  total_duration_ms: number;
  cache_hit: boolean;
}

export interface RouteStep {
  protocol: string;
  percentage: string;
}

// ========================================
// 交易相关类型
// ========================================

export interface SwapRequest {
  request_id: string;
  user_address: string;
  slippage: string;
  deadline?: string;
}

export interface SwapResponse {
  transaction_id: number;
  to: string;
  data: string;
  value: string;
  gas_limit: string;
  gas_price: string;
  nonce?: number;
}

export interface Transaction {
  id: number;
  tx_hash?: string;
  user_id: number;
  chain_id: number;
  from_token: Token;
  to_token: Token;
  aggregator: string;
  amount_in: string;
  amount_out_expected: string;
  amount_out_actual?: string;
  slippage_set: string;
  slippage_actual?: string;
  gas_limit?: number;
  gas_used?: number;
  gas_price?: string;
  gas_fee_eth?: string;
  gas_fee_usd?: string;
  price_impact: string;
  status: 'pending' | 'confirmed' | 'failed' | 'cancelled';
  user_address: string;
  block_number?: number;
  block_timestamp?: string;
  created_at: string;
  confirmed_at?: string;
}

// ========================================
// UI状态类型
// ========================================

export interface SwapState {
  fromToken: Token | null;
  toToken: Token | null;
  fromAmount: string;
  toAmount: string;
  slippage: string;
  isLoading: boolean;
  quote: QuoteResponse | null;
  error: string | null;
  lastQuoteTime: number | null;
}

export interface WalletState {
  isConnected: boolean;
  address: string | null;
  chainId: number | null;
  balance: string | null;
  isConnecting: boolean;
  error: string | null;
}

export interface AppState {
  user: User | null;
  isAuthenticated: boolean;
  tokens: Token[];
  chains: Chain[];
  selectedChain: Chain | null;
  theme: 'light' | 'dark';
  isLoading: boolean;
}

// ========================================
// 钱包相关类型
// ========================================

export interface WalletConnector {
  id: string;
  name: string;
  icon: string;
  ready: boolean;
  installed: boolean;
}

export interface NetworkConfig {
  chainId: number;
  chainName: string;
  nativeCurrency: {
    name: string;
    symbol: string;
    decimals: number;
  };
  rpcUrls: string[];
  blockExplorerUrls: string[];
}

// ========================================
// 表单类型
// ========================================

export interface TokenSearchFilters {
  search?: string;
  chain_id?: number;
  is_verified?: boolean;
  is_active?: boolean;
  page?: number;
  page_size?: number;
}

export interface TransactionFilters {
  status?: Transaction['status'];
  chain_id?: number;
  from_date?: string;
  to_date?: string;
  page?: number;
  page_size?: number;
}

// ========================================
// 工具类型
// ========================================

export type LoadingState = 'idle' | 'loading' | 'success' | 'error';

export interface PriceImpactLevel {
  level: 'low' | 'medium' | 'high';
  color: string;
  description: string;
}

export interface GasSpeed {
  speed: 'slow' | 'standard' | 'fast';
  price: string;
  time: string;
  description: string;
}

// ========================================
// 常量类型
// ========================================

export enum SUPPORTED_CHAINS {
  ETHEREUM = 1,
  POLYGON = 137,
  ARBITRUM = 42161,
  OPTIMISM = 10,
}

export enum GAS_SPEEDS {
  SLOW = 'slow',
  STANDARD = 'standard', 
  FAST = 'fast',
}

export enum TRANSACTION_STATUS {
  PENDING = 'pending',
  CONFIRMED = 'confirmed',
  FAILED = 'failed',
  CANCELLED = 'cancelled',
}

// ========================================
// 错误类型
// ========================================

export class DeFiError extends Error {
  constructor(
    public code: string,
    public message: string,
    public details?: Record<string, any>
  ) {
    super(message);
    this.name = 'DeFiError';
  }
}

export class WalletError extends Error {
  constructor(
    public code: string,
    public message: string,
    public originalError?: Error
  ) {
    super(message);
    this.name = 'WalletError';
  }
}

export class APIError extends Error {
  constructor(
    public code: string,
    public message: string,
    public statusCode?: number,
    public details?: Record<string, any>
  ) {
    super(message);
    this.name = 'APIError';
  }
}
