const API_BASE = '/api';

export interface User {
  id: number;
  username: string;
  created_at: string;
  updated_at: string;
}

export interface Tag {
  id: number;
  tag_name: string;
  description: string;
  question_count: number;
  created_at: string;
  updated_at: string;
}

export interface Question {
  id: number;
  tag_id: number;
  tag: Tag;
  content: string;
  images: string;
  images_num: number;
  is_hide: boolean;
  is_rainbow: boolean;
  is_archive: boolean;
  is_publish: boolean;
  is_spam: boolean;
  is_real_name: boolean;
  bilibili_uid?: string;
  bilibili_name?: string;
  bilibili_avatar?: string;
  emojis: string;
  likes: number;
  created_at: string;
  updated_at: string;
}

export interface Config {
  announcement: string;
  require_verified_user_to_post: boolean;
}

export interface Settings {
  deepseek_api_key: string;
  spam_prompt: string;
  require_verified_user_to_post: boolean;
}

export interface Member {
  username: string;
  bilibili_uid: string;
  bilibili_name: string;
  bilibili_avatar: string;
  verified_at: string;
  is_disabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface MembersResponse {
  users: Member[];
  total: number;
  page: number;
  page_size: number;
}

export interface BilibiliVerificationAccount {
  id: number;
  bilibili_uid: string;
  cookie_configured: boolean;
  last_checked_at: string | null;
  last_successful_at: string | null;
  last_error: string;
  created_at: string;
  updated_at: string;
}

export interface Statistics {
  total_questions: number;
  total_tags: number;
  total_users: number;
  total_members: number;
  total_images: number;
  rainbow_questions: number;
  archived_questions: number;
  published_questions: number;
  tag_stats: { tag: Tag; count: number }[];
}

export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

export interface QuestionsResponse {
  questions: Question[];
  page: number;
  page_size: number;
  total: number;
}

export async function login(username: string, password: string): Promise<ApiResponse<User>> {
  const res = await fetch(`${API_BASE}/login`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  });
  return res.json();
}

export async function logout(): Promise<ApiResponse<null>> {
  const res = await fetch(`${API_BASE}/logout`, {
    method: 'GET',
    credentials: 'include',
  });
  return res.json();
}

export async function getInfo(): Promise<ApiResponse<User>> {
  const res = await fetch(`${API_BASE}/info`, {
    method: 'GET',
    credentials: 'include',
  });
  return res.json();
}

export async function getConfig(): Promise<ApiResponse<Config>> {
  const res = await fetch(`${API_BASE}/config`, {
    method: 'GET',
    credentials: 'include',
  });
  return res.json();
}

export async function updateConfig(config: Partial<Config>): Promise<ApiResponse<null>> {
  const res = await fetch(`${API_BASE}/config`, {
    method: 'PUT',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config),
  });
  return res.json();
}

export async function getSettings(): Promise<ApiResponse<Settings>> {
  const res = await fetch(`${API_BASE}/settings`, {
    method: 'GET',
    credentials: 'include',
  });
  return res.json();
}

export async function updateSettings(settings: Settings): Promise<ApiResponse<Settings>> {
  const res = await fetch(`${API_BASE}/settings`, {
    method: 'PUT',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(settings),
  });
  return res.json();
}

export async function getQuestions(params: {
  page?: number;
  page_size?: number;
  order_by?: string;
  order?: string;
  tag_id?: number;
  is_spam?: boolean;
}): Promise<ApiResponse<QuestionsResponse>> {
  const searchParams = new URLSearchParams();
  if (params.page) searchParams.set('page', params.page.toString());
  if (params.page_size) searchParams.set('page_size', params.page_size.toString());
  if (params.order_by) searchParams.set('order_by', params.order_by);
  if (params.order) searchParams.set('order', params.order);
  if (params.tag_id) searchParams.set('tag_id', params.tag_id.toString());
  if (params.is_spam !== undefined) searchParams.set('is_spam', params.is_spam.toString());

  const res = await fetch(`${API_BASE}/question?${searchParams.toString()}`, {
    method: 'GET',
    credentials: 'include',
  });
  return res.json();
}

