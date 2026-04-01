import { useEffect, useState } from 'react';
import api from '../api/client';
import { Trash2 } from 'lucide-react';
import axios from 'axios';

export default function Settings() {
  const [formData, setFormData] = useState<any>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  
  const [aiCacheSlug, setAiCacheSlug] = useState('');

  // Password change state
  const [pwForm, setPwForm] = useState({ oldPassword: '', newPassword: '', confirmPassword: '' });
  const [pwSaving, setPwSaving] = useState(false);
  const [pwMsg, setPwMsg] = useState({ text: '', isError: false });

  // SSL state
  const [frontendSslForm, setFrontendSslForm] = useState({ certPEM: '', keyPEM: '' });
  const [adminSslForm, setAdminSslForm] = useState({ certPEM: '', keyPEM: '' });
  const [sslSaving, setSslSaving] = useState(false);
  const [sslMsgs, setSslMsgs] = useState<Record<string, { text: string; type: string }>>({});
  const [domainMsg, setDomainMsg] = useState({ text: '', type: '' });

  useEffect(() => {
    fetchSettings();
  }, []);

  const fetchSettings = async () => {
    try {
      const res = await api.get('/admin/settings');
      setFormData({
        ...res.data,
        frontendSslMode: res.data.frontendSslMode || 'off',
        adminSslMode: res.data.adminSslMode || 'off',
      });
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    try {
      await api.put('/admin/settings', formData);
      alert('Settings updated successfully!');
    } catch (err) {
      alert('Failed to update settings');
    } finally {
      setSaving(false);
    }
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFormData({ ...formData, [e.target.name]: e.target.value });
  };

  const handleClearAiCache = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!aiCacheSlug) return;
    if (window.confirm(`Clear AI cache for "${aiCacheSlug}"?`)) {
      try {
        await api.delete(`/admin/ai-cache/${aiCacheSlug}`);
        alert('Cache cleared!');
        setAiCacheSlug('');
      } catch (err) {
        alert('Failed to clear cache');
      }
    }
  };

  const handlePasswordChange = async (e: React.FormEvent) => {
    e.preventDefault();
    setPwMsg({ text: '', isError: false });

    if (pwForm.newPassword !== pwForm.confirmPassword) {
      setPwMsg({ text: 'New passwords do not match.', isError: true });
      return;
    }
    if (pwForm.newPassword.length < 6) {
      setPwMsg({ text: 'New password must be at least 6 characters.', isError: true });
      return;
    }

    setPwSaving(true);
    try {
      await api.put('/admin/password', {
        oldPassword: pwForm.oldPassword,
        newPassword: pwForm.newPassword,
      });
      setPwMsg({ text: 'Password changed successfully!', isError: false });
      setPwForm({ oldPassword: '', newPassword: '', confirmPassword: '' });
    } catch (err: any) {
      if (axios.isAxiosError(err) && err.response) {
        setPwMsg({ text: err.response.data.error || 'Failed to change password.', isError: true });
      } else {
        setPwMsg({ text: 'Network error.', isError: true });
      }
    } finally {
      setPwSaving(false);
    }
  };

  const handleCertFile = async (e: React.ChangeEvent<HTMLInputElement>, target: 'frontend' | 'admin') => {
    const file = e.target.files?.[0];
    if (file) {
      const text = await file.text();
      if (target === 'frontend') setFrontendSslForm({ ...frontendSslForm, certPEM: text });
      if (target === 'admin') setAdminSslForm({ ...adminSslForm, certPEM: text });
    }
  };

  const handleKeyFile = async (e: React.ChangeEvent<HTMLInputElement>, target: 'frontend' | 'admin') => {
    const file = e.target.files?.[0];
    if (file) {
      const text = await file.text();
      if (target === 'frontend') setFrontendSslForm({ ...frontendSslForm, keyPEM: text });
      if (target === 'admin') setAdminSslForm({ ...adminSslForm, keyPEM: text });
    }
  };

  const handleSaveSSL = async (e: React.FormEvent, target: 'frontend' | 'admin') => {
    e.preventDefault();
    setSslMsgs({ ...sslMsgs, [target]: { text: '', type: '' } });
    setSslSaving(true);
    const form = target === 'frontend' ? frontendSslForm : adminSslForm;
    try {
      const res = await api.put('/admin/ssl', { ...form, target, enabled: true });
      setSslMsgs({ ...sslMsgs, [target]: { text: res.data.restartRequired ? 'SSL cert uploaded! ⚠️ Please restart.' : 'SSL cert uploaded!', type: res.data.restartRequired ? 'warning' : 'success' } });
      
      const newMap = { ...formData };
      if (target === 'frontend') {
        newMap.frontendSslHasCert = true;
        setFrontendSslForm({ certPEM: '', keyPEM: '' });
      } else {
        newMap.adminSslHasCert = true;
        setAdminSslForm({ certPEM: '', keyPEM: '' });
      }
      setFormData(newMap);
    } catch (err: any) {
      if (axios.isAxiosError(err) && err.response) {
        setSslMsgs({ ...sslMsgs, [target]: { text: err.response.data.error || 'Failed to save SSL.', type: 'error' } });
      } else {
        setSslMsgs({ ...sslMsgs, [target]: { text: 'Network Error.', type: 'error' } });
      }
    } finally {
      setSslSaving(false);
    }
  };

  const handleRemoveSSL = async (target: 'frontend' | 'admin') => {
    if (window.confirm(`Are you sure you want to completely remove the ${target} SSL certificate?`)) {
      setSslSaving(true);
      try {
        await api.delete(`/admin/ssl?target=${target}`);
        const newMap = { ...formData };
        if (target === 'frontend') {
            newMap.frontendSslHasCert = false;
        } else {
            newMap.adminSslHasCert = false;
        }
        setFormData(newMap);
        setSslMsgs({ ...sslMsgs, [target]: { text: 'SSL removed successfully. ⚠️ Restart the server.', type: 'warning' } });
      } catch {
        setSslMsgs({ ...sslMsgs, [target]: { text: 'Failed to remove SSL.', type: 'error' } });
      } finally {
        setSslSaving(false);
      }
    }
  };

  const handleSaveDomains = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setDomainMsg({ text: '', type: '' });
    try {
      await api.put('/admin/settings', formData);
      setDomainMsg({ text: 'Domains & SSL setup saved! ⚠️ Requires server restart to fully apply.', type: 'warning' });
    } catch (err) {
      setDomainMsg({ text: 'Failed to update domains', type: 'error' });
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <p>Loading settings...</p>;

  return (
    <div style={{ display: 'flex', gap: '16px' }}>
      {/* Settings Form */}
      <div style={{ flex: 1 }}>
        <div className="window mb-4">
          <div className="title-bar"><div className="title-bar-text">⚙️ Site Settings</div></div>
          <div className="window-body">
            <form onSubmit={handleSave}>
              <fieldset>
                <legend>General</legend>
                <div className="field-row-stacked">
                  <label>Site Title</label>
                  <input name="siteTitle" type="text" value={formData.siteTitle || ''} onChange={handleChange} />
                </div>
                <div className="field-row-stacked">
                  <label>Site Description</label>
                  <input name="siteDescription" type="text" value={formData.siteDescription || ''} onChange={handleChange} />
                </div>
                <div className="field-row-stacked">
                  <label>Site Footer (supports HTML: &lt;a&gt;, &lt;b&gt;, &lt;span&gt;)</label>
                  <input name="siteFooter" type="text" value={formData.siteFooter || ''} onChange={handleChange} placeholder='© 2026 Blog | <a href="...">About</a>' />
                </div>
                <div className="field-row-stacked">
                  <label>Logo URL</label>
                  <input name="siteLogoUrl" type="text" value={formData.siteLogoUrl || ''} onChange={handleChange} placeholder="https://example.com/logo.png (or data URI)" />
                  {formData.siteLogoUrl && (
                    <div style={{ marginTop: 8, padding: 8, backgroundColor: '#f0f0f0', border: '1px solid #ccc', textAlign: 'center' }}>
                      <img src={formData.siteLogoUrl} alt="Logo Preview" style={{ maxWidth: 64, maxHeight: 64, imageRendering: 'pixelated' }} />
                    </div>
                  )}
                </div>
              </fieldset>

              <fieldset style={{ marginTop: '16px' }}>
                <legend>Content License</legend>
                <div className="field-row-stacked" style={{ marginBottom: 8 }}>
                  <label>License Type</label>
                  <select
                    name="siteLicense"
                    value={formData.siteLicense || ''}
                    onChange={(e: any) => setFormData({ ...formData, siteLicense: e.target.value })}
                    style={{ width: '100%' }}
                  >
                    <option value="">Not Set</option>
                    <option value="CC-BY-4.0">CC BY 4.0</option>
                    <option value="CC-BY-SA-4.0">CC BY-SA 4.0</option>
                    <option value="CC-BY-NC-4.0">CC BY-NC 4.0</option>
                    <option value="CC-BY-NC-SA-4.0">CC BY-NC-SA 4.0</option>
                    <option value="CC-BY-NC-ND-4.0">CC BY-NC-ND 4.0</option>
                    <option value="CC-BY-ND-4.0">CC BY-ND 4.0</option>
                    <option value="CC0-1.0">CC0 1.0 (Public Domain)</option>
                    <option value="MIT">MIT</option>
                    <option value="All Rights Reserved">All Rights Reserved</option>
                  </select>
                </div>
                <div className="field-row-stacked">
                  <label>License URL (optional, full text link)</label>
                  <input
                    name="siteLicenseUrl"
                    type="text"
                    value={formData.siteLicenseUrl || ''}
                    onChange={handleChange}
                    placeholder="https://creativecommons.org/licenses/by-nc-sa/4.0/"
                  />
                </div>
                <p style={{ fontSize: 12, color: '#666', margin: '8px 0 0 0' }}>
                  Selected license will be displayed on the frontend desktop as a clickable icon.
                </p>
              </fieldset>

              <fieldset style={{ marginTop: '16px' }}>
                <legend>AI Configuration</legend>
                <div className="field-row-stacked">
                  <label>API URL</label>
                  <input name="aiApiUrl" type="text" value={formData.aiApiUrl || ''} onChange={handleChange} />
                </div>
                <div className="field-row-stacked">
                  <label>API Key</label>
                  <input name="aiApiKey" type="password" value={formData.aiApiKey || ''} placeholder="Leave blank to keep unchanged" onChange={handleChange} />
                </div>
                <div className="field-row-stacked">
                  <label>Model Name</label>
                  <input name="aiModel" type="text" value={formData.aiModel || ''} onChange={handleChange} />
                </div>
              </fieldset>

              <fieldset style={{ marginTop: '16px' }}>
                <legend>Custom CSS</legend>
                <div className="field-row-stacked" style={{ marginBottom: 8 }}>
                  <label>Global CSS (applied to all pages)</label>
                  <textarea
                    name="globalCSS"
                    value={formData.globalCSS || ''}
                    onChange={(e: any) => setFormData({ ...formData, globalCSS: e.target.value })}
                    rows={6}
                    style={{ resize: 'vertical', width: '100%', fontFamily: 'monospace', fontSize: 13 }}
                    placeholder="/* e.g. body { font-family: 'Noto Sans SC'; } */"
                  />
                </div>
                <div style={{ marginBottom: 4 }}>
                  <label style={{ fontWeight: 'bold', fontSize: 13 }}>Enable per-post Custom CSS for types:</label>
                </div>
                {['article', 'photo', 'rating', 'page'].map((t) => {
                  const enabledTypes = (formData.customCSSEnabledTypes || 'article,photo,rating,page').split(',').map((s: string) => s.trim());
                  const checked = enabledTypes.includes(t);
                  return (
                    <div className="field-row" key={t} style={{ marginBottom: 2 }}>
                      <input
                        type="checkbox"
                        id={`css-type-${t}`}
                        checked={checked}
                        onChange={(e) => {
                          let types = enabledTypes.filter(Boolean);
                          if (e.target.checked) {
                            if (!types.includes(t)) types.push(t);
                          } else {
                            types = types.filter((x: string) => x !== t);
                          }
                          setFormData({ ...formData, customCSSEnabledTypes: types.join(',') });
                        }}
                      />
                      <label htmlFor={`css-type-${t}`}>{t}</label>
                    </div>
                  );
                })}
              </fieldset>
              
              <div style={{ marginTop: '16px', textAlign: 'right' }}>
                 <button type="submit" disabled={saving}>{saving ? 'Saving...' : 'Save Settings'}</button>
              </div>
            </form>
          </div>
        </div>

        {/* Domain & SSL Management */}
        <div className="window mb-4">
          <div className="title-bar"><div className="title-bar-text">🌐 Domain & SSL Management</div></div>
          <div className="window-body">
            
            <form onSubmit={handleSaveDomains} style={{ marginBottom: 16 }}>
              <div className="field-row-stacked" style={{ marginBottom: 8 }}>
                <label>Frontend Domain (e.g., blog.example.com)</label>
                <input 
                  name="frontendDomain" 
                  type="text" 
                  value={formData.frontendDomain || ''} 
                  onChange={handleChange} 
                  placeholder="blog.example.com" 
                  style={{ width: '100%' }} 
                />
              </div>
              <div className="field-row-stacked" style={{ marginBottom: 16 }}>
                <label>Admin Domain (e.g., admin.example.com)</label>
                <input 
                  name="adminDomain" 
                  type="text" 
                  value={formData.adminDomain || ''} 
                  onChange={handleChange} 
                  placeholder="admin.example.com" 
                  style={{ width: '100%' }} 
                />
              </div>
              <div className="field-row-stacked" style={{ marginBottom: 16 }}>
                <label>ACME Email (Used for Let's Encrypt registration & expiry notices)</label>
                <input 
                  name="acmeEmail" 
                  type="email" 
                  value={formData.acmeEmail || ''} 
                  onChange={handleChange} 
                  placeholder="admin@example.com" 
                  style={{ width: '100%' }} 
                  required={formData.frontendSslMode === 'auto' || formData.adminSslMode === 'auto'}
                />
              </div>

              <div style={{ display: 'flex', gap: '16px', flexDirection: 'column' }}>
                {['frontend', 'admin'].map((t) => {
                  const target = t as 'frontend' | 'admin';
                  const title = target === 'frontend' ? 'Frontend SSL Configuration' : 'Admin SSL Configuration';
                  const domain = target === 'frontend' ? formData.frontendDomain : formData.adminDomain;
                  const mode = target === 'frontend' ? formData.frontendSslMode : formData.adminSslMode;
                  const hasCert = target === 'frontend' ? formData.frontendSslHasCert : formData.adminSslHasCert;
                  const msg = sslMsgs[target];
                  const form = target === 'frontend' ? frontendSslForm : adminSslForm;

                  let statusIndicator = null;
                  if (mode === 'auto' && domain) statusIndicator = <span style={{ color: 'green', fontWeight: 'bold' }}>🔐 自动证书</span>;
                  else if (mode === 'manual' && hasCert) statusIndicator = <span style={{ color: 'blue', fontWeight: 'bold' }}>🔒 已配置</span>;
                  else if (mode === 'manual' && !hasCert) statusIndicator = <span style={{ color: '#d08000', fontWeight: 'bold' }}>⚠️ 待上传</span>;
                  else statusIndicator = <span style={{ color: 'gray', fontWeight: 'bold' }}>❌ 未启用</span>;

                  return (
                    <fieldset key={target} style={{ flex: 1 }}>
                      <legend style={{ display: 'flex', justifyContent: 'space-between', width: '100%' }}>
                        <span>{title}</span>
                        <span style={{ fontSize: 13, float: 'right' }}>{statusIndicator}</span>
                      </legend>

                      <div style={{ marginBottom: 12 }}>
                        <div className="field-row" style={{ marginBottom: 4 }}>
                          <input type="radio" id={`mode-off-${target}`} name={`${target}SslMode`} value="off" checked={mode === 'off'} onChange={handleChange} />
                          <label htmlFor={`mode-off-${target}`}>Off (HTTP only)</label>
                        </div>
                        <div className="field-row" style={{ marginBottom: 4 }}>
                          <input type="radio" id={`mode-auto-${target}`} name={`${target}SslMode`} value="auto" checked={mode === 'auto'} onChange={handleChange} />
                          <label htmlFor={`mode-auto-${target}`}>Auto (Let's Encrypt)</label>
                        </div>
                        <div className="field-row">
                          <input type="radio" id={`mode-manual-${target}`} name={`${target}SslMode`} value="manual" checked={mode === 'manual'} onChange={handleChange} />
                          <label htmlFor={`mode-manual-${target}`}>Manual (Upload PEM)</label>
                        </div>
                      </div>

                      {mode === 'off' && (
                        <div style={{ padding: 8, backgroundColor: '#eee', color: '#666', fontSize: 13, border: '1px solid #ccc' }}>
                          SSL 尚未启用，访问将继续使用原本的协议（通常是 HTTP），这可能会带来安全性隐患或浏览器相关API不可用。
                        </div>
                      )}

                      {mode === 'auto' && (
                        <div style={{ padding: 8, backgroundColor: '#d0ffd0', color: '#006600', fontSize: 13, border: '1px solid green' }}>
                          <strong>🔐 Let's Encrypt 自动证书</strong><br />
                          系统将尝试自动为 {domain || '该域名'} 签发证书。请确保 <code>:80</code> 必须公开暴露能由任意网络访客连通（ACME 挑战验证用）。
                          {(!formData.acmeEmail) && (
                            <div style={{ color: 'red', marginTop: 4 }}>⚠️ 请在上方填写 ACME Email 以保证顺利完成服务通讯设定。</div>
                          )}
                        </div>
                      )}

                      {mode === 'manual' && (
                        <div>
                          <p style={{ margin: '0 0 8px 0', fontSize: 13, color: '#444' }}>
                            Paste your PEM format certificate (.crt/.cer) and unencrypted private key (.key), or upload them.
                          </p>
                          <div style={{ display: 'flex', gap: '16px', marginBottom: 16 }}>
                            <div className="field-row-stacked" style={{ flex: 1 }}>
                              <label style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                                SSL Cert (PEM)
                                <input type="file" accept=".pem,.crt,.cer,.txt" onChange={(e) => handleCertFile(e, target)} style={{ fontSize: 11, width: 140 }} />
                              </label>
                              <textarea
                                value={form.certPEM}
                                onChange={(e) => {
                                  if (target === 'frontend') setFrontendSslForm({ ...frontendSslForm, certPEM: e.target.value });
                                  else setAdminSslForm({ ...adminSslForm, certPEM: e.target.value });
                                }}
                                rows={6}
                                style={{ resize: 'vertical', width: '100%', fontFamily: 'monospace', fontSize: 11 }}
                                placeholder="-----BEGIN CERTIFICATE-----\n..."
                              />
                            </div>
                            
                            <div className="field-row-stacked" style={{ flex: 1 }}>
                              <label style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                                Private Key (PEM)
                                <input type="file" accept=".pem,.key,.txt" onChange={(e) => handleKeyFile(e, target)} style={{ fontSize: 11, width: 140 }} />
                              </label>
                              <textarea
                                value={form.keyPEM}
                                onChange={(e) => {
                                  if (target === 'frontend') setFrontendSslForm({ ...frontendSslForm, keyPEM: e.target.value });
                                  else setAdminSslForm({ ...adminSslForm, keyPEM: e.target.value });
                                }}
                                rows={6}
                                style={{ resize: 'vertical', width: '100%', fontFamily: 'monospace', fontSize: 11 }}
                                placeholder="-----BEGIN PRIVATE KEY-----\n..."
                              />
                            </div>
                          </div>

                          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                            <button type="button" onClick={(e) => handleSaveSSL(e, target)} disabled={sslSaving} style={{ fontWeight: 'bold' }}>
                              {sslSaving ? 'Uploading...' : '⬆️ Upload Cert'}
                            </button>

                            {hasCert && (
                              <button type="button" onClick={() => handleRemoveSSL(target)} disabled={sslSaving} style={{ color: 'red' }}>
                                ❌ Remove SSL
                              </button>
                            )}
                          </div>

                          {msg?.text && (
                            <div style={{ 
                              marginTop: '8px', padding: 8, fontSize: 12, fontWeight: 'bold',
                              backgroundColor: msg.type === 'error' ? '#ffd0d0' : (msg.type === 'warning' ? '#ffedcc' : '#d0ffd0'),
                              border: '1px solid currentColor',
                              color: msg.type === 'error' ? 'red' : (msg.type === 'warning' ? '#b07000' : 'green')
                            }}>
                              {msg.text}
                            </div>
                          )}
                        </div>
                      )}
                    </fieldset>
                  );
                })}
              </div>

              {domainMsg?.text && (
                <div style={{ 
                  marginTop: '16px', padding: 8, fontSize: 13, fontWeight: 'bold',
                  backgroundColor: domainMsg.type === 'error' ? '#ffd0d0' : (domainMsg.type === 'warning' ? '#ffedcc' : '#d0ffd0'),
                  border: '1px solid currentColor',
                  color: domainMsg.type === 'error' ? 'red' : (domainMsg.type === 'warning' ? '#b07000' : 'green')
                }}>
                  {domainMsg.text}
                </div>
              )}

              <div style={{ marginTop: '16px', textAlign: 'right', borderTop: '1px solid #ccc', paddingTop: 16 }}>
                <button type="submit" disabled={saving} style={{ fontWeight: 'bold', padding: '4px 16px' }}>
                  {saving ? 'Saving...' : '💾 Save Domains & SSL Config'}
                </button>
              </div>
            </form>
          </div>
        </div>
      </div>

      {/* Right Column */}
      <div style={{ width: 350 }}>
        {/* AI Cache Manager */}
        <div className="window" style={{ marginBottom: '16px' }}>
          <div className="title-bar"><div className="title-bar-text">🤖 AI Cache Manager</div></div>
          <div className="window-body">
            <p>Clear the AI summary cache for a specific post.</p>
            <form onSubmit={handleClearAiCache}>
               <div className="field-row-stacked" style={{ marginTop: 12 }}>
                 <textarea 
                   placeholder="Post Slug (e.g., star-beyond)" 
                   value={aiCacheSlug} 
                   onChange={(e) => setAiCacheSlug(e.target.value)} 
                   required
                   rows={3}
                   style={{ resize: 'vertical', width: '100%', fontFamily: 'monospace', fontSize: 13 }}
                 />
               </div>
               <button type="submit" style={{ marginTop: 8, width: '100%' }}>
                 <Trash2 size={12} style={{ display: 'inline', marginRight: 4 }} /> Clear Cache
               </button>
            </form>
          </div>
        </div>

        {/* Change Password */}
        <div className="window">
          <div className="title-bar"><div className="title-bar-text">🔒 Change Password</div></div>
          <div className="window-body">
            <form onSubmit={handlePasswordChange}>
              <div className="field-row-stacked" style={{ marginBottom: 8 }}>
                <label>Current Password</label>
                <input 
                  type="password" 
                  value={pwForm.oldPassword} 
                  onChange={(e) => setPwForm({ ...pwForm, oldPassword: e.target.value })} 
                  required
                />
              </div>
              <div className="field-row-stacked" style={{ marginBottom: 8 }}>
                <label>New Password</label>
                <input 
                  type="password" 
                  value={pwForm.newPassword} 
                  onChange={(e) => setPwForm({ ...pwForm, newPassword: e.target.value })} 
                  required
                  minLength={6}
                />
              </div>
              <div className="field-row-stacked" style={{ marginBottom: 8 }}>
                <label>Confirm New Password</label>
                <input 
                  type="password" 
                  value={pwForm.confirmPassword} 
                  onChange={(e) => setPwForm({ ...pwForm, confirmPassword: e.target.value })} 
                  required
                  minLength={6}
                />
              </div>

              {pwMsg.text && (
                <p style={{ 
                  fontSize: 12, 
                  marginBottom: 8,
                  color: pwMsg.isError ? 'red' : 'green' 
                }}>
                  {pwMsg.isError ? '❌' : '✅'} {pwMsg.text}
                </p>
              )}

              <button type="submit" disabled={pwSaving} style={{ width: '100%' }}>
                {pwSaving ? 'Changing...' : 'Change Password'}
              </button>
            </form>
          </div>
        </div>
      </div>
    </div>
  );
}
