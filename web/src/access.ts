export default function access(initialState: { currentUser?: API.User }) {
  const { currentUser } = initialState || {};

  return {
    // 管理员权限
    admin: currentUser?.username === 'admin' || false,
    // DBA权限
    dba: currentUser?.username === 'dba' || currentUser?.username === 'admin' || false,
    // 普通用户权限
    user: !!currentUser,
  };
}
