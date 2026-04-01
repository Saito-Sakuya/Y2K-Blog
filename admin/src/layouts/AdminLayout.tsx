import { Outlet, NavLink, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { Home, FileText, FolderTree, Settings as SettingsIcon, LogOut, Menu, X } from 'lucide-react';
import { useEffect, useState } from 'react';
import api from '../api/client';

export default function AdminLayout() {
  const { logout, username } = useAuth();
  const navigate = useNavigate();
  const [apiOnline, setApiOnline] = useState(true);
  
  const [isMobile, setIsMobile] = useState(window.innerWidth < 768);
  const [sidebarOpen, setSidebarOpen] = useState(false);

  useEffect(() => {
    const handleResize = () => setIsMobile(window.innerWidth < 768);
    window.addEventListener('resize', handleResize);
    handleResize();
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  useEffect(() => {
    api.get('/boards')
      .then(() => setApiOnline(true))
      .catch(() => setApiOnline(false));
  }, []);

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const navLinkClass = ({isActive}: {isActive: boolean}) => isActive ? 'button active' : 'button';
  const navLinkStyle = { width: '100%', textAlign: 'left' as const, marginTop: '8px' };

  return (
    <div style={{ display: 'flex', height: '100vh', backgroundColor: '#008080' }}>
      <div className="window y2k-window" style={{ flex: 1, margin: isMobile ? 0 : '16px', display: 'flex', flexDirection: 'column' }}>
        <div className="title-bar">
          <div className="title-bar-text">🖥️ Y2K Blog Admin</div>
          <div className="title-bar-controls">
            {!isMobile && <button aria-label="Minimize" />}
            {!isMobile && <button aria-label="Maximize" />}
            <button aria-label="Close" onClick={handleLogout} />
          </div>
        </div>

        <div className="window-body" style={{ flex: 1, display: 'flex', flexDirection: isMobile ? 'column' : 'row', margin: 0, padding: 0, overflow: 'hidden', position: 'relative' }}>
          
          {/* Mobile Header / Toggle */}
          {isMobile && (
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '8px', backgroundColor: '#c0c0c0', borderBottom: '2px solid var(--border)' }}>
              <button 
                onClick={() => setSidebarOpen(!sidebarOpen)} 
                style={{ display: 'flex', alignItems: 'center', gap: '4px' }}
              >
                {sidebarOpen ? <X size={14} /> : <Menu size={14} />} 
                <span style={{ fontSize: '12px' }}>Menu</span>
              </button>
              <span style={{ fontWeight: 'bold' }}>Welcome, {username || 'Admin'}</span>
            </div>
          )}

          {/* Sidebar */}
          <div 
            style={{ 
              width: isMobile ? '200px' : '192px',
              borderRight: '2px solid var(--border)', 
              backgroundColor: '#c0c0c0', 
              padding: '8px', 
              display: 'flex', 
              flexDirection: 'column',
              // Mobile behavior properties
              position: isMobile ? 'absolute' : 'static',
              zIndex: isMobile ? 50 : 1,
              height: isMobile ? '100%' : 'auto',
              left: 0,
              top: isMobile ? '38px' : 0, /* Height of the mobile header */
              transform: isMobile && !sidebarOpen ? 'translateX(-100%)' : 'translateX(0)',
              transition: 'transform 0.2s'
            }}
          >
            {!isMobile && (
              <div style={{ marginBottom: '16px', fontWeight: 'bold' }}>
                Welcome, {username || 'Admin'}
              </div>
            )}
            
            <nav style={{ display: 'flex', flexDirection: 'column' }}>
              <NavLink to="/dashboard" onClick={() => setSidebarOpen(false)} className={navLinkClass} style={navLinkStyle}>
                <Home size={14} style={{ marginRight: '8px', display: 'inline' }} /> Dashboard
              </NavLink>
              <NavLink to="/posts" onClick={() => setSidebarOpen(false)} className={navLinkClass} style={navLinkStyle}>
                <FileText size={14} style={{ marginRight: '8px', display: 'inline' }} /> Posts
              </NavLink>
              <NavLink to="/boards" onClick={() => setSidebarOpen(false)} className={navLinkClass} style={navLinkStyle}>
                <FolderTree size={14} style={{ marginRight: '8px', display: 'inline' }} /> Boards
              </NavLink>
              <NavLink to="/settings" onClick={() => setSidebarOpen(false)} className={navLinkClass} style={navLinkStyle}>
                <SettingsIcon size={14} style={{ marginRight: '8px', display: 'inline' }} /> Settings
              </NavLink>
            </nav>

            <div style={{ marginTop: 'auto' }}>
              <button style={{ width: '100%', textAlign: 'left', marginTop: '16px' }} onClick={handleLogout}>
                <LogOut size={14} style={{ marginRight: '8px', display: 'inline' }} /> Logout
              </button>
            </div>
          </div>

          {/* Overlay for mobile to close sidebar when clicking outside */}
          {isMobile && sidebarOpen && (
             <div 
               style={{ position: 'absolute', top: '38px', right: 0, bottom: 0, left: 0, zIndex: 40, backgroundColor: 'rgba(0,0,0,0.5)' }} 
               onClick={() => setSidebarOpen(false)} 
             />
          )}

          {/* Main Content Area */}
          <div style={{ flex: 1, padding: isMobile ? '8px' : '16px', backgroundColor: '#fff', overflowY: 'auto', overflowX: 'hidden' }}>
            <Outlet />
          </div>
        </div>

        {/* Status Bar */}
        <div className="status-bar" style={{ margin: 0 }}>
          <p className="status-bar-field" style={{ fontSize: '12px' }}>API: {apiOnline ? '✅ Online' : '❌ Offline'}</p>
          <p className="status-bar-field" style={{ fontSize: '12px' }}>Y2K Pixel Blog v1.2</p>
        </div>
      </div>
    </div>
  );
}
