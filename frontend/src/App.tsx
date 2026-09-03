import { useState, useEffect } from 'react';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { IsConfigured, GetConfig, SaveSetup, CopyOutput, DismissBar, QuitApp, ValidateKey, ListModels, ScoreOnly } from '../wailsjs/go/main/App';
import FloatingBar from './views/FloatingBar';
import SettingsView from './views/SettingsView';
import SetupView from './views/SetupView';
import './style.css';

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
}

type Phase = 'idle' | 'capturing' | 'enhancing' | 'result' | 'error';

export default function App() {
  const [configured, setConfigured] = useState<boolean | null>(null);
  const [cfg, setCfg] = useState<AppConfig | null>(null);
  const [phase, setPhase] = useState<Phase>('idle');
  const [selection, setSelection] = useState('');
  const [result, setResult] = useState<EnhanceDone | null>(null);
  const [errorMsg, setErrorMsg] = useState('');
  const [showSettings, setShowSettings] = useState(false);

  useEffect(() => {
    IsConfigured().then(setConfigured);
    GetConfig().then(c => setCfg(c as AppConfig));

    // The backend drives phase changes via events after the hotkey fires.
    EventsOn('enhance:start', (text: string) => {
      setSelection(text);
      setResult(null);
      setErrorMsg('');
      setPhase('enhancing');
    });
    EventsOn('enhance:done', (data: EnhanceDone) => {
      setResult(data);
      setPhase('result');
    });
    EventsOn('enhance:error', (msg: string) => {
      setErrorMsg(msg);
      setPhase('error');
    });
  }, []);

  async function handleSetup(values: { apiKey: string; baseUrl: string; model: string; hotkey: string; autoPaste: boolean }) {
    await SaveSetup(values);
    setCfg(await GetConfig() as AppConfig);
    setConfigured(true);
  }

  function handleCopyAndDismiss() {
    if (!result) return;
    CopyOutput(result.output);
    setPhase('idle');
  }

  function handleDismiss() {
    DismissBar();
    setPhase('idle');
  }

  function handleQuit() {
    QuitApp();
  }

  if (configured === null) return null;
  if (!configured) {
    return <SetupView onSubmit={handleSetup} />;
  }

  if (showSettings) {
    return (
      <SettingsView
        cfg={cfg}
        onSaved={async () => { setCfg(await GetConfig() as AppConfig); setShowSettings(false); }}
        onBack={() => setShowSettings(false)}
      />
    );
  }

  return (
    <div className="app-root floating">
      <FloatingBar
        phase={phase}
        selection={selection}
        result={result}
        errorMsg={errorMsg}
        cfg={cfg}
        onCopy={handleCopyAndDismiss}
        onDismiss={handleDismiss}
        onSettings={() => setShowSettings(true)}
        onQuit={handleQuit}
      />
    </div>
  );
}

