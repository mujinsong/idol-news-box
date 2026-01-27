import { useState, useEffect } from 'react';
import { Card, Form, Input, Button, Descriptions, message, Modal } from 'antd';
import { EditOutlined } from '@ant-design/icons';
import { getCurrentUser, updateUser } from '../api/user';
import type { UserInfo } from '../api/user';

const Profile = () => {
  const [user, setUser] = useState<UserInfo | null>(null);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [form] = Form.useForm();

  const fetchUser = async () => {
    try {
      const res = await getCurrentUser();
      setUser(res.data);
    } catch {
      // 错误已处理
    }
  };

  useEffect(() => {
    fetchUser();
  }, []);

  const handleEdit = () => {
    form.setFieldsValue({
      nickname: user?.nickname,
      weibo_uid: user?.weibo_uid,
      weibo_cookie: '',
    });
    setModalOpen(true);
  };

  const handleSubmit = async () => {
    const values = await form.validateFields();
    if (!user) return;

    setLoading(true);
    try {
      const params: any = {};
      if (values.nickname) params.nickname = values.nickname;
      if (values.weibo_uid) params.weibo_uid = values.weibo_uid;
      if (values.weibo_cookie) params.weibo_cookie = values.weibo_cookie;
      if (values.password) params.password = values.password;

      await updateUser(user.id, params);
      message.success('更新成功');
      setModalOpen(false);
      fetchUser();
    } catch {
      // 错误已处理
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <Card
        title="个人信息"
        extra={<Button icon={<EditOutlined />} onClick={handleEdit}>编辑</Button>}
      >
        <Descriptions column={1}>
          <Descriptions.Item label="用户名">{user?.username}</Descriptions.Item>
          <Descriptions.Item label="昵称">{user?.nickname || '-'}</Descriptions.Item>
          <Descriptions.Item label="微博UID">{user?.weibo_uid || '未绑定'}</Descriptions.Item>
          <Descriptions.Item label="状态">{user?.status === 1 ? '正常' : '禁用'}</Descriptions.Item>
          <Descriptions.Item label="创建时间">{user?.created_at}</Descriptions.Item>
        </Descriptions>
      </Card>

      <Modal
        title="编辑信息"
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        confirmLoading={loading}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="nickname" label="昵称">
            <Input placeholder="请输入昵称" />
          </Form.Item>
          <Form.Item name="weibo_uid" label="微博UID">
            <Input placeholder="请输入微博UID" />
          </Form.Item>
          <Form.Item name="password" label="新密码">
            <Input.Password placeholder="不修改请留空" />
          </Form.Item>
          <Form.Item name="weibo_cookie" label="微博Cookie">
            <Input.TextArea rows={4} placeholder="从weibo.cn获取的Cookie" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default Profile;
