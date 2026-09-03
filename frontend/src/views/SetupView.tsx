import { useState } from 'react';
import { SaveSetup, ValidateKey, ListModels } from '../../wailsjs/go/main/App';

interface Props {
  onSubmit: (values: { apiKey: string; baseUrl: string; model: string; hotkey: string; autoPaste: boolean }) => Promise<void>;
}

export default function SetupView({ onSubmit }: Props) {
  const [apiKey, setApiKey] = useState('');
  const [baseUrl, setBaseUrl] = useState('https://api.gmi-serving.com/v1');
  const [models, setModels] = useState<string[]>([]);
  const [model, setModel] = useState('');
  const [hotkey, setHotkey] = useState('ctrl+shift+e');
  const [autoPaste, setAutoPaste] = useState(false);
  const [loading, setLoading] = useState(false);
  const [fetching, setFetching] = useState(false);
  const [error, setError] = useState('');
  const [fetchError, setFetchError] = useState('');

  async function handleFetchModels() {
    if (!apiKey.trim() || !baseUrl.trim()) {
      setFetchError('Enter base URL and API key first');
      return;
    }
    setFetching(true);
    setFetchError('');
    setModel('');
    try {
      const list = await ListModels(baseUrl.trim(), apiKey.trim());
      if (list.length === 0) {
        setFetchError('No models returned — check the base URL');
      } else {
        setModels(list);
        setModel(list[0]);
      }
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      setFetchError(msg);
    } finally {
      setFetching(false);
    }
  }

  async function handleValidate() {
    if (!apiKey.trim()) { setError('API key is required'); return; }
    if (!model) { setError('Select a model first'); return; }
    setLoading(true);
    setError('');
    await SaveSetup({ apiKey, baseUrl, model, hotkey, autoPaste });
    try {
      await ValidateKey();
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      setError(msg.includes('401') || msg.includes('403') || msg.includes('key')
        ? 'Invalid API key. Check your GMI Cloud key.'
        : `Validation failed: ${msg}`);
      setLoading(false);
      return;
    }
    setLoading(false);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!model) { setError('Fetch and select a model first'); return; }
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
        <p className="subtitle">AI prompt enhancement — Ctrl+Shift+E · any app</p>
      </header>

      <form className="setup-form" onSubmit={handleSubmit}>
        <div className="field">
          <label>Base URL</label>
          <input
            type="url"
            value={baseUrl}
            onChange={e => { setBaseUrl(e.target.value); setModels([]); setModel(''); }}
            placeholder="https://api.gmi-serving.com/v1"
            required
          />
        </div>

        <div className="field">
          <label>API Key</label>
          <input
            type="password"
            value={apiKey}
            onChange={e => { setApiKey(e.target.value); setModels([]); setModel(''); }}
            placeholder="sk-…"
            required
          />
        </div>

        <div className="field">
          <label>Model</label>
          <div className="model-row">
            {models.length > 0 ? (
              <select
                className="model-select"
                value={model}
                onChange={e => setModel(e.target.value)}
              >
                {models.map(m => (
                  <option key={m} value={m}>{m}</option>
                ))}
              </select>
            ) : (
              <div className="model-placeholder">
                Fetch models to see available options
              </div>
            )}
            <button
              type="button"
              className={`btn-fetch ${fetching ? 'loading' : ''}`}
              onClick={handleFetchModels}
              disabled={fetching || !apiKey.trim() || !baseUrl.trim()}
              title="Fetch available models from this base URL"
            >
              {fetching ? '↻' : '⊕'}
            </button>
          </div>
          {fetchError && <span className="fetch-error">{fetchError}</span>}
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

        <button
          type="button"
          className="btn-primary"
          onClick={handleValidate}
          disabled={loading || !model}
        >
          {loading ? 'Validating…' : '✓ Validate & Save'}
        </button>
      </form>
    </div>
  );
}
