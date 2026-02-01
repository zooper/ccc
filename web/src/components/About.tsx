import { useState, useEffect } from 'react';
import { getSiteConfig } from '../api';
import type { SiteConfig } from '../types';
import type { ThemeColors } from '../App';

interface AboutProps {
  onBack: () => void;
  colors: ThemeColors;
}

function About({ onBack, colors }: AboutProps) {
  const [config, setConfig] = useState<SiteConfig | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getSiteConfig()
      .then(setConfig)
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  const styles = {
    container: {
      padding: '20px',
    },
    header: {
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'center',
      marginBottom: '30px',
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
    content: {
      background: colors.bgCard,
      padding: '30px',
      borderRadius: '12px',
      border: `1px solid ${colors.border}`,
    },
    section: {
      marginBottom: '25px',
    },
    sectionTitle: {
      fontSize: '1.125rem',
      fontWeight: 'bold',
      marginBottom: '10px',
      color: colors.text,
    },
    paragraph: {
      color: colors.textMuted,
      lineHeight: 1.7,
      marginBottom: '15px',
      whiteSpace: 'pre-wrap' as const,
    },
    list: {
      color: colors.textMuted,
      lineHeight: 1.7,
      paddingLeft: '20px',
      marginBottom: '15px',
    },
    listItem: {
      marginBottom: '8px',
    },
    highlight: {
      color: colors.text,
      fontWeight: 'bold',
    },
    footer: {
      marginTop: '30px',
      paddingTop: '20px',
      borderTop: `1px solid ${colors.border}`,
      color: colors.textDimmed,
      fontSize: '0.875rem',
    },
    loading: {
      textAlign: 'center' as const,
      padding: '40px',
      color: colors.textMuted,
    },
    emptyState: {
      color: colors.textDimmed,
      fontStyle: 'italic' as const,
    },
  };

  if (loading) {
    return (
      <div style={styles.container}>
        <div style={styles.loading}>Loading...</div>
      </div>
    );
  }

  const siteName = config?.site_name || 'Community Connectivity Check';

  return (
    <div style={styles.container}>
      <div style={styles.header}>
        <h1 style={styles.title}>About {siteName}</h1>
        <span style={styles.backLink} onClick={onBack}>&larr; Back to Dashboard</span>
      </div>

      <div style={styles.content}>
        {config?.about_why && (
          <div style={styles.section}>
            <h2 style={styles.sectionTitle}>Why I Built This</h2>
            <p style={styles.paragraph}>{config.about_why}</p>
          </div>
        )}

        {config?.about_how_it_works && (
          <div style={styles.section}>
            <h2 style={styles.sectionTitle}>How It Works</h2>
            <p style={styles.paragraph}>{config.about_how_it_works}</p>
          </div>
        )}

        {config?.about_privacy && (
          <div style={styles.section}>
            <h2 style={styles.sectionTitle}>Privacy</h2>
            <p style={styles.paragraph}>{config.about_privacy}</p>
          </div>
        )}

        {config?.supported_isps && config.supported_isps.length > 0 && (
          <div style={styles.section}>
            <h2 style={styles.sectionTitle}>Supported ISPs</h2>
            <p style={styles.paragraph}>
              Currently, monitoring is available for residents using:
            </p>
            <ul style={styles.list}>
              {config.supported_isps.map((isp, i) => (
                <li key={i} style={styles.listItem}>
                  <span style={styles.highlight}>{isp}</span>
                </li>
              ))}
            </ul>
          </div>
        )}

        {!config?.about_why && !config?.about_how_it_works && !config?.about_privacy && (
          <div style={styles.emptyState}>
            No content has been configured yet. An administrator can set up the About page content in the admin panel.
          </div>
        )}

        {(config?.footer_text || config?.contact_email || config?.github_url) && (
          <div style={styles.footer}>
            {config.footer_text && (
              <p style={{ marginBottom: '10px', whiteSpace: 'pre-wrap' }}>{config.footer_text}</p>
            )}
            {config.contact_email && (
              <p style={{ fontSize: '0.8rem' }}>
                Questions or feedback?{' '}
                <a href={`mailto:${config.contact_email}`} style={{ color: colors.accent }}>
                  {config.contact_email}
                </a>
              </p>
            )}
            {config.github_url && (
              <p style={{ fontSize: '0.8rem', marginTop: '5px' }}>
                <a href={config.github_url} target="_blank" rel="noopener noreferrer" style={{ color: colors.accent }}>
                  View source on GitHub
                </a>
              </p>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

export default About;
