import { Tag } from 'antd';
import { useIntl } from '@/locales';

interface StatusTagProps {
  status: number;
  type?: 'order' | 'archive' | 'redis' | 'binlog';
}

export default function StatusTag({ status, type = 'order' }: StatusTagProps) {
  const intl = useIntl();

  if (type === 'order') {
    const orderIds: Record<number, string> = {
      0: 'home.status.pending',
      1: 'home.status.approved',
      2: 'home.status.rejected',
      3: 'home.status.executing',
      4: 'home.status.success',
      5: 'home.status.failed',
      6: 'home.status.review',
    };
    const orderColors: Record<number, string> = {
      0: 'orange',
      1: 'green',
      2: 'red',
      3: 'blue',
      4: 'success',
      5: 'error',
      6: 'warning',
    };
    const id = orderIds[status];
    const color = orderColors[status] || 'default';
    const text = id
      ? intl.formatMessage({ id })
      : intl.formatMessage({ id: 'home.status.unknown' });
    return <Tag color={color}>{text}</Tag>;
  }

  if (type === 'archive') {
    const map: Record<number, { id: string; color: string }> = {
      0: { id: 'archive.records.status.running', color: 'blue' },
      1: { id: 'archive.records.status.ok', color: 'green' },
      2: { id: 'archive.records.status.fail', color: 'red' },
    };
    const m = map[status] || { id: 'home.status.unknown', color: 'default' };
    return <Tag color={m.color}>{intl.formatMessage({ id: m.id })}</Tag>;
  }

  if (type === 'redis') {
    const map: Record<number, { id: string; color: string }> = {
      0: { id: 'redis.task.status.running', color: 'blue' },
      1: { id: 'redis.task.status.done', color: 'green' },
      2: { id: 'redis.task.status.fail', color: 'red' },
    };
    const m = map[status] || { id: 'home.status.unknown', color: 'default' };
    return <Tag color={m.color}>{intl.formatMessage({ id: m.id })}</Tag>;
  }

  const map: Record<number, { id: string; color: string }> = {
    0: { id: 'binlog.status.parsing', color: 'blue' },
    1: { id: 'binlog.status.ok', color: 'green' },
    2: { id: 'binlog.status.fail', color: 'red' },
  };
  const m = map[status] || { id: 'home.status.unknown', color: 'default' };
  return <Tag color={m.color}>{intl.formatMessage({ id: m.id })}</Tag>;
}
