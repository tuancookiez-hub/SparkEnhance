import { useState } from 'react';
import { SaveSetup, ValidateKey } from '../../wailsjs/go/main/App';

interface Props {
  onSubmit: (values: { apiKey: string; baseUrl: string; model: string; hotkey: string; autoPaste: boolean }) => Promise<void>;
}

const MODELS = [
  { value: 'MiniMax-M3', label: 'MiniMax-M3 (fast, general)' },
  { value: 'MiniMax-M3.5-Speculative', label: 'MiniMax-M3.5-Speculative (higher quality)' },
  { value: 'gemini-2.0-flash-thinking-exp-01-21', label: 'Gemini Thinking (reasoning)' },
];

export default function SetupView({ onSubmit }: Props) {
  const [apiKey, setApiKey] = useState('');
  const [baseUrl, setBaseUrl] = useState('https://api.gmi-serving.com/v1');
  const [model, setModel] = useState('MiniMax-M3');
  const [hotkey, setHotkey] = useState('ctrl+shift+e');
  const [autoPaste, setAutoPaste] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [validated, setValidated] = useState(false);

  async function handleValidate() {
    if (!apiKey.trim()) { setError('API key is required'); return; }
    setLoading(true);
    setError('');
    // Save temporarily so ValidateKey has a key to check
    await SaveSetup({ apiKey, baseUrl, model, hotkey, autoPaste });
    try {
      await ValidateKey();
      setValidated(true);
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      setError(msg.includes('401') || msg.includes('403') || msg.includes('key')
        ? 'Invalid API key. Check your GMI Cloud key.'
        : `Validation failed: ${msg}`);
    } finally {
      setLoading(false);
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError('');
    try {
      await onSubmit({ apiKey, baseUrl, model, hotkey, autoPaste });
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="setup-root">
      <header className="setup-header">
        <div className="app-logo">✦</div>
        <h1>SparkEnhance</h1>
        <p className="subtitle">AI prompt enhancement for Malay and English writers</p>
      </header>

      <form className="setup-form" onSubmit={handleSubmit}>
        <div className="field">
          <label>GMI Cloud Base URL</label>
          <input
            type="url"
            value={baseUrl}
            onChange={e => setBaseUrl(e.target.value)}
            placeholder="https://api.gmi-serving.com/v1"
            required
          />
        </div>

        <div className="field">
          <label>API Key</label>
          <input
            type="password"
            value={apiKey}
            onChange={e => { setApiKey(e.target.value); setValidated(false); }}
            placeholder="sk-…"
            required
          />
          <button
            type="button"
            className="btn-validate"
            onClick={handleValidate}
            disabled={loading || !apiKey.trim()}
          >
            {loading ? 'Validating…' : validated ? '✓ Valid' : 'Validate'}
          </button>
        </div>

        <div className="field">
          <label>Model</label>
          <select value={model} onChange={e => setModel(e.target.value)}>
            {MODELS.map(m => (
              <option key={m.value} value={m.value}>{m.label}</option>
            ))}
          </select>
        </div>

        <div className="field">
          <label>Global Hotkey</label>
          <input
            type="text"
            value={hotkey}
            onChange={e => setHotkey(e.target.value)}
            placeholder="ctrl+shift+e"
          />
          <span className="hint">e.g. ctrl+shift+e · alt+shift+p</span>
        </div>

        <div className="field-check">
          <label>
            <input
              type="checkbox"
              checked={autoPaste}
              onChange={e => setAutoPaste(e.target.checked)}
            />
            Auto-paste enhanced text after enhancement
          </label>
        </div>

        {error && <div className="error-msg">{error}</div>}

        <button type="submit" className="btn-primary" disabled={loading}>
          {loading ? 'Saving…' : 'Save & Start'}
        </button>
      </form>
    </div>
  );
}
