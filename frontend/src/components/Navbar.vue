<template>
  <nav class="navbar is-fixed-top is-flex is-justify-content-space-between">
    <div class="navbar-brand">
      <a
        role="button"
        class="navbar-burger"
        aria-label="menu"
        @click="openSidebar"
        aria-expanded="false"
      >
        <span aria-hidden="true"></span>
        <span aria-hidden="true"></span>
        <span aria-hidden="true"></span>
      </a>

      <router-link :to="{ name: 'dashboard' }">
        <!-- <img class="full" :src="image" /> -->
        <!-- Just trial development style -->
        <img class="favicon" src="@/assets/favicon.png" />
      </router-link>
    </div>

    <div id="navbarBasicExample" class="navbar-menu is-block p-0 is-shadowless">
      <div class="navbar-end">
        <div class="navbar-item p-1">
          <button
            @click="logout"
            class="is-flex is-justify-content-center p-1 btn has-text-weight-bold box"
          >
            <span>
              <svg style="width:24px;height:24px" viewBox="0 0 24 24">
                <path
                  fill="white"
                  d="M16,17V14H9V10H16V7L21,12L16,17M14,2A2,2 0 0,1 16,4V6H14V4H5V20H14V18H16V20A2,2 0 0,1 14,22H5A2,2 0 0,1 3,20V4A2,2 0 0,1 5,2H14Z"
                />
              </svg>
            </span>
            <span class="p-1 is-capitalized textLogout is-size-6">
              logout
            </span>
          </button>
        </div>
      </div>
    </div>
  </nav>
</template>

<script>
export default {
  data() {
    return {
      image: ""
    };
  },
  methods: {
    logout() {
      localStorage.removeItem("JWT");
      this.$router.push("/");

      // Open Alert
      this.$buefy.toast.open({
        duration: 5000,
        message: "You're Log Out",
        position: "is-top",
        type: "is-danger"
      });
    },
    openSidebar() {
      this.$emit("clickSidebar");
    }
  },
  mounted() {
    this.image = localStorage.getItem("logo");
  }
};
</script>

<style>
.btn {
  background-color: #f14668;
  border-color: transparent;
  cursor: pointer;
}

.textLogout {
  color: white;
}
</style>
