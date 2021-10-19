<template>
  <div v-show="showElement">
    <h1 class="title" v-text="'Active connections'"></h1>
    <div class="columns is-multiline">
      <template v-for="data in activeConnection">
        {{ hiddenOrShowElement(data.details) }}
        <template v-if="data.details !== null">
          <div class="column is-one-third" v-for="details in data.details" :key="details.name">
            <div class="box">
              <span
                @click="deletingData(details, data.uuid)"
                class="is-flex deletingButton is-justify-content-flex-end"
              >
                <svg style="width:35px;height:35px" viewBox="0 0 24 24">
                  <path
                    fill="currentColor"
                    d="M12,20C7.59,20 4,16.41 4,12C4,7.59 7.59,4 12,4C16.41,4 20,7.59 20,12C20,16.41 16.41,20 12,20M12,2C6.47,2 2,6.47 2,12C2,17.53 6.47,22 12,22C17.53,22 22,17.53 22,12C22,6.47 17.53,2 12,2M14.59,8L12,10.59L9.41,8L8,9.41L10.59,12L8,14.59L9.41,16L12,13.41L14.59,16L16,14.59L13.41,12L16,9.41L14.59,8Z"
                  />
                </svg>
              </span>
              <div class="is-flex is-flex-direction-column">
                <div class="">
                  <figure class="image is-128x128">
                    <img :src="details.icon" />
                  </figure>
                </div>
                <div class="title mb-0 py-2 is-capitalized is-size-5">
                  {{ details.name }}
                </div>
                <div class="">
                  <div class="is-flex is-flex-direction-column px-1">
                    <div>
                      <div class="is-capitalized">
                        <span
                          class="tagClass"
                          :class="bindingClass(details.status)"
                          v-text="details.status"
                        ></span>
                      </div>
                    </div>
                    <div class="descrip py-5">
                      <p class="description">
                        {{ details.summary }}
                      </p>
                    </div>

                    <p>{{ data.username }}</p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </template>
      </template>
    </div>
  </div>
</template>

<script>
export default {
  props: ["providers"],
  data() {
    return {
      showElement: false,
      dataProviders: "",
      activeConnection: "",
      formProvider: {}
    };
  },
  computed: {
    dataProvider() {
      return this.formProvider;
    }
  },
  methods: {
    getActiveConnection(value) {
      if (value) {
        let array = [];
        value.forEach(element => {
          element.product.forEach(product => {
            array.push(product);
          });
        });

        let connection = array.map(element => element.connection.flat());
        let arrayConnection = [];
        connection.forEach(element => {
          element.forEach(elementConnection => {
            arrayConnection.push(elementConnection);
          });
        });
        this.activeConnection = arrayConnection;
        this.showElement = true;
      }
    },

    getIndexByMessenger(messenger) {
      return this.formProvider.findIndex(
        element => element.messenger.toLowerCase() === messenger.toLowerCase()
      );
    },

    getIndexByProductName(id, product_code) {
      return this.formProvider[id].product.findIndex(
        element => element.name.toLowerCase() == product_code.toLowerCase()
      );
    },

    getIndexConnection(messengerId, productId, uuid) {
      return this.formProvider[messengerId].product[productId].connection.findIndex(
        element => element.uuid == uuid
      );
    },

    deletingData(detail, uuid) {
      let getMessenger = this.getIndexByMessenger(detail.messenger);
      let getNameProduct = this.getIndexByProductName(getMessenger, detail.product_code); // get product_code  , & get messanger

      let indexConnection = this.getIndexConnection(getMessenger, getNameProduct, uuid);

      this.$store.dispatch("providers/setRemoveConnection", {
        messengerIndex: getMessenger,
        productIndex: getNameProduct,
        conectionIndex: indexConnection
      });

      this.$emit("removeConnection");
    },

    hiddenOrShowElement(value) {
      console.log(value.length);
    },

    getSettings() {
      this.$api.getSettings().then(data => {
        const d = JSON.parse(JSON.stringify(data));

        this.formProvider = d.providers;

        this.getActiveConnection(d.providers);
      });
    }
  },
  mounted() {
    this.getSettings();

    let getData = document.querySelector(".column.is-one-third");

    console.log(getData);
  }
};
</script>

<style scoped>
.deletingButton {
  cursor: pointer;
}

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
  padding-top: 2px;
  border: 1px solid transparent;
}
.box:hover {
  border: 1px solid #7f2aff;
}
</style>
