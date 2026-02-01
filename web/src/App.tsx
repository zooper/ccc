import { useEffect, useState } from 'react';
import { getStatus, getDashboard, getEvents, getSiteConfig } from './api';
import type { StatusResponse, DashboardResponse, Event, SiteConfig } from './types';
import Dashboard from './components/Dashboard';
import OptInPrompt from './components/OptInPrompt';
import Admin from './components/Admin';
import About from './components/About';

type Theme = 'dark' | 'light';

const themes = {
  dark: {
    bg: '#0f172a',
    bgCard: '#1e293b',
    bgCardAlt: '#1e3a5f',
    border: '#334155',
    text: '#e2e8f0',
    textMuted: '#94a3b8',
    textDimmed: '#64748b',
    accent: '#3b82f6',
    error: '#7f1d1d',
    errorText: '#fecaca',
    success: '#22c55e',
    successBg: '#14532d',
    warning: '#eab308',
    warningBg: '#713f12',
    danger: '#ef4444',
    dangerBg: '#7f1d1d',
  },
  light: {
    bg: '#f8fafc',
    bgCard: '#ffffff',
    bgCardAlt: '#e0f2fe',
    border: '#e2e8f0',
    text: '#1e293b',
    textMuted: '#64748b',
    textDimmed: '#94a3b8',
    accent: '#2563eb',
    error: '#fef2f2',
    errorText: '#dc2626',
    success: '#16a34a',
    successBg: '#dcfce7',
    warning: '#ca8a04',
    warningBg: '#fef9c3',
    danger: '#dc2626',
    dangerBg: '#fee2e2',
  },
};

export type ThemeColors = typeof themes.dark;
export const ThemeContext = { colors: themes.dark, theme: 'dark' as Theme };

