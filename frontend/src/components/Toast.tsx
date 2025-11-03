// Toast通知组件 - 优雅的成功/错误提示
// 使用Tailwind CSS实现居中弹出的卡片样式通知
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
}

export const Toast: React.FC<ToastProps> = ({
  type,
  title,
  message,
  details = [],
  isVisible,
  onClose,
  autoClose = true,
  duration = 5000
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
          iconColor: 'text-green-600 dark:text-green-400',
          title: 'text-green-900 dark:text-green-100',
          message: 'text-green-800 dark:text-green-200',
          button: 'text-green-500 hover:text-green-700 dark:text-green-400 dark:hover:text-green-300'
        };
      case 'error':
        return {
          container: 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-700',
          icon: '❌',
          iconBg: 'bg-red-100 dark:bg-red-800',
          iconColor: 'text-red-600 dark:text-red-400',
          title: 'text-red-900 dark:text-red-100',
          message: 'text-red-800 dark:text-red-200',
          button: 'text-red-500 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300'
        };
      case 'warning':
        return {
          container: 'bg-yellow-50 dark:bg-yellow-900/20 border-yellow-200 dark:border-yellow-700',
          icon: '⚠️',
          iconBg: 'bg-yellow-100 dark:bg-yellow-800',
          iconColor: 'text-yellow-600 dark:text-yellow-400',
          title: 'text-yellow-900 dark:text-yellow-100',
          message: 'text-yellow-800 dark:text-yellow-200',
          button: 'text-yellow-500 hover:text-yellow-700 dark:text-yellow-400 dark:hover:text-yellow-300'
        };
      case 'info':
      default:
        return {
          container: 'bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-700',
          icon: 'ℹ️',
          iconBg: 'bg-blue-100 dark:bg-blue-800',
          iconColor: 'text-blue-600 dark:text-blue-400',
          title: 'text-blue-900 dark:text-blue-100',
          message: 'text-blue-800 dark:text-blue-200',
          button: 'text-blue-500 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300'
        };
    }
  };

  const styles = getTypeStyles();

  if (!isVisible) return null;

  return (
    <>
      {/* 无遮罩，直接固定定位的Toast */}
      <div className={`
        fixed top-4 left-1/2 transform -translate-x-1/2 z-50 max-w-md w-full mx-4
        ${styles.container}
        border rounded-xl shadow-2xl
        transition-all duration-300 ease-out
        ${isVisible ? 'translate-y-0 opacity-100' : '-translate-y-full opacity-0'}
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
          <div className="absolute bottom-0 left-0 right-0 h-1 bg-gray-200 dark:bg-gray-700 rounded-b-xl overflow-hidden">
            <div 
              className={`h-full transition-all ease-linear ${
                type === 'success' ? 'bg-green-500' :
                type === 'error' ? 'bg-red-500' :
                type === 'warning' ? 'bg-yellow-500' : 'bg-blue-500'
              }`}
              style={{
                width: '100%',
                animation: `shrink ${duration}ms linear forwards`
              }}
            />
          </div>
        )}
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
