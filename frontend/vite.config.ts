import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 9245,
    strictPort: true,
  },
  build: {
    outDir: "dist",
    target: "es2020",
  },
});
