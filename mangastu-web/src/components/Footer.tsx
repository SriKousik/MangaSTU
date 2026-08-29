import React from 'react';

export const Footer: React.FC = () => {
  return (
    <footer style={{
      marginTop: 'auto',
      borderTop: '1px solid var(--border-subtle)',
      background: 'var(--bg-secondary)',
      padding: '24px 24px',
    }}>
      <div style={{
        maxWidth: '1200px',
        margin: '0 auto',
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        flexWrap: 'wrap',
        gap: '16px',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <img src="/logo.png" alt="MangaSTU" style={{ width: '26px', height: '26px', borderRadius: '4px', objectFit: 'contain' }} />
          <span style={{ fontFamily: 'var(--font-heading)', fontWeight: 700, fontSize: '14px', color: '#fff' }}>
            MangaSTU
          </span>
          <span style={{ color: 'var(--text-muted)', fontSize: '12px' }}>
            — Backups · Merge · Track
          </span>
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: '16px', fontSize: '12px', color: 'var(--text-muted)' }}>
          <span>Tachimanga (.tmb) / Komikku / Mihon (.tachibk)</span>
        </div>
      </div>
    </footer>
  );
};
