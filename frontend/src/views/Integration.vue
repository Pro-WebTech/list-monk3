<template>
  <Layout>
    <div class="my-3 px-6-desktop">
      <h1
        class="is-size-2-desktop is-size-2-tablet is-size-4-mobile is has-text-weight-bold is-capitalized py-5"
      >
        Connect your favorite tools
      </h1>

      <AppMarket />
    </div>
  </Layout>
</template>

<script>
export default {
  components: {
    AppMarket: () => import("../components/AppMarket.vue")
  },
  data() {
    return {
      isPublic: "All Categories"
    };
  },
  watch: {
    selectedProduct(e) {
      console.log(e);
    }
  }
};
</script>

<style scoped>
.btnDropdown {
  background-color: #7f2aff;
  border-color: transparent;
  cursor: pointer;
  color: white;
}

.dropItem {
  color: #4a4a4a;
  cursor: pointer;
  width: 100%;
}

.dropItem:hover {
  color: white;
  background-color: #7f2aff;
}
</style>
