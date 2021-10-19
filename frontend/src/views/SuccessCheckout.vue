<template>
  <div class="is-fullheight">
    <div class="hero-body">
      <div class="container">
        <div class="column is-4 is-offset-4">
          <div class="box is-flex is-flex-direction-column is-justify-content-center">
            <div class="is-flex my-4 is-justify-content-center">
              <div class="successWrapper">
                <svg style="width:100px;height:100px" viewBox="0 0 24 24">
                  <path fill="white" d="M21,7L9,19L3.5,13.5L4.91,12.09L9,16.17L19.59,5.59L21,7Z" />
                </svg>
              </div>
            </div>
            <div class="is-flex my-4 is-justify-content-center">
              <span class="is-size-2 has-text-weight-bold">Success</span>
            </div>

            <div class="is-flex my-4 is-justify-content-center">
              <button
                @click="backtoDashboard"
                class="is-flex is-justify-content-center p-2 checkoutBtn has-text-weight-bold box"
              >
                <span class="p-1 is-capitalized textLogout is-size-6">
                  back to dashboard
                </span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  methods: {
    backtoDashboard() {
      this.$router.push("/dashboard");
    }
  }
};
</script>

<style scoped>
.checkoutBtn {
  background-color: #7f2aff;
  border-color: transparent;
  cursor: pointer;
}
.successWrapper {
  background-color: #34d399;
  padding: 25px;
  border-radius: 50%;
}
</style>
