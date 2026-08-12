"use client";

import { useEffect, useState } from "react";
import { Eye, EyeOff } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { getSettings, updateSettings } from "@/lib/api";

export default function SettingsPage() {
  const [deepSeekAPIKey, setDeepSeekAPIKey] = useState("");
  const [showAPIKey, setShowAPIKey] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [message, setMessage] = useState("");
  const [isSuccess, setIsSuccess] = useState(false);

  useEffect(() => {
    const loadSettings = async () => {
      try {
        const res = await getSettings();
        if (res.code === 200) {
          setDeepSeekAPIKey(res.data.deepseek_api_key || "");
        } else {
          setMessage(res.message || "加载失败");
        }
      } catch {
        setMessage("加载失败");
      } finally {
        setIsLoading(false);
      }
    };

    loadSettings();
  }, []);

  const handleSave = async () => {
    setIsSaving(true);
    setMessage("");
    setIsSuccess(false);
    try {
      const res = await updateSettings({
        deepseek_api_key: deepSeekAPIKey.trim(),
      });
      if (res.code === 200) {
        setDeepSeekAPIKey(res.data.deepseek_api_key || "");
        setMessage("保存成功");
        setIsSuccess(true);
      } else {
        setMessage(res.message || "保存失败");
      }
    } catch {
      setMessage("保存失败");
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="space-y-8">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight text-foreground">
          设置
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          管理服务所需的密钥与基础配置
        </p>
      </header>

      <Card>
        <CardHeader>
          <CardTitle>DeepSeek</CardTitle>
          <CardDescription>
            配置调用 DeepSeek 服务时使用的 API Key
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="deepseek-api-key">DeepSeek API Key</Label>
            <div className="relative max-w-2xl">
              <Input
                id="deepseek-api-key"
                type={showAPIKey ? "text" : "password"}
                value={deepSeekAPIKey}
                onChange={(event) => setDeepSeekAPIKey(event.target.value)}
                placeholder="请输入 DeepSeek API Key"
                autoComplete="off"
                disabled={isLoading}
                className="pr-10"
              />
              <button
                type="button"
                onClick={() => setShowAPIKey((value) => !value)}
                disabled={isLoading}
                className="absolute inset-y-0 right-0 flex w-10 items-center justify-center text-muted-foreground transition-colors hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50"
                aria-label={showAPIKey ? "隐藏 API Key" : "显示 API Key"}
              >
                {showAPIKey ? (
                  <EyeOff className="h-4 w-4" />
                ) : (
                  <Eye className="h-4 w-4" />
                )}
              </button>
            </div>
            <p className="text-sm text-muted-foreground">
              密钥将保存在数据库中，仅登录后的管理员可以读取或修改。
            </p>
          </div>

          {message && (
            <p className={isSuccess ? "text-[#6b7d6b]" : "text-destructive"}>
              {message}
            </p>
          )}

          <Button onClick={handleSave} disabled={isLoading || isSaving}>
            {isLoading ? "加载中..." : isSaving ? "保存中..." : "保存"}
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
