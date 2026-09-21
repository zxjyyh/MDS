<template>
  <el-dialog
    v-model="visible"
    title="理工人员报名材料生成"
    :width="dialogWidth"
    class="ligong-dialog"
    align-center
    destroy-on-close
    @open="onOpen"
  >
    <div class="lg-body">
      <!-- 1. 报名信息 -->
      <section class="lg-block">
        <div class="lg-block-title">
          <el-icon><Tickets /></el-icon>
          <span>1. 报名信息</span>
        </div>
        <div class="lg-grid">
          <div class="lg-field">
            <label>证书类型 <span class="lg-req">*</span></label>
            <el-select v-model="form.certType" style="width: 100%" @change="onCertTypeChange">
              <el-option v-for="c in options.certTypes" :key="c.value" :label="c.label" :value="c.value" />
            </el-select>
          </div>
          <div class="lg-field">
            <label>培训期数 <span class="lg-req">*</span></label>
            <el-input v-model="form.period" placeholder="只需数字，如 8" clearable />
          </div>
          <div class="lg-field">
            <label>报名日期 <span class="lg-req">*</span></label>
            <el-date-picker v-model="form.enrollDate" type="date" value-format="YYYY-MM-DD" style="width: 100%" />
          </div>
          <div class="lg-field lg-field-full">
            <label>单位名称（盖章单位） <span class="lg-req">*</span></label>
            <el-select
              v-model="form.companyName"
              filterable
              allow-create
              default-first-option
              clearable
              style="width: 100%"
              placeholder="选择报名公司，或直接输入公司全称"
              @change="onCompanyChange(true)"
            >
              <el-option v-for="c in companies" :key="c.id" :label="c.name" :value="c.name">
                <span style="float: left">{{ c.name }}</span>
                <span class="lg-opt-sub">{{ c.address || '公司地址未填' }}</span>
              </el-option>
            </el-select>
          </div>
          <div class="lg-field">
            <label>收件人</label>
            <el-input v-model="form.receiver" placeholder="收件人" clearable />
          </div>
          <div class="lg-field">
            <label>收件电话</label>
            <el-input v-model="form.receiverPhone" placeholder="默认取收件人的人员档案电话" clearable />
          </div>
          <div class="lg-field lg-field-full">
            <label>邮寄地址 <span class="lg-req">*</span></label>
            <el-input v-model="form.mailAddress" placeholder="默认取所选公司的公司地址，可手动修改" />
          </div>
          <template v-if="needsContract">
            <div class="lg-field">
              <label>劳动合同签订日期</label>
              <el-date-picker v-model="form.contractSignDate" type="date" value-format="YYYY-MM-DD" style="width: 100%" />
            </div>
            <div class="lg-field">
              <label>劳动合同甲方地址 <span class="lg-req">*</span></label>
              <el-input v-model="form.contractStationAddress" placeholder="默认取公司地址" />
            </div>
          </template>
        </div>

        <div class="lg-materials">
          <el-icon><Files /></el-icon>
          <span class="lg-materials-label">本批材料：</span>
          <el-tag v-if="mixedApplyTypes" size="small" type="danger" effect="plain">含多种报考类别</el-tag>
          <el-tag size="small" type="success" effect="plain">报名汇总表</el-tag>
          <el-tag size="small" type="success" effect="plain">安全责任书</el-tag>
          <el-tag v-for="m in materials" :key="m.key" size="small" type="warning" effect="plain">
            {{ m.name }}
            <template v-if="m.onlyDriver">
              {{ form.certType === 'env' ? '（仅引车员）' : '（有则随附）' }}
            </template>
            <template v-else-if="m.canGenerate">（优先用档案已上传件）</template>
          </el-tag>
          <span v-if="!picked.length" class="lg-materials-note">
            选择人员并设置各自的报考类别后，这里会列出对应的证件材料
          </span>
          <span v-else-if="!materials.length" class="lg-materials-note">
            所选人员均为「增项」，只需上面两份表格，不附证件与劳动合同
          </span>
        </div>
      </section>

      <!-- 2. 选择人员 -->
      <section class="lg-block">
        <div class="lg-block-title">
          <el-icon><User /></el-icon>
          <span>2. 选择报名人员</span>
          <el-tag size="small" :type="picked.length ? 'success' : 'info'" effect="plain">
            已选 {{ picked.length }} 人
          </el-tag>
          <span v-if="picked.length" class="lg-limit">（本表最多 {{ maxRows }} 人）</span>
        </div>
        <div class="lg-pick">
          <div class="lg-pick-left">
            <div class="lg-filters">
              <el-select v-model="query.department" placeholder="所属检测站" clearable style="width: 100%" @change="searchCands">
                <el-option v-for="d in departments" :key="d" :label="d" :value="d" />
              </el-select>
              <el-select v-model="query.status" placeholder="在职状态" clearable style="width: 100%" @change="searchCands">
                <el-option label="在职" value="active" />
                <el-option label="离职" value="resigned" />
              </el-select>
              <el-input v-model="query.keyword" placeholder="输入姓名模糊搜索" clearable @keyup.enter="searchCands" @clear="searchCands" />
            </div>
            <div v-loading="candLoading" class="lg-cands">
              <div
                v-for="p in cands"
                :key="p.id"
                class="lg-cand"
                :class="{ 'is-active': isPicked(p.id) }"
                @click="togglePick(p)"
              >
                <el-checkbox :model-value="isPicked(p.id)" @click.stop @change="togglePick(p)" />
                <div class="lg-cand-main">
                  <div class="lg-cand-top">
                    <span class="lg-cand-name">{{ p.name }}</span>
                    <el-tag v-if="p.status === 'active'" size="small" type="success" effect="plain">在职</el-tag>
                    <el-tag v-else size="small" type="info" effect="plain">离职</el-tag>
                  </div>
                  <div class="lg-cand-sub">{{ p.department || '未分配检测站' }} · {{ p.idCard || '无身份证' }}</div>
                </div>
              </div>
              <div v-if="!candLoading && !cands.length" class="lg-empty">没有找到匹配的人员</div>
            </div>
          </div>

          <div class="lg-pick-right">
            <div class="lg-picked-head">
              <span>已选人员</span>
              <el-button v-if="picked.length" link type="danger" @click="picked = []">清空</el-button>
            </div>
            <div v-if="!picked.length" class="lg-empty">左侧勾选需要报名的人员</div>
            <div v-else class="lg-picked-list">
              <div v-for="(row, i) in picked" :key="row.personnelId" class="lg-picked-item">
                <span class="lg-picked-idx">{{ i + 1 }}</span>
                <span class="lg-picked-name">{{ row.name }}</span>
                <span class="lg-picked-sub">{{ row.gender }} · {{ row.phone || '无电话' }}</span>
                <el-button link type="danger" @click="removePick(row.personnelId)">移除</el-button>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- 3. 逐人报名信息 -->
      <section v-if="picked.length" class="lg-block">
        <div class="lg-block-title">
          <el-icon><EditPen /></el-icon>
          <span>3. 逐人报名信息</span>
          <span class="lg-limit">
            请为每人选择「报考类别」；姓名 / 性别 / 身份证号 / 电话 / 学历 优先自动取自人员档案，
            档案里缺学历时需手工选择；「报名岗位」等信息按 {{ certLabel }} 填表要求补录
          </span>
        </div>
        <el-table :data="picked" size="small" border class="lg-table">
          <el-table-column prop="name" label="姓名" width="90" fixed />
          <el-table-column prop="gender" label="性别" width="60" />
          <el-table-column label="学历" width="130">
            <template #default="{ row }">
              <el-select v-model="row.education" size="small" filterable allow-create default-first-option clearable placeholder="未填">
                <el-option v-for="e in options.educations" :key="e" :label="e" :value="e" />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column label="职称" width="120">
            <template #default="{ row }">
              <el-input v-model="row.title" size="small" placeholder="可空" />
            </template>
          </el-table-column>
          <el-table-column v-if="form.certType === 'env'" label="检测线类别" width="150">
            <template #default="{ row }">
              <el-select v-model="row.lineType" size="small" placeholder="请选择">
                <el-option v-for="l in options.lineTypes" :key="l" :label="l" :value="l" />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column label="报考类别" width="140">
            <template #default="{ row }">
              <el-select v-model="row.applyType" size="small" @change="applyDefaults(row, true)">
                <el-option v-for="a in options.applyTypes" :key="a.value" :label="a.label" :value="a.value" />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column label="报名岗位" min-width="200">
            <template #default="{ row }">
              <el-select v-model="row.position" size="small" filterable allow-create default-first-option placeholder="请选择">
                <el-option v-for="pos in positionOptions" :key="pos" :label="pos" :value="pos" />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column v-if="form.certType === 'env'" label="准驾车型" width="120">
            <template #default="{ row }">
              <el-input
                v-model="row.driveType"
                size="small"
                :disabled="!isDriver(row.position)"
                :placeholder="driveTypePlaceholder(row)"
              />
            </template>
          </el-table-column>
          <el-table-column v-if="hasContinue" label="证书有效日期" width="180">
            <template #default="{ row }">
              <el-date-picker
                v-model="row.certExpiry"
                type="date"
                value-format="YYYY-MM-DD"
                size="small"
                style="width: 100%"
                :disabled="row.applyType !== 'continue'"
                :placeholder="row.applyType === 'continue' ? certExpiryPlaceholder : '非继续教育'"
              />
            </template>
          </el-table-column>
        </el-table>
      </section>
    </div>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="generating" :disabled="!picked.length" @click="submit">
        <el-icon style="margin-right: 4px"><Download /></el-icon>
        生成并下载整套材料
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
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

