import type { ISPStatus } from '../types';
import type { ThemeColors } from '../App';

interface StatusCardProps {
  status: ISPStatus;
  isCurrentISP: boolean;
  colors: ThemeColors;
}

function StatusCard({ status, isCurrentISP, colors }: StatusCardProps) {
  const upPercent = status.total > 0 ? (status.up / status.total) * 100 : 0;

  let statusColor: string;
  let statusBg: string;

  if (status.total === 0) {
    statusColor = colors.textMuted;
    statusBg = colors.border;
  } else if (upPercent >= 90) {
    statusColor = colors.success;
    statusBg = colors.successBg;
  } else if (upPercent >= 50) {
    statusColor = colors.warning;
    statusBg = colors.warningBg;
  } else {
    statusColor = colors.danger;
    statusBg = colors.dangerBg;
  }

  const styles = {
    card: {
      background: colors.bgCard,
      borderRadius: '12px',
      padding: '20px',
      border: isCurrentISP ? `2px solid ${colors.accent}` : `1px solid ${colors.border}`,
    },
    header: {
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'center',
      marginBottom: '15px',
    },
    ispInfo: {
      display: 'flex',
      alignItems: 'center',
      gap: '10px',
    },
    ispIcon: {
      width: '32px',
      height: '32px',
      borderRadius: '6px',
      objectFit: 'contain' as const,
    },
    ispName: {
      fontSize: '1.25rem',
      fontWeight: 'bold',
      color: colors.text,
    },
    badge: {
      background: statusBg,
      color: statusColor,
      padding: '4px 12px',
      borderRadius: '20px',
      fontSize: '0.875rem',
      fontWeight: 'bold',
    },
    stats: {
      display: 'flex',
      gap: '20px',
    },
    stat: {
      flex: 1,
    },
    statValue: {
      fontSize: '1.5rem',
      fontWeight: 'bold',
    },
    statLabel: {
      color: colors.textMuted,
      fontSize: '0.875rem',
    },
    progressBar: {
      marginTop: '15px',
      height: '8px',
      background: colors.border,
      borderRadius: '4px',
      overflow: 'hidden',
    },
    progressFill: {
      height: '100%',
      background: statusColor,
      width: `${upPercent}%`,
      transition: 'width 0.3s ease',
    },
    currentLabel: {
      marginTop: '10px',
      fontSize: '0.75rem',
      color: colors.accent,
    },
  };

  const iconUrl = status.asn ? `https://static.ui.com/asn/${status.asn}_101x101.png` : null;

  return (
    <div style={styles.card}>
      <div style={styles.header}>
        <div style={styles.ispInfo}>
          {iconUrl && (
            <img
              src={iconUrl}
              alt={status.name}
              style={styles.ispIcon}
              onError={(e) => { (e.target as HTMLImageElement).style.display = 'none'; }}
            />
          )}
          <span style={styles.ispName}>{status.name}</span>
        </div>
        <span style={styles.badge}>
          {status.total === 0 ? 'No Data' : `${Math.round(upPercent)}% Up`}
        </span>
      </div>

      <div style={styles.stats}>
        <div style={styles.stat}>
          <div style={{ ...styles.statValue, color: colors.success }}>{status.up}</div>
          <div style={styles.statLabel}>Online</div>
        </div>
        <div style={styles.stat}>
          <div style={{ ...styles.statValue, color: colors.danger }}>{status.down}</div>
          <div style={styles.statLabel}>Offline</div>
        </div>
        <div style={styles.stat}>
          <div style={{ ...styles.statValue, color: colors.text }}>{status.total}</div>
          <div style={styles.statLabel}>Total</div>
        </div>
      </div>

      <div style={styles.progressBar}>
        <div style={styles.progressFill} />
      </div>

      {isCurrentISP && (
        <div style={styles.currentLabel}>Your ISP</div>
      )}
    </div>
  );
}

export default StatusCard;
