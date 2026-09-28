"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { CSSEditor } from "@/components/css-editor";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { getConfig, updateConfig, uploadConfigAsset } from "@/lib/api";
import { DEFAULT_CUSTOM_CSS } from "@joiask/default-custom-css";
import {
  applyColorPreset,
  CUSTOM_CSS_PRESETS,
} from "@joiask/custom-css-presets";

export default function CustomPage() {
  const [siteName, setSiteName] = useState("");
  const [siteDescription, setSiteDescription] = useState("");
  const [logoUrl, setLogoUrl] = useState("");
  const [faviconUrl, setFaviconUrl] = useState("");
  const [customCSS, setCustomCSS] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [message, setMessage] = useState("");
  const [uploading, setUploading] = useState<"logo" | "favicon" | null>(null);
  const [selectedPreset, setSelectedPreset] = useState("");

  useEffect(() => {
    getConfig().then((res) => {
      if (res.code === 200) {
        setSiteName(res.data.site_name || "");
        setSiteDescription(res.data.site_description || "");
        setLogoUrl(res.data.logo_url || "");
        setFaviconUrl(res.data.favicon_url || "");
        setCustomCSS(res.data.custom_css || DEFAULT_CUSTOM_CSS);
      }
    }).catch(() => {
      setMessage("配置加载失败");
    });
  }, []);

  const handleSave = async () => {
    setIsLoading(true);
    setMessage("");
    try {
      const res = await updateConfig({
        site_name: siteName,
        site_description: siteDescription,
        logo_url: logoUrl,
        favicon_url: faviconUrl,
        custom_css: customCSS,
      });
      setMessage(res.code === 200 ? "保存成功" : res.message || "保存失败");
    } catch {
      setMessage("保存失败");
    } finally {
      setIsLoading(false);
    }
  };

  const handleReset = async () => {
    if (!window.confirm("确定恢复默认样式吗？当前自定义 CSS 将被清空。")) {
      return;
    }
    setIsLoading(true);
    setMessage("");
    try {
      const res = await updateConfig({ custom_css: "" });
      if (res.code === 200) {
        setCustomCSS(DEFAULT_CUSTOM_CSS);
        setMessage("已恢复默认样式");
      } else {
        setMessage(res.message || "恢复失败");
      }
    } catch {
      setMessage("恢复失败");
    } finally {
      setIsLoading(false);
    }
  };

  const handleAssetUpload = async (
    type: "logo" | "favicon",
    file: File | undefined,
  ) => {
    if (!file) return;
    setUploading(type);
    setMessage("");
    try {
      const res = await uploadConfigAsset(file);
      if (res.code !== 200) {
        setMessage(res.message || "图片上传失败");
      } else if (type === "logo") {
        setLogoUrl(res.data.url);
      } else {
        setFaviconUrl(res.data.url);
      }
    } catch {
      setMessage("图片上传失败");
    } finally {
      setUploading(null);
    }
  };

  const handlePresetChange = (presetId: string) => {
    const preset = CUSTOM_CSS_PRESETS.find((item) => item.id === presetId);
    if (!preset) return;
    setCustomCSS(applyColorPreset(customCSS || DEFAULT_CUSTOM_CSS, preset));
    setSelectedPreset(preset.id);
    setMessage(`${preset.label}色预设已应用，请点击保存`);
  };

  return (
    <div className="space-y-8">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight text-foreground">
          自定义
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          自定义提问箱的站点信息和品牌资源
        </p>
      </header>

      <Card>
        <CardHeader>
          <CardTitle>站点外观</CardTitle>
          <CardDescription>
            这些设置会用于浏览器标题、搜索引擎描述以及前台导航栏图标
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="site-name">站点名称</Label>
            <Input
              id="site-name"
              value={siteName}
              onChange={(event) => setSiteName(event.target.value)}
              placeholder="JoiAsk 提问箱"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="site-description">站点描述</Label>
            <Input
              id="site-description"
              value={siteDescription}
              onChange={(event) => setSiteDescription(event.target.value)}
              placeholder="JoiAsk 提问箱"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="logo-url">Logo 地址</Label>
            <div className="flex min-w-0 items-center gap-2">
              <Input
                id="logo-url"
                type="url"
                className="min-w-0 flex-1"
                value={logoUrl}
                onChange={(event) => setLogoUrl(event.target.value)}
                placeholder="/favicon.png"
              />
              <Input
                id="logo-file"
                type="file"
                className="w-auto max-w-[45%] shrink-0"
                accept="image/png,image/jpeg,image/gif,image/webp,image/svg+xml"
                disabled={uploading !== null}
                onChange={(event) => {
                  void handleAssetUpload("logo", event.target.files?.[0]);
                  event.currentTarget.value = "";
                }}
              />
            </div>
            {uploading === "logo" && (
              <p className="text-sm text-muted-foreground">Logo 上传中...</p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor="favicon-url">Favicon 地址</Label>
            <div className="flex min-w-0 items-center gap-2">
              <Input
                id="favicon-url"
                type="url"
                className="min-w-0 flex-1"
                value={faviconUrl}
                onChange={(event) => setFaviconUrl(event.target.value)}
                placeholder="/favicon.png"
              />
              <Input
                id="favicon-file"
                type="file"
                className="w-auto max-w-[45%] shrink-0"
                accept="image/png,image/jpeg,image/gif,image/webp,image/svg+xml"
                disabled={uploading !== null}
                onChange={(event) => {
                  void handleAssetUpload("favicon", event.target.files?.[0]);
                  event.currentTarget.value = "";
                }}
              />
            </div>
            {uploading === "favicon" && (
              <p className="text-sm text-muted-foreground">Favicon 上传中...</p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor="custom-css">自定义 CSS</Label>
            <div className="space-y-2 rounded-lg border border-border bg-muted/30 p-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div>
                  <p className="text-sm font-medium text-foreground">颜色预设</p>
                  <p className="text-xs text-muted-foreground">
                    只替换主题颜色，不会覆盖其他 CSS 修改
                  </p>
                </div>
                <select
                  aria-label="选择颜色预设"
                  value={selectedPreset}
                  onChange={(event) => handlePresetChange(event.target.value)}
                  className="h-9 rounded-lg border border-input bg-card px-3 text-sm text-foreground outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  <option value="">选择预设...</option>
                  {CUSTOM_CSS_PRESETS.map((preset) => (
                    <option key={preset.id} value={preset.id}>
                      {preset.label} · {preset.description}
                    </option>
                  ))}
                </select>
              </div>
              <div className="flex flex-wrap gap-2">
                {CUSTOM_CSS_PRESETS.map((preset) => (
                  <button
                    key={preset.id}
                    type="button"
                    title={`${preset.label} · ${preset.description}`}
                    aria-label={`应用${preset.label}色预设`}
                    onClick={() => handlePresetChange(preset.id)}
                    className="flex h-8 w-8 items-center justify-center rounded-full border-2 border-card shadow-sm transition-transform hover:scale-110 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    style={{ backgroundColor: preset.swatch }}
                  >
                    <span className="sr-only">{preset.label}</span>
                  </button>
                ))}
              </div>
            </div>
            <CSSEditor value={customCSS} onChange={setCustomCSS} disabled={isLoading} />
            <p className="text-sm text-muted-foreground">
              CSS 会完整替换前台默认主题。不要包含 &lt;style&gt; 标签，长度不能超过
              128 KiB。
            </p>
          </div>
          {message && (
            <div className={message === "保存成功" ? "text-[#6b7d6b]" : "text-destructive"}>
              {message}
            </div>
          )}
          <div className="flex flex-wrap gap-2">
            <Button onClick={handleSave} disabled={isLoading}>
              {isLoading ? "保存中..." : "保存"}
            </Button>
            <Button
              type="button"
              variant="outline"
              onClick={handleReset}
              disabled={isLoading}
            >
              恢复默认
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
