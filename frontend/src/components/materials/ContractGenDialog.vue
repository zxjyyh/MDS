<template>
  <el-dialog
    v-model="visible"
    title="劳动合同生成"
    :width="contractDialogWidth"
    class="contract-dialog"
    align-center
    destroy-on-close
    @open="onOpen"
  >
    <div class="contract-body">
      <div class="contract-cols" :class="{ 'with-preview': previewReady }">
        <!-- 左：选择人员 -->
        <section class="contract-pick">
          <div class="contract-sec-title">
            <el-icon><User /></el-icon>
            <span>选择人员</span>
          </div>
          <div class="ct-filters">
            <el-select v-model="ctQuery.department" placeholder="所属检测站" clearable style="width: 100%" @change="searchContractCands">
              <el-option v-for="d in departments" :key="d" :label="d" :value="d" />
            </el-select>
            <el-select v-model="ctQuery.status" placeholder="在职状态" clearable style="width: 100%" @change="searchContractCands">
              <el-option label="在职" value="active" />
              <el-option label="离职" value="resigned" />
            </el-select>
            <el-input v-model="ctQuery.keyword" placeholder="输入姓名模糊搜索" clearable @keyup.enter="searchContractCands" @clear="searchContractCands" />
            <el-button type="primary" style="width: 100%" :loading="ctLoading" @click="searchContractCands">搜索</el-button>
          </div>
          <div v-loading="ctLoading" class="ct-cands">
            <div
              v-for="p in ctCands"
              :key="p.id"
              class="ct-cand"
              :class="{ 'is-active': contractForm.personnelId === p.id }"
              @click="pickContractPerson(p)"
            >
              <div class="ct-cand-top">
                <span class="ct-radio-dot" :class="{ 'is-checked': contractForm.personnelId === p.id }" />
                <span class="ct-cand-name">{{ p.name }}</span>
                <el-tag v-if="p.status === 'active'" size="small" type="success" effect="plain">在职</el-tag>
                <el-tag v-else size="small" type="info" effect="plain">离职</el-tag>
              </div>
              <div class="ct-cand-sub">{{ p.department || '未分配检测站' }} · {{ p.idCard || '无身份证' }}</div>
            </div>
            <div v-if="!ctLoading && ctCands.length === 0" class="ct-empty">没有找到匹配的人员</div>
          </div>
        </section>

        <!-- 中：合同关键信息 -->
        <section class="contract-info">
          <div class="contract-sec-title">
            <el-icon><InfoFilled /></el-icon>
            <span>合同关键信息</span>
          </div>
          <div class="ct-fields">
            <div class="ct-field">
              <label>乙方姓名</label>
              <div class="ct-val" :class="{ 'is-empty': !contractForm.name }">{{ contractForm.name || '—（请先选择人员）' }}</div>
            </div>
            <div class="ct-field">
              <label>性别</label>
              <el-select v-model="contractForm.gender" style="width: 100%" size="small" :disabled="!contractForm.personnelId">
                <el-option label="男" value="male" />
                <el-option label="女" value="female" />
              </el-select>
            </div>
            <div class="ct-field ct-field-full">
              <label>身份证号</label>
              <div class="ct-val" :class="{ 'is-empty': !contractForm.idCard }">{{ contractForm.idCard || '—' }}</div>
            </div>
            <div class="ct-field ct-field-full">
              <label>家庭住址</label>
              <div class="ct-val" :class="{ 'is-empty': !contractForm.address }">{{ contractForm.address || '—' }}</div>
            </div>
            <div class="ct-field ct-field-full">
              <label>甲方名称（公司）</label>
              <el-select
                v-model="contractForm.stationName"
                size="small"
                style="width: 100%"
                filterable
                allow-create
                default-first-option
                clearable
                :disabled="!contractForm.personnelId"
                placeholder="选择人员后自动带出，可搜索公司或直接输入"
                @change="onStationNameChange(true)"
              >
                <el-option v-for="c in companies" :key="c.id" :label="c.name" :value="c.name">
                  <span style="float: left">{{ c.name }}</span>
                  <span class="ct-opt-sub">{{ c.address || '公司地址未填' }}</span>
                </el-option>
              </el-select>
            </div>
            <div class="ct-field ct-field-full">
              <label>
                甲方地址（公司地址）<span class="ct-req">*</span>
                <el-tag v-if="addressAutoFilled" size="small" type="success" effect="plain" style="margin-left: 6px">
                  已从公司信息带出
                </el-tag>
              </label>
              <el-input
                v-model="contractForm.stationAddress"
                size="small"
                placeholder="选择甲方名称后自动带出，可手动修改"
                @input="addressAutoFilled = false"
              />
            </div>
            <div class="ct-field">
              <label>签订日期</label>
              <el-date-picker
                v-model="contractForm.signDate"
                type="date"
                value-format="YYYY-MM-DD"
                style="width: 100%"
                size="small"
                :disabled="!contractForm.personnelId"
                @change="onSignDateChange"
              />
            </div>
            <div class="ct-field">
              <label>合同期限</label>
              <div class="ct-val">{{ contractTermText }}</div>
            </div>
          </div>
        </section>

        <!-- 右：合同预览（仅点「预览合同」后出现） -->
        <section v-if="previewReady" class="ct-preview">
          <div class="ct-preview-bar">
            <span><el-icon style="margin-right: 4px"><Document /></el-icon>合同预览</span>
            <span class="ct-preview-hint">展示效果与 Word 略有差异，以「下载合同」为准</span>
          </div>
          <div ref="contractPreviewEl" class="ct-preview-body" />
        </section>
      </div>
    </div>

    <template #footer>
      <el-button v-if="previewReady" type="primary" plain @click="collapseContractPreview">
        <el-icon style="margin-right: 4px"><Fold /></el-icon>收起合同
      </el-button>
      <el-button v-else type="primary" plain :disabled="!hasGenerated" @click="previewContractDoc">
        <el-icon style="margin-right: 4px"><View /></el-icon>预览合同
      </el-button>
      <el-button type="primary" plain :disabled="!hasGenerated" @click="downloadContract">
        <el-icon style="margin-right: 4px"><Download /></el-icon>下载合同
      </el-button>
      <el-button type="primary" :loading="generating" :disabled="!contractForm.personnelId" @click="generateContractDoc">
        <el-icon style="margin-right: 4px"><DocumentAdd /></el-icon>生成合同
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, computed, nextTick, onMounted, onUnmounted } from 'vue';
import { ElMessage } from 'element-plus';
import { personnelApi } from '../../api/personnel';
import { companyApi } from '../../api/company';
import { decodeExportFileName, triggerFileDownload } from '../../utils/download';

