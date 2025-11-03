// Toast通知组件的不同遮罩变体
// 提供多种遮罩效果供选择

import React, { useEffect } from 'react';

export interface ToastProps {
  type: 'success' | 'error' | 'info' | 'warning';
  title: string;
  message?: string;
  details?: string[];
  isVisible: boolean;
  onClose: () => void;
  autoClose?: boolean;
  duration?: number;
  variant?: 'no-overlay' | 'light-overlay' | 'blur-overlay' | 'dark-overlay';
}

export const ToastVariants: React.FC<ToastProps> = ({
  type,
  title,
  message,
  details = [],
  isVisible,
  onClose,
  autoClose = true,
  duration = 5000,
  variant = 'no-overlay'
}) => {
  // 自动关闭逻辑
  useEffect(() => {
    if (isVisible && autoClose) {
      const timer = setTimeout(onClose, duration);
      return () => clearTimeout(timer);
    }
  }, [isVisible, autoClose, duration, onClose]);

  // 根据类型设置样式
  const getTypeStyles = () => {
    switch (type) {
      case 'success':
        return {
          container: 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-700',
          icon: '🎉',
          iconBg: 'bg-green-100 dark:bg-green-800',
          title: 'text-green-900 dark:text-green-100',
          message: 'text-green-800 dark:text-green-200',
          button: 'text-green-500 hover:text-green-700 dark:text-green-400 dark:hover:text-green-300',
          progressBar: 'bg-green-500'
        };
      case 'error':
        return {
          container: 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-700',
          icon: '❌',
          iconBg: 'bg-red-100 dark:bg-red-800',
          title: 'text-red-900 dark:text-red-100',
          message: 'text-red-800 dark:text-red-200',
          button: 'text-red-500 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300',
          progressBar: 'bg-red-500'
        };
      case 'warning':
        return {
          container: 'bg-yellow-50 dark:bg-yellow-900/20 border-yellow-200 dark:border-yellow-700',
          icon: '⚠️',
          iconBg: 'bg-yellow-100 dark:bg-yellow-800',
          title: 'text-yellow-900 dark:text-yellow-100',
          message: 'text-yellow-800 dark:text-yellow-200',
          button: 'text-yellow-500 hover:text-yellow-700 dark:text-yellow-400 dark:hover:text-yellow-300',
          progressBar: 'bg-yellow-500'
        };
      case 'info':
      default:
        return {
          container: 'bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-700',
          icon: 'ℹ️',
          iconBg: 'bg-blue-100 dark:bg-blue-800',
          title: 'text-blue-900 dark:text-blue-100',
          message: 'text-blue-800 dark:text-blue-200',
          button: 'text-blue-500 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300',
          progressBar: 'bg-blue-500'
        };
    }
  };

  // 根据variant设置遮罩样式
  const getOverlayStyles = () => {
    switch (variant) {
      case 'no-overlay':
        return {
          container: 'fixed top-4 left-1/2 transform -translate-x-1/2 z-50',
          animation: isVisible ? 'translate-y-0 opacity-100' : '-translate-y-full opacity-0'
        };
      case 'light-overlay':
        return {
          container: 'fixed inset-0 bg-white bg-opacity-30 z-50 flex items-start justify-center pt-16',
          animation: isVisible ? 'scale-100 opacity-100' : 'scale-95 opacity-0'
        };
      case 'blur-overlay':
        return {
          container: 'fixed inset-0 bg-gray-500 bg-opacity-20 backdrop-blur-sm z-50 flex items-start justify-center pt-16',
          animation: isVisible ? 'scale-100 opacity-100' : 'scale-95 opacity-0'
        };
      case 'dark-overlay':
        return {
          container: 'fixed inset-0 bg-black bg-opacity-40 z-50 flex items-center justify-center',
          animation: isVisible ? 'scale-100 opacity-100' : 'scale-95 opacity-0'
        };
      default:
        return {
          container: 'fixed top-4 left-1/2 transform -translate-x-1/2 z-50',
          animation: isVisible ? 'translate-y-0 opacity-100' : '-translate-y-full opacity-0'
        };
    }
  };

  const styles = getTypeStyles();
  const overlayStyles = getOverlayStyles();

  if (!isVisible) return null;

  return (
    <>
      {/* 动态遮罩层 */}
      <div className={overlayStyles.container}>
        {/* Toast卡片 */}
        <div className={`
          relative max-w-md w-full mx-4
          ${styles.container}
          border rounded-xl shadow-2xl
          transform transition-all duration-300 ease-out
          ${overlayStyles.animation}
        `}>
          {/* 头部 */}
          <div className="flex items-start p-6">
            {/* 图标 */}
            <div className={`
              flex-shrink-0 w-10 h-10 rounded-full flex items-center justify-center
              ${styles.iconBg}
            `}>
              <span className="text-lg">{styles.icon}</span>
            </div>

            {/* 内容 */}
            <div className="ml-4 flex-1">
              {/* 标题 */}
              <h3 className={`text-lg font-semibold ${styles.title}`}>
                {title}
              </h3>

              {/* 消息 */}
              {message && (
                <p className={`mt-2 text-sm ${styles.message}`}>
                  {message}
                </p>
              )}

              {/* 详细信息 */}
              {details.length > 0 && (
                <ul className={`mt-3 space-y-1 text-sm ${styles.message}`}>
                  {details.map((detail, index) => (
                    <li key={index} className="flex items-center">
                      <span className="w-1.5 h-1.5 bg-current rounded-full mr-2 opacity-60"></span>
                      {detail}
                    </li>
                  ))}
                </ul>
              )}
            </div>

            {/* 关闭按钮 */}
            <button
              onClick={onClose}
              className={`
                flex-shrink-0 ml-4 p-1 rounded-full hover:bg-black hover:bg-opacity-10
                transition-colors duration-200 ${styles.button}
              `}
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          {/* 进度条（自动关闭时显示） */}
          {autoClose && (
            <div className="absolute bottom-0 left-0 right-0 h-1 bg-black bg-opacity-10 rounded-b-xl overflow-hidden">
              <div 
                className={`h-full ${styles.progressBar} transition-all ease-linear`}
                style={{
                  width: '100%',
                  animation: `shrink ${duration}ms linear forwards`
                }}
              />
            </div>
          )}
        </div>
      </div>

      {/* CSS动画 */}
      <style jsx>{`
        @keyframes shrink {
          from { width: 100%; }
          to { width: 0%; }
        }
      `}</style>
    </>
  );
};
