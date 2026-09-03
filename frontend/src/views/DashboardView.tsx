import { useState, useEffect, useRef } from 'react';
import type { AppConfig, EnhanceDone } from '../App';
import { SaveSetup, ScoreOnly, WriteClipboard, SimulatePaste, GetConfig, ListModels } from '../../wailsjs/go/main/App';
import { WindowHide } from '../../wailsjs/runtime/runtime';

interface Props {
  cfg: AppConfig | null;
  enhanceState: { text: string; result: EnhanceDone | null; error: string | null } | null;
  onEnhance: (text: string) => void;
  onCopyPaste: (output: string, autoPaste: boolean) => void;
  onRefreshCfg: () => void;
}

export default function DashboardView({ cfg, enhanceState, onEnhance, onCopyPaste, onRefreshCfg }: Props) {
  const [input, setInput] = useState('');
  const [liveScore, setLiveScore] = useState(0);
  const [activeTab, setActiveTab] = useState<'enhance' | 'settings'>('enhance');
  const [editingCfg, setEditingCfg] = useState<AppConfig | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // When a hotkey trigger arrives, grab clipboard as the input.
  useEffect(() => {
    if (enhanceState?.result) {
      setInput(enhanceState.result.output);
    }
  }, [enhanceState?.result]);

  function handleInputChange(e: React.ChangeEvent<HTMLTextAreaElement>) {
    const val = e.target.value;
    setInput(val);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(async () => {
      if (val.trim()) {
        const s = await ScoreOnly(val);
        setLiveScore(s);
      } else {
        setLiveScore(0);
      }
    }, 300);
  }

  function handleEnhance() {
    if (!input.trim()) return;
    onEnhance(input);
  }

  async function handleCopy(autoPaste: boolean) {
    if (!enhanceState?.result?.output) return;
    await onCopyPaste(enhanceState.result.output, autoPaste ?? false);
  }

  function handleDismiss() {
    setInput('');
    setLiveScore(0);
    WindowHide();
  }

  function startEdit() {
    if (cfg) setEditingCfg({ ...cfg });
    setActiveTab('settings');
  }

  async function handleSaveEdit(values: AppConfig) {
    await SaveSetup({
      apiKey: '',
      baseUrl: values.baseUrl,
      model: values.model,
      hotkey: values.hotkey,
      autoPaste: values.autoPaste,
    });
    onRefreshCfg();
    setEditingCfg(null);
    setActiveTab('enhance');
  }

  const isLoading = enhanceState && !enhanceState.result && !enhanceState.error;
  const result = enhanceState?.result;
  const error = enhanceState?.error;

  return (
    <div className="dashboard-root">
      {/* Header */}
      <header className="dash-header">
        <span className="app-logo-sm">✦</span>
        <h2>SparkEnhance</h2>
        <div className="header-actions">
          {cfg?.hasKey && <span className="model-badge">{cfg?.model ?? ''}</span>}
          <button className="btn-icon" onClick={startEdit} title="Settings">⚙</button>
        </div>
      </header>

      {/* Tab nav */}
      <nav className="tab-nav">
        <button className={activeTab === 'enhance' ? 'active' : ''} onClick={() => setActiveTab('enhance')}>
          Enhance
        </button>
        <button className={activeTab === 'settings' ? 'active' : ''} onClick={() => setActiveTab('settings')}>
          Settings
        </button>
      </nav>

      {/* Enhance tab */}
      {activeTab === 'enhance' && (
        <div className="enhance-pane">
          <textarea
            className="enhance-input"
            placeholder="Paste or type your prompt here… Ctrl+Shift+E to grab selection"
            value={input}
            onChange={handleInputChange}
            rows={5}
          />

          {input.trim() && (
            <div className="live-score">
              Live score: <ScoreBadge score={liveScore} />
            </div>
          )}

          <div className="enhance-actions">
            <button
              className="btn-enhance"
              onClick={handleEnhance}
              disabled={isLoading || !input.trim()}
            >
              {isLoading ? '⚡ Enhancing…' : '✦ Enhance Prompt'}
            </button>
            <button className="btn-dismiss" onClick={handleDismiss}>Dismiss</button>
          </div>

          {error && (
            <div className="result-error">
              <span className="error-label">Error</span>
              <p>{error}</p>
            </div>
          )}

          {result && (
            <div className="result-card">
              <div className="result-meta">
                <ScoreDiff before={result.before} after={result.after} />
              </div>
              <textarea
                className="result-output"
                value={result.output}
                readOnly
                rows={6}
              />
              <div className="result-actions">
                <button className="btn-copy" onClick={() => handleCopy(false)}>
                  📋 Copy
                </button>
                {(cfg?.autoPaste) && (
                  <button className="btn-copy-paste" onClick={() => handleCopy(true)}>
                    📋 Copy + Paste
                  </button>
                )}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Settings tab */}
      {activeTab === 'settings' && (
        <SettingsPane
          cfg={editingCfg ?? cfg}
          onSave={handleSaveEdit}
          onCancel={() => setActiveTab('enhance')}
        />
      )}
    </div>
  );
}

function SettingsPane({ cfg, onSave, onCancel }: {
  cfg: AppConfig | null;
  onSave: (v: AppConfig) => void;
  onCancel: () => void;
}) {
  const [baseUrl, setBaseUrl] = useState(cfg?.baseUrl ?? 'https://api.gmi-serving.com/v1');
  const [model, setModel] = useState(cfg?.model ?? 'MiniMax-M3');
  const [models, setModels] = useState<string[]>(cfg?.model ? [cfg.model] : []);
  const [hotkey, setHotkey] = useState(cfg?.hotkey ?? 'ctrl+shift+e');
  const [autoPaste, setAutoPaste] = useState(cfg?.autoPaste ?? false);
  const [fetching, setFetching] = useState(false);
  const [fetchError, setFetchError] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  async function handleFetch() {
    setFetching(true);
    setFetchError('');
    try {
      // Backend uses the stored API key when the second arg is empty.
      const list = await ListModels(baseUrl.trim(), '');
      if (list.length > 0) {
        setModels(list);
        if (!list.includes(model)) setModel(list[0]);
      } else {
        setFetchError('No models returned — check the base URL');
      }
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      setFetchError(msg);
    } finally {
      setFetching(false);
    }
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError('');
    try {
      await onSave({ baseUrl, model, hotkey, autoPaste, hasKey: true });
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  return (
    <form className="settings-pane" onSubmit={handleSave}>
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
            title="Fetch available models from this base URL"
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
          Auto-paste after enhance
        </label>
      </div>
      {error && <div className="error-msg">{error}</div>}
      <div className="settings-actions">
        <button type="submit" className="btn-primary" disabled={loading}>Save</button>
        <button type="button" className="btn-secondary" onClick={onCancel}>Cancel</button>
      </div>
    </form>
  );
}

function ScoreBadge({ score }: { score: number }) {
  const cls = score >= 70 ? 'score-good' : score >= 40 ? 'score-mid' : 'score-low';
  return <span className={`score-badge ${cls}`}>{score}</span>;
}

function ScoreDiff({ before, after }: { before: number; after: number }) {
  const diff = after - before;
  const cls = diff >= 20 ? 'diff-good' : diff >= 0 ? 'diff-neutral' : 'diff-bad';
  return (
    <span className={`score-diff ${cls}`}>
      Score {before} → {after}
      {diff >= 0 ? ` +${diff}` : ` ${diff}`}
    </span>
  );
}
