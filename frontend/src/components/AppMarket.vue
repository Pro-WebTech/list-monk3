<template>
  <div>
    <div v-if="loadingOn">
      <b-loading :is-full-page="isFullPage" v-model="isLoading" :can-cancel="true"></b-loading>
    </div>
    <template v-if="integrations.length > 0 ? true : false">
      <header class="columns">
        <div class="column is-6-mobile is-3-tablet is-2-desktop">
          <div class="field">
            <label class="label">Product</label>
            <div class="control">
              <div class="select">
                <select v-model="selectedProduct">
                  <option
                    v-for="(data, index) in dataProduct"
                    :key="index"
                    :value="data.value"
                    v-text="data.key"
                  ></option>
                </select>
              </div>
            </div>
          </div>
        </div>
      </header>

      <div class="container py-6">
        <div class="columns is-multiline">
          <div class="column is-one-third" v-for="data in integrations" :key="data.id">
            <div class="box">
              <div class="is-flex is-flex-direction-column">
                <div class="py-2">
                  <figure class="image is-128x128">
                    <img :src="data.icon" />
                  </figure>
                </div>
                <div class="title mb-0 py-2 is-capitalized is-size-5" v-text="data.name"></div>
                <div class="is-size-7 mb-3 has-text-weight-bold px-1 is-capitalized">
                  built by {{ data.builtby }}
                </div>
                <div class="descrip pb-5 pt-2">
                  <p class="description " v-html="cutDescription(data.description)"></p>
                </div>
                <div class="columns is-mobile is-tablet is-multiline">
                  <div
                    v-for="(data, index) in data.tags.tags"
                    :key="index"
                    class="column is-full-tablet px-2"
                    :class="data.length >= 10 ? 'is-5-desktop' : 'is-3-desktop'"
                  >
                    <div class="is-capitalized">
                      <span class="tagClass">{{ data }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script>
export default {
  mounted() {
    this.$api
      .getGraphql(
        "query GETALLPROVIDERS {integrations {id icon links name settings status tags updated_at builtby category description}}"
      )
      .then(res => {
        this.integrations = res.integrations;
        this.loadingOn = false;
      });
  },

  data() {
    return {
      selectedProduct: "all",
      isLoading: true,
      loadingOn: true,
      isFullPage: true,
      integrations: "",
      dataProduct: [
        { key: "All", value: "all" },
        {
          key: "Email",
          value: "email"
        },
        {
          key: "Productivity",
          value: "productivity"
        },
        {
          key: "Sales",
          value: "sales"
        },
        {
          key: "Marketing",
          value: "marketing"
        }
      ]
    };
  },
  methods: {
    cutDescription(value) {
      return value.substring(0, 92);
    }
  }
};
</script>

<style scoped>
.hid {
  height: 100px;
}

.tagClass {
  background-color: #f5f5f5;
  padding: 5px 10px;
  border-radius: 5px;
  width: auto;
  font-size: 12px;
  font-weight: bold;
}

.is-size-7 {
  opacity: 70%;
}
.description {
  font-size: 14px;
}
.box {
  cursor: pointer;
  border: 1px solid transparent;
}
.box:hover {
  border: 1px solid #7f2aff;
}
</style>
