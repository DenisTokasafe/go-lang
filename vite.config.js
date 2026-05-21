import { defineConfig } from "vite";
import tailwindcss from "@tailwindcss/vite";
import path from "path";

export default defineConfig({
  plugins: [tailwindcss()],

  // SANGAT PENTING: Base harus diatur ke '/dist/' agar import dinamis (picker, charts)
  // mencari ke path yang benar di server Golang.
  base: "/dist/",

  publicDir: false,

  build: {
    outDir: "public/dist",
    chunkSizeWarningLimit: 2000,
    emptyOutDir: true,

    // Aktifkan manifest jika Anda ingin Golang membaca nama file aslinya
    manifest: true,

    rollupOptions: {
      input: {
        style: path.resolve(__dirname, "assets/css/styles.css"),
        main: path.resolve(__dirname, "assets/js/main.js"),
      },
      output: {
        // Menghapus hash agar nama file tetap main.js dan style.css
        entryFileNames: `[name].js`,
        chunkFileNames: `assets/[name]-[hash].js`, // Modul picker dll tetap pakai hash untuk cache
        assetFileNames: `[name].[ext]`,
      },
    },
  },
});
