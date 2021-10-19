<template>
  <div class="is-flex-direction-column is-flex">
    <header class="columns">
      <div class="column is-3">
        <b-field label="Product" class="is-size-7" label-position="on-border">
          <b-select
            v-model="selectProduct"
            expanded
            class="is-size-6"
            placeholder="Select a Product"
          >
            <option
              v-for="(data, index) in dataProduct"
              :key="index"
              :value="data.value"
              v-text="data.productName"
            ></option>
          </b-select>
        </b-field>
      </div>
    </header>

    <div class="columns mt-3" v-show="tableActive">
      <div class="column column is-two-thirds">
        <b-table :data="data" hoverable @click="getData" focusable :selected.sync="selected">
          <b-table-column field="plan_qty" :label="label" v-slot="props">
            <span class="is-size-6 has-text-weight-bold">{{
              props.row.plan_qty | localString(props.row.plan_qty)
            }}</span>
          </b-table-column>

          <b-table-column field="plan_price" label="Monthly Price" v-slot="props">
            <span class="is-size-6 has-text-weight-bold">
              {{ props.row.plan_price | toUsd(props.row.plan_price) }}
            </span>
          </b-table-column>
        </b-table>
      </div>

      <div class="column is-capitalized px-3 py-5" v-show="plan_name && plan_price">
        <div
          class="is-flex  totalPrice is-size-6 has-text-weight-semibold py-4 is-justify-content-space-between"
        >
          <span v-text="plan_name"></span>
          <span>{{ plan_price | toUsd(plan_price) }}</span>
        </div>
        <div class="is-flex is-size-5 has-text-weight-bold py-5 is-justify-content-space-between">
          <span>Estimated total </span>
          <span>{{ plan_price | toUsd(plan_price) }}</span>
        </div>

        <div class="wrapper-btnCheckout">
          <a
            :href="url ? url : false"
            class="is-flex is-justify-content-center p-2 has-text-weight-bold"
            :class="activeButtton ? 'checkoutBtn' : 'disableBtn'"
          >
            <span class="p-1 is-capitalized textLogout is-size-6">
              {{ activeButtton ? "Proceed to checkout" : "Waiting ..." }}
            </span>
          </a>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  data() {
    return {
      data: [],
      plan_name: "",
      plan_price: "",
      plan_qty: null,
      selected: null,
      selectProduct: "emails",
      tableActive: true,
      label: "",
      url: "",
      product: "",
      activeButtton: false,
      dataProduct: [
        {
          productName: "Email",
          value: "emails",
          label: "Emails per month",
          key: "emails",
          plan: [
            {
              plan_name: "100,000 Emails/Month",
              plan_qty: "100000",
              plan_price: "100.00"
            },
            {
              plan_name: "200,000 Emails/Month",
              plan_qty: "200000",
              plan_price: "220.00"
            },
            {
              plan_name: "300,000 Emails/Month",
              plan_qty: "300000",
              plan_price: "350.00"
            }
          ]
        },
        {
          productName: "SMS",
          value: "sms",
          label: "SMS per month",
          key: "sms",
          plan: [
            {
              plan_name: "500 SMS/Month",
              plan_qty: "500",
              plan_price: "10.00"
            },
            {
              plan_name: "1000 SMS/Month",
              plan_qty: "1000",
              plan_price: "20.00"
            },
            {
              plan_name: "3000 SMS/Month",
              plan_qty: "300000",
              plan_price: "35.00"
            }
          ]
        },
        {
          productName: "Push Notifications",
          value: "pushNotifications",
          label: "Push per month",
          key: "pushNotifications",
          plan: [
            {
              plan_name: "1000 Push/Month",
              plan_qty: "1000",
              plan_price: "100.00"
            },
            {
              plan_name: "2000 Push/Month",
              plan_qty: "2000",
              plan_price: "220.00"
            },
            {
              plan_name: "5000 Push/Month",
              plan_qty: "5000",
              plan_price: "500.00"
            }
          ]
        },
        {
          productName: "Validations",
          value: "validations",
          label: "Validations per month",
          key: "validations",
          plan: [
            {
              plan_name: "100 Validations/Month",
              plan_qty: "100",
              plan_price: "50.00"
            },
            {
              plan_name: "200 Validations/Month",
              plan_qty: "200",
              plan_price: "20.00"
            },
            {
              plan_name: "300 Validations/Month",
              plan_qty: "300",
              plan_price: "35.00"
            }
          ]
        }
      ]
    };
  },
  methods: {
    getData(value) {
      this.statusButton(false);
      this.plan_name = value.plan_name;
      this.plan_price = value.plan_price;
      this.plan_qty = value.plan_qty;

      this.getUrlStripe();
    },

    getUrlStripe() {
      this.$api.checkout({ plan_qty: parseInt(this.plan_qty), products: this.product }).then(result => {
        let { url } = result;

        this.url = url;
        this.statusButton(true);
      });
    },

    statusButton(value) {
      this.activeButtton = value;
    },

    showTable() {
      this.tableActive = true;
    },

    setAllData(value) {
      let getData = this.dataProduct.find(element => element.value == value);
      this.label = getData.label;
      this.data = getData.plan;
      this.product = getData.value;
    }
  },
  watch: {
    selectProduct(e) {
      this.setAllData(e);
    }
  },
  mounted() {
    this.data = this.dataProduct[0].plan;
    this.label = this.dataProduct[0].label;
    this.product = this.dataProduct[0].value;
  }
};
</script>

<style scoped>
.totalPrice {
  border-bottom: 3px solid #d1d5db;
}
.checkoutBtn {
  background-color: #7f2aff;
  border-color: transparent;
  cursor: pointer;
  width: 100%;
  border-radius: 0.5rem;
}
.disableBtn {
  background-color: #d1d5db;
  border-color: transparent;
  cursor: wait;
  width: 100%;
  border-radius: 0.5rem;
}

.disableBtn:hover {
  border-color: transparent;
}

.wrapper-btnCheckout {
  width: 100%;
}
</style>