const props = defineProps<{ modelValue: boolean }>();
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>();

const visible = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v),
});

const isMobile = ref(window.innerWidth <= 768);
function syncIsMobile() {
  isMobile.value = window.innerWidth <= 768;
}
window.addEventListener('resize', syncIsMobile);
onUnmounted(() => window.removeEventListener('resize', syncIsMobile));

const departments = ref<string[]>([]);
async function loadOptions() {
  try {
    const res: any = await personnelApi.options();
    departments.value = res.data.departments || [];
  } catch {
    /* 忽略选项加载失败 */
  }
}

// ---- 甲方 = 公司：下拉数据源与「地址自动带出」 ----
// 公司名与地址统一取自「公司信息管理」模块，避免同一份信息在两处重复维护。
const companies = ref<any[]>([]);
const addressAutoFilled = ref(false);
async function loadCompanies() {
  try {
    const res: any = await companyApi.options();
    companies.value = res.data || [];
  } catch {
    /* 公司列表拉取失败不阻断生成，地址仍可手动填写 */
  }
}
function findCompanyByName(name: string): any {
  const n = (name || '').trim();
  if (!n) return null;
  return companies.value.find((c) => c.name === n) || null;
}
// 甲方名称变化时按公司名回查公司地址。
// force=true：用户在下拉里主动改选，直接用公司地址覆盖；false：自动带出，仅在地址为空时填，避免冲掉手改内容。
function onStationNameChange(force = true) {
  const c = findCompanyByName(contractForm.stationName);
  if (!c) {
    addressAutoFilled.value = false;
    return;
  }
  if (force || !contractForm.stationAddress.trim()) {
    contractForm.stationAddress = c.address || '';
    addressAutoFilled.value = !!c.address;
  }
}