const dialogWidth = computed(() => (isMobile.value ? '94%' : 'min(1180px, calc(100vw - 40px))'));

/** 报考类别 -> 该类别要收集的材料（由后端返回，避免前端与后端口径不一致） */
const materialMap = ref<Record<string, any[]>>({});

const options = reactive<{
  certTypes: Array<{ value: string; label: string }>;
  applyTypes: Array<{ value: string; label: string }>;
  positions: Record<string, string[]>;
  lineTypes: string[];
  educations: string[];
  certExpiryFields: Record<string, string>;
  defaultLineType: string;
  defaultReceiver: string;
}>({
  certTypes: [],
  applyTypes: [],
  positions: {},
  lineTypes: [],
  educations: [],
  certExpiryFields: {},
  defaultLineType: '汽车检测线',
  defaultReceiver: '',
});

const companies = ref<any[]>([]);
const departments = ref<string[]>([]);
const cands = ref<any[]>([]);
const candLoading = ref(false);
const picked = ref<any[]>([]);
const generating = ref(false);

const form = reactive({
  certType: 'security',
  period: '',
  enrollDate: '',
  companyName: '',
  receiver: '',
  receiverPhone: '',
  mailAddress: '',
  contractSignDate: '',
  contractStationAddress: '',
});

const query = reactive({ department: '', status: '', keyword: '' });

