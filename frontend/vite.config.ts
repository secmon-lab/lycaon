import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";

function preserveGitkeep() {
  return {
    name: "preserve-gitkeep",
    generateBundle() {
      this.emitFile({
        type: "asset",
        fileName: ".gitkeep",
        source: "",
      });
    },
  };
}

export default defineConfig({
  plugins: [react(), preserveGitkeep()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    allowedHosts: [
      ".ngrok-free.app",
      ".ts.net"
    ],
    proxy: {
      "/api": "http://localhost:8080",
      "/hooks": "http://localhost:8080",
      "/ws": {
        target: "http://localhost:8080",
        ws: true,
        changeOrigin: true,
        rewrite: (path) => path,
      },
    },
  },
  build: {
    outDir: "dist",
    assetsDir: "assets",
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ["react", "react-dom"],
          ui: [
            "@radix-ui/react-avatar",
            "@radix-ui/react-dialog",
            "@radix-ui/react-dropdown-menu",
            "@radix-ui/react-tabs",
          ],
        },
      },
    },
  },
});