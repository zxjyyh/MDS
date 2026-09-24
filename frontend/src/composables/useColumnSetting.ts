import { ref, watch } from 'vue';
import { useAuthStore } from '../stores/auth';

/** 可配置的展示字段：prop 与列表里的列一一对应（无 prop 的列用自定义标识，如头像） */
export interface ColumnDef {
  prop: string;
  label: string;
  /**
   * 默认不显示，需要用户在「展示设置」里手动勾选才会出现。
   *
   * 用于"字段已存在但多数人列表里并不想看"的列（如人员的安检岗位/环保岗位、
   * 设备的检定单位/检定费用）。标记后：新用户打开时不带这些列，
   * 老用户 localStorage 里已有自己的选择也不会被自动打开 —— 两种情况都符合预期。
   */
  defaultHidden?: boolean;
}

/**
 * 列表「展示设置」：勾选需要展示的字段，按用户保存、下次进入自动沿用。
 *
 * 为什么存 localStorage 而不是后端：这是纯展示偏好，按账号隔离即可，
 * 放本地可即时生效、也不用为它新建表和接口。代价是换浏览器/换设备不会跟随。
 * 换账号（username 变化）会重新加载，互不影响。
 */
export function useColumnSetting(key: string, allColumns: ColumnDef[]) {
  const authStore = useAuthStore();
  /** 默认可见集合＝全部字段减去标了 defaultHidden 的 */
  const defaultProps = () => allColumns.filter((c) => !c.defaultHidden).map((c) => c.prop);
  const storageKey = () => `mds:columns:${key}:${authStore.user?.username || 'anonymous'}`;

  const visibleColumns = ref<string[]>(defaultProps());

  function load() {
    const fallback = defaultProps();
    try {
      const raw = localStorage.getItem(storageKey());
      const saved = raw ? JSON.parse(raw) : null;
      if (Array.isArray(saved)) {
        // 过滤掉已不存在的字段（列被删改后不让脏数据残留）；全被过滤掉则回退默认
        const cleaned = saved.filter((p: string) => fallback.includes(p));
        visibleColumns.value = cleaned.length ? cleaned : fallback;
        return;
      }
    } catch {
      /* 解析失败按默认全选 */
    }
    visibleColumns.value = fallback;
  }

  function persist() {
    try {
      localStorage.setItem(storageKey(), JSON.stringify(visibleColumns.value));
    } catch {
      /* 隐私模式等写入失败时忽略，仅本次会话生效 */
    }
  }

  // 刷新页面时 profile 是守卫里异步拉的，username 可能稍后才就绪 → 就绪后重新加载
  watch(() => authStore.user?.username, load, { immediate: true });
  watch(visibleColumns, persist, { deep: true });

  function isVisible(prop: string) {
    return visibleColumns.value.includes(prop);
  }

  /** 恢复默认：回到「全部字段减去 defaultHidden」的初始状态（不是全选） */
  function resetColumns() {
    visibleColumns.value = defaultProps();
  }

  return { visibleColumns, isVisible, resetColumns };
}
