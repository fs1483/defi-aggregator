// 全局类型声明
// 扩展window对象以支持Web3相关属性

interface Window {
  ethereum?: {
    isMetaMask?: boolean;
    request?: (args: { method: string; params?: any[] }) => Promise<any>;
    on?: (event: string, handler: (data: any) => void) => void;
    removeListener?: (event: string, handler: (data: any) => void) => void;
    selectedAddress?: string;
    chainId?: string;
    networkVersion?: string;
  };
}

// 声明vite环境变量类型
interface ImportMetaEnv {
  readonly VITE_API_URL?: string;
  readonly VITE_APP_ENV?: string;
  readonly VITE_DEFAULT_CHAIN_ID?: string;
  readonly VITE_DEFAULT_SLIPPAGE?: string;
  readonly VITE_SUPPORTED_CHAIN_IDS?: string;
  readonly VITE_RPC_URL_MAINNET?: string;
  readonly VITE_RPC_URL_POLYGON?: string;
  readonly VITE_WALLETCONNECT_PROJECT_ID?: string;
  readonly VITE_MAINNET_DEFAULT_FROM_TOKEN?: string;
  readonly VITE_MAINNET_DEFAULT_TO_TOKEN?: string;
  readonly VITE_TESTNET_DEFAULT_FROM_TOKEN?: string;
  readonly VITE_TESTNET_DEFAULT_TO_TOKEN?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
