"use client";

import { useCallback, useEffect, useState } from "react";
import { CheckCircle2, RefreshCw, Save, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  BilibiliVerificationAccount,
  deleteBilibiliVerificationAccount,
  getBilibiliVerificationAccount,
  saveBilibiliVerificationAccount,
  testBilibiliVerificationAccount,
} from "@/lib/api";

function formatDate(value: string | null) {
  return value ? new Date(value).toLocaleString("zh-CN") : "尚未检查";
}

export function BilibiliVerificationSettings() {
  const [account, setAccount] = useState<BilibiliVerificationAccount | null>(null);
  const [uid, setUid] = useState("");
  const [cookie, setCookie] = useState("");
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [success, setSuccess] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const response = await getBilibiliVerificationAccount();
      if (response.code === 200) {
        setAccount(response.data);
        setUid(response.data?.bilibili_uid || "");
      } else {
        setMessage(response.message || "读取失败");
      }
    } catch {
      setMessage("读取失败");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const save = async () => {
    setBusy(true);
    setMessage("");
    setSuccess(false);
    try {
      const response = await saveBilibiliVerificationAccount({ bilibili_uid: uid.trim(), cookie: cookie.trim() });
      if (response.code === 200) {
        setAccount(response.data);
        setCookie("");
        setMessage("验证并保存成功");
        setSuccess(true);
      } else {
        setMessage(response.message || "保存失败");
      }
    } catch {
      setMessage("保存失败");
    } finally {
      setBusy(false);
    }
  };

  const test = async () => {
    setBusy(true);
    setMessage("");
    setSuccess(false);
    try {
      const response = await testBilibiliVerificationAccount();
      setMessage(response.code === 200 ? "Cookie 有效" : response.message || "检查失败");
      setSuccess(response.code === 200);
      await load();
    } catch {
      setMessage("检查失败");
    } finally {
      setBusy(false);
    }
  };

  const remove = async () => {
    const response = await deleteBilibiliVerificationAccount();
    setConfirmDelete(false);
    if (response.code === 200) {
      setAccount(null);
      setUid("");
      setCookie("");
      setMessage("验证账号已删除，新的注册验证已暂停");
      setSuccess(true);
    } else {
      setMessage(response.message || "删除失败");
      setSuccess(false);
    }
  };

  return (
    <>
      <Card id="bilibili-verification">
        <CardHeader>
          <CardTitle>B 站注册验证</CardTitle>
          <CardDescription>配置用于检查新关注记录的 B 站账号。Cookie 会加密存储且不会回显。</CardDescription>
        </CardHeader>
        <CardContent className="max-w-2xl space-y-5">
          <div className="space-y-2">
            <Label htmlFor="bilibili-uid">B 站 UID</Label>
            <Input id="bilibili-uid" value={uid} onChange={(event) => setUid(event.target.value)} inputMode="numeric" disabled={loading || busy} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="bilibili-cookie">Cookie</Label>
            <Textarea id="bilibili-cookie" value={cookie} onChange={(event) => setCookie(event.target.value)} rows={5} autoComplete="off" disabled={loading || busy} placeholder={account?.cookie_configured ? "已配置；留空可保留现有 Cookie" : "请输入完整 Cookie 请求头内容"} />
          </div>
          {account && (
            <div className="grid gap-2 rounded-md bg-muted p-4 text-sm sm:grid-cols-2">
              <span>最近检查：{formatDate(account.last_checked_at)}</span>
              <span>最近成功：{formatDate(account.last_successful_at)}</span>
              <span className="sm:col-span-2">状态：{account.last_error ? <span className="text-destructive">{account.last_error}</span> : <span className="inline-flex items-center gap-1 text-green-700"><CheckCircle2 className="h-4 w-4" />正常</span>}</span>
            </div>
          )}
          {message && <p className={success ? "text-sm text-green-700" : "text-sm text-destructive"}>{message}</p>}
          <div className="flex flex-wrap gap-2">
            <Button onClick={save} disabled={loading || busy || !uid.trim() || (!account && !cookie.trim())}><Save className="mr-2 h-4 w-4" />保存并验证</Button>
            <Button variant="outline" onClick={test} disabled={!account || busy}><RefreshCw className="mr-2 h-4 w-4" />测试 Cookie</Button>
            <Button variant="outline" onClick={() => setConfirmDelete(true)} disabled={!account || busy} className="text-destructive"><Trash2 className="mr-2 h-4 w-4" />删除</Button>
          </div>
        </CardContent>
      </Card>
      <Dialog open={confirmDelete} onOpenChange={setConfirmDelete}>
        <DialogContent>
          <DialogHeader><DialogTitle>删除验证账号</DialogTitle><DialogDescription>删除后新用户无法发起关注验证，已有用户仍可登录。</DialogDescription></DialogHeader>
          <DialogFooter><Button variant="outline" onClick={() => setConfirmDelete(false)}>取消</Button><Button variant="destructive" onClick={remove}>确认删除</Button></DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