export async function updateQuestion(id: number, data: {
  tag_id?: number;
  is_hide?: boolean;
  is_rainbow?: boolean;
  is_archive?: boolean;
  is_publish?: boolean;
  is_spam?: boolean;
}): Promise<ApiResponse<null>> {
  const res = await fetch(`${API_BASE}/question/${id}`, {
    method: 'PUT',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  return res.json();
}

export async function deleteQuestion(id: number): Promise<ApiResponse<null>> {
  const res = await fetch(`${API_BASE}/question/${id}`, {
    method: 'DELETE',
    credentials: 'include',
  });
  return res.json();
}

export async function getTags(): Promise<ApiResponse<Tag[]>> {
  const res = await fetch(`${API_BASE}/tag`, {
    method: 'GET',
    credentials: 'include',
  });
  return res.json();
}

export async function createTag(data: { tag_name: string; description: string }): Promise<ApiResponse<Tag>> {
  const res = await fetch(`${API_BASE}/tag`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  return res.json();
}

export async function updateTag(id: number, data: { tag_name: string; description: string }): Promise<ApiResponse<null>> {
  const res = await fetch(`${API_BASE}/tag/${id}`, {
    method: 'PUT',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  return res.json();
}

export async function deleteTag(id: number): Promise<ApiResponse<null>> {
  const res = await fetch(`${API_BASE}/tag/${id}`, {
    method: 'DELETE',
    credentials: 'include',
  });
  return res.json();
}

export async function getUsers(): Promise<ApiResponse<User[]>> {
  const res = await fetch(`${API_BASE}/user`, {
    method: 'GET',
    credentials: 'include',
  });
  return res.json();
}

export async function createUser(data: { username: string; password: string }): Promise<ApiResponse<User>> {
  const res = await fetch(`${API_BASE}/user`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  return res.json();
}

export async function updateUser(id: number, data: { username: string; password: string }): Promise<ApiResponse<null>> {
  const res = await fetch(`${API_BASE}/user/${id}`, {
    method: 'PUT',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  return res.json();
}

export async function deleteUser(id: number): Promise<ApiResponse<null>> {
  const res = await fetch(`${API_BASE}/user/${id}`, {
    method: 'DELETE',
    credentials: 'include',
  });
  return res.json();
}

export async function getStatistics(): Promise<ApiResponse<Statistics>> {
  const res = await fetch(`${API_BASE}/statistics`, {
    method: 'GET',
    credentials: 'include',
  });
  return res.json();
}

export async function getMembers(page = 1, pageSize = 20): Promise<ApiResponse<MembersResponse>> {
  const res = await fetch(`${API_BASE}/member?page=${page}&page_size=${pageSize}`, { credentials: 'include' });
  return res.json();
}

export async function createMember(data: { bilibili_uid: string; username: string; password: string }): Promise<ApiResponse<Member>> {
  const res = await fetch(`${API_BASE}/member`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  return res.json();
}

export async function updateMemberStatus(bilibiliUid: string, isDisabled: boolean): Promise<ApiResponse<Member>> {
  const res = await fetch(`${API_BASE}/member/${bilibiliUid}`, {
    method: 'PUT',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ is_disabled: isDisabled }),
  });
  return res.json();
}

export async function deleteMember(bilibiliUid: string): Promise<ApiResponse<null>> {
  const res = await fetch(`${API_BASE}/member/${bilibiliUid}`, {
    method: 'DELETE',
    credentials: 'include',
  });
  return res.json();
}

export async function getBilibiliVerificationAccount(): Promise<ApiResponse<BilibiliVerificationAccount | null>> {
  const res = await fetch(`${API_BASE}/bilibili-verification-account`, { credentials: 'include' });
  return res.json();
}

export async function saveBilibiliVerificationAccount(data: { bilibili_uid: string; cookie: string }): Promise<ApiResponse<BilibiliVerificationAccount>> {
  const res = await fetch(`${API_BASE}/bilibili-verification-account`, {
    method: 'PUT',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  return res.json();
}

export async function testBilibiliVerificationAccount(): Promise<ApiResponse<{ valid: boolean; checked_at: string }>> {
  const res = await fetch(`${API_BASE}/bilibili-verification-account/test`, { method: 'POST', credentials: 'include' });
  return res.json();
}

export async function deleteBilibiliVerificationAccount(): Promise<ApiResponse<null>> {
  const res = await fetch(`${API_BASE}/bilibili-verification-account`, { method: 'DELETE', credentials: 'include' });
  return res.json();
}
