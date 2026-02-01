import type { ThemeColors } from '../App';

interface AboutProps {
  onBack: () => void;
  colors: ThemeColors;
}

function About({ onBack, colors }: AboutProps) {
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
  };

  return (
    <div style={styles.container}>
      <div style={styles.header}>
        <h1 style={styles.title}>About CCC</h1>
        <span style={styles.backLink} onClick={onBack}>&larr; Back to Dashboard</span>
      </div>

      <div style={styles.content}>
        <div style={styles.section}>
          <h2 style={styles.sectionTitle}>Why I Built This</h2>
          <p style={styles.paragraph}>
            Living in a building with multiple ISP options, I noticed that when internet issues occur,
            it's often hard to tell if it's just my connection or a wider problem affecting the whole building.
            Is it my router? My ISP? Or is everyone experiencing the same issue?
          </p>
          <p style={styles.paragraph}>
            As a network engineer in my day-to-day work, I wanted to create something that helps our
            community get a better overview of the internet status in the building.
            <span style={styles.highlight}> Community Connectivity Check (CCC)</span> is the result -
            a simple tool that lets residents see at a glance whether connectivity issues are isolated
            or widespread.
          </p>
        </div>

        <div style={styles.section}>
          <h2 style={styles.sectionTitle}>How It Works</h2>
          <p style={styles.paragraph}>
            CCC monitors the connectivity of participating residents by periodically checking if their
            connection is reachable. Here's the process:
          </p>
          <ul style={styles.list}>
            <li style={styles.listItem}>
              <span style={styles.highlight}>Opt-in Monitoring:</span> Residents can choose to participate
              by joining the monitoring. Your IP is used only for connectivity checks.
            </li>
            <li style={styles.listItem}>
              <span style={styles.highlight}>Regular Checks:</span> The system performs connectivity checks
              every minute to all participating connections.
            </li>
            <li style={styles.listItem}>
              <span style={styles.highlight}>ISP Detection:</span> Your ISP is automatically detected so we
              can group results and identify if issues are ISP-specific.
            </li>
            <li style={styles.listItem}>
              <span style={styles.highlight}>Outage Detection:</span> When multiple connections from the same
              ISP go down simultaneously, the system flags a likely ISP outage.
            </li>
          </ul>
          <p style={styles.paragraph}>
            <span style={styles.highlight}>Note:</span> Some routers block ping requests for security reasons.
            If yours does, you can still join and view the dashboard - you just won't contribute to the
            monitoring data.
          </p>
        </div>

        <div style={styles.section}>
          <h2 style={styles.sectionTitle}>Privacy</h2>
          <p style={styles.paragraph}>
            Your privacy matters. CCC only stores the minimum data needed to function:
          </p>
          <ul style={styles.list}>
            <li style={styles.listItem}>Your IP address (for connectivity checks)</li>
            <li style={styles.listItem}>Your ISP name (for grouping)</li>
            <li style={styles.listItem}>Connection status (up/down)</li>
          </ul>
          <p style={styles.paragraph}>
            No personal information is collected, and the dashboard only shows aggregated,
            anonymized statistics. Individual connection details are never exposed publicly.
          </p>
          <p style={styles.paragraph}>
            <span style={styles.highlight}>Why no notifications?</span><br />
            We intentionally don't offer outage notifications because that would require collecting
            email addresses or phone numbers. We believe in keeping participation completely anonymous.
          </p>
        </div>

        <div style={styles.section}>
          <h2 style={styles.sectionTitle}>Supported ISPs</h2>
          <p style={styles.paragraph}>
            Currently, monitoring is available for residents using:
          </p>
          <ul style={styles.list}>
            <li style={styles.listItem}><span style={styles.highlight}>Comcast</span></li>
            <li style={styles.listItem}><span style={styles.highlight}>Starry</span></li>
          </ul>
          <p style={styles.paragraph}>
            If you're on a different ISP and would like to participate, let me know and I can
            look into adding support.
          </p>
        </div>

        <div style={styles.footer}>
          <p style={{ marginBottom: '10px' }}>Built with care for our building community.</p>
          <p style={{ fontSize: '0.8rem', marginBottom: '10px' }}>
            Wondering about the domain? AS215855 is my personal Autonomous System Number -
            a unique identifier used in internet routing. It's a bit of a network engineer hobby thing.
          </p>
          <p style={{ fontSize: '0.8rem' }}>
            Questions or feedback? <a href="mailto:ccc@as215855.net" style={{ color: colors.accent }}>ccc@as215855.net</a>
          </p>
        </div>
      </div>
    </div>
  );
}

export default About;