/** 本批是否需要劳动合同（增项不需要）；混报时只要有人不是增项就需要 */
const needsContract = computed(() => picked.value.some((r: any) => r.applyType !== 'add'));
const certLabel = computed(() => options.certTypes.find((c) => c.value === form.certType)?.label || '');
const positionOptions = computed(() => options.positions[form.certType] || []);
const maxRows = computed(() => (form.certType === 'env' ? 11 : 19));
/** 证书有效期留空时的提示：说明自动带出的来源（人员档案里的证书到期时间） */
const certExpiryPlaceholder = computed(() =>
  form.certType === 'env' ? '档案无环保证到期时间，请手填' : '档案无安检证到期时间，请手填'
);

/**
 * 本批要收集的材料：按**已选人员各自的报考类别**取并集。
 * 报考类别是逐人设置的（没有批次级的默认值），所以没选人时不预告任何材料。
 */
const materials = computed(() => {
  const classes = Array.from(new Set(picked.value.map((r: any) => r.applyType).filter(Boolean)));
  const seen = new Set<string>();
  const out: any[] = [];
  classes.forEach((c) => {
    (materialMap.value[c as string] || []).forEach((m: any) => {
      if (seen.has(m.key)) return;
      seen.add(m.key);
      out.push(m);
    });
  });
  return out;
});

