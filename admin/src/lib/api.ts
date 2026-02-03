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
  emojis: string;
  likes: number;
  created_at: string;
  updated_at: string;
}

export interface Config {
  announcement: string;
}

export interface Statistics {
  total_questions: number;
  total_tags: number;
  total_users: number;
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

export async function updateConfig(config: Config): Promise<ApiResponse<null>> {
  const res = await fetch(`${API_BASE}/config`, {
    method: 'PUT',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config),
  });
  return res.json();
}

export async function getQuestions(params: {
  page?: number;
  page_size?: number;
  order_by?: string;
  order?: string;
  tag_id?: number;
}): Promise<ApiResponse<QuestionsResponse>> {
  const searchParams = new URLSearchParams();
  if (params.page) searchParams.set('page', params.page.toString());
  if (params.page_size) searchParams.set('page_size', params.page_size.toString());
  if (params.order_by) searchParams.set('order_by', params.order_by);
  if (params.order) searchParams.set('order', params.order);
  if (params.tag_id) searchParams.set('tag_id', params.tag_id.toString());

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
