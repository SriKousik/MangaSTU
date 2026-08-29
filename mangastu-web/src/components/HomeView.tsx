import React, { useState } from 'react';

interface HomeViewProps {
  onOpenPuzzle: (subTab?: 'convert' | 'merge' | 'tracking') => void;
}

export const HomeView: React.FC<HomeViewProps> = ({ onOpenPuzzle }) => {
  const [activeCliTab, setActiveCliTab] = useState<'convert' | 'merge' | 'inspect' | 'validate'>('convert');
  const [copiedCmd, setCopiedCmd] = useState<string | null>(null);
  const cliTabLabels = { convert: 'Transfer', merge: 'Merge', inspect: 'Inspect', validate: 'Validate' };

  const copyToClipboard = (text: string, id: string) => {
    navigator.clipboard.writeText(text);
    setCopiedCmd(id);
    setTimeout(() => setCopiedCmd(null), 2000);
  };

  return (
    <main style={{ maxWidth: '980px', margin: '0 auto', padding: '36px 20px 80px', width: '100%' }}>
      {/* ================= HERO SECTION (Komikku-style) ================= */}
      <section style={{ textAlign: 'center', marginBottom: '48px' }}>
        {/* Crisp Hero Logo */}
        <div style={{
          display: 'inline-block',
          position: 'relative',
          marginBottom: '20px',
        }}>
          <img 
            src="/logo.png" 
            alt="MangaSTU" 
            style={{ 
              width: '110px', 
              height: '110px', 
              borderRadius: '24px',
              objectFit: 'contain',
              boxShadow: '0 8px 32px rgba(0, 0, 0, 0.4)',
              border: '1px solid rgba(255, 255, 255, 0.08)',
              display: 'block',
              margin: '0 auto',
            }} 
          />
        </div>

        {/* Hero Title */}
        <h1 style={{
          fontSize: 'clamp(36px, 6vw, 52px)',
          fontWeight: 800,
          letterSpacing: '-0.03em',
          lineHeight: 1.1,
          marginBottom: '12px',
          color: '#ffffff',
        }}>
          Manga<span style={{ color: 'var(--sky-blue)' }}>STU</span>
        </h1>

        {/* Subtitle / Mission */}
        <p style={{
          fontSize: 'clamp(15px, 2.5vw, 18px)',
          color: 'var(--text-secondary)',
          maxWidth: '560px',
          margin: '0 auto 28px',
          lineHeight: 1.5,
          fontWeight: 400,
        }}>
          Move, combine, and update manga backups while keeping your library, reading progress, and categories together.
        </p>

        {/* Action Buttons Row */}
        <div style={{
          display: 'flex',
          justifyContent: 'center',
          gap: '12px',
          flexWrap: 'wrap',
          marginBottom: '36px',
        }}>
          <button 
            onClick={() => onOpenPuzzle('convert')}
            className="btn-sky"
            style={{ padding: '11px 24px', fontSize: '14px' }}
          >
            Move a Backup
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <path d="M5 12h14M12 5l7 7-7 7"/>
            </svg>
          </button>

          <button 
            onClick={() => onOpenPuzzle('merge')}
            className="btn-red"
            style={{ padding: '11px 24px', fontSize: '14px' }}
          >
            Merge Backups
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="16 3 21 3 21 8"></polyline>
              <line x1="4" y1="20" x2="21" y2="3"></line>
            </svg>
          </button>

          <button 
            onClick={() => {
              const el = document.getElementById('cli-reference');
              if (el) el.scrollIntoView({ behavior: 'smooth' });
            }}
            className="btn-secondary"
            style={{ padding: '11px 22px', fontSize: '14px' }}
          >
            CLI Reference
          </button>
        </div>

        {/* Platform Comparison Cards */}
        <div style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          gap: '12px',
          flexWrap: 'wrap',
          maxWidth: '680px',
          margin: '0 auto',
        }}>
          {/* iOS Card */}
          <div className="card-panel mobile-full card-interactive" style={{
            padding: '14px 18px',
            flex: '1 1 200px',
            display: 'flex',
            alignItems: 'center',
            gap: '14px',
            textAlign: 'left',
          }}>
            <div style={{
              width: '40px',
              height: '40px',
              borderRadius: '10px',
              background: 'var(--sky-blue-subtle)',
              border: '1px solid var(--sky-blue-border)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: 'var(--sky-blue)',
              flexShrink: 0,
            }}>
              {/* Authentic Apple Logo */}
              <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                <path d="M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.81-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M15.97 6.38c.62-.75 1.04-1.8 0.92-2.88-.9.04-1.99.6-2.63 1.35-.58.65-1.08 1.7-0.94 2.74 1 .08 2.03-.46 2.65-1.21z"/>
              </svg>
            </div>
            <div>
              <div style={{ fontWeight: 700, fontSize: '14px', color: '#fff' }}>Tachimanga (iOS)</div>
              <div style={{ fontSize: '12px', color: 'var(--sky-blue)', fontFamily: 'var(--font-mono)' }}>.tmb (SQLite / ZIP)</div>
            </div>
          </div>

          {/* Center Bridge Indicator */}
          <div style={{
            padding: '6px 12px',
            borderRadius: '20px',
            background: 'var(--bg-secondary)',
            border: '1px solid var(--border-subtle)',
            color: 'var(--text-secondary)',
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            fontWeight: 600,
            fontSize: '12px',
            flexShrink: 0,
          }}>
            <span style={{ color: 'var(--sky-blue)' }}>←</span>
            <span>Zero Data Loss</span>
            <span style={{ color: 'var(--comic-red)' }}>→</span>
          </div>

          {/* Android Card */}
          <div className="card-panel mobile-full card-interactive" style={{
            padding: '14px 18px',
            flex: '1 1 200px',
            display: 'flex',
            alignItems: 'center',
            gap: '14px',
            textAlign: 'left',
          }}>
            <div style={{
              width: '40px',
              height: '40px',
              borderRadius: '10px',
              background: 'var(--comic-red-subtle)',
              border: '1px solid var(--comic-red-border)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: 'var(--comic-red)',
              flexShrink: 0,
            }}>
              {/* Android Robot Logo */}
              <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                <path d="M17.523 15.3414c-.5511 0-.9993-.4486-.9993-.9997s.4482-.9993.9993-.9993c.551 0 .9993.4482.9993.9993.0001.5511-.4482.9997-.9993.9997m-11.046 0c-.5511 0-.9993-.4486-.9993-.9997s.4482-.9993.9993-.9993c.5511 0 .9993.4482.9993.9993 0 .5511-.4482.9997-.9993.9997m11.4045-6.02l1.9973-3.4592a.416.416 0 00-.1521-.5676.416.416 0 00-.5676.1521l-2.0223 3.503C15.5902 8.411 13.8533 8.1 12 8.1s-3.5902.311-5.1368.8497L4.8409 5.4467a.4161.4161 0 00-.5677-.1521.4157.4157 0 00-.1521.5676l1.9973 3.4592C2.6889 11.1867.3432 14.6589 0 18.761h24c-.3432-4.1021-2.6889-7.5743-6.1185-9.4396"/>
              </svg>
            </div>
            <div>
              <div style={{ fontWeight: 700, fontSize: '14px', color: '#fff' }}>Komikku / Mihon (Android)</div>
              <div style={{ fontSize: '12px', color: 'var(--comic-red)', fontFamily: 'var(--font-mono)' }}>.tachibk (Protobuf)</div>
            </div>
          </div>
        </div>
      </section>

      {/* ================= FEATURES GRID ================= */}
      <section style={{ margin: '48px 0' }}>
        <div style={{ textAlign: 'center', marginBottom: '24px' }}>
          <h2 style={{ fontSize: '24px', fontWeight: 800, marginBottom: '6px', letterSpacing: '-0.02em' }}>Made for your library</h2>
          <p style={{ color: 'var(--text-secondary)', fontSize: '14px' }}>Keep the details that matter when moving between apps.</p>
        </div>

        <div style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))',
          gap: '12px',
        }}>
          {/* Card 1 */}
          <div className="card-panel card-interactive" style={{ padding: '20px' }}>
            <div style={{
              width: '36px',
              height: '36px',
              borderRadius: '8px',
              background: 'var(--sky-blue-subtle)',
              border: '1px solid var(--sky-blue-border)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: 'var(--sky-blue)',
              marginBottom: '12px',
            }}>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon>
              </svg>
            </div>
            <h3 style={{ fontSize: '15px', fontWeight: 700, marginBottom: '4px' }}>Fast on large libraries</h3>
            <p style={{ color: 'var(--text-secondary)', fontSize: '13px', lineHeight: 1.5 }}>
              Built to handle thousands of titles and chapters without a long wait.
            </p>
          </div>

          {/* Card 2 */}
          <div className="card-panel card-interactive" style={{ padding: '20px' }}>
            <div style={{
              width: '36px',
              height: '36px',
              borderRadius: '8px',
              background: 'var(--comic-red-subtle)',
              border: '1px solid var(--comic-red-border)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: 'var(--comic-red)',
              marginBottom: '12px',
            }}>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <polyline points="16 3 21 3 21 8"></polyline>
                <line x1="4" y1="20" x2="21" y2="3"></line>
                <polyline points="21 16 21 21 16 21"></polyline>
                <line x1="15" y1="15" x2="21" y2="21"></line>
                <line x1="4" y1="4" x2="9" y2="9"></line>
              </svg>
            </div>
            <h3 style={{ fontSize: '15px', fontWeight: 700, marginBottom: '4px' }}>Bring backups together</h3>
            <p style={{ color: 'var(--text-secondary)', fontSize: '13px', lineHeight: 1.5 }}>
              Combine libraries, progress, history, and categories into one backup.
            </p>
          </div>

          {/* Card 3 */}
          <div className="card-panel card-interactive" style={{ padding: '20px' }}>
            <div style={{
              width: '36px',
              height: '36px',
              borderRadius: '8px',
              background: 'var(--sky-blue-subtle)',
              border: '1px solid var(--sky-blue-border)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: 'var(--sky-blue)',
              marginBottom: '12px',
            }}>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path>
                <polyline points="3.27 6.96 12 12.01 20.73 6.96"></polyline>
                <line x1="12" y1="22.08" x2="12" y2="12"></line>
              </svg>
            </div>
            <h3 style={{ fontSize: '15px', fontWeight: 700, marginBottom: '4px' }}>Sources stay recognizable</h3>
            <p style={{ color: 'var(--text-secondary)', fontSize: '13px', lineHeight: 1.5 }}>
              Preserve source details so your library is ready when you import it.
            </p>
          </div>

          {/* Card 4 */}
          <div className="card-panel card-interactive" style={{ padding: '20px' }}>
            <div style={{
              width: '36px',
              height: '36px',
              borderRadius: '8px',
              background: 'rgba(255, 255, 255, 0.05)',
              border: '1px solid var(--border-subtle)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: 'var(--text-primary)',
              marginBottom: '12px',
            }}>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect>
                <path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
              </svg>
            </div>
            <h3 style={{ fontSize: '15px', fontWeight: 700, marginBottom: '4px' }}>Your files stay yours</h3>
            <p style={{ color: 'var(--text-secondary)', fontSize: '13px', lineHeight: 1.5 }}>
              Process backups locally with MangaSTU—no account or cloud library is required.
            </p>
          </div>
        </div>
      </section>

      {/* ================= CLI REFERENCE BOX ================= */}
      <section id="cli-reference" style={{ margin: '48px 0' }}>
        <div className="card-panel mobile-p-small" style={{ padding: '24px', borderRadius: '12px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '12px', marginBottom: '16px' }}>
            <div>
              <h2 style={{ fontSize: '18px', fontWeight: 700, marginBottom: '2px' }}>CLI Command Reference</h2>
              <p style={{ color: 'var(--text-secondary)', fontSize: '13px' }}>Use the same tools from your terminal.</p>
            </div>

            <div className="mobile-scroll-x" style={{ display: 'flex', gap: '4px', background: 'rgba(255, 255, 255, 0.04)', padding: '3px', borderRadius: '8px', border: '1px solid var(--border-subtle)' }}>
              {(['convert', 'merge', 'inspect', 'validate'] as const).map((tab) => (
                <button
                  key={tab}
                  onClick={() => setActiveCliTab(tab)}
                  style={{
                    background: activeCliTab === tab ? 'var(--sky-blue)' : 'transparent',
                    color: activeCliTab === tab ? '#fff' : 'var(--text-secondary)',
                    border: 'none',
                    padding: '5px 12px',
                    borderRadius: '6px',
                    fontSize: '12px',
                    fontWeight: 600,
                    cursor: 'pointer',
                    textTransform: 'capitalize',
                    whiteSpace: 'nowrap',
                    transition: 'all 0.15s ease',
                  }}
                >
                  {cliTabLabels[tab]}
                </button>
              ))}
            </div>
          </div>

          <div style={{
            background: '#151518',
            borderRadius: '8px',
            border: '1px solid var(--border-subtle)',
            padding: '16px',
            fontFamily: 'var(--font-mono)',
            fontSize: '12px',
            position: 'relative',
            overflowX: 'auto',
          }}>
            {activeCliTab === 'convert' && (
              <div>
                <div style={{ color: 'var(--text-muted)', marginBottom: '4px' }}># Move Tachimanga (.tmb) to Komikku/Mihon (.tachibk)</div>
                <div style={{ color: 'var(--sky-blue)', marginBottom: '10px', wordBreak: 'break-all' }}>mangastu convert Tachimanga_backup.tmb output.tachibk</div>
                <div style={{ color: 'var(--text-muted)', marginBottom: '4px' }}># Move Komikku (.tachibk) to Tachimanga (.tmb)</div>
                <div style={{ color: 'var(--sky-blue)', wordBreak: 'break-all' }}>mangastu convert app.komikku.tachibk tachimanga_backup.tmb</div>
              </div>
            )}

            {activeCliTab === 'merge' && (
              <div>
                <div style={{ color: 'var(--text-muted)', marginBottom: '4px' }}># Merge two or more backups into a unified .tachibk</div>
                <div style={{ color: 'var(--comic-red)', marginBottom: '10px', wordBreak: 'break-all' }}>mangastu merge old.tachibk app.komikku.tachibk -o merged.tachibk</div>
                <div style={{ color: 'var(--text-muted)', marginBottom: '4px' }}># Merge multiple backups across formats into a unified .tmb</div>
                <div style={{ color: 'var(--comic-red)', wordBreak: 'break-all' }}>mangastu merge backup1.tmb backup2.tachibk -o merged.tmb</div>
              </div>
            )}

            {activeCliTab === 'inspect' && (
              <div>
                <div style={{ color: 'var(--text-muted)', marginBottom: '4px' }}># Inspect backup contents, statistics & source counts</div>
                <div style={{ color: 'var(--sky-blue)', wordBreak: 'break-all' }}>mangastu inspect backup.tmb</div>
              </div>
            )}

            {activeCliTab === 'validate' && (
              <div>
                <div style={{ color: 'var(--text-muted)', marginBottom: '4px' }}># Check backup integrity before transferring</div>
                <div style={{ color: 'var(--sky-blue)', wordBreak: 'break-all' }}>mangastu validate backup.tachibk</div>
              </div>
            )}

            <button
              onClick={() => {
                const cmdMap = {
                  convert: 'mangastu convert backup.tmb output.tachibk',
                  merge: 'mangastu merge backup1.tmb backup2.tachibk -o merged.tachibk',
                  inspect: 'mangastu inspect backup.tmb',
                  validate: 'mangastu validate backup.tachibk',
                };
                copyToClipboard(cmdMap[activeCliTab], activeCliTab);
              }}
              style={{
                position: 'absolute',
                top: '12px',
                right: '12px',
                background: 'var(--bg-secondary)',
                border: '1px solid var(--border-subtle)',
                color: copiedCmd === activeCliTab ? '#10b981' : 'var(--text-secondary)',
                padding: '4px 10px',
                borderRadius: '6px',
                cursor: 'pointer',
                fontSize: '11px',
                fontWeight: 600,
                display: 'flex',
                alignItems: 'center',
                gap: '4px',
              }}
            >
              {copiedCmd === activeCliTab ? 'Copied' : 'Copy'}
            </button>
          </div>
        </div>
      </section>
    </main>
  );
};
