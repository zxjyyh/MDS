import { computed } from 'vue';

export type ExpiryStatus = 'expired' | 'expiring_30' | 'expiring_60' | 'expiring_90' | 'normal';

export interface ExpiryInfo {
  status: ExpiryStatus;
  daysLeft: number | null;
  label: string;
  color: string;
  type: 'danger' | 'warning' | 'success' | 'info' | 'primary';
}

export function useExpiry() {
  function calcExpiry(expiryDate: string | Date | null): ExpiryInfo {
    if (!expiryDate) {
      return { status: 'normal', daysLeft: null, label: '长期有效', color: '#909399', type: 'info' };
    }
    const expiry = new Date(expiryDate);
    const now = new Date();
    now.setHours(0, 0, 0, 0);
    expiry.setHours(0, 0, 0, 0);
    const daysLeft = Math.ceil((expiry.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));

    if (daysLeft < 0) {
      return { status: 'expired', daysLeft, label: '已过期', color: '#f56c6c', type: 'danger' };
    }
    if (daysLeft <= 30) {
      return { status: 'expiring_30', daysLeft, label: `${daysLeft}天后到期`, color: '#e6a23c', type: 'warning' };
    }
    if (daysLeft <= 60) {
      return { status: 'expiring_60', daysLeft, label: `${daysLeft}天后到期`, color: '#f0c040', type: 'warning' };
    }
    if (daysLeft <= 90) {
      return { status: 'expiring_90', daysLeft, label: `${daysLeft}天后到期`, color: '#409eff', type: 'primary' };
    }
    return { status: 'normal', daysLeft, label: `${daysLeft}天后到期`, color: '#67c23a', type: 'success' };
  }

  return { calcExpiry };
}

export const CERT_TYPES: Record<string, string> = {
  id_card: '身份证',
  diploma: '毕业证',
  driver_license: '驾驶证',
  training: '培训证书',
  other: '其他',
};

export const EQUIP_STATUS: Record<string, string> = {
  in_use: '使用中',
  idle: '闲置',
  repair: '维修中',
  scrapped: '已报废',
};

export const PERSONNEL_STATUS: Record<string, string> = {
  active: '在职',
  resigned: '离职',
};
