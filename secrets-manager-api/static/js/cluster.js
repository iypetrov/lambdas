document.addEventListener("alpine:init", () => {
  Alpine.store("cluster", {
    selectedCluster: null,

    init() {
      const saved = localStorage.getItem("selectedCluster");
      if (saved) {
        this.selectedCluster = saved;
      }
    },

    setCluster(cluster) {
      this.selectedCluster = cluster;
      localStorage.setItem("selectedCluster", cluster);
      htmx.trigger("body", "refresh");
      htmx.trigger("body", "refreshStaticSecretsTable");
      htmx.trigger("body", "refreshTLSCertificatesTable");
    },

    getCluster() {
      return this.selectedCluster || "";
    },
  });

  Alpine.data("clusterDropdown", (env) => ({
    init() {
      this.$watch("$store.cluster.selectedCluster", (value) => {
        const select = this.$el.querySelector("select");
        if (select && value) {
          select.value = value;
        }
      });

      this.$el.addEventListener("htmx:after-swap", (event) => {
        const select = event.target;
        if (select.tagName === "SELECT") {
          const currentCluster = this.$store.cluster.getCluster();
          if (currentCluster && select.querySelector(`option[value="${currentCluster}"]`)) {
            select.value = currentCluster;
          } else if (select.options.length > 0 && select.options[0].value) {
            const firstCluster = select.options[0].value;
            if (firstCluster) {
              this.$store.cluster.setCluster(firstCluster);
              select.value = firstCluster;
            }
          }
        }
      });
    },

    handleChange(event) {
      const cluster = event.target.value;
      if (cluster) {
        this.$store.cluster.setCluster(cluster);
      }
    },
  }));
});
