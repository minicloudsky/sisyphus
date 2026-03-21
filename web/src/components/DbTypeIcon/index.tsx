import React from 'react';
import { DatabaseOutlined } from '@ant-design/icons';

/**
 * 数据库类型徽标（Simple Icons 官方矢量，见 public/icons/db/，MIT）
 * HBase 无独立 Simple Icon，使用 Apache 徽标代表其生态。
 */
const ICON_SRC: Record<string, string> = {
  mysql: '/icons/db/mysql.svg',
  redis: '/icons/db/redis.svg',
  clickhouse: '/icons/db/clickhouse.svg',
  hbase: '/icons/db/hbase.svg',
};

interface Props {
  dbType: string;
  size?: number;
  style?: React.CSSProperties;
  className?: string;
}

const DbTypeIcon: React.FC<Props> = ({ dbType, size = 18, style, className }) => {
  const t = (dbType || '').toLowerCase();
  const src = ICON_SRC[t];

  if (!src) {
    return (
      <DatabaseOutlined
        className={className}
        style={{
          fontSize: size,
          color: '#8c8c8c',
          verticalAlign: 'middle',
          flexShrink: 0,
          ...style,
        }}
      />
    );
  }

  return (
    <img
      src={src}
      width={size}
      height={size}
      alt=""
      title={t.toUpperCase()}
      className={className}
      loading="lazy"
      decoding="async"
      style={{
        display: 'inline-block',
        verticalAlign: 'middle',
        flexShrink: 0,
        objectFit: 'contain',
        ...style,
      }}
    />
  );
};

export default DbTypeIcon;
