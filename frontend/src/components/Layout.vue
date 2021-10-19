<template>
  <div id="app">
    <Navbar @clickSidebar="open"></Navbar>
    <div class="wrapper" v-if="$root.isLoaded">
      <Sidebar :openSidebar="sidebar" />
      <div class="main">
        <div class="global-notices" v-if="serverConfig.needs_restart || serverConfig.update">
          <div v-if="serverConfig.needs_restart" class="notification is-danger">
            {{ $t("settings.needsRestart") }}
            &mdash;
            <b-button
              class="is-primary"
              size="is-small"
              @click="$utils.confirm($t('settings.confirmRestart'), reloadApp)"
            >
              {{ $t("settings.restart") }}
            </b-button>
          </div>
          <div v-if="serverConfig.update" class="notification is-success">
            {{ $t("settings.updateAvailable", { version: serverConfig.update.version }) }}
            <a :href="serverConfig.update.url" target="_blank">View</a>
          </div>
        </div>

        <slot />

        <!-- :key="$route.fullPath" -->

        <b-loading v-if="!$root.isLoaded" active />
      </div>
    </div>
  </div>
</template>

<script>
import { mapState } from "vuex";
import Navbar from "../components/Navbar.vue";
export default {
  name: "Layout",

  components: {
    Navbar,
    Sidebar: () => import("../components/Sidebar.vue")
  },

  data() {
    return {
      activeItem: {},
      activeGroup: {},
      sidebar: false
    };
  },

  watch: {
    $route(to) {
      // Set the current route name to true for active+expanded keys in the
      // menu to pick up.
      this.activeItem = { [to.name]: true };
      if (to.meta.group) {
        this.activeGroup = { [to.meta.group]: true };
      } else {
        // Reset activeGroup to collapse menu items on navigating
        // to non group items from sidebar
        this.activeGroup = {};
      }
    }
  },

  methods: {
    toggleGroup(group, state) {
      this.activeGroup = state ? { [group]: true } : {};
    },
    open() {
      this.sidebar = !this.sidebar;
    },
    reloadApp() {
      this.$api.reloadApp().then(() => {
        this.$utils.toast("Reloading app ...");

        // Poll until there's a 200 response, waiting for the app
        // to restart and come back up.
        const pollId = setInterval(() => {
          this.$api.getHealth().then(() => {
            clearInterval(pollId);
            document.location.reload();
          });
        }, 500);
      });
    }
  },

  computed: {
    ...mapState(["serverConfig"]),

    version() {
      return process.env.VUE_APP_VERSION;
    }
  },

  mounted() {
    // Lists is required across different views. On app load, fetch the lists
    // and have them in the store.
    if (localStorage.getItem("JWT")) {
      this.$api.initSetting();
    }
  }
};
</script>

<style lang="scss">
// @import "assets/style.scss";
// @import "assets/icons/fontello.css";
</style>
