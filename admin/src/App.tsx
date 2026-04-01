import { BrowserRouter as Router, Routes, Route, Navigate, useLocation } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext';
import { useEffect, useState } from 'react';
import api from './api/client';

import Setup from './pages/Setup';
import Login from './pages/Login';
import AdminLayout from './layouts/AdminLayout';
import Dashboard from './pages/Dashboard';
import PostList from './pages/PostList';
import PostEditor from './pages/PostEditor';
import Boards from './pages/Boards';
import Settings from './pages/Settings';

// Protected Route wrapper
const ProtectedRoute = ({ children }: { children: React.ReactNode }) => {
  const { isAuthenticated } = useAuth();
  const location = useLocation();

  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  return <>{children}</>;
};

const AppRoutes = () => {
  const [loading, setLoading] = useState(true);
  const [needsSetup, setNeedsSetup] = useState(false);
  const { isAuthenticated } = useAuth();

  useEffect(() => {
    // Check setup status before rendering
    api.get('/setup/status')
      .then((res) => {
        setNeedsSetup(res.data.needsSetup);
        setLoading(false);
      })
      .catch((err) => {
        console.error("Failed to fetch setup status", err);
        setLoading(false);
      });
  }, []);

  if (loading) {
    return (
      <div className="window" style={{ margin: '20px auto', width: 300 }}>
        <div className="title-bar">
          <div className="title-bar-text">Loading...</div>
        </div>
        <div className="window-body">
          <p>Please wait...</p>
        </div>
      </div>
    );
  }

  // Redirect to setup if necessary
  if (needsSetup && window.location.pathname !== '/setup') {
    return <Navigate to="/setup" replace />;
  }

  // Redirect to login if not setup but visiting setup
  if (!needsSetup && window.location.pathname === '/setup') {
    return <Navigate to="/login" replace />;
  }

  return (
    <Routes>
      <Route path="/setup" element={<Setup />} />
      <Route path="/login" element={isAuthenticated && !needsSetup ? <Navigate to="/" replace /> : <Login />} />
      
      <Route
        path="/"
        element={
          <ProtectedRoute>
            <AdminLayout />
          </ProtectedRoute>
        }
      >
        <Route index element={<Navigate to="/dashboard" replace />} />
        <Route path="dashboard" element={<Dashboard />} />
        <Route path="posts" element={<PostList />} />
        <Route path="posts/new" element={<PostEditor />} />
        <Route path="posts/edit/:slug" element={<PostEditor />} />
        <Route path="boards" element={<Boards />} />
        <Route path="settings" element={<Settings />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
};

function App() {
  return (
    <AuthProvider>
      <Router>
        <AppRoutes />
      </Router>
    </AuthProvider>
  );
}

export default App;
