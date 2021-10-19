<template>
  <div>
    <router-view />
    <!-- :key="$route.fullPath" -->
  </div>
</template>

<script>
import Vue from "vue";
import { mapState } from "vuex";

export default Vue.extend({
  name: "App",

  data() {
    return {
      activeItem: {},
      activeGroup: {},
      isLoginPage: ""
    };
  },

  mounted() {
    // Lists is required across different views. On app load, fetch the lists
    // and have them in the store.
    if (localStorage.getItem("JWT")) {
      this.$api.initLists();
    }
  }
});
</script>

<style lang="scss">
@import "assets/style.scss";
@import "assets/icons/fontello.css";
</style>