const ctQuery = reactive({ department: '', status: '', keyword: '' });
const ctCands = ref<any[]>([]);
const ctLoading = ref(false);
const contractForm = reactive({
  personnelId: 0,
  name: '',
  gender: 'male', // male/female（档案 unknown 时按身份证推断后落这里，可手动改）
  idCard: '',
  address: '',
  stationName: '', // 甲方名称 = 所属检测站（自动带出，可改）
  stationAddress: '', // 甲方地址 = 检测站地址（手填）
  signDate: '', // YYYY-MM-DD
});
const generating = ref(false);
const hasGenerated = ref(false); // 是否已成功生成（未下载前可预览/下载）
const previewReady = ref(false);
const previewBlob = ref<Blob | null>(null);
const previewFileName = ref('');
const contractPreviewEl = ref<HTMLElement>();
// 弹窗宽度：未预览两栏居中；预览展开后加宽到三栏（全局 flex 规则保证始终屏幕居中）
const contractDialogWidth = computed(() => {
  if (isMobile.value) return '94%';
  return previewReady.value ? 'min(1660px, calc(100vw - 40px))' : 'min(940px, calc(100vw - 40px))';
});

const CONTRACT_YEARS = 3;

function fmtDay(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}
function yesterdayStr(): string {
  const d = new Date();
  d.setDate(d.getDate() - 1);
  return fmtDay(d);
}
function yearsEndStr(startDate: string, years: number): string {
  const d = new Date(`${startDate}T00:00:00`);
  if (Number.isNaN(d.getTime())) return '';
  d.setFullYear(d.getFullYear() + years);
  d.setDate(d.getDate() - 1);
  return fmtDay(d);
}
// 档案性别 male/female 直接用；unknown 按身份证第 17 位推断（奇男偶女），推断失败默认 male（可手改）
function inferGender(p: any): string {
  if (p.gender === 'male' || p.gender === 'female') return p.gender;
  const c = String(p.idCard || '');
  if (c.length >= 18) {
    const n = Number(c[16]);
    if (!Number.isNaN(n)) return n % 2 === 1 ? 'male' : 'female';
  }
  return 'male';
}

function resetContractForm() {
  Object.assign(contractForm, {
    personnelId: 0, name: '', gender: 'male', idCard: '', address: '',
    stationName: '', stationAddress: '', signDate: yesterdayStr(),
  });
  hasGenerated.value = false;
  previewReady.value = false;
  previewBlob.value = null;
  previewFileName.value = '';
}

function onOpen() {
  resetContractForm();
  ctQuery.department = '';
  ctQuery.status = '';
  ctQuery.keyword = '';
  if (!departments.value.length) loadOptions();
  // 每次打开都刷新公司列表：用户可能刚在「公司信息管理」新增公司或改了地址
  loadCompanies();
  searchContractCands();
}

async function searchContractCands() {
  ctLoading.value = true;
  try {
    const res: any = await personnelApi.list({
      page: 1,
      pageSize: 30,
      keyword: ctQuery.keyword || '',
      department: ctQuery.department || '',
      status: ctQuery.status || '',
    });
    ctCands.value = res.data.list || [];
  } finally {
    ctLoading.value = false;
  }
}

