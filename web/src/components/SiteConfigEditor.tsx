import { useState, useEffect } from 'react';
import type { SiteConfig } from '../types';
import type { ThemeColors } from '../App';

interface SiteConfigEditorProps {
  siteConfig: SiteConfig | null;
  onSave: (config: SiteConfig) => Promise<void>;
  saving: boolean;
  colors: ThemeColors;
}

function SiteConfigEditor({ siteConfig, onSave, saving, colors }: SiteConfigEditorProps) {
  const [localConfig, setLocalConfig] = useState<SiteConfig>(siteConfig || {
    site_name: '',
    site_description: '',
    about_why: '',
    about_how_it_works: '',
    about_privacy: '',
    supported_isps: [],
    contact_email: '',
    footer_text: '',
    github_url: '',
  });
  const [ispInput, setIspInput] = useState('');

  useEffect(() => {
    if (siteConfig) {
      setLocalConfig(siteConfig);
    }
  }, [siteConfig]);

  const handleChange = (field: keyof SiteConfig, value: string | string[]) => {
    setLocalConfig(prev => ({ ...prev, [field]: value }));
  };

  const handleAddISP = () => {
    if (ispInput.trim() && !localConfig.supported_isps.includes(ispInput.trim())) {
      handleChange('supported_isps', [...localConfig.supported_isps, ispInput.trim()]);
      setIspInput('');
    }
  };

  const handleRemoveISP = (isp: string) => {
    handleChange('supported_isps', localConfig.supported_isps.filter(i => i !== isp));
  };

  const styles = {
    section: {
      background: colors.bgCard,
      padding: '20px',
      borderRadius: '12px',
      marginBottom: '20px',
      border: `1px solid ${colors.border}`,
    },
    sectionTitle: {
      fontSize: '1.125rem',
      fontWeight: 'bold' as const,
      marginBottom: '15px',
      color: colors.text,
    },
    input: {
      width: '100%',
      padding: '10px',
      borderRadius: '6px',
      border: `1px solid ${colors.border}`,
      background: colors.bg,
      color: colors.text,
      fontSize: '1rem',
      marginBottom: '15px',
    },
    textarea: {
      width: '100%',
      padding: '10px',
      borderRadius: '6px',
      border: `1px solid ${colors.border}`,
      background: colors.bg,
      color: colors.text,
      fontSize: '1rem',
      marginBottom: '15px',
      minHeight: '100px',
      resize: 'vertical' as const,
      fontFamily: 'inherit',
    },
    label: {
      display: 'block',
      marginBottom: '5px',
      color: colors.text,
      fontWeight: 'bold' as const,
      fontSize: '0.875rem',
    },
    help: {
      fontSize: '0.75rem',
      color: colors.textDimmed,
      marginTop: '-10px',
      marginBottom: '15px',
    },
    button: {
      padding: '10px 20px',
      borderRadius: '6px',
      border: 'none',
      cursor: 'pointer',
      fontSize: '1rem',
      fontWeight: 'bold' as const,
      background: colors.success,
      color: 'white',
    },
    tag: {
      background: colors.border,
      padding: '4px 10px',
      borderRadius: '20px',
      fontSize: '0.875rem',
      display: 'flex' as const,
      alignItems: 'center' as const,
      gap: '6px',
    },
  };

  return (
    <div style={styles.section}>
      <div style={styles.sectionTitle}>Site Configuration</div>
      <p style={{ color: colors.textMuted, marginBottom: '20px', lineHeight: 1.6 }}>
        Customize the site name, About page content, and other public-facing text.
      </p>

      <div style={{ marginBottom: '20px' }}>
        <label style={styles.label}>Site Name</label>
        <input
          type="text"
          value={localConfig.site_name}
          onChange={(e) => handleChange('site_name', e.target.value)}
          placeholder="Community Connectivity Check"
          style={styles.input}
        />

        <label style={styles.label}>Site Description</label>
        <input
          type="text"
          value={localConfig.site_description}
          onChange={(e) => handleChange('site_description', e.target.value)}
          placeholder="Monitor ISP connectivity in our building"
          style={styles.input}
        />

        <label style={styles.label}>About: Why I Built This</label>
        <textarea
          value={localConfig.about_why}
          onChange={(e) => handleChange('about_why', e.target.value)}
          placeholder="Explain why you created this tool..."
          style={styles.textarea}
        />
        <p style={styles.help}>Displayed on the About page. Use line breaks for paragraphs.</p>

        <label style={styles.label}>About: How It Works</label>
        <textarea
          value={localConfig.about_how_it_works}
          onChange={(e) => handleChange('about_how_it_works', e.target.value)}
          placeholder="Explain how the monitoring works..."
          style={styles.textarea}
        />

        <label style={styles.label}>About: Privacy</label>
        <textarea
          value={localConfig.about_privacy}
          onChange={(e) => handleChange('about_privacy', e.target.value)}
          placeholder="Explain your privacy policy..."
          style={styles.textarea}
        />

        <label style={styles.label}>Supported ISPs</label>
        <div style={{ display: 'flex', gap: '10px', marginBottom: '10px' }}>
          <input
            type="text"
            value={ispInput}
            onChange={(e) => setIspInput(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), handleAddISP())}
            placeholder="Add ISP name..."
            style={{ ...styles.input, marginBottom: 0, flex: 1 }}
          />
          <button onClick={handleAddISP} style={styles.button}>
            Add
          </button>
        </div>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px', marginBottom: '15px' }}>
          {localConfig.supported_isps.map((isp) => (
            <span key={isp} style={styles.tag}>
              {isp}
              <span
                onClick={() => handleRemoveISP(isp)}
                style={{ cursor: 'pointer', opacity: 0.7 }}
              >
                ×
              </span>
            </span>
          ))}
        </div>

        <label style={styles.label}>Contact Email</label>
        <input
          type="email"
          value={localConfig.contact_email}
          onChange={(e) => handleChange('contact_email', e.target.value)}
          placeholder="contact@example.com"
          style={styles.input}
        />

        <label style={styles.label}>Footer Text</label>
        <textarea
          value={localConfig.footer_text}
          onChange={(e) => handleChange('footer_text', e.target.value)}
          placeholder="Additional text shown at the bottom of the About page..."
          style={{ ...styles.textarea, minHeight: '60px' }}
        />

        <label style={styles.label}>GitHub URL</label>
        <input
          type="url"
          value={localConfig.github_url}
          onChange={(e) => handleChange('github_url', e.target.value)}
          placeholder="https://github.com/your/repo"
          style={styles.input}
        />

        <button
          onClick={() => onSave(localConfig)}
          disabled={saving}
          style={{
            ...styles.button,
            width: '100%',
            marginTop: '10px',
            opacity: saving ? 0.6 : 1,
            cursor: saving ? 'not-allowed' : 'pointer',
          }}
        >
          {saving ? 'Saving...' : 'Save Site Configuration'}
        </button>
      </div>
    </div>
  );
}

export default SiteConfigEditor;
