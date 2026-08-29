import React from 'react';

interface NavbarProps {
  activeTab: 'home' | 'puzzle';
  setActiveTab: (tab: 'home' | 'puzzle') => void;
}

export const Navbar: React.FC<NavbarProps> = ({
  activeTab,
  setActiveTab,
}) => {
  return (
    <header style={{
      position: 'sticky',
      top: 0,
      zIndex: 100,
      background: 'rgba(27, 27, 31, 0.88)',
      backdropFilter: 'blur(16px)',
      WebkitBackdropFilter: 'blur(16px)',
      borderBottom: '1px solid var(--border-subtle)',
    }}>
      <div style={{
        maxWidth: '1100px',
        margin: '0 auto',
        padding: '12px 20px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        gap: '12px',
      }}>
        {/* Brand with Logo */}
        <div 
          onClick={() => setActiveTab('home')}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '12px',
            cursor: 'pointer',
            userSelect: 'none',
          }}
        >
          <img 
            src="/logo.png" 
            alt="MangaSTU Logo" 
            style={{ 
              width: '44px', 
              height: '44px', 
              borderRadius: '10px', 
              objectFit: 'contain',
              display: 'block',
              boxShadow: '0 2px 8px rgba(0, 0, 0, 0.4)',
            }}
          />
          <div>
            <div style={{
              fontFamily: 'var(--font-heading)',
              fontWeight: 800,
              fontSize: '19px',
              color: '#fff',
              letterSpacing: '-0.02em',
              lineHeight: 1.1,
            }}>
              Manga<span style={{ color: 'var(--sky-blue)' }}>STU</span>
            </div>
            <span 
              className="mobile-hide"
              style={{
                fontSize: '11px',
                color: 'var(--text-muted)',
                display: 'block',
                fontWeight: 500,
                letterSpacing: '0.03em',
                marginTop: '2px',
              }}
            >
              Backups · Merge · Track
            </span>
          </div>
        </div>

        {/* Navigation Tabs & Action Button */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <nav style={{
            display: 'flex',
            alignItems: 'center',
            gap: '4px',
            background: 'rgba(255, 255, 255, 0.04)',
            padding: '3px',
            borderRadius: '20px',
            border: '1px solid var(--border-subtle)',
          }}>
            <button
              onClick={() => setActiveTab('home')}
              style={{
                background: activeTab === 'home' ? 'var(--sky-blue)' : 'transparent',
                color: activeTab === 'home' ? '#fff' : 'var(--text-secondary)',
                border: 'none',
                padding: '6px 14px',
                borderRadius: '16px',
                fontFamily: 'var(--font-heading)',
                fontWeight: 600,
                fontSize: '13px',
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                gap: '6px',
                transition: 'all 0.15s ease',
              }}
            >
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="m3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>
                <polyline points="9 22 9 12 15 12 15 22"/>
              </svg>
              Home
            </button>

            <button
              onClick={() => setActiveTab('puzzle')}
              style={{
                background: activeTab === 'puzzle' ? 'var(--sky-blue)' : 'transparent',
                color: activeTab === 'puzzle' ? '#fff' : 'var(--text-secondary)',
                border: 'none',
                padding: '6px 14px',
                borderRadius: '16px',
                fontFamily: 'var(--font-heading)',
                fontWeight: 600,
                fontSize: '13px',
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                gap: '6px',
                transition: 'all 0.15s ease',
              }}
            >
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <rect width="18" height="18" x="3" y="3" rx="2"/>
                <path d="M3 9h18"/>
                <path d="M9 21V9"/>
              </svg>
              Workspace
            </button>
          </nav>
        </div>
      </div>
    </header>
  );
};
