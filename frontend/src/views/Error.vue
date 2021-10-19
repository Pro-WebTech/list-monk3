<template>
  <div class="is-fullheight">
    <div class="hero-body">
      <div class="container">
        <div class="column is-4 is-offset-4">
          <div class="p-3 is-flex is-flex-direction-column is-justify-content-center">
            <div class="has-text-centered">
              <h1 class="errorCode">404</h1>
              <p class="is-capitalized is-size-2 has-text-weight-bold">Page not found</p>
            </div>
            <div class="my-2 py-5">
              <b-button type="is-primary" @click="backButton" size="is-medium" class="box mx-auto"
                >Back to Home</b-button
              >
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
    backButton() {
      this.$router.push("/");
    }
  }
};
</script>

<style>
.errorCode {
  font-size: 15rem;
}
</style>
