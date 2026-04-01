import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import api from '../api/client';

export default function PostList() {
  const [posts, setPosts] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [currentTab, setCurrentTab] = useState<'published' | 'draft' | 'trashed'>('published');
  
  // Modals
  const [deleteModal, setDeleteModal] = useState({ isOpen: false, slug: '', checked: false });
  const [emptyTrashModal, setEmptyTrashModal] = useState({ isOpen: false, checked: false });

  const navigate = useNavigate();

  useEffect(() => {
    fetchPosts();
  }, [currentTab]);

  const fetchPosts = async () => {
    try {
      setLoading(true);
      const res = await api.get(`/admin/posts?status=${currentTab}&limit=100`);
      setPosts(res.data.results || []);
    } catch (err) {
      setError('Failed to fetch posts');
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  // Actions
  const handlePublish = async (slug: string) => {
    try {
      await api.put(`/admin/posts/${slug}`, { status: 'published' });
      fetchPosts();
    } catch (err) {
      alert('Failed to publish post');
    }
  };

  const handleTrash = async (slug: string) => {
    try {
      await api.post(`/admin/posts/${slug}/trash`);
      fetchPosts();
    } catch (err) {
      alert('Failed to move post to trash');
    }
  };

  const handleRestore = async (slug: string) => {
    try {
      await api.post(`/admin/posts/${slug}/restore`);
      fetchPosts();
    } catch (err) {
      alert('Failed to restore post');
    }
  };

  const handlePreview = async (slug: string) => {
    try {
      const res = await api.post(`/admin/preview/${slug}`);
      const { token, previewUrl } = res.data;
      // Open the blog frontend preview page in a new tab
      window.open(previewUrl || `http://localhost:3000/preview/${token}`, '_blank');
    } catch (err) {
      alert('Failed to generate preview token');
    }
  };

  const handleDeleteClick = (slug: string) => {
    setDeleteModal({ isOpen: true, slug, checked: false });
  };

  const confirmDelete = async () => {
    if (!deleteModal.checked) return;
    try {
      await api.delete(`/admin/posts/${deleteModal.slug}`);
      setPosts(posts.filter((p) => p.slug !== deleteModal.slug));
      setDeleteModal({ isOpen: false, slug: '', checked: false });
    } catch (err) {
      alert('Failed to delete post');
    }
  };

  const confirmEmptyTrash = async () => {
    if (!emptyTrashModal.checked) return;
    try {
      await api.delete('/admin/trash');
      setEmptyTrashModal({ isOpen: false, checked: false });
      fetchPosts();
    } catch (err) {
      alert('Failed to empty trash');
    }
  };

  return (
    <div>
      <div className="flex justify-between items-center" style={{ borderBottom: '2px solid var(--border)', paddingBottom: '8px', marginBottom: '16px' }}>
        <h2>📝 Posts Management</h2>
        <div className="flex gap-2">
          {currentTab === 'trashed' && (
            <button onClick={() => setEmptyTrashModal({ isOpen: true, checked: false })}>
              Empty Trash
            </button>
          )}
          <button onClick={() => navigate('/posts/new')}>
            New Post
          </button>
        </div>
      </div>

      <menu role="tablist" className="mb-4">
        <li role="tab" aria-selected={currentTab === 'published'}>
          <a href="#published" onClick={(e) => { e.preventDefault(); setCurrentTab('published'); }}>Published</a>
        </li>
        <li role="tab" aria-selected={currentTab === 'draft'}>
          <a href="#drafts" onClick={(e) => { e.preventDefault(); setCurrentTab('draft'); }}>Drafts</a>
        </li>
        <li role="tab" aria-selected={currentTab === 'trashed'}>
          <a href="#trash" onClick={(e) => { e.preventDefault(); setCurrentTab('trashed'); }}>Trash</a>
        </li>
      </menu>

      {error && <p className="error-text">{error}</p>}

      {loading ? (
        <p>Loading posts...</p>
      ) : (
        <div className="window w-full">
          <div className="title-bar">
            <div className="title-bar-text">
               {currentTab === 'published' ? 'Published Posts' : currentTab === 'draft' ? 'Drafts' : 'Recycle Bin'}
            </div>
          </div>
          <div className="window-body" style={{ padding: 0 }}>
            <table className="w-full" style={{ borderCollapse: 'collapse', textAlign: 'left', fontSize: '14px' }}>
              <thead>
                <tr style={{ borderBottom: '1px solid gray', backgroundColor: '#e0dfdf' }}>
                  <th className="p-2">Title</th>
                  <th className="p-2 w-24">Type</th>
                  <th className="p-2 w-24">Date</th>
                  <th className="p-2">Tags</th>
                  <th className="p-2 w-16 text-center">Score</th>
                  <th className="p-2 text-center w-56">Actions</th>
                </tr>
              </thead>
              <tbody>
                {posts.map((post) => (
                  <tr key={post.slug} style={{ borderBottom: '1px solid #ccc' }} className="hover:bg-gray-100">
                    <td className="p-2 font-medium">{post.title}</td>
                    <td className="p-2">
                      <span className="tag-badge" style={{ backgroundColor: 'var(--border)' }}>{post.type}</span>
                    </td>
                    <td className="p-2">{post.date}</td>
                    <td className="p-2">
                      {post.tags?.map((tag: string) => (
                        <span key={tag} className="tag-badge">{tag}</span>
                      ))}
                    </td>
                    <td className="p-2 text-center text-sm">{post.score ? post.score.toFixed(1) : '-'}</td>
                    <td className="p-2 text-center flex justify-center gap-1">
                      {/* Published Actions */}
                      {currentTab === 'published' && (
                        <>
                          <button onClick={() => navigate(`/posts/edit/${post.slug}`)}>Edit</button>
                          <button onClick={() => handleTrash(post.slug)}>Trash</button>
                        </>
                      )}

                      {/* Draft Actions */}
                      {currentTab === 'draft' && (
                        <>
                          <button onClick={() => handlePreview(post.slug)}>Preview</button>
                          <button onClick={() => navigate(`/posts/edit/${post.slug}`)}>Edit</button>
                          <button onClick={() => handlePublish(post.slug)}>Publish</button>
                        </>
                      )}

                      {/* Trashed Actions */}
                      {currentTab === 'trashed' && (
                        <>
                          <button onClick={() => handleRestore(post.slug)}>Restore</button>
                        </>
                      )}
                      
                      <button onClick={() => handleDeleteClick(post.slug)}>Delete</button>

                    </td>
                  </tr>
                ))}
                {posts.length === 0 && (
                  <tr>
                    <td colSpan={6} className="p-4 text-center">No posts found in this section.</td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* MODALS */}
      {deleteModal.isOpen && (
        <div style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, backgroundColor: 'rgba(0,0,0,0.5)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000 }}>
          <div className="window" style={{ width: 350 }}>
            <div className="title-bar">
              <div className="title-bar-text">Confirm Delete</div>
              <div className="title-bar-controls">
                <button aria-label="Close" onClick={() => setDeleteModal({ isOpen: false, slug: '', checked: false })} />
              </div>
            </div>
            <div className="window-body">
              <p>Are you sure you want to delete the post <strong>{deleteModal.slug}</strong>?</p>
              <p className="error-text mb-4">This action cannot be undone!</p>
              
              <div className="field-row" style={{ marginBottom: 16 }}>
                <input 
                  type="checkbox" 
                  id="confirm-del" 
                  checked={deleteModal.checked} 
                  onChange={(e) => setDeleteModal({ ...deleteModal, checked: e.target.checked })} 
                />
                <label htmlFor="confirm-del">Yes, I am sure I want to delete this</label>
              </div>
              
              <div className="flex justify-end gap-2">
                <button onClick={() => setDeleteModal({ isOpen: false, slug: '', checked: false })}>Cancel</button>
                <button disabled={!deleteModal.checked} onClick={confirmDelete}>OK</button>
              </div>
            </div>
          </div>
        </div>
      )}

      {emptyTrashModal.isOpen && (
        <div style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, backgroundColor: 'rgba(0,0,0,0.5)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000 }}>
          <div className="window" style={{ width: 350 }}>
            <div className="title-bar">
              <div className="title-bar-text">Empty Trash</div>
              <div className="title-bar-controls">
                <button aria-label="Close" onClick={() => setEmptyTrashModal({ isOpen: false, checked: false })} />
              </div>
            </div>
            <div className="window-body">
              <p>Are you sure you want to permanently delete ALL posts in the recycle bin?</p>
              <p className="error-text mb-4">This action cannot be undone!</p>
              
              <div className="field-row" style={{ marginBottom: 16 }}>
                <input 
                  type="checkbox" 
                  id="confirm-empty" 
                  checked={emptyTrashModal.checked} 
                  onChange={(e) => setEmptyTrashModal({ ...emptyTrashModal, checked: e.target.checked })} 
                />
                <label htmlFor="confirm-empty">Yes, destroy everything</label>
              </div>
              
              <div className="flex justify-end gap-2">
                <button onClick={() => setEmptyTrashModal({ isOpen: false, checked: false })}>Cancel</button>
                <button disabled={!emptyTrashModal.checked} onClick={confirmEmptyTrash}>Empty</button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