/** 已选人员里是否混了多种报考类别 */
const mixedApplyTypes = computed(() => {
  const set = new Set(picked.value.map((r: any) => r.applyType).filter(Boolean));
  return set.size > 1;
});

/** 是否存在「继续教育」的人 —— 决定「证书有效日期」这一列要不要显示 */
const hasContinue = computed(() => picked.value.some((r: any) => r.applyType === 'continue'));

function isDriver(position: string): boolean {
  const p = (position || '').trim();
  return p.includes('引车员') || p.includes('驾驶员');
}

/** 准驾车型输入框的提示：档案里没有就明确告诉用户要手填，别让人以为系统忘了带 */
function driveTypePlaceholder(row: any): string {
  if (!isDriver(row.position)) return '非引车员';
  return row._src?.driverLicense ? '如 C1' : '档案未录驾驶证，请手填';
}

function fmtDay(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}
function todayStr(): string {
  return fmtDay(new Date());
}
function yesterdayStr(): string {
  const d = new Date();
  d.setDate(d.getDate() - 1);
  return fmtDay(d);
}

function isPicked(id: number): boolean {
  return picked.value.some((p) => p.personnelId === id);
}
function removePick(id: number) {
  picked.value = picked.value.filter((p) => p.personnelId !== id);
}
/**
 * 按人员档案 / 约定默认值补齐「学历 / 检测线类别 / 准驾车型 / 证书有效期」：
 * - 学历：档案有就带出来；档案里没录则**留空**，由用户在表格里手工选择。
 *   不再兜底成某个默认学历 —— 报名表上的学历必须是真的（用户 2026-09-21 要求）
 * - 检测线类别：环保默认「汽车检测线」（摩托车线是少数，由用户主动改选）。
 *   该字段只有环保用，且取值与证书类型无关，所以**不随证书类型切换而清空**
 * - 准驾车型：取人员档案里的驾驶证字段（只有引车员用得上）。**这里必须一并重填**，
 *   否则切换证书类型时被清空后就再也回不来了（2026-09-20 修的实际 bug）
 * - 证书有效期：只在「继续教育」时有意义（模板填表说明：继续教育人员要在备注栏填证书有效日期），
 *   取值按证书类型而定 —— 环保取人员档案里的「环保证到期时间」、安检取「安检证到期时间」
 *   （注意是人员基本信息里的字段，不是证件上传件里的到期日），查不到就留空由用户填
 *
 * reset：切换证书类型 / 报考类别时用 —— 这两个维度会改变取值来源，需要重取。
 */
function applyDefaults(row: any, reset = false) {
  const src = row._src || {};
  if (!row.education) row.education = src.education || '';

  if (form.certType === 'env' && !row.lineType) {
    row.lineType = options.defaultLineType || '汽车检测线';
  }

  // 准驾车型：档案里录了就直接带出来，省得用户再手填一遍
  if (reset || !row.driveType) row.driveType = src.driverLicense || '';

  // 证书有效期按**本人**的报考类别判断（混报时同批里新人员不需要填）
  const applicable = row.applyType === 'continue';
  const field = options.certExpiryFields[form.certType];
  const def = applicable && field ? src[field] || '' : '';
  if (reset || !applicable) row.certExpiry = def;
  else if (!row.certExpiry) row.certExpiry = def;
}

