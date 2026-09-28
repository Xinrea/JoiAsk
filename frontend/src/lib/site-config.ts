import { cache } from 'react';
import { DEFAULT_CUSTOM_CSS } from '@joiask/default-custom-css';

export interface SiteConfig {
  site_name: string;
  site_description: string;
  logo_url: string;
  favicon_url: string;
  custom_css: string;
}

export const DEFAULT_SITE_CONFIG: SiteConfig = {
  site_name: 'JoiAsk 提问箱',
  site_description: 'JoiAsk 提问箱',
  logo_url: '/favicon.png',
  favicon_url: '/favicon.png',
  custom_css: DEFAULT_CUSTOM_CSS,
};

const CONFIG_URL = process.env.SITE_CONFIG_API_URL || 'http://127.0.0.1:8080/api/config';

export const getSiteConfig = cache(async (): Promise<SiteConfig> => {
  try {
    const response = await fetch(CONFIG_URL, { cache: 'no-store' });
    if (!response.ok) return DEFAULT_SITE_CONFIG;

    const result = await response.json() as {
      code: number;
      data?: Partial<SiteConfig>;
    };
    if (result.code !== 200 || !result.data) return DEFAULT_SITE_CONFIG;

    return {
      site_name: result.data.site_name?.trim() || DEFAULT_SITE_CONFIG.site_name,
      site_description: result.data.site_description?.trim() || DEFAULT_SITE_CONFIG.site_description,
      logo_url: result.data.logo_url?.trim() || DEFAULT_SITE_CONFIG.logo_url,
      favicon_url: result.data.favicon_url?.trim() || DEFAULT_SITE_CONFIG.favicon_url,
      custom_css: result.data.custom_css?.trim() || DEFAULT_SITE_CONFIG.custom_css,
    };
  } catch {
    return DEFAULT_SITE_CONFIG;
  }
});

