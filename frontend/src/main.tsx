import React from "react";
import { createRoot } from "react-dom/client";
import { BarScreen, SettingsScreen } from "./App";
import "./style.css";

function Router() {
  const path = window.location.pathname;
  if (path.includes("settings")) {
    return <SettingsScreen />;
  }
  return <BarScreen />;
}

const root = createRoot(document.getElementById("root")!);
root.render(<Router />);