function togglePick(p: any) {
  if (isPicked(p.id)) {
    removePick(p.id);
    return;
  }
  if (picked.value.length >= maxRows.value) {
    ElMessage.warning(`${certLabel.value}报名汇总表最多容纳 ${maxRows.value} 人`);
    return;
  }
  const row: any = {
    personnelId: p.id,
    name: p.name,
    gender: p.gender === 'female' ? '女' : '男',
    idCard: p.idCard || '',
    phone: p.phone || '',
    department: p.department || '',
    // 报考类别逐人设置（没有批次默认值），这里先留空，由用户在各行选择
    applyType: '',
    // 学历 / 检测线类别 / 准驾车型 / 证书有效期 都自动带出（见 applyDefaults）
    education: '',
    title: '',
    position: '',
    lineType: '',
    driveType: p.driverLicense || '',
    certExpiry: '',
    // 保留原始档案行，切换证书类型时要用它重取「证书有效期」
    _src: p,
  };
  applyDefaults(row, true);
  picked.value.push(row);
  // 单位名称还没选时，用第一个人员所属检测站带出
  if (!form.companyName.trim() && p.department) {
    form.companyName = p.department;
    onCompanyChange(false);
  }
}

function onCertTypeChange() {
  // 报名岗位的口径随证书类型变化（安检是主任/普通检验员，环保是引车员/登录员…），切换后必须重选。
  // 检测线类别与准驾车型、证书有效期会由 applyDefaults 按新证书类型处理。
  picked.value.forEach((row) => {
    row.position = '';
    applyDefaults(row, true);
  });
}

/** 单位名称变化 → 同步带出公司地址（force=true 时覆盖已填内容） */
function onCompanyChange(force = true) {
  const c = companies.value.find((x) => x.name === form.companyName.trim());
  const addr = c?.address || '';
  if (!addr) return;
  if (force || !form.mailAddress.trim()) form.mailAddress = addr;
  if (force || !form.contractStationAddress.trim()) form.contractStationAddress = addr;
}

async function loadOptions() {
  try {
    const res: any = await personnelApi.ligongOptions();
    const d = res.data || {};
    options.certTypes = d.certTypes || [];
    options.applyTypes = d.applyTypes || [];
    options.positions = d.positions || {};
    options.lineTypes = d.lineTypes || [];
    options.educations = d.educations || [];
    options.certExpiryFields = d.certExpiryFields || {};
    options.defaultLineType = d.defaultLineType || '汽车检测线';
    options.defaultReceiver = d.defaultReceiver || '';
    // 收件人默认值来自部署配置（异步返回），首次打开时若还没填过就补上并带出电话
    if (!form.receiver && options.defaultReceiver) {
      form.receiver = options.defaultReceiver;
      loadDefaultReceiverPhone();
    }
    materialMap.value = d.materials || {};
    // 选项是异步来的，可能晚于第一次勾选：补一遍默认值
    picked.value.forEach((row) => applyDefaults(row));
  } catch {
    /* 选项加载失败不阻断，错误由拦截器提示 */
  }
}

async function loadCompanies() {
  try {
    const res: any = await companyApi.options();
    companies.value = res.data || [];
  } catch {
    /* 公司列表拉取失败不阻断，单位名称与地址仍可手填 */
  }
}

async function loadDepartments() {
  try {
    const res: any = await personnelApi.options();
    departments.value = res.data.departments || [];
  } catch {
    /* 忽略 */
  }
}

/** 收件电话默认取「收件人」在人员档案里的联系电话（收件人可由部署配置预填） */
async function loadDefaultReceiverPhone() {
  const name = form.receiver.trim();
  if (form.receiverPhone.trim() || !name) return;
  try {
    const res: any = await personnelApi.list({ page: 1, pageSize: 5, keyword: name });
    const hit = (res.data.list || []).find((p: any) => p.name === name) || (res.data.list || [])[0];
    if (hit?.phone) form.receiverPhone = hit.phone;
  } catch {
    /* 查不到就留给用户手填 */
  }
}

