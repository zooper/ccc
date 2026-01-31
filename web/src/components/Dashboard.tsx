import type { DashboardResponse, Event } from '../types';
import type { ThemeColors } from '../App';
import StatusCard from './StatusCard';

interface DashboardProps {
  data: DashboardResponse;
  currentISP?: string;
  colors: ThemeColors;
  events: Event[];
}

function Dashboard({ data, currentISP, colors, events }: DashboardProps) {
  const styles = {
    container: {
      display: 'flex',
      flexDirection: 'column' as const,
      gap: '20px',
    },
    outageAlert: {
      background: colors.dangerBg,
      border: `1px solid ${colors.danger}`,
      color: colors.errorText,
      padding: '20px',
      borderRadius: '12px',
      textAlign: 'center' as const,
    },
    outageTitle: {
      fontSize: '1.25rem',
      fontWeight: 'bold',
      marginBottom: '5px',
    },
    outageSubtitle: {
      fontSize: '0.875rem',
      opacity: 0.9,
    },
    grid: {
      display: 'grid',
      gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))',
      gap: '15px',
    },
    noData: {
      textAlign: 'center' as const,
      padding: '40px',
      color: colors.textMuted,
    },
    footer: {
      textAlign: 'center' as const,
      color: colors.textDimmed,
      fontSize: '0.75rem',
      marginTop: '20px',
    },
    eventsSection: {
      background: colors.bgCard,
      borderRadius: '12px',
      padding: '20px',
      border: `1px solid ${colors.border}`,
    },
    eventsTitle: {
      fontSize: '1rem',
      fontWeight: 'bold',
      marginBottom: '15px',
      color: colors.text,
    },
    eventsList: {
      display: 'flex',
      flexDirection: 'column' as const,
      gap: '8px',
      maxHeight: '250px',
      overflowY: 'auto' as const,
    },
    eventItem: {
      display: 'flex',
      alignItems: 'center',
      gap: '12px',
      padding: '10px 12px',
      background: colors.bg,
      borderRadius: '8px',
      fontSize: '0.875rem',
    },
    eventIcon: {
      width: '24px',
      height: '24px',
      borderRadius: '50%',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      fontSize: '12px',
      flexShrink: 0,
    },
    eventContent: {
      flex: 1,
    },
    eventMessage: {
      color: colors.text,
    },
    eventTime: {
      color: colors.textMuted,
      fontSize: '0.75rem',
    },
    noEvents: {
      color: colors.textMuted,
      textAlign: 'center' as const,
      padding: '20px',
    },
  };

  const formatTime = (isoString: string) => {
    try {
      return new Date(isoString).toLocaleTimeString();
    } catch {
      return 'Unknown';
    }
  };

  const formatDateTime = (isoString: string) => {
    try {
      const date = new Date(isoString);
      const now = new Date();
      const isToday = date.toDateString() === now.toDateString();
      if (isToday) {
        return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
      }
      return date.toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
    } catch {
      return 'Unknown';
    }
  };

  const getEventStyle = (eventType: string) => {
    switch (eventType) {
      case 'down':
        return { bg: colors.dangerBg, color: colors.danger, icon: '↓' };
      case 'up':
        return { bg: colors.successBg, color: colors.success, icon: '↑' };
      case 'outage':
        return { bg: colors.dangerBg, color: colors.danger, icon: '!' };
      case 'recovery':
        return { bg: colors.successBg, color: colors.success, icon: '✓' };
      default:
        return { bg: colors.border, color: colors.textMuted, icon: '•' };
    }
  };

  return (
    <div style={styles.container}>
      {data.likely_outage && (
        <div style={styles.outageAlert}>
          <div style={styles.outageTitle}>Possible Outage Detected</div>
          <div style={styles.outageSubtitle}>
            One or more ISPs are experiencing connectivity issues
          </div>
        </div>
      )}

      {data.isps.length === 0 ? (
        <div style={styles.noData}>
          No monitoring data yet. Be the first to register!
        </div>
      ) : (
        <div style={styles.grid}>
          {data.isps.map((isp) => (
            <StatusCard
              key={isp.name}
              status={isp}
              isCurrentISP={isp.name === currentISP}
              colors={colors}
            />
          ))}
        </div>
      )}

      {/* Events Section */}
      <div style={styles.eventsSection}>
        <div style={styles.eventsTitle}>Recent Events</div>
        {events.length === 0 ? (
          <div style={styles.noEvents}>No events yet. Status changes will appear here.</div>
        ) : (
          <div style={styles.eventsList}>
            {events.slice(0, 15).map((event) => {
              const eventStyle = getEventStyle(event.event_type);
              return (
                <div key={event.id} style={styles.eventItem}>
                  <div style={{
                    ...styles.eventIcon,
                    background: eventStyle.bg,
                    color: eventStyle.color,
                  }}>
                    {eventStyle.icon}
                  </div>
                  <div style={styles.eventContent}>
                    <div style={styles.eventMessage}>{event.message}</div>
                    <div style={styles.eventTime}>{formatDateTime(event.timestamp)}</div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      <div style={styles.footer}>
        Last updated: {formatTime(data.last_updated)}
      </div>
    </div>
  );
}

export default Dashboard;
