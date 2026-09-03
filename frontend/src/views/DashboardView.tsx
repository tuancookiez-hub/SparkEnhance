import { useState, useEffect, useRef } from 'react';
import type { AppConfig, EnhanceDone } from '../App';
import { SaveSetup, ScoreOnly, WriteClipboard, SimulatePaste, GetConfig } from '../../wailsjs/go/main/App';
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
                <span className="prev-clip-hint">
                  {result.prevClip ? 'Previous clipboard restored' : ''}
                </span>
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
  const [hotkey, setHotkey] = useState(cfg?.hotkey ?? 'ctrl+shift+e');
  const [autoPaste, setAutoPaste] = useState(cfg?.autoPaste ?? false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

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
        <select value={model} onChange={e => setModel(e.target.value)}>
          <option value="MiniMax-M3">MiniMax-M3</option>
          <option value="MiniMax-M3.5-Speculative">MiniMax-M3.5-Speculative</option>
          <option value="gemini-2.0-flash-thinking-exp-01-21">Gemini Thinking</option>
        </select>
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
