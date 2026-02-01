import { useState } from 'react';
import { register } from '../api';
import type { ThemeColors } from '../App';

interface OptInPromptProps {
  isp: string;
  onRegistered: () => void;
  colors: ThemeColors;
}

function OptInPrompt({ isp, onRegistered, colors }: OptInPromptProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleRegister = async () => {
    setLoading(true);
    setError(null);

    try {
      await register();
      onRegistered();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to join monitoring');
    } finally {
      setLoading(false);
    }
  };

  const styles = {
    container: {
      background: colors.bgCardAlt,
      border: `1px solid ${colors.accent}`,
      borderRadius: '12px',
      padding: '25px',
      marginBottom: '20px',
      textAlign: 'center' as const,
    },
    title: {
      fontSize: '1.25rem',
      fontWeight: 'bold',
      marginBottom: '10px',
      color: colors.text,
    },
    description: {
      color: colors.textMuted,
      marginBottom: '20px',
      lineHeight: 1.6,
    },
    ispBadge: {
      display: 'inline-block',
      background: colors.border,
      color: colors.text,
      padding: '4px 12px',
      borderRadius: '20px',
      marginBottom: '15px',
      fontSize: '0.875rem',
    },
    button: {
      background: colors.accent,
      color: 'white',
      border: 'none',
      padding: '12px 30px',
      borderRadius: '8px',
      fontSize: '1rem',
      fontWeight: 'bold',
      cursor: 'pointer',
      transition: 'opacity 0.2s',
    },
    buttonDisabled: {
      opacity: 0.6,
      cursor: 'not-allowed',
    },
    error: {
      color: colors.errorText,
      marginTop: '10px',
      fontSize: '0.875rem',
    },
    privacy: {
      marginTop: '15px',
      fontSize: '0.75rem',
      color: colors.textDimmed,
    },
  };

  return (
    <div style={styles.container}>
      <div style={styles.ispBadge}>
        Detected ISP: {isp}
      </div>
      <h2 style={styles.title}>Join Community Monitoring</h2>
      <p style={styles.description}>
        Help monitor connectivity in our building by allowing periodic pings
        to your connection. Your IP is stored securely and never shared.
      </p>
      <button
        style={{
          ...styles.button,
          ...(loading ? styles.buttonDisabled : {}),
        }}
        onClick={handleRegister}
        disabled={loading}
      >
        {loading ? 'Joining...' : 'Join Monitoring'}
      </button>
      {error && (
        <div style={styles.error}>{error}</div>
      )}
      <p style={styles.privacy}>
        Inactive connections are automatically removed after 30 days.
      </p>
    </div>
  );
}

export default OptInPrompt;