async function searchCands() {
  candLoading.value = true;
  try {
    const res: any = await personnelApi.list({
      page: 1,
      pageSize: 50,
      keyword: query.keyword || '',
      department: query.department || '',
      status: query.status || '',
    });
    cands.value = res.data.list || [];
  } finally {
    candLoading.value = false;
  }
}

function onOpen() {
  Object.assign(form, {
    certType: 'security',
    period: '',
    enrollDate: todayStr(),
    companyName: '',
    receiver: options.defaultReceiver || '',
    receiverPhone: '',
    mailAddress: '',
    contractSignDate: yesterdayStr(),
    contractStationAddress: '',
  });
  picked.value = [];
  Object.assign(query, { department: '', status: '', keyword: '' });

  if (!options.certTypes.length) loadOptions();
  loadCompanies(); // 每次打开都刷新：用户可能刚在「公司信息管理」改了地址
  if (!departments.value.length) loadDepartments();
  loadDefaultReceiverPhone();
  searchCands();
}

/** 提交前的本地校验，尽量与后端提示保持一致，避免白跑一次请求 */
function validate(): string {
  if (!/^\d+$/.test(form.period.replace(/[第期\s]/g, ''))) return '请填写培训期数（只需数字，如 8）';
  if (!form.enrollDate) return '请选择报名日期';
  if (!form.companyName.trim()) return '请填写单位名称（报名公司）';
  if (!form.mailAddress.trim()) return '请填写邮寄地址，或先在「公司信息管理」为该公司补全地址';
  // 劳动合同甲方地址不在这里校验：只有当本批真的需要重新生成劳动合同时才必填
  // （档案里已有可用上传件时不需要，见 resolveContractDecision）
  for (let i = 0; i < picked.value.length; i += 1) {
    const r = picked.value[i];
    // 学历不再兜底默认值：档案里没录就必须手工选，避免把臆造的学历填进官方报名表
    if (!r.education) {
      return `「${r.name}」缺学历：人员档案里没有学历，请手工选择`;
    }
    if (!r.applyType) return `请为「${r.name}」选择报考类别`;
    if (!r.phone) return `「${r.name}」未填写联系电话，请先完善人员档案`;
    if (!r.position) return `请为「${r.name}」选择报名岗位`;
    if (form.certType === 'env') {
      if (!r.lineType) return `请为「${r.name}」选择检测线类别`;
      if (isDriver(r.position) && !r.driveType) {
        return (
          `「${r.name}」报名岗位含引车员，必须填写准驾车型` +
          (r._src?.driverLicense ? '' : '（人员档案里未录入驾驶证准驾车型，请手工填写）')
        );
      }
    }
    // 继续教育：模板填表说明要求备注栏填「证书有效日期」，档案查不到时必须手工补
    if (r.applyType === 'continue' && !r.certExpiry) {
      return `「${r.name}」缺「证书有效日期」：人员档案里没有${certLabel.value}到期时间，请手工选择`;
    }
  }
  return '';
}

/**
 * 劳动合同预检 + 二次确认（用户 2026-09-20 确认的口径）：
 *   - 档案里有仍在有效期内的劳动合同上传件 → 直接打包那份，不重新生成
 *   - 不存在 / 已过期 → 问用户是否重新生成；选「否」则本次不出，由用户自行打印提供
 *
 * 返回：true 表示重新生成，false 表示复用上传件或不生成；null 表示用户取消了整个操作。
 */
