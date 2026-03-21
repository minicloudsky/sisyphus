import React, { useState, useRef, useEffect, useLayoutEffect, useCallback, useMemo } from 'react';
import { PageContainer } from '@ant-design/pro-components';
import {
  Card,
  Select,
  Button,
  message,
  Space,
  Table,
  Modal,
  Tree,
  Input,
  Tag,
  Alert,
  List,
  Spin,
  Divider,
  Typography,
  Tabs,
  Popconfirm,
  Form,
  Dropdown,
} from 'antd';
import type { MenuProps } from 'antd';
import Editor from '@monaco-editor/react';
import {
  PlayCircleOutlined,
  DownloadOutlined,
  DownOutlined,
  HistoryOutlined,
  StarOutlined,
  FolderOutlined,
  EditOutlined,
  DeleteOutlined,
  BugOutlined,
  TableOutlined,
  BulbOutlined,
  AlignLeftOutlined,
  CodeOutlined,
  DatabaseOutlined,
  EllipsisOutlined,
} from '@ant-design/icons';
import DbTypeIcon from '@/components/DbTypeIcon';
import { formatDateTime } from '@/utils/format';
import { isCommonResponseOk } from '@/utils/kratosResponse';
import {
  listInstances,
  listDatabases,
  listTables,
  getTableDDL,
  getTableColumns,
  getAdvisorSuggestions,
  mysqlQuery,
  redisQuery,
  clickHouseQuery,
  hbaseQuery,
  exportQuery,
  downloadQueryExport,
  listQueryHistory,
  listFavorites,
  createFavorite,
  updateFavorite,
  deleteFavorite,
  getExplain,
  generateMybatisXML,
} from '@/services/api';
import { useRequest } from 'ahooks';
import { saveAs } from 'file-saver';
import { useSplitPanePercent } from '@/hooks/useSplitPanePercent';
import { useIntl } from '@/locales';
import './index.less';

const { Text } = Typography;
const { TextArea } = Input;

const levelColor: Record<string, string> = {
  error: 'error',
  warning: 'warning',
  info: 'processing',
};


interface TreeNode {
  title: string | React.ReactNode;
  key: string;
  children?: TreeNode[];
  isLeaf?: boolean;
}

// ─── Tab data model ──────────────────────────────────────────────────────────

interface QueryTab {
  id: string;
  title: string;
  sql: string;
  instanceId: string;
  database: string;
}

const TABS_KEY = 'query_tabs_v2';
const ACTIVE_TAB_KEY = 'query_active_tab_v2';

const genId = () => `tab_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`;

function initTabs(): QueryTab[] {
  try {
    const raw = localStorage.getItem(TABS_KEY);
    if (raw) {
      const arr = JSON.parse(raw) as QueryTab[];
      if (Array.isArray(arr) && arr.length > 0) return arr;
    }
  } catch {}
  return [{ id: genId(), title: '', sql: '', instanceId: '', database: '' }];
}

function initActiveTabId(tabs: QueryTab[]): string {
  const saved = localStorage.getItem(ACTIVE_TAB_KEY);
  return tabs.find((t) => t.id === saved)?.id || tabs[0].id;
}

// ─────────────────────────────────────────────────────────────────────────────

