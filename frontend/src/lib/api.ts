const API_BASE = '/api';

export interface Tag {
  id: number;
  tag_name: string;
  description: string;
  question_count: number;
  created_at: string;
  updated_at: string;
}

export interface EmojiData {
  value: string;
  count: number;
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

export async function getInfo(): Promise<ApiResponse<{ id: number; username: string }>> {
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

export async function getQuestions(params: {
  page?: number;
  size?: number;
  order_by?: string;
  order?: string;
  tag_id?: number;
  publish?: boolean;
  rainbow?: boolean;
  archive?: boolean;
  search?: string;
}): Promise<ApiResponse<QuestionsResponse>> {
  const searchParams = new URLSearchParams();
  if (params.page) searchParams.set('page', params.page.toString());
  if (params.size) searchParams.set('size', params.size.toString());
  if (params.order_by) searchParams.set('order_by', params.order_by);
  if (params.order) searchParams.set('order', params.order);
  if (params.tag_id) searchParams.set('tag_id', params.tag_id.toString());
  if (params.publish !== undefined) searchParams.set('publish', params.publish.toString());
  if (params.rainbow !== undefined) searchParams.set('rainbow', params.rainbow.toString());
  if (params.archive !== undefined) searchParams.set('archive', params.archive.toString());
  if (params.search) searchParams.set('search', params.search);

  const res = await fetch(`${API_BASE}/question?${searchParams.toString()}`, {
    method: 'GET',
    credentials: 'include',
  });
  return res.json();
}

export async function createQuestion(data: FormData): Promise<Response> {
  return fetch(`${API_BASE}/question`, {
    method: 'POST',
    credentials: 'include',
    body: data,
  });
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

export async function addEmoji(questionId: number, emoji: string): Promise<ApiResponse<null>> {
  const formData = new FormData();
  formData.append('emoji', emoji);
  const res = await fetch(`${API_BASE}/question/${questionId}/emoji`, {
    method: 'POST',
    credentials: 'include',
    body: formData,
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
