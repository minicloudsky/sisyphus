import { PageContainer, ProCard, ProTable } from '@ant-design/pro-components';
import { Button, Col, Row, Statistic, Tag, Space } from 'antd';
import {
  DatabaseOutlined,
  FileTextOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  PlusOutlined,
  SearchOutlined,
  RollbackOutlined,
  CalendarOutlined,
} from '@ant-design/icons';
import { useRequest } from 'ahooks';
import { useIntl } from '@/locales';
import { listSQLOrders, listInstances } from '@/services/api';
import { useNavigate } from 'react-router-dom';
import type { ProColumns } from '@ant-design/pro-components';
import { useMemo } from 'react';
import './index.less';

export default () => {
  const navigate = useNavigate();
  const intl = useIntl();
  const locale = intl.locale;

  const statusMap = useMemo(
    () =>
      ({
        0: { text: intl.formatMessage({ id: 'home.status.pending' }), color: 'orange' },
        1: { text: intl.formatMessage({ id: 'home.status.approved' }), color: 'green' },
        2: { text: intl.formatMessage({ id: 'home.status.rejected' }), color: 'red' },
        3: { text: intl.formatMessage({ id: 'home.status.executing' }), color: 'blue' },
        4: { text: intl.formatMessage({ id: 'home.status.success' }), color: 'success' },
        5: { text: intl.formatMessage({ id: 'home.status.failed' }), color: 'error' },
        6: { text: intl.formatMessage({ id: 'home.status.review' }), color: 'warning' },
      }) as Record<number, { text: string; color: string }>,
    [locale, intl]
  );

  const { data: instancesData } = useRequest(() => listInstances({ page: 1, pageSize: 1 }));

  const { data: pendingData } = useRequest(() =>
    listSQLOrders({ page: 1, pageSize: 1, status: 0 })
  );

  const { data: todayData } = useRequest(() => listSQLOrders({ page: 1, pageSize: 100 }));

  const { data: monthData } = useRequest(() => listSQLOrders({ page: 1, pageSize: 100 }));

  const { data: recentData, loading: recentLoading } = useRequest(() =>
    listSQLOrders({ page: 1, pageSize: 5 })
  );

  const instanceTotal =
    instancesData?.pageResponse?.total || instancesData?.list?.length || 0;

  const pendingTotal =
    pendingData?.data?.page?.total || (pendingData?.data?.list || []).length || 0;

  const today = new Date().toISOString().slice(0, 10);
  const allOrders: API.SQLOrder[] = todayData?.data?.list || [];
  const todayExecuted = allOrders.filter(
    (o) =>
      (o.status === 4 || o.status === 5 || o.status === 3) && o.createdAt?.startsWith(today)
  ).length;

  const thisMonth = new Date().toISOString().slice(0, 7);
  const monthOrders: API.SQLOrder[] = monthData?.data?.list || [];
  const monthTotal = monthOrders.filter((o) => o.createdAt?.startsWith(thisMonth)).length;

  const recentOrders: API.SQLOrder[] = recentData?.data?.list || [];

  const recentColumns: ProColumns<API.SQLOrder>[] = useMemo(
    () => [
      {
        title: intl.formatMessage({ id: 'home.col.workId' }),
        dataIndex: 'workId',
        width: 160,
        render: (_, record) => (
          <a onClick={() => navigate(`/sql-order/detail/${record.id}`)}>{record.workId}</a>
        ),
      },
      {
        title: intl.formatMessage({ id: 'home.col.database' }),
        dataIndex: 'databaseName',
        width: 120,
      },
      {
        title: intl.formatMessage({ id: 'home.col.type' }),
        dataIndex: 'orderType',
        width: 80,
        render: (_, record) => {
          const typeMap = [
            intl.formatMessage({ id: 'home.orderType.ddl' }),
            intl.formatMessage({ id: 'home.orderType.dml' }),
            intl.formatMessage({ id: 'home.orderType.export' }),
          ];
          return typeMap[record.orderType] || '-';
        },
      },
      {
        title: intl.formatMessage({ id: 'home.col.status' }),
        dataIndex: 'status',
        width: 100,
        render: (_, record) => {
          const s = statusMap[record.status] || {
            text: intl.formatMessage({ id: 'home.status.unknown' }),
            color: 'default',
          };
          return <Tag color={s.color}>{s.text}</Tag>;
        },
      },
      {
        title: intl.formatMessage({ id: 'home.col.createdAt' }),
        dataIndex: 'createdAt',
        width: 180,
        valueType: 'dateTime',
      },
      {
        title: intl.formatMessage({ id: 'home.col.action' }),
        valueType: 'option',
        width: 80,
        render: (_, record) => (
          <Button type="link" size="small" onClick={() => navigate(`/sql-order/detail/${record.id}`)}>
            {intl.formatMessage({ id: 'home.view' })}
          </Button>
        ),
      },
    ],
    [locale, intl, navigate, statusMap]
  );

  return (
    <PageContainer title={intl.formatMessage({ id: 'home.platformTitle' })} className="home-container">
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <ProCard>
            <Statistic
              title={intl.formatMessage({ id: 'home.stat.instances' })}
              value={instanceTotal}
              prefix={<DatabaseOutlined />}
              valueStyle={{ color: '#1677ff' }}
            />
          </ProCard>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <ProCard>
            <Statistic
              title={intl.formatMessage({ id: 'home.stat.pendingOrders' })}
              value={pendingTotal}
              prefix={<ClockCircleOutlined />}
              valueStyle={{ color: '#faad14' }}
            />
          </ProCard>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <ProCard>
            <Statistic
              title={intl.formatMessage({ id: 'home.stat.todayExecuted' })}
              value={todayExecuted}
              prefix={<CalendarOutlined />}
              valueStyle={{ color: '#52c41a' }}
            />
          </ProCard>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <ProCard>
            <Statistic
              title={intl.formatMessage({ id: 'home.stat.monthOrders' })}
              value={monthTotal}
              prefix={<FileTextOutlined />}
              valueStyle={{ color: '#722ed1' }}
            />
          </ProCard>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={8}>
          <ProCard title={intl.formatMessage({ id: 'home.quickLinks' })} headerBordered>
            <Space wrap size={[12, 12]}>
              <Button type="primary" icon={<PlusOutlined />} onClick={() => navigate('/sql-order/create-ddl')}>
                {intl.formatMessage({ id: 'home.createDdl' })}
              </Button>
              <Button
                type="primary"
                ghost
                icon={<PlusOutlined />}
                onClick={() => navigate('/sql-order/create-dml')}
              >
                {intl.formatMessage({ id: 'home.createDml' })}
              </Button>
              <Button icon={<SearchOutlined />} onClick={() => navigate('/query')}>
                {intl.formatMessage({ id: 'home.query' })}
              </Button>
              <Button icon={<RollbackOutlined />} onClick={() => navigate('/binlog')}>
                {intl.formatMessage({ id: 'home.binlog' })}
              </Button>
              <Button icon={<CheckCircleOutlined />} onClick={() => navigate('/sql-order/list')}>
                {intl.formatMessage({ id: 'home.orderList' })}
              </Button>
            </Space>
          </ProCard>
        </Col>

        <Col xs={24} lg={16}>
          <ProCard title={intl.formatMessage({ id: 'home.recentOrders' })} headerBordered>
            <ProTable<API.SQLOrder>
              columns={recentColumns}
              dataSource={recentOrders}
              rowKey="id"
              loading={recentLoading}
              search={false}
              options={false}
              pagination={false}
              size="small"
              locale={{ emptyText: intl.formatMessage({ id: 'home.emptyOrders' }) }}
            />
          </ProCard>
        </Col>
      </Row>
    </PageContainer>
  );
};
