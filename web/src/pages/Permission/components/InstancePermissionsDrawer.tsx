import { useEffect, useState, useCallback } from 'react';
import { Drawer, Table, Button, Checkbox, Select, Space, message, Spin } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useIntl } from '@/locales';
import {
  listInstances,
  getUserInstancePermissions,
  replaceUserInstancePermissions,
  getRoleInstancePermissions,
  replaceRoleInstancePermissions,
} from '@/services/api';
import { isCommonResponseOk } from '@/utils/kratosResponse';

const PERM_TYPES = ['query', 'ddl', 'dml', 'export'] as const;

export type InstancePermissionsDrawerProps = {
  open: boolean;
  onClose: () => void;
  mode: 'user' | 'role';
  targetId: string;
  targetLabel?: string;
};

type Row = { instanceId: number; types: Set<string> };

/** Kratos 列表在顶层 list；自定义 writeSuccess 可能在 data.list */
function extractInstanceList(res: unknown): { id?: number; name?: string }[] {
  if (!res || typeof res !== 'object') return [];
  const r = res as { list?: unknown; data?: { list?: unknown } };
  const raw = r.list ?? r.data?.list;
  return Array.isArray(raw) ? (raw as { id?: number; name?: string }[]) : [];
}

type InstancePermListItem = {
  instance_id?: number;
  instanceId?: number;
  permission_type?: string;
  permissionType?: string;
};

function normalizeItems(list: InstancePermListItem[]): Row[] {
  const byInst = new Map<number, Set<string>>();
  for (const it of list || []) {
    const iid = Number(it.instance_id ?? it.instanceId ?? 0);
    const pt = String(it.permission_type ?? it.permissionType ?? '').trim();
    if (!iid || !pt) continue;
    if (!byInst.has(iid)) byInst.set(iid, new Set());
    byInst.get(iid)!.add(pt);
  }
  return Array.from(byInst.entries()).map(([instanceId, types]) => ({ instanceId, types }));
}

