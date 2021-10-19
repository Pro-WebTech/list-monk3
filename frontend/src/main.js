import "./assets/icons/fontello.css";
import "./assets/style.scss";
import Vue from "vue";
import Buefy from "buefy";
import VueI18n from "vue-i18n";
import Validate from "./Plugins/validation";

import App from "./App.vue";
import router from "./router";
import store from "./store";
import * as api from "./api";
import Utils from "./utils";
import Layout from "./components/Layout.vue";

// Internationalisation.
Vue.use(VueI18n);
const i18n = new VueI18n();

Vue.use(Buefy, { defaultIconPack: "mdi" });
Vue.config.productionTip = false;
Vue.use(Validate);

//Global layout
Vue.component("Layout", Layout);

// Global Mixin
Vue.mixin({
  computed: {
    hasFormChanged: {
      get() {
        if (!this.formCopy) {
          // console.log(this.formCopy);  // Duplicate data from form
          return this.submitButton;
        }
        return (this.submitButton = JSON.stringify(this.form) !== this.formCopy);
      },
      set(value) {}
    }
  },
  filters: {
    toUsd(value) {
      let numberObject = new Number(value);
      let myObj = {
        style: "currency",
        currency: "USD"
      };

      return numberObject.toLocaleString("en-US", myObj);
    },
    localString(value) {
      return Number(value).toLocaleString("en-US");
    }
  },
  mounted() {},
  methods: {
    loadConfig() {
      api.getServerConfig().then(data => {
        api.getLang(data.lang).then(lang => {
          i18n.locale = data.lang;
          i18n.setLocaleMessage(i18n.locale, lang);
          this.isLoaded = true;
        });
      });
    }
  }
});

// Globals.
Vue.prototype.$utils = new Utils(i18n);
Vue.prototype.$api = api;

// Import the config for the auth
// eslint-disable-next-line
// import config from './config';

new Vue({
  router,
  store,
  i18n,

  //   http: api.httpApi,
  //   config,
  render: h => h(App),

  data: {
    isLoaded: false
  },

  methods: {
    loadConfig() {
      api.getServerConfig().then(data => {
        api.getLang(data.lang).then(lang => {
          i18n.locale = data.lang;
          i18n.setLocaleMessage(i18n.locale, lang);
          this.isLoaded = true;
        });
      });
    }
  },

  mounted() {},

  created() {
    if (localStorage.getItem("JWT")) {
      api.initSetting();
      this.loadConfig();
    }
  }
}).$mount("#app");
