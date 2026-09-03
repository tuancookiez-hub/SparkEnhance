import { useState, useEffect } from 'react';
import type { AppConfig } from '../App';
import { GetConfig, SaveSetup, ListModels } from '../../wailsjs/go/main/App';

interface Props {
  cfg: AppConfig | null;
  onSaved: () => void;
  onBack: () => void;
}

export default function SettingsView({ cfg, onSaved, onBack }: Props) {
  const [baseUrl, setBaseUrl] = useState(cfg?.baseUrl ?? 'https://api.gmi-serving.com/v1');
  const [model, setModel] = useState(cfg?.model ?? 'MiniMax-M3');
  const [models, setModels] = useState<string[]>(cfg?.model ? [cfg.model] : []);
  const [hotkey, setHotkey] = useState(cfg?.hotkey ?? 'ctrl+shift+e');
  const [autoPaste, setAutoPaste] = useState(cfg?.autoPaste ?? false);
  const [fetching, setFetching] = useState(false);
  const [fetchError, setFetchError] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    setBaseUrl(cfg?.baseUrl ?? 'https://api.gmi-serving.com/v1');
    setModel(cfg?.model ?? 'MiniMax-M3');
    setHotkey(cfg?.hotkey ?? 'ctrl+shift+e');
    setAutoPaste(cfg?.autoPaste ?? false);
  }, [cfg]);

  async function handleFetch() {
    setFetching(true);
    setFetchError('');
    try {
      const list = await ListModels(baseUrl.trim(), '');
      if (list.length > 0) {
        setModels(list);
        if (!list.includes(model)) setModel(list[0]);
      } else {
        setFetchError('No models returned — check the base URL');
      }
    } catch (e: unknown) {
      setFetchError(e instanceof Error ? e.message : String(e));
    } finally {
      setFetching(false);
    }
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setError('');
    try {
      await SaveSetup({
        apiKey: '',
        baseUrl,
        model,
        hotkey,
        autoPaste,
      });
      onSaved();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="settings-root">
      <header className="setup-header">
        <div className="app-logo">⚙</div>
        <h1>Settings</h1>
        <p className="subtitle">SparkEnhance — config</p>
      </header>

      <form className="setup-form" onSubmit={handleSave}>
        <div className="field">
          <label>Base URL</label>
          <input type="url" value={baseUrl} onChange={e => setBaseUrl(e.target.value)} required />
        </div>
        <div className="field">
          <label>Model</label>
          <div className="model-row">
            <select className="model-select" value={model} onChange={e => setModel(e.target.value)}>
              {models.length === 0 ? (
                <option value={model}>{model}</option>
              ) : models.map(m => (
                <option key={m} value={m}>{m}</option>
              ))}
            </select>
            <button
              type="button"
              className={`btn-fetch ${fetching ? 'loading' : ''}`}
              onClick={handleFetch}
              disabled={fetching}
              title="Fetch available models"
            >
              {fetching ? '↻' : '⊕'}
            </button>
          </div>
          {fetchError && <span className="fetch-error">{fetchError}</span>}
        </div>
        <div className="field">
          <label>Hotkey</label>
          <input type="text" value={hotkey} onChange={e => setHotkey(e.target.value)} />
        </div>
        <div className="field-check">
          <label>
            <input type="checkbox" checked={autoPaste} onChange={e => setAutoPaste(e.target.checked)} />
            Auto-paste enhanced text at the original selection
          </label>
        </div>
        {error && <div className="error-msg">{error}</div>}
        <div style={{ display: 'flex', gap: 10 }}>
          <button type="submit" className="btn-primary" disabled={saving} style={{ flex: 1 }}>
            {saving ? 'Saving…' : 'Save'}
          </button>
          <button type="button" className="btn-secondary" onClick={onBack}>Back</button>
        </div>
      </form>
    </div>
  );
}
