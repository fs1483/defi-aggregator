/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html", 
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // DeFi主题色彩 (Tailwind v4 支持嵌套变量)
        primary: 'oklch(0.65 0.15 262)', // Blue-600 equivalent
        'primary-50': 'oklch(0.98 0.02 262)',
        'primary-100': 'oklch(0.94 0.05 262)',
        'primary-500': 'oklch(0.65 0.15 262)',
        'primary-600': 'oklch(0.60 0.17 262)',
        'primary-700': 'oklch(0.55 0.19 262)',
        
        success: 'oklch(0.70 0.17 142)', // Green-500 equivalent
        'success-50': 'oklch(0.98 0.02 142)',
        'success-600': 'oklch(0.65 0.19 142)',
        
        warning: 'oklch(0.75 0.15 65)', // Orange-500 equivalent
        'warning-50': 'oklch(0.98 0.02 65)',
        'warning-600': 'oklch(0.70 0.17 65)',
        
        error: 'oklch(0.65 0.20 25)', // Red-500 equivalent
        'error-50': 'oklch(0.98 0.02 25)',
        'error-600': 'oklch(0.60 0.22 25)',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
      animation: {
        'spin-slow': 'spin 3s linear infinite',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'bounce-slow': 'bounce 2s infinite',
      },
      boxShadow: {
        'glow': '0 0 20px oklch(0.65 0.15 262 / 0.5)',
        'glow-green': '0 0 20px oklch(0.70 0.17 142 / 0.5)',
        'glow-red': '0 0 20px oklch(0.65 0.20 25 / 0.5)',
      },
      // Tailwind v4 特有的自定义属性
      screens: {
        'xs': '475px',
      },
    },
  },
  plugins: [
    // Tailwind v4 的插件语法
    '@tailwindcss/typography',
  ],
}

