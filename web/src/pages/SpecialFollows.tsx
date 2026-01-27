import { useState, useEffect } from 'react';
import { Card, Table, Button, message } from 'antd';
import { SyncOutlined, DownloadOutlined } from '@ant-design/icons';
import { getSpecialFollowsFromDB, syncSpecialFollows, submitCrawlTask } from '../api/spider';
import type { SpecialFollowDBUser } from '../api/spider';

const SpecialFollows = () => {
  const [data, setData] = useState<SpecialFollowDBUser[]>([]);
  const [loading, setLoading] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [crawling, setCrawling] = useState<Record<string, boolean>>({});

  const fetchData = async () => {
    setLoading(true);
    try {
      const res = await getSpecialFollowsFromDB();
      setData(res.data.users || []);
    } catch {
      // 错误已处理
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleSync = async () => {
    setSyncing(true);
    try {
      await syncSpecialFollows();
      message.success('同步成功');
      fetchData();
    } catch {
      // 错误已处理
    } finally {
      setSyncing(false);
    }
  };

  const handleCrawl = async (userId: string, nickname: string) => {
    setCrawling(prev => ({ ...prev, [userId]: true }));
    try {
      await submitCrawlTask(userId, true);
      message.success(`已提交 ${nickname} 的爬取任务`);
    } catch {
      // 错误已处理
    } finally {
      setCrawling(prev => ({ ...prev, [userId]: false }));
    }
  };

  const columns = [
    { title: '微博UID', dataIndex: 'user_id', key: 'user_id' },
    { title: '昵称', dataIndex: 'nickname', key: 'nickname' },
    {
      title: '同步时间',
      dataIndex: 'synced_at',
      key: 'synced_at',
      render: (text: string) => text ? new Date(text).toLocaleString() : '-',
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: SpecialFollowDBUser) => (
        <Button
          type="primary"
          size="small"
          icon={<DownloadOutlined />}
          loading={crawling[record.user_id]}
          onClick={() => handleCrawl(record.user_id, record.nickname)}
        >
          爬取微博
        </Button>
      ),
    },
  ];

  return (
    <Card
      title="特别关注"
      extra={
        <Button type="primary" icon={<SyncOutlined />} loading={syncing} onClick={handleSync}>
          同步特别关注
        </Button>
      }
    >
      <Table
        columns={columns}
        dataSource={data}
        rowKey="id"
        loading={loading}
        pagination={{ pageSize: 10 }}
      />
    </Card>
  );
};

export default SpecialFollows;
