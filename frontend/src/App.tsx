import { useState, useEffect } from 'react';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { IsConfigured, GetConfig, SaveSetup, EnhanceText, ValidateKey, WriteClipboard, SimulatePaste, ReadClipboard, ScoreOnly } from '../wailsjs/go/main/App';
import SetupView from './views/SetupView';
import DashboardView from './views/DashboardView';

export interface AppConfig {
  baseUrl: string;
  model: string;
  hotkey: string;
  autoPaste: boolean;
  hasKey: boolean;
}

export interface EnhanceDone {
  selection: string;
  output: string;
  before: number;
  after: number;
  prevClip: string;
}

export default function App() {
  const [configured, setConfigured] = useState<boolean | null>(null); // null = loading
  const [cfg, setCfg] = useState<AppConfig | null>(null);
  const [enhanceState, setEnhanceState] = useState<{ text: string; result: EnhanceDone | null; error: string | null } | null>(null);

  useEffect(() => {
    IsConfigured().then(setConfigured);

    // Listen for hotkey results from the backend.
    EventsOn('enhance:start', (text: string) => {
      setEnhanceState({ text, result: null, error: null });
    });
    EventsOn('enhance:done', (data: EnhanceDone) => {
      setEnhanceState(prev => prev ? { ...prev, result: data, error: null } : null);
    });
    EventsOn('enhance:error', (msg: string) => {
      setEnhanceState(prev => prev ? { ...prev, error: msg } : null);
    });
  }, []);

  async function handleSetup(values: { apiKey: string; baseUrl: string; model: string; hotkey: string; autoPaste: boolean }) {
    await SaveSetup(values);
    const cfg = await GetConfig();
    setCfg(cfg as AppConfig);
    setConfigured(true);
  }

  async function handleEnhance(text: string) {
    try {
      const res = await EnhanceText({ text });
      const result: EnhanceDone = {
        selection: text,
        output: res.output,
        before: await ScoreOnly(text),
        after: res.score,
        prevClip: '',
      };
      setEnhanceState({ text, result, error: null });
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      setEnhanceState(prev => prev ? { ...prev, error: msg } : null);
    }
  }

  async function handleCopyAndPaste(output: string, autoPaste: boolean) {
    await WriteClipboard(output);
    if (autoPaste) {
      await SimulatePaste(output);
    }
  }

  if (configured === null) return null;

  return (
    <div className="app-root">
      {!configured
        ? <SetupView onSubmit={handleSetup} />
        : <DashboardView
            cfg={cfg}
            enhanceState={enhanceState}
            onEnhance={handleEnhance}
            onCopyPaste={handleCopyAndPaste}
            onRefreshCfg={async () => { const c = await GetConfig(); setCfg(c as AppConfig); }}
          />
      }
    </div>
  );
}
