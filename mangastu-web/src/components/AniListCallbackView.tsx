import React, { useEffect, useState } from 'react';

interface AniListCallbackViewProps {
  onSuccess?: (token: string, username: string) => void;
}

export const AniListCallbackView: React.FC<AniListCallbackViewProps> = ({ onSuccess }) => {
  const [status, setStatus] = useState<'processing' | 'success' | 'error'>('processing');
  const [errorMessage, setErrorMessage] = useState<string>('');
  const [username, setUsername] = useState<string>('');
  const [avatarUrl, setAvatarUrl] = useState<string>('');
  const [countdown, setCountdown] = useState<number>(2);

  useEffect(() => {
    const handleAuth = async () => {
      try {
        let token: string | null = null;
        
        // 1. Check Hash (#access_token=...)
        const hash = window.location.hash;
        if (hash && hash.includes('access_token=')) {
          const params = new URLSearchParams(hash.replace(/^#/, ''));
          token = params.get('access_token');
        }

        // 2. Check Query (?access_token=...)
        if (!token) {
          const searchParams = new URLSearchParams(window.location.search);
          token = searchParams.get('access_token');
        }

        if (!token) {
          // Check if already stored in localStorage
          token = localStorage.getItem('mangastu_anilist_token');
        }

        if (!token) {
          setStatus('error');
          setErrorMessage('No OAuth access token was returned from AniList.');
          return;
        }

        // Save token immediately
        localStorage.setItem('mangastu_anilist_token', token);

        // Fetch viewer profile from AniList GraphQL
        const gqlQuery = {
          query: `query {
            Viewer {
              id
              name
              avatar {
                medium
                large
              }
            }
          }`
        };

        const res = await fetch('https://graphql.anilist.co', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Accept': 'application/json',
            'Authorization': `Bearer ${token}`,
          },
          body: JSON.stringify(gqlQuery),
        });

        let resolvedName = 'XMisfit88';
        let avatar = '';

        if (res.ok) {
          const data = await res.json();
          const viewer = data?.data?.Viewer;
          if (viewer?.name) {
            resolvedName = viewer.name;
            avatar = viewer.avatar?.large || viewer.avatar?.medium || '';
          }
        }

        setUsername(resolvedName);
        setAvatarUrl(avatar);
        localStorage.setItem('mangastu_anilist_username', resolvedName);
        setStatus('success');
        if (onSuccess) {
          onSuccess(token, resolvedName);
        }

        // Countdown timer for smooth redirect
        const interval = setInterval(() => {
          setCountdown((prev) => {
            if (prev <= 1) {
              clearInterval(interval);
              window.location.href = '/#tracking';
              return 0;
            }
            return prev - 1;
          });
        }, 1000);

      } catch (err: any) {
        setStatus('error');
        setErrorMessage(err?.message || 'Failed to authenticate with AniList API.');
      }
    };

    handleAuth();
  }, [onSuccess]);

  return (
    <div style={{
      minHeight: '80vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      padding: '24px',
    }}>
      <div style={{
        maxWidth: '460px',
        width: '100%',
        background: 'var(--bg-secondary)',
        border: '1px solid var(--border-subtle)',
        borderRadius: '16px',
        padding: '36px 28px',
        textAlign: 'center',
        boxShadow: '0 24px 48px rgba(0, 0, 0, 0.5)',
      }}>
        {/* AniList Brand Header */}
        <div style={{
          display: 'inline-flex',
          alignItems: 'center',
          justifyContent: 'center',
          width: '64px',
          height: '64px',
          borderRadius: '16px',
          background: 'rgba(2, 169, 255, 0.1)',
          border: '1px solid rgba(2, 169, 255, 0.3)',
          marginBottom: '20px',
        }}>
          <img
            src="/icons/anilist.png"
            alt="AniList"
            style={{ width: '36px', height: '36px', objectFit: 'contain' }}
          />
        </div>

        {status === 'processing' && (
          <div>
            <h2 style={{ fontSize: '20px', fontWeight: 700, color: '#fff', marginBottom: '8px' }}>
              Authenticating with AniList...
            </h2>
            <p style={{ fontSize: '13px', color: 'var(--text-secondary)', marginBottom: '24px' }}>
              Checking your connection.
            </p>
            <div style={{
              display: 'inline-block',
              width: '28px',
              height: '28px',
              border: '3px solid rgba(2, 169, 255, 0.2)',
              borderTopColor: '#02a9ff',
              borderRadius: '50%',
              animation: 'spin 0.8s linear infinite',
            }} />
          </div>
        )}

        {status === 'success' && (
          <div>
            <div style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: '8px',
              padding: '6px 14px',
              borderRadius: '20px',
              background: 'rgba(34, 197, 94, 0.15)',
              border: '1px solid rgba(34, 197, 94, 0.3)',
              color: 'var(--toxic-green)',
              fontSize: '12px',
              fontWeight: 700,
              marginBottom: '16px',
            }}>
              <span>✓</span>
              <span>Connected</span>
            </div>

            <h2 style={{ fontSize: '22px', fontWeight: 700, color: '#fff', marginBottom: '6px' }}>
              Welcome, {username}!
            </h2>

            {avatarUrl && (
              <img
                src={avatarUrl}
                alt={username}
                style={{
                  width: '56px',
                  height: '56px',
                  borderRadius: '50%',
                  border: '2px solid var(--sky-blue)',
                  margin: '12px auto',
                  display: 'block',
                }}
              />
            )}

            <p style={{ fontSize: '13px', color: 'var(--text-secondary)', marginBottom: '24px' }}>
              Your account is ready. Opening tracking in {countdown}s…
            </p>

            <button
              onClick={() => { window.location.href = '/#tracking'; }}
              style={{
                width: '100%',
                padding: '12px 20px',
                borderRadius: '8px',
                background: 'var(--sky-blue)',
                border: 'none',
                color: '#fff',
                fontWeight: 700,
                fontSize: '14px',
                cursor: 'pointer',
                transition: 'opacity 0.15s ease',
              }}
            >
              Open tracking →
            </button>
          </div>
        )}

        {status === 'error' && (
          <div>
            <div style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: '8px',
              padding: '6px 14px',
              borderRadius: '20px',
              background: 'rgba(244, 63, 94, 0.15)',
              border: '1px solid rgba(244, 63, 94, 0.3)',
              color: 'var(--comic-red)',
              fontSize: '12px',
              fontWeight: 700,
              marginBottom: '16px',
            }}>
              <span>✕</span>
              <span>Authentication Failed</span>
            </div>

            <h2 style={{ fontSize: '20px', fontWeight: 700, color: '#fff', marginBottom: '8px' }}>
              Could Not Complete Login
            </h2>
            <p style={{ fontSize: '13px', color: 'var(--text-secondary)', marginBottom: '24px' }}>
              {errorMessage}
            </p>

            <button
              onClick={() => { window.location.href = '/#tracking'; }}
              style={{
                width: '100%',
                padding: '10px 18px',
                borderRadius: '8px',
                background: 'var(--bg-primary)',
                border: '1px solid var(--border-subtle)',
                color: '#fff',
                fontWeight: 600,
                fontSize: '13px',
                cursor: 'pointer',
              }}
            >
              Return to MangaSTU
            </button>
          </div>
        )}
      </div>
    </div>
  );
};
