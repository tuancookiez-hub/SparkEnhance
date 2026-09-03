import type { AppConfig, EnhanceDone } from '../App';

interface Props {
  phase: 'idle' | 'capturing' | 'enhancing' | 'result' | 'error';
  selection: string;
  result: EnhanceDone | null;
  errorMsg: string;
  cfg: AppConfig | null;
  onCopy: () => void;
  onDismiss: () => void;
  onSettings: () => void;
  onQuit: () => void;
}

export default function FloatingBar({ phase, result, errorMsg, cfg, onCopy, onDismiss, onSettings, onQuit }: Props) {
  if (phase === 'idle') {
    return (
      <div className="bar-idle">
        <div className="bar-idle-content">
          <span className="bar-icon">✦</span>
          <div>
            <div className="bar-title">SparkEnhance</div>
            <div className="bar-hint">
              Press <kbd>{cfg?.hotkey ?? 'ctrl+shift+e'}</kbd> to enhance any selected text
            </div>
          </div>
        </div>
        <div className="bar-actions">
          <button className="bar-btn-icon" onClick={onSettings} title="Settings">⚙</button>
          <button className="bar-btn-icon" onClick={onQuit} title="Quit">×</button>
        </div>
      </div>
    );
  }

  if (phase === 'capturing' || phase === 'enhancing') {
    return (
      <div className="bar-loading">
        <div className="bar-spinner" />
        <div>
          <div className="bar-title">{phase === 'capturing' ? 'Capturing selection…' : 'Enhancing with M3…'}</div>
          <div className="bar-hint">This usually takes 2-5 seconds</div>
        </div>
        <button className="bar-btn-icon" onClick={onDismiss}>×</button>
      </div>
    );
  }

  if (phase === 'error') {
    return (
      <div className="bar-error">
        <span className="bar-icon">⚠</span>
        <div className="bar-error-text">{errorMsg}</div>
        <button className="bar-btn" onClick={onDismiss}>Close</button>
      </div>
    );
  }

  // phase === 'result'
  return (
    <div className="bar-result">
      <div className="bar-header">
        <span className="bar-icon">✦</span>
        <div className="bar-header-text">
          <div className="bar-title">Enhanced</div>
          <div className="bar-score">
            {result && (
              <>
                <span className="bar-score-before">{result.before}</span>
                <span className="bar-score-arrow">→</span>
                <span className="bar-score-after">{result.after}</span>
              </>
            )}
          </div>
        </div>
        <button className="bar-btn-icon" onClick={onDismiss} title="Dismiss">×</button>
      </div>
      <textarea
        className="bar-output"
        value={result?.output ?? ''}
        readOnly
        rows={4}
        onClick={(e) => (e.target as HTMLTextAreaElement).select()}
      />
      <div className="bar-actions">
        <button className="bar-btn-primary" onClick={onCopy}>
          {cfg?.autoPaste ? '📋 Replace' : '📋 Copy'}
        </button>
        <button className="bar-btn" onClick={onDismiss}>Dismiss</button>
      </div>
    </div>
  );
}
