document.addEventListener("alpine:init", () => {
  Alpine.data("toast", () => ({
    toasts: [],

    add(event) {
      this.toasts.push({
        id: `toast-${Math.random().toString(16).slice(2)}`,
        message: event.detail.message,
        statusCode: event.detail.statusCode,
        show: false,
      });
    },

    show(id) {
      const toast = this.toasts.find((toast) => toast.id === id);
      const index = this.toasts.findIndex((toast) => toast.id === id);
      this.toasts.splice(index, 1, { ...toast, show: true });
    },

    remove(id) {
      const toast = this.toasts.find((toast) => toast.id === id);
      const index = this.toasts.findIndex((toast) => toast.id === id);
      this.toasts.splice(index, 1, { ...toast, show: false });
    },

    dismiss(id) {
      let that = this;
      setTimeout(function () {
        that.remove(id);
      }, 50);
    },

    toastInit(el) {
      const id = el.getAttribute("id");
      let that = this;
      setTimeout(function () {
        that.show(id);
      }, 50);
      setTimeout(function () {
        that.remove(id);
      }, 5050);
    },

    globalInit() {
      window.toast = function (message, statusCode = 200) {
        window.dispatchEvent(
          new CustomEvent("add-toast", {
            detail: {
              message: message,
              statusCode: statusCode,
            },
          }),
        );
      };
    },
  }));
});
