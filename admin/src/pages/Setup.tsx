import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import api from '../api/client';
import axios from 'axios';

export default function Setup() {
  const [formData, setFormData] = useState({
    siteTitle: 'Y2K Pixel Blog',
    siteDescription: '一个复古像素风格的博客',
    siteFooter: '© 2026 Y2K Pixel Blog',
    adminUsername: 'admin',
    adminPassword: '',
    confirmPassword: '',
    aiApiUrl: '',
    aiApiKey: '',
    aiModel: '',
    firstBoardSlug: 'start',
    firstBoardName: 'Start',
    firstBoardIcon: '⭐',
  });
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [loading, setLoading] = useState(false);
  
  const navigate = useNavigate();

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFormData({ ...formData, [e.target.name]: e.target.value });
  };

  const handleSetup = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    
    if (formData.adminPassword !== formData.confirmPassword) {
      setError('Passwords do not match');
      return;
    }

    if (formData.adminPassword.length < 6) {
      setError('Password must be at least 6 characters');
      return;
    }

    setLoading(true);

    try {
      const response = await api.post('/setup/initialize', {
        siteTitle: formData.siteTitle,
        siteDescription: formData.siteDescription,
        siteFooter: formData.siteFooter,
        adminUsername: formData.adminUsername,
        adminPassword: formData.adminPassword,
        aiApiUrl: formData.aiApiUrl,
        aiApiKey: formData.aiApiKey,
        aiModel: formData.aiModel,
        firstBoardSlug: formData.firstBoardSlug,
        firstBoardName: formData.firstBoardName,
        firstBoardIcon: formData.firstBoardIcon,
      });

      if (response.status === 201 || response.status === 200) {
        setSuccess('🎉 Installation Complete! Redirecting to login...');
        setTimeout(() => {
          navigate('/login');
        }, 2000);
      }
    } catch (err: any) {
      if (axios.isAxiosError(err) && err.response) {
        setError(err.response.data.error || 'Set up failed');
      } else {
        setError('Network error or server down');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex h-screen items-center justify-center p-4 overflow-y-auto">
      <div className="window y2k-window" style={{ width: '100%', maxWidth: 500 }}>
        <div className="title-bar">
          <div className="title-bar-text">🖥️ Y2K Pixel Blog — Initial Setup</div>
          <div className="title-bar-controls">
            <button aria-label="Close" />
          </div>
        </div>

        <div className="window-body">
          <p>Welcome! Please complete the initial setup to start using your blog.</p>

          <form onSubmit={handleSetup} className="flex-col gap-4 mt-4">
            <fieldset>
              <legend>📌 Step 1: Site Info</legend>
              <div className="field-row-stacked">
                <label>Site Title</label>
                <input name="siteTitle" type="text" value={formData.siteTitle} onChange={handleChange} required />
              </div>
              <div className="field-row-stacked">
                <label>Site Description</label>
                <input name="siteDescription" type="text" value={formData.siteDescription} onChange={handleChange} />
              </div>
              <div className="field-row-stacked">
                <label>Site Footer</label>
                <input name="siteFooter" type="text" value={formData.siteFooter} onChange={handleChange} />
              </div>
            </fieldset>

            <fieldset>
              <legend>👤 Step 2: Admin Account</legend>
              <div className="field-row-stacked">
                <label>Username</label>
                <input name="adminUsername" type="text" value={formData.adminUsername} onChange={handleChange} required />
              </div>
              <div className="field-row-stacked">
                <label>Password</label>
                <input name="adminPassword" type="password" value={formData.adminPassword} onChange={handleChange} required minLength={6} />
              </div>
              <div className="field-row-stacked">
                <label>Confirm Password</label>
                <input name="confirmPassword" type="password" value={formData.confirmPassword} onChange={handleChange} required minLength={6} />
              </div>
            </fieldset>

            <fieldset>
              <legend>🤖 Step 3: AI Config (Optional)</legend>
              <div className="field-row-stacked">
                <label>API URL</label>
                <input name="aiApiUrl" type="text" value={formData.aiApiUrl} onChange={handleChange} placeholder="https://api..." />
              </div>
              <div className="field-row-stacked">
                <label>API Key</label>
                <input name="aiApiKey" type="password" value={formData.aiApiKey} onChange={handleChange} placeholder="sk-..." />
              </div>
              <div className="field-row-stacked">
                <label>Model Name</label>
                <input name="aiModel" type="text" value={formData.aiModel} onChange={handleChange} placeholder="deepseek-v3" />
              </div>
            </fieldset>

            <fieldset>
              <legend>📁 Step 4: First Board (Optional)</legend>
              <div className="field-row-stacked">
                <label>Board Slug</label>
                <input name="firstBoardSlug" type="text" value={formData.firstBoardSlug} onChange={handleChange} placeholder="my-blog" />
              </div>
              <div className="field-row-stacked">
                <label>Board Name</label>
                <input name="firstBoardName" type="text" value={formData.firstBoardName} onChange={handleChange} placeholder="My Blog" />
              </div>
              <div className="field-row-stacked">
                <label>Board Icon (emoji or symbol)</label>
                <input name="firstBoardIcon" type="text" value={formData.firstBoardIcon} onChange={handleChange} placeholder="⭐ 💻 📝" maxLength={2} style={{ width: 80 }} />
              </div>
            </fieldset>

            {error && <p className="error-text">❌ {error}</p>}
            {success && <p style={{ color: 'green', fontWeight: 'bold' }}>{success}</p>}

            <div className="field-row mt-4" style={{ justifyContent: 'center' }}>
              <button disabled={loading || !!success} type="submit" style={{ width: 150 }}>
                {loading ? 'Processing...' : '🚀 Let\'s Go!'}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}
