import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import { authApi, type UserInfo } from '../api/auth';

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '');
  const refreshToken = ref(localStorage.getItem('refreshToken') || '');
  const user = ref<UserInfo | null>(null);

  const isLoggedIn = computed(() => !!token.value);
  const isAdmin = computed(() => user.value?.role === 'admin');

  async function login(username: string, password: string) {
    const res: any = await authApi.login({ username, password });
    token.value = res.data.accessToken;
    refreshToken.value = res.data.refreshToken;
    user.value = res.data.user;
    localStorage.setItem('token', res.data.accessToken);
    localStorage.setItem('refreshToken', res.data.refreshToken);
    return res.data;
  }

  function logout() {
    token.value = '';
    refreshToken.value = '';
    user.value = null;
    localStorage.removeItem('token');
    localStorage.removeItem('refreshToken');
  }

  async function fetchProfile() {
    try {
      const res: any = await authApi.getProfile();
      user.value = res.data;
      return res.data;
    } catch {
      logout();
      return null;
    }
  }

  return {
    token,
    refreshToken,
    user,
    isLoggedIn,
    isAdmin,
    login,
    logout,
    fetchProfile,
  };
});
