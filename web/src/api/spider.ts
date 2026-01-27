import request from './request';

export type SpecialFollowUser = {
  id: string;
  nickname: string;
}

export type SpecialFollowDBUser = {
  id: number;
  owner_id: string;
  user_id: string;
  nickname: string;
  synced_at: string;
  created_at: string;
}

// 从微博获取特别关注
export const getSpecialFollows = () => {
  return request.get<any, { data: { users: SpecialFollowUser[]; total: number } }>('/special-follows');
};

// 同步特别关注到数据库
export const syncSpecialFollows = () => {
  return request.post('/special-follows/sync');
};

// 从数据库获取特别关注
export const getSpecialFollowsFromDB = () => {
  return request.get<any, { data: { users: SpecialFollowDBUser[]; total: number } }>('/special-follows/db');
};

// 提交爬取任务
export const submitCrawlTask = (userId: string, downloadMedia: boolean = true) => {
  return request.post('/weibos', {
    user_id: userId,
    download_media: downloadMedia,
  });
};
