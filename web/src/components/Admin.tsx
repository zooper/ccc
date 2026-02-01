import { useState, useEffect } from 'react';
import { adminListEndpoints, adminAddEndpoint, adminDeleteEndpoint, adminGetMetrics, adminGetSettings, adminUpdateSettings, adminGetSiteConfig, adminUpdateSiteConfig } from '../api';
import type { AdminEndpoint, AdminMetrics, AdminSettings, SiteConfig } from '../types';
import type { ThemeColors } from '../App';
import SiteConfigEditor from './SiteConfigEditor';

interface AdminProps {
  onBack: () => void;
  colors: ThemeColors;
}

function Admin({ onBack, colors }: AdminProps) {
  const [password, setPassword] = useState(() => sessionStorage.getItem('adminPassword') || '');
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [endpoints, setEndpoints] = useState<AdminEndpoint[]>([]);
  const [metrics, setMetrics] = useState<AdminMetrics | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [newIP, setNewIP] = useState('');
  const [newISP, setNewISP] = useState('');
  const [adding, setAdding] = useState(false);
  const [loginPassword, setLoginPassword] = useState('');
  const [activeTab, setActiveTab] = useState<'metrics' | 'endpoints' | 'settings' | 'site-config'>('metrics');
  const [settings, setSettings] = useState<AdminSettings | null>(null);
  const [savingSettings, setSavingSettings] = useState(false);
  const [siteConfig, setSiteConfig] = useState<SiteConfig | null>(null);
  const [savingSiteConfig, setSavingSiteConfig] = useState(false);

  const styles = {
    container: {
      padding: '20px',
    },
    header: {
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'center',
      marginBottom: '20px',
    },
    title: {
      fontSize: '1.5rem',
      fontWeight: 'bold',
      color: colors.text,
    },
    backLink: {
      color: colors.accent,
      textDecoration: 'none',
      cursor: 'pointer',
    },
    loginForm: {
      background: colors.bgCard,
      padding: '30px',
      borderRadius: '12px',
      maxWidth: '400px',
      margin: '40px auto',
      textAlign: 'center' as const,
      border: `1px solid ${colors.border}`,
    },
    loginTitle: {
      fontSize: '1.25rem',
      fontWeight: 'bold',
      marginBottom: '20px',
      color: colors.text,
    },
    addForm: {
      background: colors.bgCard,
      padding: '20px',
      borderRadius: '12px',
      marginBottom: '20px',
      border: `1px solid ${colors.border}`,
    },
    formRow: {
      display: 'flex',
      gap: '10px',
      marginBottom: '10px',
    },
    input: {
      flex: 1,
      padding: '10px',
      borderRadius: '6px',
      border: `1px solid ${colors.border}`,
      background: colors.bg,
      color: colors.text,
      fontSize: '1rem',
    },
    button: {
      padding: '10px 20px',
      borderRadius: '6px',
      border: 'none',
      cursor: 'pointer',
      fontSize: '1rem',
      fontWeight: 'bold',
    },
    loginButton: {
      background: colors.accent,
      color: 'white',
      width: '100%',
      marginTop: '10px',
    },
    addButton: {
      background: colors.success,
      color: 'white',
    },
    deleteButton: {
      background: colors.danger,
      color: 'white',
      padding: '6px 12px',
      fontSize: '0.875rem',
    },
    logoutButton: {
      background: colors.textDimmed,
      color: 'white',
      padding: '6px 12px',
      fontSize: '0.875rem',
    },
    table: {
      width: '100%',
      borderCollapse: 'collapse' as const,
      background: colors.bgCard,
      borderRadius: '12px',
      overflow: 'hidden',
      border: `1px solid ${colors.border}`,
    },
    th: {
      padding: '12px',
      textAlign: 'left' as const,
      background: colors.border,
      fontWeight: 'bold',
      color: colors.text,
    },
    td: {
      padding: '12px',
      borderTop: `1px solid ${colors.border}`,
      color: colors.text,
    },
    statusUp: {
      color: colors.success,
      fontWeight: 'bold',
    },
    statusDown: {
      color: colors.danger,
      fontWeight: 'bold',
    },
    statusUnknown: {
      color: colors.textMuted,
    },
    error: {
      background: colors.error,
      color: colors.errorText,
      padding: '10px',
      borderRadius: '6px',
      marginBottom: '10px',
    },
    loading: {
      textAlign: 'center' as const,
      padding: '40px',
      color: colors.textMuted,
    },
    noData: {
      textAlign: 'center' as const,
      padding: '40px',
      color: colors.textMuted,
    },
    tabs: {
      display: 'flex',
      gap: '10px',
      marginBottom: '20px',
    },
    tab: {
      padding: '10px 20px',
      borderRadius: '6px',
      border: 'none',
      cursor: 'pointer',
      fontSize: '1rem',
      fontWeight: 'bold',
      background: colors.border,
      color: colors.text,
    },
    tabActive: {
      background: colors.accent,
      color: 'white',
    },
    metricsGrid: {
      display: 'grid',
      gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
      gap: '15px',
      marginBottom: '20px',
    },
    metricCard: {
      background: colors.bgCard,
      padding: '20px',
      borderRadius: '12px',
      border: `1px solid ${colors.border}`,
    },
    metricLabel: {
      fontSize: '0.875rem',
      color: colors.textMuted,
      marginBottom: '5px',
    },
    metricValue: {
      fontSize: '1.5rem',
      fontWeight: 'bold',
      color: colors.text,
    },
    metricSmall: {
      fontSize: '0.75rem',
      color: colors.textDimmed,
      marginTop: '5px',
    },
    section: {
      background: colors.bgCard,
      padding: '20px',
      borderRadius: '12px',
      marginBottom: '20px',
      border: `1px solid ${colors.border}`,
    },
    sectionTitle: {
      fontSize: '1.125rem',
      fontWeight: 'bold',
      marginBottom: '15px',
      color: colors.text,
    },
    ispGrid: {
      display: 'grid',
      gridTemplateColumns: 'repeat(auto-fit, minmax(250px, 1fr))',
      gap: '15px',
    },
    ispCard: {
      padding: '15px',
      borderRadius: '8px',
      border: `1px solid ${colors.border}`,
      background: colors.bg,
    },
    ispName: {
      fontSize: '1rem',
      fontWeight: 'bold',
      marginBottom: '10px',
      textTransform: 'uppercase' as const,
      color: colors.text,
    },
    ispStats: {
      display: 'flex',
      gap: '15px',
      fontSize: '0.875rem',
    },
    ispStat: {
      display: 'flex',
      flexDirection: 'column' as const,
    },
    uptimeBar: {
      height: '8px',
      borderRadius: '4px',
      background: colors.border,
      marginTop: '10px',
      overflow: 'hidden',
    },
    uptimeFill: {
      height: '100%',
      borderRadius: '4px',
    },
    historyList: {
      maxHeight: '200px',
      overflowY: 'auto' as const,
    },
    historyItem: {
      display: 'flex',
      justifyContent: 'space-between',
      padding: '8px 0',
      borderBottom: `1px solid ${colors.border}`,
      fontSize: '0.875rem',
    },
  };

  const fetchData = async (pwd: string) => {
    setLoading(true);
    try {
      const [endpointsData, metricsData, settingsData, siteConfigData] = await Promise.all([
        adminListEndpoints(pwd),
        adminGetMetrics(pwd),
        adminGetSettings(pwd),
        adminGetSiteConfig(pwd),
      ]);
      setEndpoints(endpointsData || []);
      setMetrics(metricsData);
      setSettings(settingsData);
      setSiteConfig(siteConfigData);
      setError(null);
      setIsLoggedIn(true);
      sessionStorage.setItem('adminPassword', pwd);
      setPassword(pwd);
    } catch (err) {
      if (err instanceof Error && err.message === 'Authentication required') {
        setIsLoggedIn(false);
        sessionStorage.removeItem('adminPassword');
        setError('Invalid password');
      } else {
        setError(err instanceof Error ? err.message : 'Failed to load data');
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (password) {
      fetchData(password);
    }
  }, []);

  useEffect(() => {
    if (!isLoggedIn) return;
    const interval = setInterval(() => fetchData(password), 10000);
    return () => clearInterval(interval);
  }, [isLoggedIn, password]);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    await fetchData(loginPassword);
  };

  const handleLogout = () => {
    sessionStorage.removeItem('adminPassword');
    setPassword('');
    setIsLoggedIn(false);
    setEndpoints([]);
    setMetrics(null);
    setSettings(null);
    setSiteConfig(null);
  };

  const handleSaveSettings = async (newThreshold: number) => {
    setSavingSettings(true);
    setError(null);
    try {
      const updated = await adminUpdateSettings(password, { outage_threshold: newThreshold });
      setSettings(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save settings');
    } finally {
      setSavingSettings(false);
    }
  };

  const handleSaveSiteConfig = async (config: SiteConfig) => {
    setSavingSiteConfig(true);
    setError(null);
    try {
      const updated = await adminUpdateSiteConfig(password, config);
      setSiteConfig(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save site config');
    } finally {
      setSavingSiteConfig(false);
    }
  };

  const formatBytes = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
  };

  const getUptimeColor = (pct: number) => {
    if (pct >= 90) return colors.success;
    if (pct >= 70) return colors.warning;
    return colors.danger;
  };

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newIP.trim()) return;

    setAdding(true);
    setError(null);

    try {
      await adminAddEndpoint(password, {
        ipv4: newIP.trim(),
        isp: newISP.trim() || undefined,
      });
      setNewIP('');
      setNewISP('');
      await fetchData(password);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add endpoint');
    } finally {
      setAdding(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm(`Delete endpoint ${id}?`)) return;

    try {
      await adminDeleteEndpoint(password, id);
      await fetchData(password);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete endpoint');
    }
  };

  const formatTime = (isoString?: string) => {
    if (!isoString || isoString.startsWith('0001')) return '-';
    try {
      return new Date(isoString).toLocaleString();
    } catch {
      return '-';
    }
  };

  const getStatusStyle = (status: string) => {
    switch (status) {
      case 'up': return styles.statusUp;
      case 'down': return styles.statusDown;
      default: return styles.statusUnknown;
    }
  };

  if (!isLoggedIn) {
    return (
      <div style={styles.container}>
        <div style={styles.header}>
          <h1 style={styles.title}>Admin Login</h1>
          <span style={styles.backLink} onClick={onBack}>&larr; Back to Dashboard</span>
        </div>

        <div style={styles.loginForm}>
          <div style={styles.loginTitle}>Enter Admin Password</div>
          {error && <div style={styles.error}>{error}</div>}
          <form onSubmit={handleLogin}>
            <input
              type="password"
              placeholder="Password"
              value={loginPassword}
              onChange={(e) => setLoginPassword(e.target.value)}
              style={{ ...styles.input, width: '100%', marginBottom: '10px' }}
              autoFocus
            />
            <button
              type="submit"
              style={{ ...styles.button, ...styles.loginButton }}
              disabled={loading}
            >
              {loading ? 'Logging in...' : 'Login'}
            </button>
          </form>
        </div>
      </div>
    );
  }

  const renderMetrics = () => {
    if (!metrics) {
      return <div style={styles.loading}>Loading metrics...</div>;
    }

    return (
      <>
        {/* Overview Stats */}
        <div style={styles.metricsGrid}>
          <div style={styles.metricCard}>
            <div style={styles.metricLabel}>Total Endpoints</div>
            <div style={styles.metricValue}>{metrics.total_endpoints}</div>
          </div>
          <div style={styles.metricCard}>
            <div style={styles.metricLabel}>Endpoints Up</div>
            <div style={{ ...styles.metricValue, color: colors.success }}>
              {metrics.endpoints_up}
            </div>
          </div>
          <div style={styles.metricCard}>
            <div style={styles.metricLabel}>Endpoints Down</div>
            <div style={{ ...styles.metricValue, color: metrics.endpoints_down > 0 ? colors.danger : colors.text }}>
              {metrics.endpoints_down}
            </div>
          </div>
          <div style={styles.metricCard}>
            <div style={styles.metricLabel}>Overall Uptime</div>
            <div style={{ ...styles.metricValue, color: getUptimeColor(metrics.overall_uptime_pct) }}>
              {metrics.overall_uptime_pct.toFixed(1)}%
            </div>
          </div>
        </div>

        {/* Monitoring Stats */}
        <div style={styles.section}>
          <div style={styles.sectionTitle}>Monitoring</div>
          <div style={styles.metricsGrid}>
            <div style={styles.metricCard}>
              <div style={styles.metricLabel}>Last Ping</div>
              <div style={styles.metricValue}>{formatTime(metrics.last_ping_time)}</div>
            </div>
            <div style={styles.metricCard}>
              <div style={styles.metricLabel}>Ping Interval</div>
              <div style={styles.metricValue}>{metrics.ping_interval}</div>
            </div>
            <div style={styles.metricCard}>
              <div style={styles.metricLabel}>Next Ping</div>
              <div style={styles.metricValue}>{formatTime(metrics.next_ping_time)}</div>
            </div>
            <div style={styles.metricCard}>
              <div style={styles.metricLabel}>Total Ping Cycles</div>
              <div style={styles.metricValue}>{metrics.total_ping_cycles.toLocaleString()}</div>
            </div>
          </div>
          <div style={styles.metricsGrid}>
            <div style={styles.metricCard}>
              <div style={styles.metricLabel}>Direct Monitored</div>
              <div style={styles.metricValue}>{metrics.direct_monitored}</div>
              <div style={styles.metricSmall}>Endpoints pinged directly</div>
            </div>
            <div style={styles.metricCard}>
              <div style={styles.metricLabel}>Hop Monitored</div>
              <div style={styles.metricValue}>{metrics.hop_monitored}</div>
              <div style={styles.metricSmall}>Endpoints via traceroute hop</div>
            </div>
            <div style={styles.metricCard}>
              <div style={styles.metricLabel}>Shared Hops</div>
              <div style={styles.metricValue}>{metrics.shared_hops}</div>
              <div style={styles.metricSmall}>Common infrastructure points</div>
            </div>
          </div>
        </div>

        {/* ISP Breakdown */}
        <div style={styles.section}>
          <div style={styles.sectionTitle}>ISP Status</div>
          <div style={styles.ispGrid}>
            {metrics.isp_stats.map((isp) => (
              <div key={isp.name} style={styles.ispCard}>
                <div style={styles.ispName}>
                  {isp.name}
                  {isp.likely_outage && (
                    <span style={{ marginLeft: '8px', color: colors.danger, fontSize: '0.75rem' }}>
                      OUTAGE
                    </span>
                  )}
                </div>
                <div style={styles.ispStats}>
                  <div style={styles.ispStat}>
                    <span style={{ color: colors.textMuted }}>Total</span>
                    <span style={{ fontWeight: 'bold' }}>{isp.total}</span>
                  </div>
                  <div style={styles.ispStat}>
                    <span style={{ color: colors.success }}>Up</span>
                    <span style={{ fontWeight: 'bold' }}>{isp.up}</span>
                  </div>
                  <div style={styles.ispStat}>
                    <span style={{ color: colors.danger }}>Down</span>
                    <span style={{ fontWeight: 'bold' }}>{isp.down}</span>
                  </div>
                  <div style={styles.ispStat}>
                    <span style={{ color: colors.textMuted }}>Unknown</span>
                    <span style={{ fontWeight: 'bold' }}>{isp.unknown}</span>
                  </div>
                </div>
                <div style={styles.uptimeBar}>
                  <div
                    style={{
                      ...styles.uptimeFill,
                      width: `${isp.uptime_pct}%`,
                      background: getUptimeColor(isp.uptime_pct),
                    }}
                  />
                </div>
                <div style={{ ...styles.metricSmall, marginTop: '5px' }}>
                  {isp.uptime_pct.toFixed(1)}% uptime
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* System Info */}
        <div style={styles.section}>
          <div style={styles.sectionTitle}>System</div>
          <div style={styles.metricsGrid}>
            <div style={styles.metricCard}>
              <div style={styles.metricLabel}>Version</div>
              <div style={styles.metricValue}>{metrics.version}</div>
            </div>
            <div style={styles.metricCard}>
              <div style={styles.metricLabel}>Server Uptime</div>
              <div style={styles.metricValue}>{metrics.server_uptime || '-'}</div>
            </div>
            <div style={styles.metricCard}>
              <div style={styles.metricLabel}>Database Size</div>
              <div style={styles.metricValue}>{formatBytes(metrics.database_size_bytes)}</div>
            </div>
            <div style={styles.metricCard}>
              <div style={styles.metricLabel}>Database Path</div>
              <div style={{ ...styles.metricValue, fontSize: '0.875rem', wordBreak: 'break-all' }}>
                {metrics.database_path}
              </div>
            </div>
          </div>
        </div>
      </>
    );
  };

  const renderEndpoints = () => (
    <>
      <div style={styles.addForm}>
        <form onSubmit={handleAdd}>
          <div style={styles.formRow}>
            <input
              type="text"
              placeholder="IP Address (e.g., 1.2.3.4)"
              value={newIP}
              onChange={(e) => setNewIP(e.target.value)}
              style={styles.input}
              required
            />
            <input
              type="text"
              placeholder="ISP (optional, auto-detect)"
              value={newISP}
              onChange={(e) => setNewISP(e.target.value)}
              style={{ ...styles.input, flex: 0.5 }}
            />
            <button
              type="submit"
              style={{ ...styles.button, ...styles.addButton }}
              disabled={adding}
            >
              {adding ? 'Adding...' : 'Add Endpoint'}
            </button>
          </div>
        </form>
      </div>

      {loading && endpoints.length === 0 ? (
        <div style={styles.loading}>Loading...</div>
      ) : endpoints.length === 0 ? (
        <div style={styles.noData}>No endpoints registered yet.</div>
      ) : (
        <table style={styles.table}>
          <thead>
            <tr>
              <th style={styles.th}>ID</th>
              <th style={styles.th}>IP Address</th>
              <th style={styles.th}>ISP</th>
              <th style={styles.th}>Status</th>
              <th style={styles.th}>Monitor Target</th>
              <th style={styles.th}>Last Seen</th>
              <th style={styles.th}>Last OK</th>
              <th style={styles.th}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {endpoints.map((ep) => (
              <tr key={ep.id}>
                <td style={styles.td}>{ep.id}</td>
                <td style={styles.td}>{ep.ipv4}</td>
                <td style={styles.td}>{ep.isp.toUpperCase()}</td>
                <td style={{ ...styles.td, ...getStatusStyle(ep.status) }}>
                  {ep.status.toUpperCase()}
                </td>
                <td style={styles.td}>
                  {ep.use_hop && ep.monitored_hop
                    ? `Hop ${ep.hop_number}: ${ep.monitored_hop}`
                    : 'Direct'}
                </td>
                <td style={styles.td}>{formatTime(ep.last_seen)}</td>
                <td style={styles.td}>{formatTime(ep.last_ok)}</td>
                <td style={styles.td}>
                  <button
                    style={{ ...styles.button, ...styles.deleteButton }}
                    onClick={() => handleDelete(ep.id)}
                  >
                    Delete
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </>
  );

  const renderSettings = () => {
    const currentThreshold = settings?.outage_threshold ?? 0.5;
    const thresholdPercent = Math.round(currentThreshold * 100);

    return (
      <div style={styles.section}>
        <div style={styles.sectionTitle}>Outage Detection</div>
        <p style={{ color: colors.textMuted, marginBottom: '20px', lineHeight: 1.6 }}>
          Configure the threshold for detecting ISP outages. When the percentage of down endpoints
          for an ISP exceeds this threshold, the dashboard will show a "likely outage" warning.
        </p>
        <div style={{ marginBottom: '20px' }}>
          <div style={{ marginBottom: '10px', display: 'flex', alignItems: 'center', gap: '15px' }}>
            <label style={{ color: colors.text, fontWeight: 'bold' }}>
              Outage Threshold: {thresholdPercent}%
            </label>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '15px' }}>
            <span style={{ color: colors.textMuted, fontSize: '0.875rem' }}>10%</span>
            <input
              type="range"
              min="10"
              max="90"
              step="5"
              value={thresholdPercent}
              onChange={(e) => {
                const newThreshold = parseInt(e.target.value) / 100;
                handleSaveSettings(newThreshold);
              }}
              disabled={savingSettings}
              style={{ flex: 1, cursor: savingSettings ? 'not-allowed' : 'pointer' }}
            />
            <span style={{ color: colors.textMuted, fontSize: '0.875rem' }}>90%</span>
          </div>
          <p style={{ color: colors.textDimmed, fontSize: '0.75rem', marginTop: '10px' }}>
            {savingSettings ? 'Saving...' : `Outage will be detected when >${thresholdPercent}% of endpoints for an ISP are down.`}
          </p>
        </div>
      </div>
    );
  };

  return (
    <div style={styles.container}>
      <div style={styles.header}>
        <h1 style={styles.title}>Admin Dashboard</h1>
        <div>
          <button
            style={{ ...styles.button, ...styles.logoutButton, marginRight: '10px' }}
            onClick={handleLogout}
          >
            Logout
          </button>
          <span style={styles.backLink} onClick={onBack}>&larr; Back</span>
        </div>
      </div>

      {error && <div style={styles.error}>{error}</div>}

      <div style={styles.tabs}>
        <button
          style={{
            ...styles.tab,
            ...(activeTab === 'metrics' ? styles.tabActive : {}),
          }}
          onClick={() => setActiveTab('metrics')}
        >
          Metrics
        </button>
        <button
          style={{
            ...styles.tab,
            ...(activeTab === 'endpoints' ? styles.tabActive : {}),
          }}
          onClick={() => setActiveTab('endpoints')}
        >
          Endpoints ({endpoints.length})
        </button>
        <button
          style={{
            ...styles.tab,
            ...(activeTab === 'settings' ? styles.tabActive : {}),
          }}
          onClick={() => setActiveTab('settings')}
        >
          Settings
        </button>
        <button
          style={{
            ...styles.tab,
            ...(activeTab === 'site-config' ? styles.tabActive : {}),
          }}
          onClick={() => setActiveTab('site-config')}
        >
          Site Config
        </button>
      </div>

      {activeTab === 'metrics' && renderMetrics()}
      {activeTab === 'endpoints' && renderEndpoints()}
      {activeTab === 'settings' && renderSettings()}
      {activeTab === 'site-config' && (
        <SiteConfigEditor
          siteConfig={siteConfig}
          onSave={handleSaveSiteConfig}
          saving={savingSiteConfig}
          colors={colors}
        />
      )}
    </div>
  );
}

export default Admin;
