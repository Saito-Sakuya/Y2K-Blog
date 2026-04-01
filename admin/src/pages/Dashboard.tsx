import { useEffect, useState } from 'react';
import api from '../api/client';

export default function Dashboard() {
  const [stats, setStats] = useState({ posts: 0, boards: 0, tags: 0 });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const [boardsRes, tagsRes] = await Promise.all([
          api.get('/boards'),
          api.get('/tags')
        ]);
        
        let postCount = 0;
        let boardCount = boardsRes.data.boards?.length || 0;
        
        // Count posts recursively
        const countPosts = (boards: any[]) => {
          boards.forEach(b => {
             postCount += b.postCount || 0;
             if (b.children) countPosts(b.children);
          });
        };
        
        if (boardsRes.data.boards) {
          countPosts(boardsRes.data.boards);
        }

        setStats({
          posts: postCount,
          boards: boardCount,
          tags: tagsRes.data.total || 0
        });
      } catch (err) {
        console.error("Failed to fetch dashboard stats", err);
      } finally {
        setLoading(false);
      }
    };

    fetchStats();
  }, []);

  return (
    <div>
      <h2 style={{ borderBottom: '2px solid var(--border)', paddingBottom: '8px' }}>Dashboard</h2>
      
      {loading ? (
        <p>Loading stats...</p>
      ) : (
        <div className="flex gap-4 mt-4" style={{ flexWrap: 'wrap' }}>
          <div className="window" style={{ width: 200 }}>
            <div className="title-bar">
              <div className="title-bar-text">📝 Posts</div>
            </div>
            <div className="window-body text-center">
              <h1 style={{ margin: '16px 0', fontSize: '3rem', color: 'var(--y2k-dark-purple)' }}>{stats.posts}</h1>
              <p>Total published posts</p>
            </div>
          </div>
          
          <div className="window" style={{ width: 200 }}>
            <div className="title-bar">
              <div className="title-bar-text">📁 Boards</div>
            </div>
            <div className="window-body text-center">
              <h1 style={{ margin: '16px 0', fontSize: '3rem', color: 'var(--y2k-dark-purple)' }}>{stats.boards}</h1>
              <p>Top-level boards</p>
            </div>
          </div>

          <div className="window" style={{ width: 200 }}>
            <div className="title-bar">
              <div className="title-bar-text">🏷️ Tags</div>
            </div>
            <div className="window-body text-center">
              <h1 style={{ margin: '16px 0', fontSize: '3rem', color: 'var(--y2k-dark-purple)' }}>{stats.tags}</h1>
              <p>Unique tags used</p>
            </div>
          </div>
        </div>
      )}
      
      <div className="window mt-4 bg-gray-100 p-4">
        <div className="title-bar">
          <div className="title-bar-text">System Info</div>
        </div>
        <div className="window-body">
           <p>Welcome to the Y2K Pixel Blog admin panel.</p>
           <p>You can manage posts, edit site settings, and organize boards using the sidebar navigation.</p>
        </div>
      </div>
    </div>
  );
}
