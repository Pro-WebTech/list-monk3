<template>
  <div>
    <ActiveProviders :providers="dataForm" @removeConnection="removeConnection" />

    <header class="columns">
      <b-field label="category" label-position="on-border">
        <b-select v-model="selectedProduct" name="upload.provider">
          <option
            v-for="(data, index) in dataCategory"
            :key="index"
            :value="data.key"
            v-text="data.value"
          ></option>
        </b-select>
      </b-field>
    </header>

    <header class="columns">
      <div class="column is-3">
        <div class="field">
          <label class="label">Category</label>
          <div class="control">
            <div class="select">
              <select v-model="selectedProduct">
                <option
                  v-for="(data, index) in dataCategory"
                  :key="index"
                  :value="data.key"
                  v-text="data.value"
                ></option>
              </select>
            </div>
          </div>
        </div>
      </div>
    </header>

    <div class="container py-6">
      <div class="">
        <div>
          <b-loading :is-full-page="isFullPage" v-model="isLoading" :can-cancel="true"></b-loading>
        </div>

        <div class="columns is-multiline">
          <div class="column is-one-third" v-for="data in allProviders" :key="data.id">
            <div class="box" @[dynamicalClick(data)]="nextStep(data)">
              <div class="is-flex is-flex-direction-column">
                <div class="">
                  <figure class="image is-128x128">
                    <img :src="data.icon" />
                  </figure>
                </div>
                <div class="title mb-0 py-2 is-capitalized is-size-5" v-text="data.name"></div>
                <div class="">
                  <div class="is-flex is-flex-direction-column px-1">
                    <div>
                      <div class="is-capitalized">
                        <span
                          class="tagClass"
                          :class="bindingClass(data.status)"
                          v-text="data.status"
                        ></span>
                      </div>
                    </div>
                    <div class="descrip py-5">
                      <p class="description" v-html="data.summary"></p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  components: {
    Steps: () => import("../components/Steps.vue"),
    ActiveProviders: () => import("../components/ActiveProviders.vue")
  },
  data() {
    return {
      stepHidden: false,
      selectedProduct: "all",
      isLoading: true,
      isFullPage: true,
      dataCategory: "",
      dataProviders: "",
      oneData: "",
      message: `Optional array of e-mail headers to include in all messages sent from this server. eg: [{"X-Custom": "value"}, {"X-Custom2": "value"}] `
    };
  },

  mounted() {
    this.$api
      .getGraphql(
        "query  {providers { id links name settings status summary updated_at icon product_code created_at category messenger}}"
      )
      .then(res => {
        this.isLoading = false;

        this.dataProviders = res.providers;
        this.createNewArray(res.providers);
      });
  },

  watch: {
    selectedProduct(value) {
      let getone = this.dataProviders.filter(element => element.messenger == value);

      value === "all" ? (this.oneData = "") : (this.oneData = getone);
    }
  },
  methods: {
    bindingClass(dataValue) {
      switch (dataValue) {
        case "enabled":
          return "tagClassEnabled";
        case "comingsoon":
          return "tagClassComingSoon";
        case "disabled":
          return "tagClassDisabled";
      }
    },
    createNewArray(value) {
      let oldArray = [{ key: "all", value: "ALL" }];

      let newArray = value.map(element => {
        let obj = {};
        obj["key"] = element.messenger;
        obj["value"] = element.category;
        return obj;
      });

      this.dataCategory = oldArray.concat(newArray);
    },
    nextStep(value) {
      this.$emit("changeComponent", "FormIdentity", value);
    },
    dynamicalClick(value) {
      return value.status == "disabled" ? null : "click";
    },
    removeConnection() {
      this.$emit("removeConnection");
    },
    successToast() {
      this.$buefy.toast.open({
        duration: 5000,
        message: "Your connection details have been successfully saved!",
        position: "is-top",
        type: "is-success"
      });
    }
  },
  computed: {
    allProviders() {
      return this.oneData === "" ? this.dataProviders : this.oneData;
    },
    dataForm() {
      return this.$store.getters["providers/dataForm"].providers;
    }
  }
};
</script>

<style scoped>
.hid {
  height: 100px;
  overflow: hidden;
}

.tagClass {
  padding: 5px 10px;
  border-radius: 5px;
  width: auto;
  font-size: 12px;
  font-weight: bold;
}
.tagClassEnabled {
  background-color: #7f2aff;
  color: white;
}
.tagClassComingSoon {
  background-color: #ffe08a;
  color: black;
}
.tagClassDisabled {
  background-color: #f14668;
  color: white;
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
