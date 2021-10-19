<template>
  <keep-alive>
    <component
      :is="currentComponent"
      @changeComponent="reactiveComponent"
      @addNewProvider="$emit('submitProvider')"
      @removeConnection="$emit('removeConnection')"
      :dataHost="host"
    ></component>
  </keep-alive>
</template>

<script>
export default {
  props: ["dataForm"],
  components: {
    FormIdentity: () => import("../components/FormIdentity.vue"),
    ProvidersMarket: () => import("../components/ProvidersMarket.vue")
  },
  data() {
    return {
      currentComponent: "ProvidersMarket",
      host: ""
    };
  },

  methods: {
    toThis(messenger, name, dataHost) {
      //console.log(messenger, name, dataHost);
      this.$emit("submitProvider", messenger, name, dataHost);
    },
    reactiveComponent(componentPosition, host) {
      this.currentComponent = componentPosition;

      host !== undefined ? (this.host = host) : "";
    }
  }
};
</script>

<style></style>
