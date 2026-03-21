import { PageContainer } from '@ant-design/pro-components';
import {
  Button,
  Card,
  Col,
  Form,
  Input,
  message,
  Modal,
  Row,
  Select,
  Space,
  Spin,
} from 'antd';
import { MinusCircleOutlined, PlusOutlined } from '@ant-design/icons';
import { useNavigate, useSearchParams } from 'react-router-dom';
import {
  createWidetableConfig,
  getWidetableConfig,
  listInstances,
  previewWidetableSQL,
  updateWidetableConfig,
} from '@/services/api';
import { useEffect, useState } from 'react';
import { useRequest } from 'ahooks';
import { useIntl } from '@/locales';

const { TextArea } = Input;

const JOIN_TYPE_OPTIONS = [
  { label: 'LEFT JOIN', value: 'LEFT JOIN' },
  { label: 'INNER JOIN', value: 'INNER JOIN' },
  { label: 'RIGHT JOIN', value: 'RIGHT JOIN' },
  { label: 'FULL JOIN', value: 'FULL JOIN' },
];

export default () => {
  const intl = useIntl();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const editId = searchParams.get('id');
  const isEdit = !!editId;

  const [form] = Form.useForm();
  const [submitting, setSubmitting] = useState(false);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [previewSQL, setPreviewSQL] = useState('');
  const [previewVisible, setPreviewVisible] = useState(false);
  const [configLoading, setConfigLoading] = useState(false);

  const { data: instancesData } = useRequest(
    () => listInstances({ page: 1, pageSize: 1000 }),
  );
  const instances = (instancesData as any)?.list || (instancesData as any)?.data?.list || [];

  useEffect(() => {
    if (isEdit && editId) {
      setConfigLoading(true);
      getWidetableConfig(Number(editId))
        .then((resp: any) => {
          const config = resp.data || resp;
          if (config) {
            let joinConfig: any[] = [];
            let fieldConfig: any[] = [];
            try {
              joinConfig = JSON.parse(config.join_config || '[]');
            } catch {}
            try {
              fieldConfig = JSON.parse(config.field_config || '[]');
            } catch {}
            form.setFieldsValue({
              name: config.name,
              description: config.description,
              source_instance_id: config.source_instance_id,
              main_database: config.main_database,
              main_table: config.main_table,
              sync_condition: config.sync_condition,
              target_instance_id: config.target_instance_id,
              target_database: config.target_database,
              target_table: config.target_table,
              join_config: joinConfig,
              field_config: fieldConfig,
            });
          }
        })
        .catch(() => {})
        .finally(() => setConfigLoading(false));
    }
  }, [editId, isEdit]);

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setSubmitting(true);

      const payload = {
        ...values,
        join_config: JSON.stringify(values.join_config || []),
        field_config: JSON.stringify(values.field_config || []),
      };

      let response: any;
      if (isEdit) {
        response = await updateWidetableConfig(Number(editId), payload);
      } else {
        response = await createWidetableConfig(payload);
      }

      if (response.code === 200) {
        message.success(isEdit ? intl.formatMessage({ id: 'common.updateOk' }) : intl.formatMessage({ id: 'instance.createOk' }));
        navigate('/widetable/list');
      } else {
        message.error(response.message || (isEdit ? intl.formatMessage({ id: 'common.operationFailed' }) : intl.formatMessage({ id: 'instance.createFail' })));
      }
    } catch (e: any) {
      if (!e?.errorFields) {
      }
    } finally {
      setSubmitting(false);
    }
  };

  const handlePreviewSQL = async () => {
    if (!isEdit) {
      message.warning(intl.formatMessage({ id: 'widetable.warnPreview' }));
      return;
    }
    setPreviewLoading(true);
    try {
      const response: any = await previewWidetableSQL(Number(editId), { limit: 10 });
      if (response.code === 200) {
        setPreviewSQL(response.data?.sql || response.sql || '');
        setPreviewVisible(true);
      } else {
        message.error(response.message || intl.formatMessage({ id: 'widetable.previewFail' }));
      }
    } catch {
    } finally {
      setPreviewLoading(false);
    }
  };

  return (
    <PageContainer title={isEdit ? intl.formatMessage({ id: 'widetable.create.titleEdit' }) : intl.formatMessage({ id: 'widetable.create.titleNew' })}>
      <Spin spinning={configLoading}>
        <Form form={form} layout="vertical" style={{ maxWidth: 900 }}>
          <Card title={intl.formatMessage({ id: 'widetable.card.basic' })} style={{ marginBottom: 16 }}>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item
                  name="name"
                  label={intl.formatMessage({ id: 'common.name' })}
                  rules={[{ required: true, message: intl.formatMessage({ id: 'widetable.rule.name' }) }]}
                >
                  <Input placeholder={intl.formatMessage({ id: 'widetable.ph.name' })} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="description" label={intl.formatMessage({ id: 'widetable.label.desc' })}>
                  <Input placeholder={intl.formatMessage({ id: 'widetable.ph.desc' })} />
                </Form.Item>
              </Col>
            </Row>
            <Row gutter={16}>
              <Col span={8}>
                <Form.Item
                  name="source_instance_id"
                  label={intl.formatMessage({ id: 'archive.label.sourceInst' })}
                  rules={[{ required: true, message: intl.formatMessage({ id: 'widetable.rule.sourceInst' }) }]}
                >
                  <Select
                    placeholder={intl.formatMessage({ id: 'widetable.rule.sourceInst' })}
                    showSearch
                    filterOption={(input, option) =>
                      (option?.label as string)?.toLowerCase().includes(input.toLowerCase())
                    }
                    options={instances.map((inst: any) => ({
                      label: `${inst.name} (${inst.host}:${inst.port})`,
                      value: inst.id,
                    }))}
                  />
                </Form.Item>
              </Col>
              <Col span={8}>
                <Form.Item
                  name="main_database"
                  label={intl.formatMessage({ id: 'widetable.label.mainDb' })}
                  rules={[{ required: true, message: intl.formatMessage({ id: 'widetable.rule.mainDb' }) }]}
                >
                  <Input placeholder="main_db" />
                </Form.Item>
              </Col>
              <Col span={8}>
                <Form.Item
                  name="main_table"
                  label={intl.formatMessage({ id: 'widetable.label.mainTable' })}
                  rules={[{ required: true, message: intl.formatMessage({ id: 'widetable.rule.mainTable' }) }]}
                >
                  <Input placeholder="orders" />
                </Form.Item>
              </Col>
            </Row>
          </Card>

          <Card title={intl.formatMessage({ id: 'widetable.card.join' })} style={{ marginBottom: 16 }}>
            <Form.List name="join_config">
              {(fields, { add, remove }) => (
                <>
                  {fields.map(({ key, name, ...restField }) => (
                    <div key={key} style={{ background: '#fafafa', padding: 16, marginBottom: 12, borderRadius: 6, border: '1px solid #f0f0f0' }}>
                      <Row gutter={12} align="middle">
                        <Col span={4}>
                          <Form.Item
                            {...restField}
                            name={[name, 'database']}
                            label={intl.formatMessage({ id: 'widetable.label.dbName' })}
                            rules={[{ required: true, message: intl.formatMessage({ id: 'widetable.rule.dbName' }) }]}
                            style={{ marginBottom: 0 }}
                          >
                            <Input placeholder="database" />
                          </Form.Item>
                        </Col>
                        <Col span={4}>
                          <Form.Item
                            {...restField}
                            name={[name, 'table']}
                            label={intl.formatMessage({ id: 'widetable.label.tableName' })}
                            rules={[{ required: true, message: intl.formatMessage({ id: 'widetable.rule.tableName' }) }]}
                            style={{ marginBottom: 0 }}
                          >
                            <Input placeholder="table" />
                          </Form.Item>
                        </Col>
                        <Col span={3}>
                          <Form.Item
                            {...restField}
                            name={[name, 'alias']}
                            label={intl.formatMessage({ id: 'widetable.label.alias' })}
                            style={{ marginBottom: 0 }}
                          >
                            <Input placeholder="t1" />
                          </Form.Item>
                        </Col>
                        <Col span={4}>
                          <Form.Item
                            {...restField}
                            name={[name, 'join_type']}
                            label={intl.formatMessage({ id: 'widetable.label.joinType' })}
                            rules={[{ required: true, message: intl.formatMessage({ id: 'widetable.rule.joinType' }) }]}
                            style={{ marginBottom: 0 }}
                          >
                            <Select options={JOIN_TYPE_OPTIONS} />
                          </Form.Item>
                        </Col>
                        <Col span={8}>
                          <Form.Item
                            {...restField}
                            name={[name, 'on']}
                            label="ON"
                            rules={[{ required: true, message: intl.formatMessage({ id: 'widetable.rule.on' }) }]}
                            style={{ marginBottom: 0 }}
                          >
                            <Input placeholder="main.id = t1.order_id" />
                          </Form.Item>
                        </Col>
                        <Col span={1} style={{ paddingTop: 30 }}>
                          <MinusCircleOutlined
                            style={{ color: '#ff4d4f', cursor: 'pointer', fontSize: 18 }}
                            onClick={() => remove(name)}
                          />
                        </Col>
                      </Row>
                    </div>
                  ))}
                  <Button
                    type="dashed"
                    onClick={() => add({ join_type: 'LEFT JOIN' })}
                    icon={<PlusOutlined />}
                  >
                    {intl.formatMessage({ id: 'widetable.addJoin' })}
                  </Button>
                </>
              )}
            </Form.List>
          </Card>

          <Card title={intl.formatMessage({ id: 'widetable.card.fields' })} style={{ marginBottom: 16 }}>
            <Form.List name="field_config">
              {(fields, { add, remove }) => (
                <>
                  {fields.map(({ key, name, ...restField }) => (
                    <Row gutter={12} key={key} align="middle" style={{ marginBottom: 8 }}>
                      <Col span={11}>
                        <Form.Item
                          {...restField}
                          name={[name, 'source']}
                          label={name === 0 ? intl.formatMessage({ id: 'widetable.label.sourceExpr' }) : ''}
                          rules={[{ required: true, message: intl.formatMessage({ id: 'widetable.rule.sourceExpr' }) }]}
                          style={{ marginBottom: 0 }}
                        >
                          <Input placeholder="main.id" />
                        </Form.Item>
                      </Col>
                      <Col span={11}>
                        <Form.Item
                          {...restField}
                          name={[name, 'alias']}
                          label={name === 0 ? intl.formatMessage({ id: 'widetable.label.fieldAlias' }) : ''}
                          rules={[{ required: true, message: intl.formatMessage({ id: 'widetable.rule.fieldAlias' }) }]}
                          style={{ marginBottom: 0 }}
                        >
                          <Input placeholder="order_id" />
                        </Form.Item>
                      </Col>
                      <Col span={2} style={{ paddingTop: name === 0 ? 30 : 0 }}>
                        <MinusCircleOutlined
                          style={{ color: '#ff4d4f', cursor: 'pointer', fontSize: 18 }}
                          onClick={() => remove(name)}
                        />
                      </Col>
                    </Row>
                  ))}
                  <Button
                    type="dashed"
                    onClick={() => add()}
                    icon={<PlusOutlined />}
                    style={{ marginTop: 8 }}
                  >
                    {intl.formatMessage({ id: 'widetable.addField' })}
                  </Button>
                </>
              )}
            </Form.List>
          </Card>

          <Card title={intl.formatMessage({ id: 'widetable.card.syncCond' })} style={{ marginBottom: 16 }}>
            <Form.Item name="sync_condition" label={intl.formatMessage({ id: 'widetable.label.syncWhere' })}>
              <TextArea rows={3} placeholder={intl.formatMessage({ id: 'widetable.ph.syncWhere' })} />
            </Form.Item>
          </Card>

          <Card title={intl.formatMessage({ id: 'widetable.card.target' })} style={{ marginBottom: 16 }}>
            <Row gutter={16}>
              <Col span={8}>
                <Form.Item name="target_instance_id" label={intl.formatMessage({ id: 'widetable.label.targetInst' })}>
                  <Select
                    placeholder={intl.formatMessage({ id: 'widetable.ph.targetInst' })}
                    allowClear
                    showSearch
                    filterOption={(input, option) =>
                      (option?.label as string)?.toLowerCase().includes(input.toLowerCase())
                    }
                    options={instances.map((inst: any) => ({
                      label: `${inst.name} (${inst.host}:${inst.port})`,
                      value: inst.id,
                    }))}
                  />
                </Form.Item>
              </Col>
              <Col span={8}>
                <Form.Item name="target_database" label={intl.formatMessage({ id: 'widetable.label.targetDb' })}>
                  <Input placeholder={intl.formatMessage({ id: 'widetable.ph.targetDb' })} />
                </Form.Item>
              </Col>
              <Col span={8}>
                <Form.Item name="target_table" label={intl.formatMessage({ id: 'widetable.label.targetTable' })}>
                  <Input placeholder={intl.formatMessage({ id: 'widetable.ph.targetTable' })} />
                </Form.Item>
              </Col>
            </Row>
          </Card>

          <Space>
            <Button
              type="primary"
              onClick={handleSubmit}
              loading={submitting}
            >
              {isEdit ? intl.formatMessage({ id: 'widetable.saveEdit' }) : intl.formatMessage({ id: 'widetable.createBtn' })}
            </Button>
            {isEdit && (
              <Button
                onClick={handlePreviewSQL}
                loading={previewLoading}
              >
                {intl.formatMessage({ id: 'widetable.previewSql' })}
              </Button>
            )}
            <Button onClick={() => navigate('/widetable/list')}>{intl.formatMessage({ id: 'widetable.cancel' })}</Button>
          </Space>
        </Form>
      </Spin>

      <Modal
        title={intl.formatMessage({ id: 'widetable.previewTitle' })}
        open={previewVisible}
        onCancel={() => setPreviewVisible(false)}
        footer={<Button onClick={() => setPreviewVisible(false)}>{intl.formatMessage({ id: 'widetable.close' })}</Button>}
        width={900}
      >
        <pre
          style={{
            background: '#1e1e1e',
            color: '#d4d4d4',
            padding: 16,
            borderRadius: 4,
            overflowX: 'auto',
            fontFamily: 'monospace',
            fontSize: 13,
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-all',
          }}
        >
          {previewSQL || intl.formatMessage({ id: 'widetable.emptySql' })}
        </pre>
      </Modal>
    </PageContainer>
  );
};