async function resolveContractDecision(): Promise<boolean | null> {
  if (!needsContract.value) return false;
  let pre: any = null;
  try {
    const res: any = await personnelApi.ligongPrecheck({
      enrollDate: form.enrollDate,
      // 逐人传报考类别：增项的人不涉及劳动合同，后端会跳过
      persons: picked.value.map((r) => ({ personnelId: r.personnelId, applyType: r.applyType })),
    });
    pre = res.data;
  } catch {
    // 预检失败不阻断：交给后端按同一规则再判一次
    return false;
  }
  const need = (pre?.needDecision || []) as any[];
  if (!need.length) return false;

  const list = need.map((r) => `${r.name}（${r.stateText}）`).join('、');
  try {
    await ElMessageBox.confirm(
      `以下人员的劳动合同在档案里缺失或已过期：${list}。是否由系统重新生成劳动合同？` +
        `选择「否」则本次不生成，由您自行打印提供。`,
      '劳动合同处理方式',
      {
        confirmButtonText: '是，重新生成',
        cancelButtonText: '否，自行提供',
        type: 'warning',
        distinguishCancelAndClose: true,
      }
    );
  } catch (action) {
    if (action === 'cancel') return false; // 明确选了「否」
    return null; // 关闭弹窗 = 放弃本次生成
  }
  if (!form.contractStationAddress.trim()) {
    ElMessage.warning('需要重新生成劳动合同，请先填写「劳动合同甲方地址」');
    return null;
  }
  return true;
}

async function submit() {
  const err = validate();
  if (err) {
    ElMessage.warning(err);
    return;
  }

  generating.value = true;
  try {
    const regenerate = await resolveContractDecision();
    if (regenerate === null) return; // 用户取消了

    const payload: any = {
      certType: form.certType,
      period: form.period,
      enrollDate: form.enrollDate,
      companyName: form.companyName.trim(),
      receiver: form.receiver.trim(),
      receiverPhone: form.receiverPhone.trim(),
      mailAddress: form.mailAddress.trim(),
      regenerateContract: regenerate,
      contractSignDate: form.contractSignDate,
      contractStationAddress: form.contractStationAddress.trim(),
      persons: picked.value.map((r) => ({
        personnelId: r.personnelId,
        // 报考类别逐人下传：同一证书类型下允许混报（新人员 + 继续教育 + 增项）
        applyType: r.applyType,
        education: r.education,
        title: r.title,
        position: r.position,
        lineType: r.lineType,
        driveType: r.driveType,
        certExpiry: r.certExpiry,
      })),
    };

    const res: any = await personnelApi.generateLigongEnroll(payload);
    const blob = res.data as Blob;
    if (blob.type && blob.type.includes('json')) {
      ElMessage.error('生成失败');
      return;
    }
    const headers = res.headers || {};
    const filename = decodeExportFileName(headers['content-disposition']) || '理工报名材料.zip';
    triggerFileDownload(blob, filename);
    reportResult(headers);
    visible.value = false;
  } catch {
    // 后端错误已由请求拦截器统一提示
  } finally {
    generating.value = false;
  }
}

/** 下载后的结果提示：缺件明细 + 劳动合同这批是怎么处理的 */
function reportResult(headers: Record<string, any>) {
  const decode = (v: any): string[] => {
    const s = String(v || '');
    if (!s) return [];
    try {
      return decodeURIComponent(s).split(';').filter(Boolean);
    } catch {
      return [];
    }
  };
  const missing = decode(headers['x-ligong-missing']);
  const fromArchive = decode(headers['x-ligong-contract-archive']);
  const generated = decode(headers['x-ligong-contract-generated']);
  const self = decode(headers['x-ligong-contract-self']);

  const parts: string[] = [];
  if (missing.length) parts.push(`${missing.length} 项证件缺失（${missing.slice(0, 5).join('、')}）`);
  if (fromArchive.length) parts.push(`劳动合同直接复用档案上传件（${fromArchive.join('、')}）`);
  if (generated.length) parts.push(`劳动合同已重新生成（${generated.join('、')}）`);
  if (self.length) parts.push(`劳动合同未生成、需自行打印提供（${self.join('、')}）`);

  if (parts.length) {
    ElMessage.warning({
      message: `材料已下载。${parts.join('；')}。详细说明见压缩包内「00_报名说明.txt」`,
      duration: 9000,
    });
  } else {
    ElMessage.success('报名材料已生成并开始下载');
  }
}

