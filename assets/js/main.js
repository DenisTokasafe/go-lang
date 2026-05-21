import Alpine from "alpinejs";

// 1. Toastify (Lazy Loaded)
window.showToast = async (message, type = "success") => {
  const { showToast } = await import("./modules/notifications");
  showToast(message, type);
};

// 2. Datepicker (Lazy Loaded)
window.setupDatepicker = async (config = {}) => {
  const { setupDatepicker } = await import("./modules/pickers");
  return setupDatepicker(config);
};
// Daftarkan fungsi ke window SEBELUM Alpine.start()
window.initMonthPicker = async (el, initialValue, onSelect) => {
  // Import fungsi baru dari modules/pickers
  const { initMonthPicker } = await import("./modules/pickers");
  return initMonthPicker(el, initialValue, onSelect);
};

window.initRangePicker = async (el, config) => {
  const { initRangePicker } = await import("./modules/pickers");
  return initRangePicker(el, config);
};

// 3. File Input (Langsung dimuat karena sangat ringan)
window.setupFileInput = function () {
  return {
    errorMessage: "",
    isInvalid: false,
    validateFile(e) {
      const file = e.target.files[0];
      this.errorMessage = "";
      this.isInvalid = false;
      if (!file) return;
      const maxSize = 2 * 1024 * 1024;
      if (file.size > maxSize) {
        this.errorMessage = "Ukuran file maksimal adalah 2MB.";
        this.isInvalid = true;
        e.target.value = "";
        return;
      }
      const allowedTypes = ["image/png", "application/pdf"];
      if (!allowedTypes.includes(file.type)) {
        this.errorMessage = "Format file harus PNG atau PDF.";
        this.isInvalid = true;
        e.target.value = "";
        return;
      }
      this.isInvalid = false;
    },
  };
};

// 4. Global Library Registration (Lazy Loaded saat dipanggil di window)
// Jika Anda memanggil window.echarts di script inline, ini akan memuatnya secara on-demand
window.getEchartsInstance = async () => {
  const module = await import("./modules/charts");
  return await module.getEcharts();
};
window.initCKEditor = async (el, config = {}) => {
  // Memicu lazy load melalui getter yang sudah Anda buat
  const ClassicEditor = await window.ClassicEditor;

  const defaultConfig = {
    toolbar: [
      "heading",
      "|",
      "bold",
      "italic",
      "link",
      "bulletedList",
      "numberedList",
      "blockQuote",
    ],
  };

  return ClassicEditor.create(el, { ...defaultConfig, ...config });
};

Object.defineProperty(window, "ClassicEditor", {
  get: async () => {
    const { default: ClassicEditor } =
      await import("@ckeditor/ckeditor5-build-classic");
    return ClassicEditor;
  },
});

window.Alpine = Alpine;
Alpine.start();
