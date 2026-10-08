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
import { Textarea } from "@/components/ui/textarea";
import { Checkbox } from "@/components/ui/checkbox";
import { BilibiliVerificationSettings } from "@/components/bilibili-verification-settings";
import { EmailSettings, getSettings, updateSettings } from "@/lib/api";

export default function SettingsPage() {
  const [email, setEmail] = useState<EmailSettings>({
    enabled: false,
    host: "",
    port: 587,
    username: "",
    password: "",
    password_configured: false,
    security: "starttls",
    from_email: "",
    from_name: "",
    site_url: "",
  });
  const updateEmail = <K extends keyof EmailSettings>(
    key: K,
    value: EmailSettings[K],
  ) => setEmail((current) => ({ ...current, [key]: value }));
  const [deepSeekAPIKey, setDeepSeekAPIKey] = useState("");
  const [spamPrompt, setSpamPrompt] = useState("");
  const [requireVerifiedUserToPost, setRequireVerifiedUserToPost] =
    useState(false);
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
          if (res.data.email) setEmail(res.data.email);
          setDeepSeekAPIKey(res.data.deepseek_api_key || "");
          setSpamPrompt(res.data.spam_prompt || "");
          setRequireVerifiedUserToPost(
            !!res.data.require_verified_user_to_post,
          );
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
    if (!spamPrompt.trim()) {
      setMessage("低质量提问判定标准不能为空");
      setIsSuccess(false);
      return;
    }
    setIsSaving(true);
    setMessage("");
    setIsSuccess(false);
    try {
      const res = await updateSettings({
        email,
        deepseek_api_key: deepSeekAPIKey.trim(),
        spam_prompt: spamPrompt.trim(),
        require_verified_user_to_post: requireVerifiedUserToPost,
      });
      if (res.code === 200) {
        if (res.data.email) setEmail(res.data.email);
        setDeepSeekAPIKey(res.data.deepseek_api_key || "");
        setSpamPrompt(res.data.spam_prompt || "");
        setRequireVerifiedUserToPost(!!res.data.require_verified_user_to_post);
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
          管理投稿权限、B 站注册验证与外部服务配置
        </p>
      </header>

      <Card>
        <CardHeader>
          <CardTitle>投稿权限</CardTitle>
          <CardDescription>
            控制游客是否可以直接投稿，不影响登录和注册入口。
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex max-w-2xl items-start gap-3">
            <Checkbox
              id="require-verified-user"
              checked={requireVerifiedUserToPost}
              onCheckedChange={(checked) =>
                setRequireVerifiedUserToPost(!!checked)
              }
              disabled={isLoading}
            />
            <div className="space-y-1">
              <Label htmlFor="require-verified-user">投稿需要已验证用户</Label>
              <p className="text-sm text-muted-foreground">
                开启后仅已完成 B 站验证并登录的用户可以投稿。
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      <BilibiliVerificationSettings />

      <Card>
        <CardHeader>
          <CardTitle>邮件服务（SMTP）</CardTitle>
          <CardDescription>
            开启后，已注册用户可以绑定邮箱，并通过邮件链接重置密码。注册流程保持不变。
          </CardDescription>
        </CardHeader>
        <CardContent className="max-w-3xl space-y-5">
          <div className="flex items-center gap-3">
            <Checkbox
              id="email-enabled"
              checked={email.enabled}
              onCheckedChange={(checked) => updateEmail("enabled", !!checked)}
              disabled={isLoading || isSaving}
            />
            <Label htmlFor="email-enabled">开启邮件功能</Label>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="smtp-host">SMTP 主机</Label>
              <Input
                id="smtp-host"
                value={email.host}
                onChange={(e) => updateEmail("host", e.target.value)}
                placeholder="smtp.example.com"
                disabled={isLoading || isSaving}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="smtp-port">SMTP 端口</Label>
              <Input
                id="smtp-port"
                type="number"
                min={1}
                max={65535}
                value={email.port || ""}
                onChange={(e) => updateEmail("port", Number(e.target.value))}
                disabled={isLoading || isSaving}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="smtp-security">连接加密</Label>
              <select
                id="smtp-security"
                value={email.security}
                onChange={(e) =>
                  updateEmail(
                    "security",
                    e.target.value as EmailSettings["security"],
                  )
                }
                disabled={isLoading || isSaving}
                className="h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
              >
                <option value="starttls">STARTTLS（通常为 587）</option>
                <option value="tls">TLS / SSL（通常为 465）</option>
                <option value="none">不加密（仅用于本地邮件中继）</option>
              </select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="smtp-username">SMTP 用户名</Label>
              <Input
                id="smtp-username"
                value={email.username}
                onChange={(e) => updateEmail("username", e.target.value)}
                autoComplete="off"
                placeholder="可留空以使用无需认证的中继"
                disabled={isLoading || isSaving}
              />
            </div>
            <div className="space-y-2 sm:col-span-2">
              <Label htmlFor="smtp-password">SMTP 密码 / 授权码</Label>
              <Input
                id="smtp-password"
                type="password"
                value={email.password}
                onChange={(e) => updateEmail("password", e.target.value)}
                autoComplete="new-password"
                placeholder={
                  email.password_configured
                    ? "已配置，留空保留现有密码"
                    : "请输入 SMTP 密码或授权码"
                }
                disabled={isLoading || isSaving}
              />
              <p className="text-xs text-muted-foreground">
                密码保存在数据库中，接口不会返回已保存的密码。留空表示保留原值。
              </p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="smtp-from-email">发件邮箱</Label>
              <Input
                id="smtp-from-email"
                type="email"
                value={email.from_email}
                onChange={(e) => updateEmail("from_email", e.target.value)}
                placeholder="noreply@example.com"
                disabled={isLoading || isSaving}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="smtp-from-name">发件人名称</Label>
              <Input
                id="smtp-from-name"
                value={email.from_name}
                onChange={(e) => updateEmail("from_name", e.target.value)}
                placeholder="JoiAsk 提问箱"
                disabled={isLoading || isSaving}
              />
            </div>
            <div className="space-y-2 sm:col-span-2">
              <Label htmlFor="email-site-url">网站公开地址</Label>
              <Input
                id="email-site-url"
                type="url"
                value={email.site_url}
                onChange={(e) => updateEmail("site_url", e.target.value)}
                placeholder="https://ask.example.com"
                disabled={isLoading || isSaving}
              />
              <p className="text-xs text-muted-foreground">
                填写用户访问前台的完整地址，用于生成邮件链接。链接有效期为 30
                分钟。关闭邮件功能后，相关入口隐藏，已有邮件链接也暂不可用。
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

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

          <div className="space-y-2">
            <Label htmlFor="spam-prompt">低质量提问判定标准</Label>
            <Textarea
              id="spam-prompt"
              value={spamPrompt}
              onChange={(event) => setSpamPrompt(event.target.value)}
              placeholder={"例如：\n广告或恶意引流\n重复灌水\n明显无意义的内容"}
              rows={8}
              disabled={isLoading}
              className="max-w-2xl"
            />
            <p className="text-sm text-muted-foreground">
              只需描述什么样的提问属于低质量内容，无需编写“请判断”等任务说明。
              审核逻辑、输入字段说明和 is_spam JSON 返回格式由系统自动补充。
            </p>
          </div>
        </CardContent>
      </Card>

      <div className="flex flex-wrap items-center gap-3">
        <Button onClick={handleSave} disabled={isLoading || isSaving}>
          {isLoading ? "加载中..." : isSaving ? "保存中..." : "保存设置"}
        </Button>
        {message && (
          <p className={isSuccess ? "text-[#6b7d6b]" : "text-destructive"}>
            {message}
          </p>
        )}
      </div>
    </div>
  );
}
