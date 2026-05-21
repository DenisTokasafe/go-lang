import Toastify from "toastify-js";
import "toastify-js/src/toastify.css";

export function showToast(message, type = "success") {
  const style = getComputedStyle(document.body);
  let bgColor;

  switch (type) {
    case "error":
      bgColor = `oklch(${style.getPropertyValue("--er")})`;
      break;
    case "info":
      bgColor = `oklch(${style.getPropertyValue("--in")})`;
      break;
    case "warning":
      bgColor = `oklch(${style.getPropertyValue("--wa")})`;
      break;
    default:
      bgColor = `oklch(${style.getPropertyValue("--su")})`;
      break;
  }

  Toastify({
    text: message,
    duration: 3000,
    close: true,
    gravity: "top",
    position: "right",
    stopOnFocus: true,
    style: {
      background: bgColor,
      color: `oklch(${style.getPropertyValue("--nc")})`,
      borderRadius: "12px",
      fontSize: "13px",
      fontWeight: "500",
      boxShadow: "0 10px 15px -3px rgba(0, 0, 0, 0.1)",
    },
  }).showToast();
}
