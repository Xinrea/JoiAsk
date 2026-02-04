import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: 'export',
  basePath: '/admin',
  trailingSlash: true,
};

export default nextConfig;
