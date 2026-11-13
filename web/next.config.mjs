import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'export',
  basePath: '/guardinfo',
  trailingSlash: true,
  images: {
    unoptimized: true,
  },
  webpack: (config, { webpack }) => {
    config.plugins.push(
      new webpack.NormalModuleReplacementPlugin(
        /app-router-announcer/,
        path.resolve(__dirname, 'lib/app-router-announcer.js')
      )
    );
    return config;
  },
};

export default nextConfig;
