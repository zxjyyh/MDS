/**
 * 全局固定选项常量。
 *
 * 这里的取值需要与后端保持一致（后端同类枚举见 `internal/ligong/rules.go`）。
 */

/**
 * 学历选项。
 *
 * 2026-09-21 由三档「高中/中专 / 大专 / 本科及以上」收敛为四档：
 * 原「高中/中专」把高中与中专混在一格里，且「本科及以上」无法区分本科与硕士。
 * 存量数据已在后端启动迁移中收敛（高中/中专→高中、本科及以上→本科）。
 */
export const EDUCATION_OPTIONS = ['中专', '高中', '大专', '本科'];

/**
 * 人员岗位选项（多选，非必选）。
 *
 * 安检岗位 / 环保岗位分别挂在「安检证」「环保证」两本证上，描述持证可从事的岗位。
 * 注意与「理工报名」里的「检验岗位」区分：那个是每次报名时逐人选的，属于一次性动作；
 * 这里是人员档案上的长期属性，存在 personnel.security_positions / env_positions。
 *
 * 安检岗位按「主任 → 作业」排序，与安检报名表填表说明里列举的先后一致。
 * 用词刻意写成「主任检验员」而非报名表里的「机动车主任检验员」：
 * 两者目前互不联动（报名岗位仍逐人手选），将来若要打通需先统一措辞。
 *
 * 选项取自培训机构的报名表说明，调整前需与业务口径确认。
 * 注意「登录员、OBD查验员」是**一个**选项（中间是顿号），所以存储分隔符必须用半角逗号，
 * 不能用顿号，否则拆分会把它切碎。
 */
export const SECURITY_POSITION_OPTIONS = ['主任检验员', '引车员', '外观检验员', '底盘检验员', '登录员'];
export const ENV_POSITION_OPTIONS = ['引车员', '登录员、OBD查验员'];

/**
 * 多选字段的表单↔存储转换。
 *
 * 表单里用 string[]（配合 el-select multiple），库里存逗号分隔的 TEXT。
 * 空数组转成 null 而不是空串：与「清空即置空」的语义一致，避免库里混入 ''。
 */
export function joinMultiSelect(v?: string[] | null): string | null {
  return v && v.length ? v.join(',') : null;
}

export function splitMultiSelect(v?: string | null): string[] {
  return v ? v.split(',').filter(Boolean) : [];
}
