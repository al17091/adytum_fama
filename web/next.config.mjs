/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'export',
  basePath: '/guardinfo',
  trailingSlash: true,
  images: {
    unoptimized: true,
  },
};

export default nextConfig;