export default () => {
  const intl = useIntl();

  // ── Tabs state ─────────────────────────────────────────────────────────────
  const [tabs, setTabsState] = useState<QueryTab[]>(initTabs);
  const [activeTabId, setActiveTabIdState] = useState<string>(() => initActiveTabId(initTabs()));
  const [renamingTabId, setRenamingTabId] = useState<string | null>(null);
  const [renameValue, setRenameValue] = useState('');

  const persistTabs = useCallback((next: QueryTab[]) => {
    localStorage.setItem(TABS_KEY, JSON.stringify(next));
  }, []);

  const setTabs = useCallback((updater: QueryTab[] | ((prev: QueryTab[]) => QueryTab[])) => {
    setTabsState((prev) => {
      const next = typeof updater === 'function' ? updater(prev) : updater;
      persistTabs(next);
      return next;
    });
  }, [persistTabs]);

  const setActiveTabId = useCallback((id: string) => {
    setActiveTabIdState(id);
    localStorage.setItem(ACTIVE_TAB_KEY, id);
  }, []);

  useLayoutEffect(() => {
    setTabs((prev) => {
      let changed = false;
      const next = prev.map((t, i) => {
        if (!t.title?.trim()) {
          changed = true;
          return {
            ...t,
            title: intl.formatMessage({ id: 'queryMain.tabTitle' }, { n: i + 1 }),
          };
        }
        return t;
      });
      return changed ? next : prev;
    });
  }, [intl.locale, setTabs]);

  const activeTab = tabs.find((t) => t.id === activeTabId) || tabs[0];

  const updateTab = useCallback((id: string, patch: Partial<QueryTab>) => {
    setTabs((prev) => prev.map((t) => (t.id === id ? { ...t, ...patch } : t)));
  }, [setTabs]);

  const updateActiveTab = useCallback((patch: Partial<QueryTab>) => {
    updateTab(activeTabId, patch);
  }, [activeTabId, updateTab]);

  // Derived "current" values from active tab
  const sqlContent = activeTab?.sql || '';
  const setSqlContent = (sql: string) => updateActiveTab({ sql });
  const selectedInstanceId = activeTab?.instanceId || '';
  const selectedDatabase = activeTab?.database || '';

  const {
    percent: querySidebarPct,
    dragging: querySidebarDragging,
    onSplitterMouseDown: onQuerySidebarSplitterMouseDown,
  } = useSplitPanePercent({
    defaultPct: 12,
    minPct: 9,
    maxPct: 36,
    storageKey: 'query_page_sidebar_pct_v2',
  });

  const addTab = () => {
    const newTab: QueryTab = {
      id: genId(),
      title: intl.formatMessage({ id: 'queryMain.tabTitle' }, { n: tabs.length + 1 }),
      sql: '',
      instanceId: selectedInstanceId,
      database: selectedDatabase,
    };
    setTabs((prev) => [...prev, newTab]);
    setActiveTabId(newTab.id);
    setQueryResult(null);
    setAdvisorVisible(false);
    setHistoryVisible(false);
  };

  const removeTab = (targetId: string) => {
    if (tabs.length === 1) return;
    setTabs((prev) => {
      const next = prev.filter((t) => t.id !== targetId);
      if (activeTabId === targetId) {
        const idx = prev.findIndex((t) => t.id === targetId);
        const newActive = (prev[idx - 1] || prev[idx + 1] || next[0]).id;
        setActiveTabIdState(newActive);
        localStorage.setItem(ACTIVE_TAB_KEY, newActive);
      }
      return next;
    });
    setQueryResult(null);
  };

  const handleTabChange = (key: string) => {
    setActiveTabId(key);
    setQueryResult(null);
    setAdvisorVisible(false);
    setHistoryVisible(false);
    setSelectedTable('');
  };

  const handleTabEdit = (
    targetKey: React.MouseEvent | React.KeyboardEvent | string,
    action: 'add' | 'remove',
  ) => {
    if (action === 'add') addTab();
    else removeTab(targetKey as string);
  };

  // ── Other UI state ─────────────────────────────────────────────────────────
  const [selectedTable, setSelectedTable] = useState<string>('');
  const [queryResult, setQueryResult] = useState<any>(null);
  const [favoritesVisible, setFavoritesVisible] = useState(false);
  const [editFavVisible, setEditFavVisible] = useState(false);
  const [editingFav, setEditingFav] = useState<API.Favorite | null>(null);
  const [editFavSaving, setEditFavSaving] = useState(false);
  const [editFavForm] = Form.useForm();
  const [historyVisible, setHistoryVisible] = useState(false);
  const [explainVisible, setExplainVisible] = useState(false);
  const [explainResult, setExplainResult] = useState<any>('');
  const [mybatisPickerVisible, setMybatisPickerVisible] = useState(false);
  const [mybatisPickerInstance, setMybatisPickerInstance] = useState<string>('');
  const [mybatisPickerDbs, setMybatisPickerDbs] = useState<string[]>([]);
  const [mybatisPickerDb, setMybatisPickerDb] = useState<string>('');
  const [mybatisPickerTables, setMybatisPickerTables] = useState<string[]>([]);
  const [mybatisPickerTable, setMybatisPickerTable] = useState<string>('');
  const [mybatisPickerDbLoading, setMybatisPickerDbLoading] = useState(false);
  const [mybatisPickerTableLoading, setMybatisPickerTableLoading] = useState(false);
  const [mybatisPickerGenerating, setMybatisPickerGenerating] = useState(false);
  const [mybatisXML, setMybatisXML] = useState('');
  const [structureVisible, setStructureVisible] = useState(false);
  const [structureColumns, setStructureColumns] = useState<any[]>([]);
  const [structureColumnsLoading, setStructureColumnsLoading] = useState(false);
  const [structureDDL, setStructureDDL] = useState('');
  const [structureActiveTab, setStructureActiveTab] = useState('columns');
  const [advisorResult, setAdvisorResult] = useState<any[]>([]);
  const [advisorLoading, setAdvisorLoading] = useState(false);
  const [advisorVisible, setAdvisorVisible] = useState(false);
  const [hbaseLimit, setHbaseLimit] = useState<number>(100);
  const [exportInsertModalOpen, setExportInsertModalOpen] = useState(false);
  const [exportInsertTableName, setExportInsertTableName] = useState('query_export');
  const [exportRunning, setExportRunning] = useState(false);
  const [treeSearch, setTreeSearch] = useState('');

  // ── Per-instance database/table cache ─────────────────────────────────────
  const [instanceDatabasesCache, setInstanceDatabasesCache] = useState<Record<string, string[]>>({});
  const [databasesLoading, setDatabasesLoading] = useState(false);
  // instanceTablesCache[instanceId][database] = string[]
  const [instanceTablesCache, setInstanceTablesCache] = useState<Record<string, Record<string, string[]>>>({});
  const [loadingTablesMap, setLoadingTablesMap] = useState<Record<string, boolean>>({});

  const databases = instanceDatabasesCache[selectedInstanceId] || [];
  const databaseTablesMap = instanceTablesCache[selectedInstanceId] || {};

  // ── Editor refs for autocomplete ──────────────────────────────────────────
  const editorRef = useRef<any>(null);
  const monacoRef = useRef<any>(null);
  const completionProviderRef = useRef<any>(null);
  const databasesRef = useRef<string[]>([]);
  const tablesRef = useRef<Record<string, string[]>>({});
  const selectedDbRef = useRef<string>('');

  useEffect(() => { return () => { completionProviderRef.current?.dispose(); }; }, []);
  useEffect(() => { databasesRef.current = databases; }, [databases]);
  useEffect(() => { tablesRef.current = databaseTablesMap; }, [databaseTablesMap]);
  useEffect(() => { selectedDbRef.current = selectedDatabase; }, [selectedDatabase]);

  // ── Data fetching ──────────────────────────────────────────────────────────

  const { data: instancesData, loading: instancesLoading } = useRequest(
    () => listInstances({ page: 1, pageSize: 1000, instancePermissionScope: 'query' }),
    { refreshDeps: [] }
  );
  const instances = instancesData?.list || [];

  const noDbRequired = (dbType?: string) => dbType === 'hbase' || dbType === 'redis';
  const selectedInstance = instances.find((i) => i.id === selectedInstanceId) || null;

  const loadDatabasesForInstance = async (instanceId: string) => {
    if (!instanceId || instanceDatabasesCache[instanceId]) return;
    setDatabasesLoading(true);
    try {
      const res = await listDatabases(instanceId);
      const dbs = (res as any).databases || [];
      setInstanceDatabasesCache((prev) => ({ ...prev, [instanceId]: dbs }));
    } catch {
    } finally {
      setDatabasesLoading(false);
    }
  };

  // When switching tabs or instances list loads, ensure databases are loaded for active tab
  useEffect(() => {
    if (instances.length === 0 || !selectedInstanceId) return;
    const instance = instances.find((i) => i.id === selectedInstanceId);
    if (instance && !noDbRequired(instance.dbType)) {
      loadDatabasesForInstance(selectedInstanceId);
    }
  }, [activeTabId, instances, selectedInstanceId]); // eslint-disable-line react-hooks/exhaustive-deps

  const loadDatabaseTables = async (database: string) => {
    if (!selectedInstanceId || instanceTablesCache[selectedInstanceId]?.[database]) return;
    setLoadingTablesMap((prev) => ({ ...prev, [database]: true }));
    try {
      const response = await listTables(selectedInstanceId, database);
      if (response.code === 200) {
        setInstanceTablesCache((prev) => ({
          ...prev,
          [selectedInstanceId]: {
            ...(prev[selectedInstanceId] || {}),
            [database]: (response as any).tables || [],
          },
        }));
      }
    } catch {
    } finally {
      setLoadingTablesMap((prev) => ({ ...prev, [database]: false }));
    }
  };

  const handleInstanceChange = (instanceId: string) => {
    const instance = instances.find((inst) => inst.id === instanceId);
    updateActiveTab({ instanceId, database: '' });
    setSelectedTable('');
    setQueryResult(null);
    if (instanceId && !noDbRequired(instance?.dbType)) {
      loadDatabasesForInstance(instanceId);
    }
  };

  // ── Editor ────────────────────────────────────────────────────────────────

  const registerCompletionProvider = (monaco: any, intlApi: typeof intl) => {
    completionProviderRef.current?.dispose();
    completionProviderRef.current = monaco.languages.registerCompletionItemProvider('sql', {
      triggerCharacters: [' ', '.', '\n', '\t'],
      provideCompletionItems: (model: any, position: any) => {
        const word = model.getWordUntilPosition(position);
        const range = {
          startLineNumber: position.lineNumber,
          endLineNumber: position.lineNumber,
          startColumn: word.startColumn,
          endColumn: word.endColumn,
        };
        const suggestions: any[] = [];
        const CK = monaco.languages.CompletionItemKind;
        const keywords = [
          'SELECT', 'FROM', 'WHERE', 'AND', 'OR', 'NOT', 'IN', 'NOT IN',
          'LIKE', 'NOT LIKE', 'IS NULL', 'IS NOT NULL', 'BETWEEN',
          'ORDER BY', 'GROUP BY', 'HAVING', 'LIMIT', 'OFFSET',
          'JOIN', 'LEFT JOIN', 'RIGHT JOIN', 'INNER JOIN', 'ON',
          'AS', 'DISTINCT', 'UNION', 'UNION ALL',
          'COUNT', 'SUM', 'AVG', 'MAX', 'MIN', 'COALESCE', 'IFNULL',
          'CASE', 'WHEN', 'THEN', 'ELSE', 'END',
          'INSERT INTO', 'VALUES', 'UPDATE', 'SET', 'DELETE FROM',
          'CREATE TABLE', 'DROP TABLE', 'ALTER TABLE',
          'NULL', 'NOT NULL', 'DEFAULT', 'AUTO_INCREMENT',
          'SHOW DATABASES', 'SHOW TABLES', 'SHOW CREATE TABLE', 'DESCRIBE', 'EXPLAIN',
        ];
        const kwDetail = intlApi.formatMessage({ id: 'queryMain.completion.keyword' });
        const dbDetail = intlApi.formatMessage({ id: 'queryMain.completion.database' });
        keywords.forEach((kw) => {
          suggestions.push({ label: kw, kind: CK.Keyword, insertText: kw, detail: kwDetail, range });
        });
        databasesRef.current.forEach((db) => {
          suggestions.push({ label: db, kind: CK.Module, insertText: db, detail: dbDetail, range });
        });
        const currentDb = selectedDbRef.current;
        const allTables = tablesRef.current;
        if (currentDb && allTables[currentDb]) {
          allTables[currentDb].forEach((t) => {
            suggestions.push({
              label: t,
              kind: CK.Class,
              insertText: t,
              detail: intlApi.formatMessage({ id: 'queryMain.completion.table' }, { db: currentDb }),
              range,
            });
          });
        } else {
          Object.entries(allTables).forEach(([db, tables]) => {
            tables.forEach((t) => {
              suggestions.push({
                label: t,
                kind: CK.Class,
                insertText: t,
                detail: intlApi.formatMessage({ id: 'queryMain.completion.table' }, { db }),
                range,
              });
            });
          });
        }
        return { suggestions };
      },
    });
  };

  const handleEditorMount = (editor: any, monaco: any) => {
    editorRef.current = editor;
    monacoRef.current = monaco;
    registerCompletionProvider(monaco, intl);
  };

  useEffect(() => {
    if (monacoRef.current) {
      registerCompletionProvider(monacoRef.current, intl);
    }
  }, [intl.locale]);

  const handleFormatSQL = () => {
    if (!sqlContent.trim()) return;
    const keywords = ['SELECT', 'FROM', 'WHERE', 'AND', 'OR', 'LEFT JOIN', 'RIGHT JOIN',
      'INNER JOIN', 'JOIN', 'ON', 'GROUP BY', 'ORDER BY', 'HAVING',
      'LIMIT', 'INSERT INTO', 'VALUES', 'UPDATE', 'SET', 'DELETE FROM', 'UNION ALL', 'UNION'];
    let sql = sqlContent.replace(/\s+/g, ' ').trim();
    keywords.forEach((kw) => {
      const re = new RegExp(`\\b(${kw})\\b`, 'gi');
      sql = sql.replace(re, `\n$1`);
    });
    sql = sql.replace(/,\s*/g, ',\n  ').replace(/\n{3,}/g, '\n').trim();
    setSqlContent(sql);
  };

  // ── Tree ──────────────────────────────────────────────────────────────────

  const handleTreeExpand = (expandedKeys: React.Key[]) => {
    expandedKeys.forEach((key) => {
      const keyStr = key as string;
      if (keyStr.startsWith('db-')) loadDatabaseTables(keyStr.replace('db-', ''));
    });
  };

  const handleTreeSelect = (selectedKeys: React.Key[]) => {
    const key = selectedKeys[0] as string;
    if (!key) return;
    if (key.startsWith('db-')) {
      const db = key.replace('db-', '');
      updateActiveTab({ database: db });
      setSelectedTable('');
      loadDatabaseTables(db);
    } else if (key.startsWith('table-')) {
      const parts = key.replace('table-', '').split('-');
      const db = parts[0];
      const table = parts.slice(1).join('-');
      updateActiveTab({ database: db });
      setSelectedTable(table);
      handleOpenStructure(db, table);
    }
  };

  const buildTreeData = (): TreeNode[] => {
    if (!selectedInstanceId || databases.length === 0) return [];
    const search = treeSearch.toLowerCase();
    const filteredDbs = search
      ? databases.filter((db) => {
          if (db.toLowerCase().includes(search)) return true;
          return (databaseTablesMap[db] || []).some((t) => t.toLowerCase().includes(search));
        })
      : databases;
    return filteredDbs.map((db) => {
      const tables = databaseTablesMap[db] || [];
      const filteredTables = search
        ? tables.filter((t) => t.toLowerCase().includes(search) || db.toLowerCase().includes(search))
        : tables;
      return {
        title: (
          <span>
            <DatabaseOutlined style={{ color: '#1677ff', marginRight: 6 }} />
            {db}
            {loadingTablesMap[db] && (
              <span style={{ marginLeft: 8, color: '#999', fontSize: '12px' }}>
                {intl.formatMessage({ id: 'common.loading' })}
              </span>
            )}
          </span>
        ),
        key: `db-${db}`,
        children: filteredTables.map((table) => ({
          title: (
            <span>
              <TableOutlined style={{ color: '#52c41a', marginRight: 6 }} />
              {table}
            </span>
          ),
          key: `table-${db}-${table}`,
          isLeaf: true,
        })),
      };
    });
  };

  // ── Query ─────────────────────────────────────────────────────────────────

  const { loading: queryLoading, run: handleQuery } = useRequest(
    async () => {
      const instance = instances.find((inst) => inst.id === selectedInstanceId);
      if (!selectedInstanceId) { message.warning(intl.formatMessage({ id: 'queryMain.warnInstance' })); return; }
      if (!instance) { message.error(intl.formatMessage({ id: 'queryMain.errNoInstance' })); return; }
      if (!noDbRequired(instance.dbType) && !selectedDatabase) { message.warning(intl.formatMessage({ id: 'queryMain.warnDatabase' })); return; }
      if (!sqlContent.trim()) { message.warning(intl.formatMessage({ id: 'queryMain.warnSql' })); return; }
      let response: any;
      if (instance.dbType === 'mysql') {
        response = await mysqlQuery({ instanceId: selectedInstanceId, database: selectedDatabase, sql: sqlContent.trim() });
      } else if (instance.dbType === 'clickhouse') {
        response = await clickHouseQuery({ instanceId: selectedInstanceId, database: selectedDatabase, sql: sqlContent.trim() });
      } else if (instance.dbType === 'redis') {
        response = await redisQuery({ instanceId: selectedInstanceId, command: sqlContent.trim() });
      } else if (instance.dbType === 'hbase') {
        response = await hbaseQuery({ instanceId: selectedInstanceId, sql: sqlContent.trim(), limit: hbaseLimit });
      } else {
        message.error(intl.formatMessage({ id: 'queryMain.errDbType' }));
        return;
      }
      if (response.code === 200 && response.data) {
        setQueryResult({ columns: response.data.columns || [], rows: response.data.rows || [] });
        const rowCount = response.data.rows?.length || 0;
        message.success(
          response.data.time
            ? intl.formatMessage(
                { id: 'queryMain.okRowsTime' },
                { rows: rowCount, time: response.data.time }
              )
            : intl.formatMessage({ id: 'queryMain.okRows' }, { rows: rowCount })
        );
      } else {
        message.error(response.message || intl.formatMessage({ id: 'queryMain.fail' }));
      }
    },
    { manual: true }
  );

  // ── Export ────────────────────────────────────────────────────────────────

  const runQueryExport = useCallback(
    async (format: 'excel' | 'csv' | 'json' | 'sql', insertTableName?: string): Promise<boolean> => {
      if (!sqlContent.trim()) {
        message.warning(intl.formatMessage({ id: 'queryMain.warnSqlFirst' }));
        return false;
      }
      const inst = instances.find((i) => i.id === selectedInstanceId);
      if (!inst) {
        message.warning(intl.formatMessage({ id: 'queryMain.warnInstance' }));
        return false;
      }
      if (inst.dbType !== 'mysql' && inst.dbType !== 'clickhouse') {
        message.error(intl.formatMessage({ id: 'queryMain.errExportEngines' }));
        return false;
      }
      if ((inst.dbType === 'mysql' || inst.dbType === 'clickhouse') && !selectedDatabase) {
        message.warning(intl.formatMessage({ id: 'queryMain.warnInstanceDb' }));
        return false;
      }
      if (format === 'sql' && inst.dbType !== 'mysql') {
        message.error(intl.formatMessage({ id: 'queryMain.errExportInsertMysql' }));
        return false;
      }
      try {
        setExportRunning(true);
        const payload: Record<string, unknown> = {
          instanceId: selectedInstanceId,
          database: selectedDatabase,
          sql: sqlContent.trim(),
          format,
        };
        if (format === 'sql') {
          payload.table_name = (insertTableName ?? 'query_export').trim() || 'query_export';
        }
        const res = await exportQuery(payload as any);
        if (!isCommonResponseOk(res)) {
          message.error(res?.message || intl.formatMessage({ id: 'queryMain.exportFail' }));
          return false;
        }
        const d = res.data;
        const downloadUrl = d?.downloadUrl || d?.download_url;
        if (!downloadUrl) {
          message.error(intl.formatMessage({ id: 'queryMain.exportFail' }));
          return false;
        }
        const blob = await downloadQueryExport(downloadUrl);
        const ext = format === 'excel' ? 'xlsx' : format;
        const fileName = d.fileName || d.file_name || `query_result.${ext}`;
        saveAs(blob, fileName);
        message.success(intl.formatMessage({ id: 'queryMain.exportOk' }));
        return true;
      } catch {
        return false;
      } finally {
        setExportRunning(false);
      }
    },
    [sqlContent, selectedInstanceId, selectedDatabase, instances, intl],
  );

  const exportMenuItems: MenuProps['items'] = useMemo(() => {
    const items: MenuProps['items'] = [
      { key: 'excel', label: intl.formatMessage({ id: 'queryMain.export.excel' }) },
      { key: 'csv', label: intl.formatMessage({ id: 'queryMain.export.csv' }) },
      { key: 'json', label: intl.formatMessage({ id: 'queryMain.export.json' }) },
    ];
    if (selectedInstance?.dbType === 'mysql') {
      items.push({ type: 'divider' });
      items.push({
        key: 'sql',
        label: intl.formatMessage({ id: 'queryMain.export.insertSql' }),
      });
    }
    return items;
  }, [intl, selectedInstance?.dbType]);

  const onExportMenuClick: MenuProps['onClick'] = useCallback(
    ({ key }) => {
      if (key === 'sql') {
        setExportInsertTableName('query_export');
        setExportInsertModalOpen(true);
        return;
      }
      void runQueryExport(key as 'excel' | 'csv' | 'json');
    },
    [runQueryExport],
  );

  const getSelectedSQL = () => {
    if (editorRef.current) {
      const selection = editorRef.current.getSelection();
      if (selection && !selection.isEmpty()) {
        return editorRef.current.getModel()?.getValueInRange(selection) || '';
      }
    }
    return '';
  };

  // ── Explain ───────────────────────────────────────────────────────────────

  const { loading: explainLoading, run: handleExplain } = useRequest(
    async () => {
      if (!selectedInstanceId || !selectedDatabase) { message.warning(intl.formatMessage({ id: 'queryMain.warnInstanceDb' })); return; }
      const sqlToExplain = getSelectedSQL() || sqlContent.trim();
      if (!sqlToExplain) { message.warning(intl.formatMessage({ id: 'queryMain.warnSqlFirst' })); return; }
      try {
        const response = await getExplain(selectedInstanceId, selectedDatabase, sqlToExplain);
        if (response.code === 200) {
          try { setExplainResult(JSON.parse(response.explain || '[]')); }
          catch { setExplainResult(response.explain || ''); }
          setExplainVisible(true);
        } else {
          message.error(response.message || intl.formatMessage({ id: 'queryMain.explainFail' }));
        }
      } catch {
      }
    },
    { manual: true }
  );

  // ── History ───────────────────────────────────────────────────────────────

  const { data: historyData, loading: historyLoading, run: loadHistory } = useRequest(
    () => listQueryHistory({ page: 1, pageSize: 50 }),
    { manual: true }
  );
  const historyList = historyData?.data?.list || [];

  // ── Favorites ─────────────────────────────────────────────────────────────

  const { data: favoritesData, loading: favoritesLoading, run: loadFavorites } = useRequest(
    () => listFavorites({ page: 1, pageSize: 100 }),
    { manual: true }
  );
  const favoritesList = favoritesData?.favorites || [];

  const handleSaveFavorite = async () => {
    if (!sqlContent.trim()) { message.warning(intl.formatMessage({ id: 'queryMain.warnSqlFirst' })); return; }
    try {
      const response = await createFavorite({
        instanceId: selectedInstanceId,
        databaseName: selectedDatabase,
        sqlText: sqlContent.trim(),
        name: intl.formatMessage({ id: 'queryMain.favTitlePrefix' }, { time: formatDateTime(new Date()) }),
      });
      if (Number((response as { id?: number }).id) > 0) {
        message.success(intl.formatMessage({ id: 'queryMain.favOk' }));
        loadFavorites();
      } else {
        message.error((response as { message?: string }).message || intl.formatMessage({ id: 'queryMain.favFail' }));
      }
    } catch {
    }
  };

  const handleUseFavorite = (sql: string) => {
    setSqlContent(sql);
    setFavoritesVisible(false);
    message.success(intl.formatMessage({ id: 'queryMain.favLoaded' }));
  };

  const handleEditFav = (record: API.Favorite) => {
    setEditingFav(record);
    const r = record as API.Favorite & { title?: string };
    editFavForm.setFieldsValue({
      name: r.name || r.title || '',
      sqlText: record.sqlText,
    });
    setEditFavVisible(true);
  };

  const handleSaveEditFav = async () => {
    if (!editingFav) return;
    const values = await editFavForm.validateFields();
    setEditFavSaving(true);
    try {
      const res = await updateFavorite(editingFav.id, {
        name: values.name,
        sqlText: values.sqlText,
      });
      if (isCommonResponseOk(res as { code?: number; message?: string })) {
        message.success(intl.formatMessage({ id: 'queryMain.favEditOk' }));
        setEditFavVisible(false);
        setEditingFav(null);
        loadFavorites();
      } else {
        message.error((res as any).message || intl.formatMessage({ id: 'queryMain.favEditFail' }));
      }
    } catch {
    } finally {
      setEditFavSaving(false);
    }
  };

  const handleDeleteFav = async (id: string) => {
    try {
      const res = await deleteFavorite(id);
      if (isCommonResponseOk(res as { code?: number; message?: string })) {
        message.success(intl.formatMessage({ id: 'queryMain.favDeleted' }));
        loadFavorites();
      } else {
        message.error((res as any).message || intl.formatMessage({ id: 'queryMain.favDeleteFail' }));
      }
    } catch {
    }
  };

  // ── Table structure ───────────────────────────────────────────────────────

  const handleOpenStructure = async (db: string, table: string, tab = 'columns') => {
    setStructureVisible(true);
    setStructureActiveTab(tab);
    setStructureColumns([]);
    setStructureDDL('');
    setStructureColumnsLoading(true);
    try {
      const [colRes, ddlRes] = await Promise.all([
        getTableColumns(selectedInstanceId, db, table),
        getTableDDL(selectedInstanceId, db, table),
      ]);
      setStructureColumns(colRes?.columns || []);
      setStructureDDL(ddlRes?.ddl || '');
    } catch {
    } finally {
      setStructureColumnsLoading(false);
    }
  };

  const handleGetAdvisor = async () => {
    const sqlToAnalyze = getSelectedSQL() || sqlContent.trim();
    if (!sqlToAnalyze) { message.warning(intl.formatMessage({ id: 'queryMain.advisorNeedSql' })); return; }
    if (!selectedInstanceId) { message.warning(intl.formatMessage({ id: 'queryMain.warnInstance' })); return; }
    setAdvisorLoading(true);
    setAdvisorVisible(true);
    setAdvisorResult([]);
    try {
      const res = await getAdvisorSuggestions(selectedInstanceId, {
        database: selectedDatabase,
        table: selectedTable,
        sql: sqlToAnalyze,
      });
      if (res?.code === 200) {
        setAdvisorResult(res.suggestions || []);
        if (!res.suggestions?.length) message.info(intl.formatMessage({ id: 'queryMain.advisorNoTips' }));
      } else {
        message.error(res?.message || intl.formatMessage({ id: 'queryMain.advisorFail' }));
      }
    } catch {
    } finally {
      setAdvisorLoading(false);
    }
  };

  // ── MyBatis picker ────────────────────────────────────────────────────────

  const handleOpenMybatisPicker = () => {
    setMybatisPickerInstance(selectedInstanceId || '');
    setMybatisPickerDb('');
    setMybatisPickerTable('');
    setMybatisPickerDbs([]);
    setMybatisPickerTables([]);
    setMybatisXML('');
    setMybatisPickerVisible(true);
    if (selectedInstanceId) {
      setMybatisPickerDbLoading(true);
      listDatabases(selectedInstanceId)
        .then((res) => setMybatisPickerDbs((res as any).databases || []))
        .catch(() => {})
        .finally(() => setMybatisPickerDbLoading(false));
    }
  };

  const handleMybatisPickerInstanceChange = async (instanceId: string) => {
    setMybatisPickerInstance(instanceId);
    setMybatisPickerDb('');
    setMybatisPickerTable('');
    setMybatisPickerDbs([]);
    setMybatisPickerTables([]);
    if (!instanceId) return;
    setMybatisPickerDbLoading(true);
    try {
      const res = await listDatabases(instanceId);
      setMybatisPickerDbs((res as any).databases || []);
    } catch {
    } finally {
      setMybatisPickerDbLoading(false);
    }
  };

  const handleMybatisPickerDbChange = async (db: string) => {
    setMybatisPickerDb(db);
    setMybatisPickerTable('');
    setMybatisPickerTables([]);
    if (!db || !mybatisPickerInstance) return;
    setMybatisPickerTableLoading(true);
    try {
      const res = await listTables(mybatisPickerInstance, db);
      setMybatisPickerTables((res as any).tables || []);
    } catch {
    } finally {
      setMybatisPickerTableLoading(false);
    }
  };

  const handleMybatisGenerate = async () => {
    if (!mybatisPickerInstance || !mybatisPickerDb || !mybatisPickerTable) {
      message.warning(intl.formatMessage({ id: 'queryMain.mybatisPick' }));
      return;
    }
    setMybatisPickerGenerating(true);
    try {
      const ddlRes = await getTableDDL(mybatisPickerInstance, mybatisPickerDb, mybatisPickerTable);
      if (!ddlRes?.ddl) { message.error(intl.formatMessage({ id: 'queryMain.ddlFail' })); return; }
      const xmlRes = await generateMybatisXML({ ddl: ddlRes.ddl });
      if (xmlRes.code === 200 && xmlRes.data) {
        setMybatisXML(xmlRes.data.xml || '');
        message.success(intl.formatMessage({ id: 'queryMain.mybatisOk' }));
      } else {
        message.error(xmlRes.message || intl.formatMessage({ id: 'queryMain.mybatisFail' }));
      }
    } catch {
    } finally {
      setMybatisPickerGenerating(false);
    }
  };

  // ── Result columns ────────────────────────────────────────────────────────

  const resultColumns = queryResult?.columns?.map((col: string) => ({
    title: col, dataIndex: col, key: col, ellipsis: true,
  })) || [];

  const resultDataSource = queryResult?.rows?.map((row: any, index: number) => {
    const record: any = { key: index };
    const values: string[] = Array.isArray(row) ? row : (row?.values || []);
    queryResult.columns.forEach((col: string, i: number) => { record[col] = values[i]; });
    return record;
  }) || [];

  const structureColumnDefs = useMemo(
    () => [
      { title: intl.formatMessage({ id: 'queryMain.struct.col.name' }), dataIndex: 'name', key: 'name', width: 150 },
      {
        title: intl.formatMessage({ id: 'queryMain.struct.col.type' }),
        dataIndex: 'type',
        key: 'type',
        width: 150,
        render: (t: string) => <Text code>{t}</Text>,
      },
      {
        title: intl.formatMessage({ id: 'queryMain.struct.col.nullable' }),
        dataIndex: 'null',
        key: 'null',
        width: 70,
        render: (v: string) =>
          v === 'YES' ? (
            <Tag>{intl.formatMessage({ id: 'common.yes' })}</Tag>
          ) : (
            <Tag color="green">{intl.formatMessage({ id: 'common.no' })}</Tag>
          ),
      },
      {
        title: intl.formatMessage({ id: 'queryMain.struct.col.key' }),
        dataIndex: 'key',
        key: 'key',
        width: 60,
        render: (v: string) =>
          v ? <Tag color={v === 'PRI' ? 'gold' : v === 'UNI' ? 'blue' : 'default'}>{v}</Tag> : null,
      },
      {
        title: intl.formatMessage({ id: 'queryMain.struct.col.default' }),
        dataIndex: 'default',
        key: 'default',
        width: 120,
        render: (v: string) => v || <Text type="secondary">NULL</Text>,
      },
      {
        title: intl.formatMessage({ id: 'queryMain.struct.col.comment' }),
        dataIndex: 'comment',
        key: 'comment',
        ellipsis: true,
      },
    ],
    [intl]
  );

  // ── Tab label with double-click rename ────────────────────────────────────

  const renderTabLabel = (tab: QueryTab) => {
    if (renamingTabId === tab.id) {
      return (
        <Input
          size="small"
          autoFocus
          value={renameValue}
          style={{ width: 80, fontSize: 13 }}
          onChange={(e) => setRenameValue(e.target.value)}
          onBlur={() => {
            if (renameValue.trim()) updateTab(tab.id, { title: renameValue.trim() });
            setRenamingTabId(null);
          }}
          onPressEnter={() => {
            if (renameValue.trim()) updateTab(tab.id, { title: renameValue.trim() });
            setRenamingTabId(null);
          }}
          onClick={(e) => e.stopPropagation()}
        />
      );
    }
    return (
      <span
        onDoubleClick={(e) => {
          e.stopPropagation();
          setRenamingTabId(tab.id);
          setRenameValue(tab.title);
        }}
        title={intl.formatMessage({ id: 'queryMain.tabDblclickRename' })}
      >
        {tab.title}
      </span>
    );
  };

  // ── JSX ───────────────────────────────────────────────────────────────────

  const dbType = selectedInstance?.dbType;
  const sidebarTitleInstance = instances.find((i) => i.id === selectedInstanceId);

  return (
    <PageContainer
      title={intl.formatMessage({ id: 'pageTitle.query' })}
      extra={
        <Space>
          <Button type="primary" icon={<PlayCircleOutlined />} loading={queryLoading} onClick={handleQuery}>
            {intl.formatMessage({ id: 'queryMain.btn.run' })}
          </Button>
          <Dropdown menu={{ items: exportMenuItems, onClick: onExportMenuClick }}>
            <Button icon={<DownloadOutlined />} style={{ color: '#13c2c2', borderColor: '#13c2c2' }}>
              {intl.formatMessage({ id: 'queryMain.btn.export' })} <DownOutlined />
            </Button>
          </Dropdown>
          <Button icon={<AlignLeftOutlined />} onClick={handleFormatSQL}>
            {intl.formatMessage({ id: 'queryMain.btn.format' })}
          </Button>
          <Button icon={<BugOutlined />} style={{ color: '#722ed1', borderColor: '#722ed1' }} loading={explainLoading} onClick={handleExplain}>
            {intl.formatMessage({ id: 'queryMain.btn.explain' })}
          </Button>
          <Button
            icon={<HistoryOutlined />}
            onClick={() => { if (!historyVisible) loadHistory(); setHistoryVisible((v) => !v); }}
          >
            {intl.formatMessage({ id: 'queryMain.btn.history' })}
          </Button>
          <Button icon={<StarOutlined />} style={{ color: '#fa8c16', borderColor: '#fa8c16' }} onClick={handleSaveFavorite}>
            {intl.formatMessage({ id: 'queryMain.btn.fav' })}
          </Button>
          <Button icon={<FolderOutlined />} onClick={() => { loadFavorites(); setFavoritesVisible(true); }}>
            {intl.formatMessage({ id: 'queryMain.btn.favorites' })}
          </Button>
          <Button icon={<BulbOutlined />} style={{ color: '#fa8c16', borderColor: '#fa8c16' }} loading={advisorLoading} onClick={handleGetAdvisor}>
            {intl.formatMessage({ id: 'queryMain.btn.advisor' })}
          </Button>
          <Button icon={<CodeOutlined />} style={{ color: '#531dab', borderColor: '#531dab' }} onClick={handleOpenMybatisPicker}>
            {intl.formatMessage({ id: 'queryMain.btn.mybatis' })}
          </Button>
        </Space>
      }
    >
      <div className="query-page-split">
        {/* Left: instance selector + DB tree */}
        <div
          className="query-page-split__sidebar"
          style={{
            width: `${querySidebarPct}%`,
            flex: `0 0 ${querySidebarPct}%`,
            transition: querySidebarDragging ? 'none' : 'width 0.15s ease-out',
          }}
        >
          <Card
            size="small"
            className="query-page-split__sidebar-card"
            title={
              <Space size={8}>
                {sidebarTitleInstance ? (
                  <DbTypeIcon dbType={sidebarTitleInstance.dbType} size={20} />
                ) : (
                  <DatabaseOutlined style={{ color: '#1677ff', fontSize: 18 }} />
                )}
                <span>{intl.formatMessage({ id: 'queryMain.sidebarDb' })}</span>
              </Space>
            }
          >
            <Select
              placeholder={intl.formatMessage({ id: 'queryMain.ph.instance' })}
              style={{ width: '100%', marginBottom: 10 }}
              loading={instancesLoading}
              value={selectedInstanceId || undefined}
              onChange={handleInstanceChange}
              showSearch
              filterOption={(input, option) =>
                String(option?.label ?? '').toLowerCase().includes(input.toLowerCase())
              }
              options={instances.map((inst) => ({
                label: (
                  <Space size={4}>
                    <DbTypeIcon dbType={inst.dbType} />
                    {inst.name}
                    <Text type="secondary" style={{ fontSize: '12px' }}>({inst.dbType})</Text>
                  </Space>
                ),
                value: inst.id,
              }))}
            />

            {selectedInstanceId && dbType === 'redis' && (
              <div style={{ padding: '8px', background: '#fff7e6', borderRadius: 4, border: '1px solid #ffd591' }}>
                <div style={{ fontSize: '12px', color: '#d46b08', marginBottom: 4 }}>{intl.formatMessage({ id: 'queryMain.redisHint' })}</div>
                <div style={{ fontSize: '12px', color: '#999', lineHeight: '1.8' }}>
                  <div>GET key</div><div>HGETALL key</div><div>KEYS pattern</div><div>LRANGE key 0 10</div>
                </div>
              </div>
            )}

            {selectedInstanceId && dbType === 'hbase' && (
              <div style={{ padding: '8px', background: '#f6ffed', borderRadius: 4, border: '1px solid #b7eb8f' }}>
                <div style={{ fontSize: '12px', color: '#389e0d', marginBottom: 4 }}>{intl.formatMessage({ id: 'queryMain.hbaseHint' })}</div>
                <div style={{ fontSize: '12px', color: '#999', lineHeight: '1.8' }}>
                  <div>GET 'table', 'rowkey'</div><div>SCAN 'table'</div>
                </div>
                <div style={{ marginTop: 8 }}>
                  <div style={{ marginBottom: 4, fontSize: '12px', color: '#666' }}>{intl.formatMessage({ id: 'queryMain.hbaseLimit' })}</div>
                  <Input
                    type="number" size="small" value={hbaseLimit}
                    onChange={(e) => setHbaseLimit(Number(e.target.value) || 100)}
                    style={{ width: '100%' }} min={1} max={10000}
                  />
                </div>
              </div>
            )}

            {selectedInstanceId && !noDbRequired(dbType) && (
              <>
                <Input.Search
                  placeholder={intl.formatMessage({ id: 'queryMain.searchTree' })}
                  allowClear size="small" style={{ marginBottom: 8 }}
                  value={treeSearch}
                  onChange={(e) => setTreeSearch(e.target.value)}
                />
                <Tree
                  treeData={buildTreeData()}
                  onSelect={handleTreeSelect}
                  onExpand={handleTreeExpand}
                  showLine
                  defaultExpandAll={false}
                />
              </>
            )}

            {selectedTable && (
              <div style={{ marginTop: 10, padding: '6px 8px', background: '#f0f5ff', borderRadius: 4, border: '1px solid #adc6ff', fontSize: '12px', color: '#1d3461' }}>
                <TableOutlined style={{ marginRight: 4, color: '#52c41a' }} />
                <Text code style={{ fontSize: '11px' }}>{selectedDatabase}.{selectedTable}</Text>
              </div>
            )}
          </Card>
        </div>

        <div
          className={`query-page-split__handle${querySidebarDragging ? ' query-page-split__handle--active' : ''}`}
          onMouseDown={onQuerySidebarSplitterMouseDown}
        />

        {/* Right: query tabs + editor + results */}
        <div className="query-page-split__main">
          {/* Query window tabs */}
          <Tabs
            type="editable-card"
            activeKey={activeTabId}
            onChange={handleTabChange}
            onEdit={handleTabEdit}
            style={{ marginBottom: 0 }}
            items={tabs.map((tab) => ({
              key: tab.id,
              label: renderTabLabel(tab),
              closable: tabs.length > 1,
            }))}
          />

          <Card
            title={intl.formatMessage({ id: 'queryMain.editorTitle' })}
            style={{ borderTopLeftRadius: 0 }}
            styles={{ header: { borderTopLeftRadius: 0 } }}
          >
            <Editor
              key={activeTabId}
              height="300px"
              defaultLanguage="sql"
              value={sqlContent}
              onChange={(value) => setSqlContent(value || '')}
              onMount={handleEditorMount}
              theme="vs"
              options={{
                minimap: { enabled: false },
                fontSize: 14,
                wordWrap: 'on',
                automaticLayout: true,
                suggestOnTriggerCharacters: true,
                quickSuggestions: { other: true, comments: false, strings: true },
                tabCompletion: 'on',
              }}
            />
          </Card>

          {/* Inline query history */}
          {historyVisible && (
            <Card
              style={{ marginTop: 12 }}
              title={<Space><HistoryOutlined style={{ color: '#1677ff' }} /><span>{intl.formatMessage({ id: 'queryHist.pageTitle' })}</span></Space>}
              extra={<Button size="small" onClick={() => setHistoryVisible(false)}>{intl.formatMessage({ id: 'queryMain.close' })}</Button>}
              size="small"
            >
              <Table
                columns={[
                  { title: intl.formatMessage({ id: 'queryMain.history.sqlCol' }), dataIndex: 'sqlText', key: 'sqlText', ellipsis: true, render: (text) => <div style={{ maxWidth: 500, wordBreak: 'break-all', fontSize: '12px' }}>{text}</div> },
                  { title: intl.formatMessage({ id: 'queryMain.sidebarDb' }), dataIndex: 'databaseName', key: 'databaseName', width: 130 },
                  { title: intl.formatMessage({ id: 'common.time' }), dataIndex: 'createdAt', key: 'createdAt', width: 170, render: (text: string) => formatDateTime(text) },
                  { title: intl.formatMessage({ id: 'home.col.action' }), key: 'action', width: 80, render: (_: any, record: any) => <Button type="link" size="small" onClick={() => { setSqlContent(record.sqlText || ''); setHistoryVisible(false); }}>{intl.formatMessage({ id: 'common.use' })}</Button> },
                ]}
                dataSource={historyList}
                loading={historyLoading}
                rowKey="id"
                pagination={{ pageSize: 10, showSizeChanger: true }}
                size="small"
              />
            </Card>
          )}

          {/* Inline advisor results */}
          {advisorVisible && (
            <Card
              style={{ marginTop: 12 }}
              title={
                <Space>
                  <BulbOutlined style={{ color: '#fa8c16' }} />
                  <span>{intl.formatMessage({ id: 'queryMain.advisorTitle' })}</span>
                  {selectedTable && (
                    <span style={{ fontSize: '12px', color: '#8c8c8c', fontWeight: 400 }}>
                      {intl.formatMessage(
                        { id: 'queryMain.advisorBased' },
                        { schema: selectedDatabase, table: selectedTable }
                      )}
                    </span>
                  )}
                </Space>
              }
              extra={<Button size="small" onClick={() => { setAdvisorVisible(false); setAdvisorResult([]); }}>{intl.formatMessage({ id: 'queryMain.close' })}</Button>}
              size="small"
            >
              <Spin spinning={advisorLoading}>
                {advisorResult.length > 0 ? (
                  <List
                    size="small"
                    dataSource={advisorResult}
                    renderItem={(item: any) => (
                      <List.Item style={{ padding: '4px 0' }}>
                        <Alert
                          type={item.level === 'error' ? 'error' : item.level === 'warning' ? 'warning' : 'info'}
                          message={<Space><Tag color={levelColor[item.level] || 'default'}>{item.rule}</Tag>{item.message}</Space>}
                          description={item.detail}
                          style={{ width: '100%' }}
                          showIcon
                        />
                      </List.Item>
                    )}
                  />
                ) : (
                  !advisorLoading && (
                    <div style={{ textAlign: 'center', padding: '16px', color: '#52c41a' }}>
                      {intl.formatMessage({ id: 'queryMain.advisorGood' })}
                    </div>
                  )
                )}
              </Spin>
            </Card>
          )}

          <Card style={{ marginTop: 16 }}>
            {queryResult ? (
              <div>
                <div style={{ marginBottom: 8, color: '#666', fontSize: '14px' }}>
                  {intl.formatMessage({ id: 'queryMain.records' }, { n: resultDataSource.length })}
                </div>
                <Table
                  columns={resultColumns}
                  dataSource={resultDataSource}
                  scroll={{ x: 'max-content', y: 400 }}
                  pagination={{ pageSize: 100, showSizeChanger: true, showTotal: (total) => intl.formatMessage({ id: 'queryMain.showTotal' }, { n: total }), pageSizeOptions: ['50', '100', '200', '500'] }}
                  size="small"
                />
              </div>
            ) : (
              <div style={{ textAlign: 'center', padding: '40px', color: '#999' }}>
                {intl.formatMessage({ id: 'queryMain.emptyHint' })}
              </div>
            )}
          </Card>
        </div>
      </div>

      <Modal
        title={intl.formatMessage({ id: 'queryMain.export.insertTitle' })}
        open={exportInsertModalOpen}
        onCancel={() => setExportInsertModalOpen(false)}
        confirmLoading={exportRunning}
        destroyOnClose
        onOk={async () => {
          const ok = await runQueryExport('sql', exportInsertTableName);
          if (ok) setExportInsertModalOpen(false);
        }}
      >
        <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
          {intl.formatMessage({ id: 'queryMain.export.tableNameTip' })}
        </Typography.Paragraph>
        <Input
          placeholder="query_export"
          value={exportInsertTableName}
          onChange={(e) => setExportInsertTableName(e.target.value)}
          onPressEnter={async () => {
            const ok = await runQueryExport('sql', exportInsertTableName);
            if (ok) setExportInsertModalOpen(false);
          }}
        />
      </Modal>

      {/* Favorites modal */}
      <Modal
        title={<Space><StarOutlined style={{ color: '#fa8c16' }} /><span>{intl.formatMessage({ id: 'queryMain.favModal' })}</span></Space>}
        open={favoritesVisible}
        onCancel={() => setFavoritesVisible(false)}
        footer={null}
        width={960}
        styles={{ body: { paddingTop: 12 } }}
      >
        <Table
          columns={[
            {
              title: intl.formatMessage({ id: 'queryMain.favName' }),
              dataIndex: 'name',
              key: 'name',
              width: 160,
              ellipsis: true,
              render: (text: string, record: API.Favorite & { title?: string }) =>
                text || record.title || '—',
            },
            {
              title: intl.formatMessage({ id: 'common.sql' }),
              dataIndex: 'sqlText',
              key: 'sqlText',
              ellipsis: true,
              render: (text: string) => (
                <Text
                  code
                  style={{ fontSize: 12, color: '#595959', display: 'block', maxWidth: 420 }}
                  ellipsis={{ tooltip: text }}
                >
                  {text}
                </Text>
              ),
            },
            { title: intl.formatMessage({ id: 'queryMain.sidebarDb' }), dataIndex: 'databaseName', key: 'databaseName', width: 120, ellipsis: true },
            {
              title: intl.formatMessage({ id: 'home.col.action' }),
              key: 'action',
              width: 96,
              fixed: 'right',
              align: 'center',
              render: (_: any, record: API.Favorite) => {
                const menuItems: MenuProps['items'] = [
                  {
                    key: 'use',
                    label: intl.formatMessage({ id: 'queryMain.favUseSql' }),
                    onClick: () => handleUseFavorite(record.sqlText || ''),
                  },
                  {
                    key: 'edit',
                    label: intl.formatMessage({ id: 'common.edit' }),
                    icon: <EditOutlined />,
                    onClick: () => handleEditFav(record),
                  },
                  { type: 'divider' },
                  {
                    key: 'del',
                    danger: true,
                    label: intl.formatMessage({ id: 'common.delete' }),
                    icon: <DeleteOutlined />,
                    onClick: () => {
                      Modal.confirm({
                        title: intl.formatMessage({ id: 'queryMain.favConfirmDel' }),
                        okText: intl.formatMessage({ id: 'common.delete' }),
                        okButtonProps: { danger: true },
                        onOk: () => handleDeleteFav(record.id),
                      });
                    },
                  },
                ];
                return (
                  <Dropdown menu={{ items: menuItems }} trigger={['click']}>
                    <Button type="link" size="small" icon={<EllipsisOutlined />}>
                      {intl.formatMessage({ id: 'home.col.action' })}
                    </Button>
                  </Dropdown>
                );
              },
            },
          ]}
          dataSource={favoritesList}
          loading={favoritesLoading}
          rowKey="id"
          pagination={{ pageSize: 10, showSizeChanger: true, showTotal: (t) => intl.formatMessage({ id: 'queryMain.showTotal' }, { n: t }) }}
          size="small"
          scroll={{ x: 820 }}
        />
      </Modal>

      {/* Edit favorite modal */}
      <Modal
        title={<Space><EditOutlined style={{ color: '#1677ff' }} /><span>{intl.formatMessage({ id: 'queryMain.favEditModal' })}</span></Space>}
        open={editFavVisible}
        onCancel={() => { setEditFavVisible(false); setEditingFav(null); }}
        onOk={handleSaveEditFav}
        okText={intl.formatMessage({ id: 'common.save' })}
        confirmLoading={editFavSaving}
        width={720}
        destroyOnClose
      >
        <Form form={editFavForm} layout="vertical" style={{ marginTop: 12 }}>
          <Form.Item
            name="name"
            label={intl.formatMessage({ id: 'queryMain.favNameLabel' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'queryMain.favNameRule' }) }]}
          >
            <Input placeholder={intl.formatMessage({ id: 'queryMain.favNamePh' })} maxLength={100} showCount />
          </Form.Item>
          <Form.Item
            name="sqlText"
            label={intl.formatMessage({ id: 'queryMain.favSqlLabel' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'queryMain.favSqlRule' }) }]}
          >
            <TextArea
              rows={10}
              placeholder="SELECT ..."
              style={{ fontFamily: 'monospace', fontSize: 13 }}
              autoSize={{ minRows: 6, maxRows: 20 }}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* Explain modal */}
      <Modal title={intl.formatMessage({ id: 'queryMain.explainTitle' })} open={explainVisible} onCancel={() => { setExplainVisible(false); setExplainResult(''); }} footer={null} width={1000}>
        {explainResult ? (
          Array.isArray(explainResult) ? (
            <Table
              dataSource={explainResult.map((item: any, i: number) => ({ ...item, key: i }))}
              columns={Object.keys(explainResult[0] || {}).map((key) => ({ title: key, dataIndex: key, key, ellipsis: true }))}
              pagination={false} size="small" scroll={{ x: 'max-content' }}
            />
          ) : (
            <pre style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word', maxHeight: '600px', overflow: 'auto' }}>
              {typeof explainResult === 'object' ? JSON.stringify(explainResult, null, 2) : explainResult}
            </pre>
          )
        ) : (
          <div style={{ textAlign: 'center', padding: '40px', color: '#999' }}>{intl.formatMessage({ id: 'queryMain.explainEmpty' })}</div>
        )}
      </Modal>

      {/* Table structure modal */}
      <Modal
        title={
          <>
            <TableOutlined style={{ color: '#1677ff', marginRight: 8 }} />
            {selectedTable
              ? intl.formatMessage(
                  { id: 'queryMain.struct.modalWithTable' },
                  { schema: selectedDatabase, table: selectedTable }
                )
              : intl.formatMessage({ id: 'queryMain.struct.modal' })}
          </>
        }
        open={structureVisible}
        onCancel={() => setStructureVisible(false)}
        footer={null}
        width={900}
        styles={{ body: { padding: '12px 24px 24px' } }}
      >
        <Tabs
          activeKey={structureActiveTab}
          onChange={setStructureActiveTab}
          items={[
            {
              key: 'columns',
              label: <><TableOutlined /> {intl.formatMessage({ id: 'queryMain.structColInfo' })}</>,
              children: (
                <Spin spinning={structureColumnsLoading}>
                  <Table columns={structureColumnDefs} dataSource={structureColumns} rowKey="name" size="small" pagination={false} scroll={{ y: 420 }} />
                </Spin>
              ),
            },
            {
              key: 'ddl',
              label: intl.formatMessage({ id: 'queryMain.structDdl' }),
              children: (
                <Editor height="420px" defaultLanguage="sql" value={structureDDL} theme="vs" options={{ readOnly: true, minimap: { enabled: false }, fontSize: 13 }} />
              ),
            },
          ]}
        />
      </Modal>

      {/* MyBatis picker modal */}
      <Modal
        title={<><CodeOutlined style={{ color: '#531dab', marginRight: 8 }} />{intl.formatMessage({ id: 'queryMain.mybatisModalTitle' })}</>}
        open={mybatisPickerVisible}
        onCancel={() => setMybatisPickerVisible(false)}
        footer={null}
        width={900}
      >
        <Space style={{ marginBottom: 12 }} wrap>
          <Select
            placeholder={intl.formatMessage({ id: 'queryMain.mybatisPhInst' })} style={{ width: 220 }} value={mybatisPickerInstance || undefined}
            onChange={handleMybatisPickerInstanceChange} showSearch
            options={instances.map((inst) => ({ label: <Space size={4}>{<DbTypeIcon dbType={inst.dbType} />}{inst.name}</Space>, value: inst.id }))}
          />
          <Select
            placeholder={intl.formatMessage({ id: 'queryMain.mybatisPhDb' })} style={{ width: 160 }} value={mybatisPickerDb || undefined}
            onChange={handleMybatisPickerDbChange} loading={mybatisPickerDbLoading}
            disabled={!mybatisPickerInstance} showSearch
            options={mybatisPickerDbs.map((db) => ({ label: <Space size={4}><DatabaseOutlined style={{ color: '#1677ff' }} />{db}</Space>, value: db }))}
          />
          <Select
            placeholder={intl.formatMessage({ id: 'queryMain.mybatisPhTable' })} style={{ width: 200 }} value={mybatisPickerTable || undefined}
            onChange={setMybatisPickerTable} loading={mybatisPickerTableLoading}
            disabled={!mybatisPickerDb} showSearch
            options={mybatisPickerTables.map((t) => ({ label: <Space size={4}><TableOutlined style={{ color: '#52c41a' }} />{t}</Space>, value: t }))}
          />
          <Button
            type="primary" icon={<CodeOutlined />} loading={mybatisPickerGenerating}
            disabled={!mybatisPickerInstance || !mybatisPickerDb || !mybatisPickerTable}
            onClick={handleMybatisGenerate}
            style={{ background: '#531dab', borderColor: '#531dab' }}
          >
            {intl.formatMessage({ id: 'queryMain.mybatisGen' })}
          </Button>
        </Space>
        {mybatisXML && (
          <>
            <Divider />
            <Editor height="500px" defaultLanguage="xml" value={mybatisXML} theme="vs" options={{ readOnly: true, minimap: { enabled: false }, fontSize: 12, wordWrap: 'on' }} />
          </>
        )}
      </Modal>
    </PageContainer>
  );
};
