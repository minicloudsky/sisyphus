/**
 * api.v1.CommonResponse：成功时常为 code=0 或省略 code，仅 message: "success"。
 */
export function isCommonResponseOk(
  res: { code?: number; message?: string } | null | undefined
): boolean {
  if (!res || typeof res !== 'object') return false;
  const c = res.code;
  if (c === 200 || c === 0) return true;
  if (
    (c === undefined || c === null) &&
    String(res.message || '').toLowerCase() === 'success'
  ) {
    return true;
  }
  return false;
}