onMounted(() => {
  loadOptions();
  loadCompanies();
  loadDepartments();
});
</script>

<style scoped lang="scss">
.lg-body {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.lg-block {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  padding: 14px;
  background: var(--el-fill-color-blank);
}
.lg-block-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 12px;
  flex-wrap: wrap;
  .el-icon {
    color: var(--el-color-primary);
  }
}
.lg-limit {
  font-size: 12px;
  font-weight: 400;
  color: var(--el-text-color-secondary);
}
.lg-grid {
  display: grid;
  /* 三个字段一行（证书类型 / 培训期数 / 报名日期），登记信息更宽松好填 */
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px 14px;
}
.lg-field {
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
.lg-field-full {
  grid-column: 1 / -1;
}
.lg-req {
  color: var(--el-color-danger);
}
.lg-opt-sub {
  float: right;
  margin-left: 16px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.lg-materials {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 12px;
  padding-top: 10px;
  border-top: 1px dashed var(--el-border-color-lighter);
  font-size: 12.5px;
  color: var(--el-text-color-secondary);
  .el-icon {
    color: var(--el-color-primary);
  }
}
.lg-materials-label {
  font-weight: 600;
  color: var(--el-text-color-regular);
}
.lg-materials-note {
  color: var(--el-text-color-placeholder);
}
.lg-pick {
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) minmax(0, 1fr);
  gap: 14px;
}
.lg-pick-left {
  min-width: 0;
}
.lg-filters {
  display: grid;
  grid-template-columns: 1fr 1fr 1.2fr;
  gap: 8px;
}
.lg-cands {
  margin-top: 10px;
  max-height: 260px;
  overflow-y: auto;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 6px;
}
.lg-cand {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: 8px;
  cursor: pointer;
  border: 1px solid transparent;
  transition: background 0.18s;
  &:hover {
    background: var(--el-fill-color-light);
  }
  &.is-active {
    background: var(--el-color-primary-light-9);
    border-color: var(--el-color-primary);
  }
}
.lg-cand-main {
  min-width: 0;
  flex: 1;
}
.lg-cand-top {
  display: flex;
  align-items: center;
  gap: 8px;
}
.lg-cand-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.lg-cand-sub {
  margin-top: 2px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.lg-pick-right {
  min-width: 0;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 10px;
  display: flex;
  flex-direction: column;
}
.lg-picked-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-regular);
  margin-bottom: 6px;
}
.lg-picked-list {
  max-height: 232px;
  overflow-y: auto;
}
.lg-picked-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 0;
  border-bottom: 1px dashed var(--el-border-color-lighter);
  font-size: 13px;
  &:last-child {
    border-bottom: none;
  }
}
.lg-picked-idx {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  font-size: 11px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.lg-picked-name {
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.lg-picked-sub {
  flex: 1;
  min-width: 0;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.lg-empty {
  padding: 24px 0;
  text-align: center;
  color: var(--el-text-color-placeholder);
  font-size: 13px;
  flex: 1;
}
.lg-table {
  width: 100%;
}
/* 弹窗限高：内容超高时在弹窗内部滚动，保证始终屏幕居中 */
:global(.ligong-dialog) {
  max-height: calc(100vh - 40px);
  display: flex;
  flex-direction: column;
  margin: auto !important;
}
:global(.ligong-dialog .el-dialog__header),
:global(.ligong-dialog .el-dialog__footer) {
  flex-shrink: 0;
}
:global(.ligong-dialog .el-dialog__body) {
  overflow: auto;
}

@media screen and (max-width: 1200px) {
  .lg-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
@media screen and (max-width: 900px) {
  .lg-pick {
    grid-template-columns: 1fr;
  }
}
@media screen and (max-width: 640px) {
  .lg-grid {
    grid-template-columns: 1fr;
  }
  .lg-field-full {
    grid-column: auto;
  }
  .lg-filters {
    grid-template-columns: 1fr;
  }
}
</style>