export function InstancePermissionsDrawer({
  open,
  onClose,
  mode,
  targetId,
  targetLabel,
}: InstancePermissionsDrawerProps) {
  const intl = useIntl();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [instanceOptions, setInstanceOptions] = useState<{ label: string; value: number }[]>([]);
  const [rows, setRows] = useState<Row[]>([]);

  const load = useCallback(async () => {
    if (!open || !targetId) return;
    setLoading(true);
    try {
      const instRes = await listInstances({ page: 1, pageSize: 500 } as Record<string, unknown>);
      const opts = extractInstanceList(instRes).map((x) => ({
        value: Number(x.id),
        label: `${x.name || x.id} (#${x.id})`,
      }));
      setInstanceOptions(opts);

      const permRes =
        mode === 'user'
          ? await getUserInstancePermissions(targetId)
          : await getRoleInstancePermissions(targetId);
      const pr = permRes as { code?: number; message?: string; list?: unknown };
      if (pr.code === 200) {
        const permList = Array.isArray(pr.list) ? (pr.list as InstancePermListItem[]) : [];
        setRows(normalizeItems(permList));
      } else {
        message.error(pr.message || intl.formatMessage({ id: 'perm.instance.loadFail' }));
        setRows([]);
      }
    } catch {
      message.error(intl.formatMessage({ id: 'perm.instance.loadFail' }));
      setRows([]);
    } finally {
      setLoading(false);
    }
  }, [open, targetId, mode, intl]);

  useEffect(() => {
    load();
  }, [load]);

  const toggleType = (instanceId: number, perm: string, checked: boolean) => {
    setRows((prev) =>
      prev.map((r) => {
        if (r.instanceId !== instanceId) return r;
        const next = new Set(r.types);
        if (checked) next.add(perm);
        else next.delete(perm);
        return { ...r, types: next };
      }),
    );
  };

  const addRow = () => {
    setRows((prev) => [...prev, { instanceId: 0, types: new Set() }]);
  };

  const setRowInstance = (index: number, instanceId: number) => {
    setRows((prev) => {
      const next = [...prev];
      if (next[index]) next[index] = { ...next[index], instanceId };
      return next;
    });
  };

  const removeRow = (index: number) => {
    setRows((prev) => prev.filter((_, i) => i !== index));
  };

  const handleSave = async () => {
    const items: { instance_id: number; permission_type: string }[] = [];
    for (const r of rows) {
      if (!r.instanceId) continue;
      for (const t of r.types) {
        items.push({ instance_id: r.instanceId, permission_type: t });
      }
    }
    setSaving(true);
    try {
      const res =
        mode === 'user'
          ? await replaceUserInstancePermissions(targetId, items)
          : await replaceRoleInstancePermissions(targetId, items);
      if (isCommonResponseOk(res)) {
        message.success(intl.formatMessage({ id: 'perm.instance.saveOk' }));
        onClose();
      } else {
        message.error((res as any).message || intl.formatMessage({ id: 'perm.instance.saveFail' }));
      }
    } catch {
      message.error(intl.formatMessage({ id: 'perm.instance.saveFail' }));
    } finally {
      setSaving(false);
    }
  };

  const title =
    mode === 'user'
      ? intl.formatMessage({ id: 'perm.instance.drawerUser' }, { name: targetLabel || targetId })
      : intl.formatMessage({ id: 'perm.instance.drawerRole' }, { name: targetLabel || targetId });

  const columns = [
    {
      title: intl.formatMessage({ id: 'instance.title' }),
      key: 'inst',
      width: 280,
      render: (_: unknown, record: Row, index: number) => {
        if (!record.instanceId) {
          const used = new Set(rows.filter((r) => r.instanceId > 0).map((r) => r.instanceId));
          return (
            <Select
              showSearch
              optionFilterProp="label"
              placeholder={intl.formatMessage({ id: 'perm.instance.pickInstance' })}
              style={{ width: '100%' }}
              options={instanceOptions.filter((o) => !used.has(o.value))}
              onChange={(v) => setRowInstance(index, Number(v))}
            />
          );
        }
        const label = instanceOptions.find((o) => o.value === record.instanceId)?.label;
        return label || `#${record.instanceId}`;
      },
    },
    ...PERM_TYPES.map((pt) => ({
      title: intl.formatMessage({ id: `perm.instance.type.${pt}` }),
      key: pt,
      width: 90,
      align: 'center' as const,
      render: (_: unknown, record: Row) =>
        record.instanceId ? (
          <Checkbox
            checked={record.types.has(pt)}
            onChange={(e) => toggleType(record.instanceId, pt, e.target.checked)}
          />
        ) : null,
    })),
    {
      title: intl.formatMessage({ id: 'home.col.action' }),
      key: 'act',
      width: 72,
      render: (_: unknown, __: Row, index: number) => (
        <Button type="link" danger size="small" onClick={() => removeRow(index)}>
          {intl.formatMessage({ id: 'common.delete' })}
        </Button>
      ),
    },
  ];

  return (
    <Drawer
      title={title}
      width={720}
      open={open}
      onClose={onClose}
      destroyOnClose
      extra={
        <Space>
          <Button onClick={onClose}>{intl.formatMessage({ id: 'common.cancel' })}</Button>
          <Button type="primary" loading={saving} onClick={handleSave}>
            {intl.formatMessage({ id: 'common.save' })}
          </Button>
        </Space>
      }
    >
      <p style={{ marginBottom: 12, color: 'var(--ant-color-text-secondary)' }}>
        {intl.formatMessage({ id: 'perm.instance.hint' })}
      </p>
      <Space style={{ marginBottom: 12 }}>
        <Button icon={<PlusOutlined />} onClick={addRow}>
          {intl.formatMessage({ id: 'perm.instance.addRow' })}
        </Button>
      </Space>
      <Spin spinning={loading}>
        <Table
          rowKey={(_, i) => String(i)}
          dataSource={rows}
          columns={columns}
          pagination={false}
          size="small"
        />
      </Spin>
    </Drawer>
  );
}
