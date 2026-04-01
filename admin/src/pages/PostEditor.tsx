import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import api from '../api/client';
import axios from 'axios';

export default function PostEditor() {
  const { slug } = useParams<{ slug: string }>();
  const isEdit = !!slug;
  const navigate = useNavigate();

  const [formData, setFormData] = useState<any>({
    slug: '',
    title: '',
    type: 'article',
    status: 'published',
    date: new Date().toISOString().split('T')[0],
    tags: '',
    boards: 'start',
    excerpt: '',
    content: '',
    customFooter: '',
    // Rating
    cover: '',
    summary: '',
    score: 0.0,
    radarCharts: [],
    // Photo
    pages: [],
    // Page
    icon: '',
    order: 0,
    showInMenu: false,
    // Custom CSS
    customCSS: '',
    cssEnabled: false
  });

  const [loading, setLoading] = useState(isEdit);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  // Setup options
  const [availableBoards, setAvailableBoards] = useState<any[]>([]);

  useEffect(() => {
    // Fetch boards for dropdown
    api.get('/boards').then(res => {
      const flatten = (nodes: any[]): any[] => {
        return nodes.reduce((acc, curr) => {
           acc.push({ slug: curr.slug, name: curr.name });
           if (curr.children) {
             acc = acc.concat(flatten(curr.children));
           }
           return acc;
        }, []);
      };
      setAvailableBoards(flatten(res.data.boards || []));
    }).catch(console.error);

    if (isEdit) {
      api.get(`/posts/${slug}`).then(res => {
        const post = res.data;
        setFormData({
          ...post,
          status: post.status || 'published',
          tags: (post.tags || []).join(', '),
          boards: (post.boards || []).join(', '),
          content: post.contentRaw || post.content || '',
          radarCharts: post.radarCharts || [],
          pages: post.pages || [],
          customCSS: post.customCSS || '',
          cssEnabled: post.cssEnabled || false
        });
      }).catch(() => {
        setError("Failed to load post");
      }).finally(() => setLoading(false));
    }
  }, [slug, isEdit]);

  const handleChange = (e: any) => {
    const { name, value, type, checked } = e.target;
    setFormData((prev: any) => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value
    }));
  };

  const submitPost = async (targetStatus: string) => {
    setSaving(true);
    setError('');

    try {
      const payload = {
        ...formData,
        status: targetStatus,
        tags: formData.tags.split(',').map((t: string) => t.trim()).filter(Boolean),
        boards: formData.boards.split(',').map((b: string) => b.trim()).filter(Boolean),
        score: parseFloat(formData.score) || 0
      };

      if (isEdit) {
        await api.put(`/admin/posts/${slug}`, payload);
        alert('Post updated successfully!');
      } else {
        await api.post(`/admin/posts`, payload);
        alert('Post created successfully!');
      }
      navigate('/posts');
    } catch (err: any) {
      if (axios.isAxiosError(err) && err.response) {
        setError(err.response.data.error || 'Saved failed');
      } else {
        setError('Network error');
      }
    } finally {
      setSaving(false);
    }
  };

  const handlePublish = (e: React.MouseEvent) => {
    e.preventDefault();
    submitPost('published');
  };

  const handleSaveDraft = (e: React.MouseEvent) => {
    e.preventDefault();
    submitPost('draft');
  };

  // Mini helpers for dynamic arrays
  const addRadarChart = () => setFormData({ ...formData, radarCharts: [...formData.radarCharts, { name: 'New Axis', axes: [] }] });
  const updateRadarChartName = (idx: number, name: string) => {
    const newArr = [...formData.radarCharts];
    newArr[idx].name = name;
    setFormData({ ...formData, radarCharts: newArr });
  };
  const addRadarAxis = (idx: number) => {
    const newArr = [...formData.radarCharts];
    newArr[idx].axes.push({ label: 'Label', score: 5.0 });
    setFormData({ ...formData, radarCharts: newArr });
  };
  const updateRadarAxis = (chartIdx: number, axisIdx: number, field: string, value: any) => {
    const newArr = [...formData.radarCharts];
    newArr[chartIdx].axes[axisIdx][field] = value;
    setFormData({ ...formData, radarCharts: newArr });
  };

  const addPhotoPage = () => setFormData({ ...formData, pages: [...formData.pages, { image: '', textRaw: '' }] });
  const updatePhotoPage = (idx: number, field: string, value: any) => {
    const newArr = [...formData.pages];
    newArr[idx][field] = value;
    setFormData({ ...formData, pages: newArr });
  };

  if (loading) return <p>Loading editor...</p>;

  return (
    <div>
      <div className="flex justify-between items-center" style={{ borderBottom: '2px solid var(--border)', paddingBottom: '8px', marginBottom: '16px' }}>
        <h2>📝 {isEdit ? `Edit Post: ${slug}` : 'Create New Post'}</h2>
      </div>

      {error && <p className="error-text">❌ {error}</p>}

      <form className="flex gap-4">
        {/* Left Column (Main content) */}
        <div className="flex-1">
           <div className="window mb-4">
             <div className="title-bar"><div className="title-bar-text">Basic Info</div></div>
             <div className="window-body">
               <div className="field-row-stacked">
                 <label>Title</label>
                 <input type="text" name="title" value={formData.title} onChange={handleChange} required />
               </div>
               <div className="field-row-stacked">
                 <label>Excerpt</label>
                 <textarea name="excerpt" value={formData.excerpt} onChange={handleChange} rows={2} />
               </div>
             </div>
           </div>

           {(formData.type === 'article' || formData.type === 'rating' || formData.type === 'page') && (
             <div className="window mb-4">
               <div className="title-bar"><div className="title-bar-text">Markdown Content</div></div>
               <div className="window-body">
                 <div className="field-row-stacked">
                   <textarea name="content" value={formData.content} onChange={handleChange} rows={15} style={{ fontFamily: 'monospace' }} />
                 </div>
               </div>
             </div>
           )}

           {formData.type === 'rating' && (
             <div className="window mb-4">
               <div className="title-bar"><div className="title-bar-text">Rating Specifics</div></div>
               <div className="window-body">
                 <div className="field-row-stacked"><label>Cover URL</label><input type="text" name="cover" value={formData.cover} onChange={handleChange} /></div>
                 <div className="field-row-stacked"><label>Summary</label><input type="text" name="summary" value={formData.summary} onChange={handleChange} /></div>
                 <div className="field-row-stacked"><label>Score (0-10)</label><input type="number" step="0.1" max="10" name="score" value={formData.score} onChange={handleChange} /></div>
                 
                 <div className="mt-4">
                   <strong>Radar Charts</strong>
                   {formData.radarCharts.map((chart: any, cIdx: number) => (
                     <div key={cIdx} className="mb-2 p-2" style={{ border: '1px outset white' }}>
                       <div className="flex justify-between mb-2">
                         <input value={chart.name} onChange={(e) => updateRadarChartName(cIdx, e.target.value)} />
                         <button type="button" onClick={() => addRadarAxis(cIdx)}>Add Axis</button>
                       </div>
                       {chart.axes.map((axis: any, aIdx: number) => (
                         <div key={aIdx} className="flex gap-2 mb-1">
                           <input value={axis.label} onChange={(e) => updateRadarAxis(cIdx, aIdx, 'label', e.target.value)} placeholder="Label" />
                           <input type="number" step="0.1" value={axis.score} onChange={(e) => updateRadarAxis(cIdx, aIdx, 'score', parseFloat(e.target.value))} placeholder="Score" style={{ width: 80 }} />
                         </div>
                       ))}
                     </div>
                   ))}
                   <button type="button" onClick={addRadarChart}>Add Radar Chart</button>
                 </div>
               </div>
             </div>
           )}

           {formData.type === 'photo' && (
             <div className="window mb-4">
               <div className="title-bar"><div className="title-bar-text">Photo Pages</div></div>
               <div className="window-body">
                 {formData.pages.map((p: any, idx: number) => (
                   <div key={idx} className="mb-2 p-2" style={{ border: '1px solid gray' }}>
                     <div className="field-row-stacked"><label>Image URL</label><input type="text" value={p.image} onChange={(e) => updatePhotoPage(idx, 'image', e.target.value)} /></div>
                     <div className="field-row-stacked"><label>Text (Markdown)</label><textarea value={p.textRaw} onChange={(e) => updatePhotoPage(idx, 'textRaw', e.target.value)} rows={3} /></div>
                   </div>
                 ))}
                 <button type="button" onClick={addPhotoPage}>Add Page</button>
               </div>
             </div>
           )}

           {formData.type === 'page' && (
             <div className="window mb-4">
               <div className="title-bar"><div className="title-bar-text">Page Specifics</div></div>
               <div className="window-body">
                 <div className="field-row-stacked"><label>Icon (emoji or symbol)</label><input type="text" name="icon" value={formData.icon} onChange={handleChange} placeholder="📝 🏠" maxLength={2} style={{ width: 80 }} /></div>
                 <div className="field-row-stacked"><label>Order</label><input type="number" name="order" value={formData.order} onChange={handleChange} /></div>
                 <div className="field-row" style={{ marginTop: 8 }}>
                   <input type="checkbox" id="showInMenu" name="showInMenu" checked={formData.showInMenu} onChange={handleChange} />
                   <label htmlFor="showInMenu">Show In Menu</label>
                 </div>
               </div>
             </div>
           )}

            {/* Custom CSS */}
            <div className="window mb-4">
              <div className="title-bar"><div className="title-bar-text">🎨 Custom CSS</div></div>
              <div className="window-body">
                <div className="field-row" style={{ marginBottom: 8 }}>
                  <input
                    type="checkbox"
                    id="cssEnabled"
                    name="cssEnabled"
                    checked={formData.cssEnabled}
                    onChange={handleChange}
                  />
                  <label htmlFor="cssEnabled">Enable Custom CSS for this post</label>
                </div>
                <div className="field-row-stacked">
                  <label>CSS Code</label>
                  <textarea
                    name="customCSS"
                    value={formData.customCSS}
                    onChange={(e: any) => setFormData({ ...formData, customCSS: e.target.value })}
                    rows={8}
                    style={{ resize: 'vertical', width: '100%', fontFamily: 'monospace', fontSize: 13 }}
                    placeholder="/* Scoped CSS for this post */"
                    disabled={!formData.cssEnabled}
                  />
                </div>
              </div>
            </div>
        </div>

        {/* Right Column (Meta) */}
        <div style={{ width: 300 }}>
          <div className="window mb-4">
             <div className="title-bar"><div className="title-bar-text">Metadata</div></div>
             <div className="window-body">
               
               <div className="field-row-stacked">
                 <label>Slug (URL Identifier)</label>
                 <input type="text" name="slug" value={formData.slug} onChange={handleChange} required disabled={isEdit} />
               </div>

               <div className="field-row-stacked">
                 <label>Type</label>
                 <select name="type" value={formData.type} onChange={handleChange} disabled={isEdit}>
                   <option value="article">Article</option>
                   <option value="rating">Rating</option>
                   <option value="photo">Photo</option>
                   <option value="page">Page</option>
                 </select>
               </div>

               <div className="field-row-stacked">
                 <label>Date (YYYY-MM-DD)</label>
                 <input type="date" name="date" value={formData.date} onChange={handleChange} required />
               </div>

               <div className="field-row-stacked">
                 <label>Boards (Comma separated)</label>
                 <input type="text" name="boards" value={formData.boards} onChange={handleChange} />
                 <small>Available: {availableBoards.map(b => b.slug).join(', ')}</small>
               </div>

               <div className="field-row-stacked">
                 <label>Tags (Comma separated)</label>
                 <input type="text" name="tags" value={formData.tags} onChange={handleChange} />
               </div>

             </div>
          </div>
          <div className="flex flex-col gap-2 mt-4">
            <button type="button" onClick={handlePublish} style={{ height: 40, fontWeight: 'bold' }} disabled={saving}>
              {saving ? 'Processing...' : '🚀 Publish Post'}
            </button>
            <button type="button" onClick={handleSaveDraft} disabled={saving}>
              {saving ? 'Processing...' : '💾 Save as Draft'}
            </button>
            <button type="button" onClick={() => navigate('/posts')} disabled={saving}>
              Cancel
            </button>
          </div>
        </div>
      </form>
    </div>
  );
}
