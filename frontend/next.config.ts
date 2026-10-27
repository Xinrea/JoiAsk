import type { NextConfig } from "next";
import path from "node:path";

// Match Caddy's upload policy even when Next is accessed without the proxy.
const uploadCSP = "sandbox; default-src 'none'; img-src data:; style-src 'unsafe-inline'";
const uploadImageTypes = [
  ['[pP][nN][gG]', 'image/png'],
  ['[jJ][pP][eE]?[gG]', 'image/jpeg'],
  ['[gG][iI][fF]', 'image/gif'],
  ['[wW][eE][bB][pP]', 'image/webp'],
  ['[sS][vV][gG]', 'image/svg+xml'],
];

const nextConfig: NextConfig = {
  output: 'standalone',
  async headers() {
    return [
      {
        source: '/upload-img/:path*',
        headers: [
          { key: 'Content-Security-Policy', value: uploadCSP },
          { key: 'X-Content-Type-Options', value: 'nosniff' },
          { key: 'Content-Type', value: 'application/octet-stream' },
          { key: 'Content-Disposition', value: 'attachment' },
        ],
      },
      // Later matches override only MIME/disposition, never the sandbox.
      ...uploadImageTypes.map(([extension, contentType]) => ({
        source: `/upload-img/:path(.*\\.${extension})`,
        headers: [
          { key: 'Content-Type', value: contentType },
          { key: 'Content-Disposition', value: 'inline' },
        ],
      })),
    ];
  },
  trailingSlash: true,
  turbopack: {
    root: path.resolve(__dirname, '..'),
  },
};

export default nextConfig;
