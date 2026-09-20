// 文件下载相关工具：从响应头解析文件名 + 触发浏览器下载

/** 解析 Content-Disposition 中的文件名（优先 RFC5987 的 filename*） */
export function decodeExportFileName(disposition?: string): string {
  if (!disposition) return '';
  const utf8 = disposition.match(/filename\*=UTF-8''([^;]+)/i);
  if (utf8?.[1]) {
    try {
      return decodeURIComponent(utf8[1]);
    } catch {
      /* 忽略解析失败，走 fallback */
    }
  }
  const fallback = disposition.match(/filename="?([^";]+)"?/i);
  return fallback?.[1] || '';
}

/** 用 Blob 触发浏览器下载（自动回收 objectURL） */
export function triggerFileDownload(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  window.setTimeout(() => URL.revokeObjectURL(url), 4000);
}