function pickContractPerson(p: any) {
  contractForm.personnelId = p.id;
  contractForm.name = p.name;
  contractForm.gender = inferGender(p);
  contractForm.idCard = p.idCard || '';
  contractForm.address = p.address || '';
  contractForm.stationName = p.department || '';
  contractForm.stationAddress = '';
  contractForm.signDate = yesterdayStr();
  // 按甲方（公司）名称自动带出公司地址；此时地址刚被清空，命中公司即会填入
  onStationNameChange(false);
  // 切换人员后，旧合同失效，需重新生成
  hasGenerated.value = false;
  previewReady.value = false;
  previewBlob.value = null;
}

const contractTermText = computed(() => {
  if (!contractForm.signDate) return '—';
  const end = yearsEndStr(contractForm.signDate, CONTRACT_YEARS);
  return end ? `${contractForm.signDate} 至 ${end}（${CONTRACT_YEARS}年）` : '—';
});

function onSignDateChange() {
  // 签订日期变化会改变合同期限与落款，旧合同一并作废
  hasGenerated.value = false;
  previewReady.value = false;
  previewBlob.value = null;
}

async function generateContractDoc() {
  if (!contractForm.personnelId) {
    ElMessage.warning('请先在左侧选择需要签订合同的人员');
    return;
  }
  if (!contractForm.stationAddress.trim()) {
    ElMessage.warning('请填写甲方地址，或先在「公司信息管理」为该甲方补全公司地址');
    return;
  }
  generating.value = true;
  try {
    const res: any = await personnelApi.generateContract({
      personnelId: contractForm.personnelId,
      signDate: contractForm.signDate,
      stationName: contractForm.stationName,
      stationAddress: contractForm.stationAddress.trim(),
      years: CONTRACT_YEARS,
    });
    const blob = res.data as Blob;
    if (blob.type && blob.type.includes('json')) {
      // 理论不会走到：后端校验失败走 HTTP 400，由拦截器统一提示
      ElMessage.error('合同生成失败');
      return;
    }
    previewBlob.value = blob;
    previewFileName.value =
      decodeExportFileName(res.headers?.['content-disposition']) ||
      `${contractForm.name}-聘用劳动合同-${(contractForm.signDate || '').replace(/-/g, '')}.docx`;
    // 生成后不自动展开预览，由用户通过「预览合同」/「下载合同」按钮按需触发
    previewReady.value = false;
    hasGenerated.value = true;
    ElMessage.success('合同已生成，可点击「预览合同」查看或「下载合同」保存');
  } catch {
    // 后端错误已由请求拦截器统一提示
  } finally {
    generating.value = false;
  }
}

// 按钮触发式预览：仅当用户点击「预览合同」时才渲染 docx（节省资源）
async function previewContractDoc() {
  if (!hasGenerated.value || !previewBlob.value) {
    ElMessage.warning('请先点击「生成合同」');
    return;
  }
  previewReady.value = true;
  await nextTick();
  const { renderAsync } = await import('docx-preview');
  if (contractPreviewEl.value) {
    contractPreviewEl.value.innerHTML = '';
    try {
      await renderAsync(previewBlob.value, contractPreviewEl.value);
    } catch {
      ElMessage.error('合同预览渲染失败，可直接点击「下载合同」保存文件');
    }
  }
}

// 收起预览：仅隐藏右侧预览列；已生成的文件保留，仍可重新展开或直接下载
function collapseContractPreview() {
  previewReady.value = false;
}

function downloadContract() {
  if (!previewBlob.value) return;
  triggerFileDownload(previewBlob.value, previewFileName.value || '聘用劳动合同.docx');
}

onMounted(() => {
  loadOptions();
  loadCompanies();
});
</script>

