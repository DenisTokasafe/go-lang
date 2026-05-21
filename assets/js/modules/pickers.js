import flatpickr from "flatpickr";
import "flatpickr/dist/flatpickr.css"; // WAJIB ADA
import "flatpickr/dist/plugins/monthSelect/style.css";
import monthSelectPlugin from "flatpickr/dist/plugins/monthSelect";
import "flatpickr/dist/plugins/monthSelect/style.css";

/**
 * Helper untuk deteksi device mobile
 */
const isMobileDevice = () =>
  /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(
    navigator.userAgent,
  );

// ==========================================
// DIRECT INITIALIZATION FUNCTIONS
// (Digunakan oleh window.init... di main.js)
// ==========================================

/**
 * Inisialisasi Flatpickr Month langsung ke Elemen
 */
export function initMonthPicker(el, initialValue, onSelect) {
  return flatpickr(el, {
    disableMobile: true,
    plugins: [
      new monthSelectPlugin({
        shorthand: true,
        dateFormat: "Y-m-d",
        altFormat: "F Y",
        theme: "light",
      }),
    ],
    altInput: true,
    altInputClass:
      "input input-bordered input-xs w-full focus:outline-none focus:border-info focus:ring-info",
    static: true,
    defaultDate: initialValue || null,
    onChange: (selectedDates, dateStr) => {
      if (onSelect) onSelect(dateStr);
    },
  });
}

/**
 * Inisialisasi Flatpickr Range langsung ke Elemen
 */
export function initRangePicker(el, config = {}) {
  return flatpickr(el, {
    mode: "range",
    dateFormat: "Y-m-d",
    altInput: true,
    altFormat: "d M Y",
    altInputClass:
      "input input-bordered input-xs w-full max-w-sm focus:border-info focus:ring-info",
    ...config,
  });
}

// ==========================================
// ALPINE COMPONENT WRAPPERS
// (Untuk penggunaan x-data="setup...")
// ==========================================

/**
 * Standard DateTime Picker
 */
export function setupDatepicker(config = {}) {
  return {
    value: "",
    init() {
      this.$nextTick(() => {
        const el = this.$refs.datepicker;
        if (!el) return;

        if (isMobileDevice()) {
          el.type = "datetime-local";
          el.removeAttribute("readonly");
          el.addEventListener("input", (e) => {
            this.value = e.target.value.replace("T", " ");
          });
          return;
        }

        this._fp = flatpickr(el, {
          enableTime: true,
          altInput: true,
          altFormat: "j F, Y H:i",
          dateFormat: "Y-m-d H:i",
          time_24hr: true,
          disableMobile: true,
          ...config,
          onChange: (selectedDates, dateStr) => {
            this.value = dateStr;
          },
        });

        this.$watch("value", (val) => {
          if (this._fp && val !== this._fp.input.value) {
            this._fp.setDate(val, false);
          }
        });
      });
    },
  };
}

/**
 * Month Picker
 */
export function setupMonthPicker(initialValue = "") {
  return {
    month: initialValue,
    init() {
      this.$nextTick(() => {
        const el = this.$refs.monthpicker;
        if (!el) return;
        this._fp = initMonthPicker(el, this.month, (dateStr) => {
          this.month = dateStr;
          if (typeof this.clearInvalid === "function")
            this.clearInvalid("month");
        });
      });
    },
  };
}

/**
 * Range Picker
 */
export function setupRangePicker(startVal = "", endVal = "") {
  return {
    startDate: startVal,
    endDate: endVal,
    init() {
      this.$nextTick(() => {
        const el = this.$refs.rangepicker;
        if (!el) return;

        this._fp = initRangePicker(el, {
          defaultDate:
            this.startDate && this.endDate
              ? [this.startDate, this.endDate]
              : null,
          onClose: (selectedDates, dateStr, instance) => {
            if (selectedDates.length === 2) {
              this.startDate = instance.formatDate(selectedDates[0], "Y-m-d");
              this.endDate = instance.formatDate(selectedDates[1], "Y-m-d");
              if (typeof this.doSearch === "function") this.doSearch();
            }
          },
        });
      });
    },
  };
}
