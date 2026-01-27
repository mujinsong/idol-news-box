import request from './request';

export type LoginParams = {
  username: string;
  password: string;
}

export type UserInfo = {
  id: number;
  username: string;
  nickname: string;
  weibo_uid: string;
  weibo_cookie: string;
  status: number;
  created_at: string;
  updated_at: string;
}

export type UpdateUserParams = {
  nickname?: string;
  password?: string;
  weibo_uid?: string;
  weibo_cookie?: string;
}

// 登录
export const login = (params: LoginParams) => {
  return request.post('/auth/login', params);
};

// 获取当前用户信息
export const getCurrentUser = () => {
  return request.get<any, { data: UserInfo }>('/users/me');
};

// 更新用户信息
export const updateUser = (id: number, params: UpdateUserParams) => {
  return request.put(`/users/${id}`, params);
};