<style scoped lang="scss">
.contract-body {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.contract-cols {
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
  gap: 16px;
  align-items: start;
}
/* 预览展开 → 三栏：选人 / 关键信息 / 右侧合同预览 */
.contract-cols.with-preview {
  grid-template-columns: 280px minmax(320px, 1fr) 830px;
}
.contract-pick,
.contract-info,
.ct-preview {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  background: var(--el-fill-color-blank);
  min-width: 0;
}
.contract-pick {
  padding: 14px;
  display: flex;
  flex-direction: column;
}
.contract-info {
  padding: 14px;
}
.contract-sec-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 12px;
}
.contract-sec-title .el-icon {
  color: var(--el-color-primary);
}
.ct-filters {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.ct-cands {
  margin-top: 10px;
  max-height: 330px;
  overflow-y: auto;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 6px;
  flex: 1;
}
.ct-cand {
  padding: 8px 10px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.18s;
  border: 1px solid transparent;
  &:hover {
    background: var(--el-fill-color-light);
  }
  &.is-active {
    background: var(--el-color-primary-light-9);
    border-color: var(--el-color-primary);
  }
}
.ct-cand-top {
  display: flex;
  align-items: center;
  gap: 8px;
}
.ct-radio-dot {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  border: 1px solid var(--el-border-color);
  flex-shrink: 0;
  background: var(--el-color-white);
  box-sizing: border-box;
  position: relative;
  &.is-checked {
    border-color: var(--el-color-primary);
    &::after {
      content: '';
      position: absolute;
      inset: 2px;
      border-radius: 50%;
      background: var(--el-color-primary);
    }
  }
}
.ct-cand-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.ct-cand-sub {
  margin-top: 3px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ct-empty {
  padding: 28px 0;
  text-align: center;
  color: var(--el-text-color-placeholder);
  font-size: 13px;
}
.ct-fields {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px 14px;
}
.ct-field {
  min-width: 0;
  label {
    display: block;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    margin-bottom: 4px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}
.ct-field-full {
  grid-column: 1 / -1;
}
.ct-val {
  min-height: 24px;
  display: flex;
  align-items: center;
  font-size: 13px;
  color: var(--el-text-color-primary);
  background: var(--el-fill-color-light);
  border-radius: 4px;
  padding: 3px 8px;
  line-height: 1.5;
  word-break: break-all;
  &.is-empty {
    color: var(--el-text-color-placeholder);
  }
}
.ct-req {
  color: var(--el-color-danger);
  margin-left: 2px;
}
/* 甲方下拉里右侧的公司地址提示 */
.ct-opt-sub {
  float: right;
  margin-left: 16px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.ct-preview {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  align-self: stretch;
}
.ct-preview-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--el-fill-color-light);
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.ct-preview-hint {
  font-size: 12px;
  font-weight: 400;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.ct-preview-body {
  height: 60vh;
  max-height: 780px;
  min-height: 420px;
  overflow: auto;
  padding: 14px 18px;
  background: #e8eaed;
  .docx-wrapper {
    background: #fff;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.12);
    padding: 24px 0;
    margin: 0 auto;
  }
}
/* 合同弹窗：限高保证始终屏幕居中（超高内容在弹窗内部滚动） */
:global(.contract-dialog) {
  max-height: calc(100vh - 40px);
  display: flex;
  flex-direction: column;
  margin: auto !important;
}
:global(.contract-dialog .el-dialog__header),
:global(.contract-dialog .el-dialog__footer) {
  flex-shrink: 0;
}
:global(.contract-dialog .el-dialog__body) {
  overflow: auto;
}
/* 中屏：三栏放不下时，预览列整行移至下方 */
@media screen and (max-width: 1560px) {
  .contract-cols.with-preview {
    grid-template-columns: 280px minmax(0, 1fr);
  }
  .contract-cols.with-preview .ct-preview {
    grid-column: 1 / -1;
    width: 100%;
  }
}
@media screen and (max-width: 900px) {
  .contract-cols,
  .contract-cols.with-preview {
    grid-template-columns: 1fr;
  }
  .contract-cols.with-preview .ct-preview {
    grid-column: auto;
  }
  .contract-pick,
  .contract-info {
    width: 100%;
  }
  .ct-fields {
    grid-template-columns: 1fr;
  }
  .ct-field-full {
    grid-column: auto;
  }
  .ct-preview-body {
    height: 55vh;
  }
}
</style>
