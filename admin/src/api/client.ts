import axios from 'axios';

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/api',
});

api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('adminToken');
    if (token) {
      config.headers['Authorization'] = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

api.interceptors.response.use(
  (response) => {
    return response;
  },
  (error) => {
    if (error.response?.status === 401) {
      console.warn("Unauthorized API call, redirecting to login...");
      // A simple redirect mechanism; Context handles state, but this is a global fallback
      // Using window.location to redirect, but avoid looping if we're already on /login
      if (window.location.pathname !== '/login' && window.location.pathname !== '/setup'
          && window.location.pathname !== '/admin/login' && window.location.pathname !== '/admin/setup') {
        localStorage.removeItem('adminToken');
        // Detect if we're under /admin/ prefix
        const base = window.location.pathname.startsWith('/admin') ? '/admin/login' : '/login';
        window.location.href = base;
      }
    }
    return Promise.reject(error);
  }
);

export default api;
