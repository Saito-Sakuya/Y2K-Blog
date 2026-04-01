import { useEffect, useState } from 'react';
import api from '../api/client';

export default function Boards() {
  const [boardsTree, setBoardsTree] = useState<any[]>([]);
  const [flatBoards, setFlatBoards] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  
  const [isEditing, setIsEditing] = useState(false);
  const [formData, setFormData] = useState({ slug: '', name: '', color: '#8b7aab', icon: '', order: 0, parent: '' });

  useEffect(() => {
    fetchBoards();
  }, []);

  const fetchBoards = async () => {
    try {
      const res = await api.get('/boards');
      setBoardsTree(res.data.boards || []);
      
      const flatten = (nodes: any[]): any[] => {
        return nodes.reduce((acc, curr) => {
           acc.push(curr);
           if (curr.children) acc = acc.concat(flatten(curr.children));
           return acc;
        }, []);
      };
      setFlatBoards(flatten(res.data.boards || []));
      
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleEdit = (board?: any) => {
    setIsEditing(true);
    if (board) {
      setFormData({
        slug: board.slug,
        name: board.name,
        color: board.color || '#8b7aab',
        icon: board.icon || '',
        order: board.order || 0,
        parent: board.parent || '' // Need parent to be editable if API returns it, though the API tree might not return parent slug explicitly. Assuming empty string is none.
      });
    } else {
      setFormData({ slug: '', name: '', color: '#8b7aab', icon: '', order: 0, parent: '' });
    }
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await api.post('/admin/boards', {
        ...formData,
        parent: formData.parent === '' ? undefined : formData.parent
      });
      alert('Board saved!');
      setIsEditing(false);
      fetchBoards();
    } catch (err) {
      alert('Failed to save board');
    }
  };

  const renderBoardTree = (nodes: any[], depth = 0) => {
    return nodes.map((node: any) => (
      <div key={node.slug} style={{ marginLeft: depth * 20, marginBottom: 4 }}>
        <div className="flex justify-between items-center" style={{ border: '1px outset #fff', padding: '4px', background: 'var(--border)' }}>
          <div>
             <span style={{ color: node.color, fontWeight: 'bold' }}>{node.name}</span>
             <small className="ml-2 text-gray-500">({node.slug}) - {node.postCount || 0} posts</small>
          </div>
          <button onClick={() => handleEdit(node)}>Edit</button>
        </div>
        {node.children && node.children.length > 0 && (
          <div className="mt-1">
            {renderBoardTree(node.children, depth + 1)}
          </div>
        )}
      </div>
    ));
  };

  return (
    <div className="flex gap-4">
      <div className="flex-1">
        <div className="flex justify-between items-center mb-4">
          <h2>📁 Boards</h2>
          <button onClick={() => handleEdit()}>New Board</button>
        </div>
        
        {loading ? <p>Loading...</p> : (
          <div className="window w-full">
            <div className="title-bar"><div className="title-bar-text">Board Tree</div></div>
            <div className="window-body">
               {boardsTree.length > 0 ? renderBoardTree(boardsTree) : <p>No boards found.</p>}
            </div>
          </div>
        )}
      </div>

      {isEditing && (
        <div style={{ width: 300 }}>
          <div className="window">
             <div className="title-bar">
               <div className="title-bar-text">{formData.slug ? 'Edit' : 'New'} Board</div>
               <div className="title-bar-controls"><button aria-label="Close" onClick={() => setIsEditing(false)}/></div>
             </div>
             <div className="window-body">
               <form onSubmit={handleSave}>
                 <div className="field-row-stacked">
                   <label>Slug</label>
                   <input required name="slug" value={formData.slug} onChange={(e) => setFormData({...formData, slug: e.target.value})} disabled={!!formData.slug && isEditing /* theoretically slug is fixed but wait API is POST for UPSERT, so usually slug is PK */} />
                 </div>
                 <div className="field-row-stacked">
                   <label>Name</label>
                   <input required name="name" value={formData.name} onChange={(e) => setFormData({...formData, name: e.target.value})} />
                 </div>
                 <div className="field-row-stacked">
                   <label>Color</label>
                   <input type="color" name="color" value={formData.color} onChange={(e) => setFormData({...formData, color: e.target.value})} />
                 </div>
                 <div className="field-row-stacked">
                   <label>Icon (emoji or symbol)</label>
                   <input name="icon" value={formData.icon} onChange={(e) => setFormData({...formData, icon: e.target.value})} placeholder="⭐ 💻 📝 🎮" maxLength={2} style={{ width: 80 }} />
                 </div>
                 <div className="field-row-stacked">
                   <label>Order</label>
                   <input type="number" name="order" value={formData.order} onChange={(e) => setFormData({...formData, order: parseInt(e.target.value)})} />
                 </div>
                 <div className="field-row-stacked">
                   <label>Parent Board</label>
                   <select name="parent" value={formData.parent} onChange={(e) => setFormData({...formData, parent: e.target.value})}>
                     <option value="">-- None (Top Level) --</option>
                     {flatBoards.map(b => (
                       <option key={b.slug} value={b.slug}>{b.name}</option>
                     ))}
                   </select>
                 </div>
                 <div className="field-row mt-4 justify-between">
                   <button type="submit" style={{ flex: 1 }}>Save</button>
                 </div>
               </form>
             </div>
          </div>
        </div>
      )}
    </div>
  );
}
