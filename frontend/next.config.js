/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  // 配置反向代理，将 /api 请求转发到后端
  async rewrites() {
    const backendUrl = process.env.BACKEND_URL || 'http://localhost:8080'
    return [
      {
        source: '/api/:path*',
        destination: `${backendUrl}/api/:path*`,
      },
    ]
  },
}

module.exports = nextConfig

