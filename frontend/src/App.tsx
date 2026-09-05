// SparkEnhance — React frontend (Wails v3).
// Talks to the Go side via window.go (auto-injected by wails dev server).
import { useState, useEffect } from "react";
import { Events } from "@wails/runtime";

export function BarScreen() {
  const [state, setState] = useState<"idle" | "loading" | "result" | "error">("idle");
  const [sel, setSel] = useState("");
  const [result, setResult] = useState("");
  const [errMsg, setErrMsg] = useState("");

  useEffect(() => {
    const off = [
      Events.On("enhance:start", (data: any) => {
        setSel(data.selection ?? "");
        setState("loading");
        Events.Emit("bar:user-activity");
      }),
      Events.On("enhance:result", (data: any) => {
        setResult(data.result ?? "");
        setState("result");
        Events.Emit("bar:user-activity");
      }),
      Events.On("enhance:error", (data: any) => {
        setErrMsg(data.error ?? "Unknown error");
        setState("error");
        Events.Emit("bar:user-activity");
      }),
      Events.On("enhance:idle", () => setState("idle")),
    ];
    return () => off.forEach(f => f());
  }, []);

  if (state === "loading") {
    return (
      <div className="screen">
        <div className="bar">
          <span className="spinner" />
          <span className="label">{sel || "Enhancing…"}</span>
        </div>
      </div>
    );
  }

  if (state === "error") {
    return (
      <div className="screen">
        <div className="bar">
          <span className="label error">{errMsg}</span>
          <div className="actions">
            <button className="btn" onClick={() => setState("idle")}>Dismiss</button>
          </div>
        </div>
      </div>
    );
  }

  if (state === "result") {
    return (
      <div className="screen">
        <div className="bar">
          <span className="label">
            <div className="result">{result}</div>
          </span>
          <div className="actions">
            <button className="btn primary" onClick={copyResult}>Copy</button>
            <button className="btn danger" onClick={dismiss}>Close</button>
          </div>
        </div>
      </div>
    );
  }

  // idle
  return (
    <div className="screen">
      <div className="bar">
        <span className="label">Hover any selected text, press Ctrl+Shift+E</span>
      </div>
    </div>
  );

  async function copyResult() {
    try {
      await navigator.clipboard.writeText(result);
    } catch {}
  }
  function dismiss() {
    setState("idle");
    Events.Emit("bar:user-activity");
  }
}

export function SettingsScreen() {
  const [apiKey, setApiKey] = useState("");
  const [baseURL, setBaseURL] = useState("https://api.gmi-serving.com/v1");
  const [model, setModel] = useState("MiniMaxAI/MiniMax-M3");
  const [status, setStatus] = useState("");
  const [statusKind, setStatusKind] = useState<"ok" | "err" | "">("");

  async function save() {
    try {
      // @ts-ignore — window.go is auto-injected
      await window.go.config.SaveConfig({ apiKey, baseURL, model });
      setStatus("Saved.");
      setStatusKind("ok");
    } catch (e: any) {
      setStatus("Save failed: " + (e?.message || String(e)));
      setStatusKind("err");
    }
  }

  return (
    <div className="settings">
      <h1>SparkEnhance</h1>
      <p className="hint">
        Hover-enhance any selected text using MiniMax M3 on GMI Cloud. Your API key
        is stored locally in <code>%APPDATA%\SparkEnhance\config.json</code> and is
        never sent to anywhere except <code>{baseURL}</code>.
      </p>

      <div className="row">
        <label>API Key</label>
        <input
          type="password"
          value={apiKey}
          onChange={e => setApiKey(e.target.value)}
          placeholder="sk-…"
        />
      </div>

      <div className="row">
        <label>Base URL</label>
        <input value={baseURL} onChange={e => setBaseURL(e.target.value)} />
      </div>

      <div className="row">
        <label>Model</label>
        <input value={model} onChange={e => setModel(e.target.value)} />
      </div>

      <div className="footer">
        {status && <span className={"status " + statusKind}>{status}</span>}
        <span className="spacer" />
        <button className="btn primary" onClick={save}>Save</button>
      </div>
    </div>
  );
}