function App() {
  const [status, setStatus] = useState<StatusResponse | null>(null);
  const [dashboard, setDashboard] = useState<DashboardResponse | null>(null);
  const [events, setEvents] = useState<Event[]>([]);
  const [siteConfig, setSiteConfig] = useState<SiteConfig | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [showAdmin, setShowAdmin] = useState(false);
  const [showAbout, setShowAbout] = useState(false);
  const [theme, setTheme] = useState<Theme>(() => {
    const saved = localStorage.getItem('theme') as Theme;
    return saved || 'light';
  });

  const colors = themes[theme];

  useEffect(() => {
    localStorage.setItem('theme', theme);
    document.body.style.background = colors.bg;
    document.body.style.color = colors.text;
  }, [theme, colors]);

  const fetchData = async () => {
    try {
      const [statusData, dashboardData, eventsData, siteConfigData] = await Promise.all([
        getStatus(),
        getDashboard(),
        getEvents(),
        getSiteConfig(),
      ]);
      setStatus(statusData);
      setDashboard(dashboardData);
      setEvents(eventsData.events || []);
      setSiteConfig(siteConfigData);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load data');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (window.location.pathname === '/admin') {
      setShowAdmin(true);
    } else if (window.location.pathname === '/about') {
      setShowAbout(true);
    }
    fetchData();
    const interval = setInterval(fetchData, 30000);
    return () => clearInterval(interval);
  }, []);

  const handleRegistered = () => {
    fetchData();
  };

  const navigateToHome = () => {
    window.history.pushState({}, '', '/');
    setShowAdmin(false);
    setShowAbout(false);
    fetchData();
  };

  const toggleTheme = () => {
    setTheme(t => t === 'dark' ? 'light' : 'dark');
  };

  const styles = {
    container: {
      maxWidth: '900px',
      margin: '0 auto',
      padding: '20px',
      minHeight: '100vh',
    },
    header: {
      textAlign: 'center' as const,
      marginBottom: '30px',
    },
    title: {
      fontSize: '2rem',
      fontWeight: 'bold',
      marginBottom: '10px',
      color: colors.text,
    },
    subtitle: {
      color: colors.textMuted,
      fontSize: '1rem',
    },
    error: {
      background: colors.error,
      color: colors.errorText,
      padding: '15px',
      borderRadius: '8px',
      marginBottom: '20px',
    },
    loading: {
      textAlign: 'center' as const,
      padding: '40px',
      color: colors.textMuted,
    },
    nav: {
      display: 'flex',
      justifyContent: 'flex-end',
      alignItems: 'center',
      gap: '15px',
      marginBottom: '20px',
      padding: '10px 0',
    },
    navLink: {
      background: colors.bgCard,
      border: `1px solid ${colors.border}`,
      borderRadius: '6px',
      padding: '6px 12px',
      cursor: 'pointer',
      color: colors.text,
      fontSize: '0.875rem',
      textDecoration: 'none',
    },
    themeToggle: {
      background: 'transparent',
      border: 'none',
      borderRadius: '6px',
      padding: '6px',
      cursor: 'pointer',
      color: colors.textMuted,
      fontSize: '1.25rem',
      lineHeight: 1,
    },
    registered: {
      background: colors.bgCardAlt,
      padding: '15px',
      borderRadius: '8px',
      marginBottom: '20px',
      textAlign: 'center' as const,
      color: colors.text,
    },
  };

  if (showAdmin) {
    return (
      <div style={styles.container}>
        <button style={styles.themeToggle} onClick={toggleTheme} title={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}>
          {theme === 'dark' ? '☀️' : '🌙'}
        </button>
        <Admin onBack={navigateToHome} colors={colors} />
      </div>
    );
  }

  if (showAbout) {
    return (
      <div style={styles.container}>
        <button style={styles.themeToggle} onClick={toggleTheme} title={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}>
          {theme === 'dark' ? '☀️' : '🌙'}
        </button>
        <About onBack={navigateToHome} colors={colors} />
      </div>
    );
  }

  if (loading) {
    return (
      <div style={styles.container}>
        <div style={styles.loading}>Loading...</div>
      </div>
    );
  }

  return (
    <div style={styles.container}>
      <nav style={styles.nav}>
        <span
          style={styles.navLink}
          onClick={() => { window.history.pushState({}, '', '/about'); setShowAbout(true); }}
        >
          About
        </span>
        {siteConfig?.github_url && (
          <a
            href={siteConfig.github_url}
            target="_blank"
            rel="noopener noreferrer"
            style={{ ...styles.themeToggle, textDecoration: 'none', display: 'flex', alignItems: 'center' }}
            title="View on GitHub"
          >
            <svg height="20" width="20" viewBox="0 0 16 16" fill="currentColor">
              <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/>
            </svg>
          </a>
        )}
        <button style={styles.themeToggle} onClick={toggleTheme} title={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}>
          {theme === 'dark' ? '☀️' : '🌙'}
        </button>
      </nav>

      <header style={styles.header}>
        <h1 style={styles.title}>{siteConfig?.site_name || 'Community Connectivity Check'}</h1>
        <p style={styles.subtitle}>
          {siteConfig?.site_description || 'Monitor ISP connectivity in our building'}
        </p>
      </header>

      {error && (
        <div style={styles.error}>
          Error: {error}
        </div>
      )}

      {status && !status.registered && status.can_register && (
        <OptInPrompt isp={status.isp} onRegistered={handleRegistered} colors={colors} />
      )}

      {status && status.registered && (
        <div style={styles.registered}>
          {status.endpoint_status === 'unreachable' ? (
            <>
              <div style={{ marginBottom: '8px' }}>
                You've joined monitoring ({status.isp}), but your connection doesn't respond to our checks.
              </div>
              <div style={{ fontSize: '0.875rem', color: colors.textMuted }}>
                This is normal if your router blocks ping requests. You can still view the dashboard and see how others are doing.
              </div>
            </>
          ) : status.endpoint_status === 'up' ? (
            <>You're participating in monitoring and online ({status.isp})</>
          ) : status.endpoint_status === 'down' ? (
            <>You're participating but currently appear offline ({status.isp})</>
          ) : (
            <>You're participating in monitoring ({status.isp})</>
          )}
        </div>
      )}

      {dashboard && (
        <Dashboard data={dashboard} currentISP={status?.isp} colors={colors} events={events} />
      )}

    </div>
  );
}

export default App;
