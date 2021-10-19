<template>
  <div>
    <ValidationObserver ref="form">
      <form action="">
        <label class="label py-2">{{ $t("settings.smtp.host") }}</label>
        <b-field
          class="is-size-7"
          label-position="on-border"
          :message="$t('settings.smtp.hostHelp')"
        >
          <b-select
            v-model="selectedHostname"
            expanded
            class="is-size-6"
            placeholder="Select a Hostname"
          >
            <option
              v-for="(value, key) in dataProviders"
              :key="key"
              @input="getDataHost"
              :value="value"
              v-text="key"
            ></option>
          </b-select>
        </b-field>
        <ValidationProvider
          ref="usernameProvider"
          v-slot="{ errors }"
          name="Username"
          rules="required"
        >
          <div class="field mb-2">
            <label class="label py-2">Username</label>
            <div class="control">
              <input
                type="text"
                @input="validate"
                @change="validate"
                ref="username"
                :class="{ 'is-danger': errors[0] }"
                placeholder="Username"
                class="input"
                v-model="username"
              />
            </div>
            <span v-show="errors[0]" class="warningColor">{{ errors[0] }}</span>
          </div>
        </ValidationProvider>

        <ValidationProvider
          ref="passwordProvider"
          v-slot="{ errors }"
          name="Password"
          rules="required"
        >
          <div class="field mb-2">
            <label class="label py-2">Password</label>
            <div class="control">
              <input
                @input="validate"
                @change="validate"
                class="input"
                :class="{ 'is-danger': errors[0] }"
                type="password"
                placeholder="Password"
                v-model="password"
              />
            </div>
            <span v-show="errors[0]" class="warningColor">{{ errors[0] }}</span>
          </div>
        </ValidationProvider>

        <b-field label="Headers" :message="message">
          <b-input
            :value="headers"
            v-model="headers"
            @input="checkHeaders"
            class="fieldHeader"
            name="email_headers"
            type="textarea"
            placeholder='[{"X-Custom": "value"}, {"X-Custom2": "value"}]'
          />
        </b-field>

        <b-field label="Tags" class="fieldTags" :type="{ 'is-danger': danger }">
          <b-taginput
            @typing="checkTag"
            @blur="checkErrorTag"
            :before-adding="checkTag"
            :allow-duplicates="duplicated"
            maxtags="3"
            class="is-size-5 getTags"
            v-model="tags"
          >
          </b-taginput>
        </b-field>
      </form>
    </ValidationObserver>

    <div class="mt-5 is-flex is-justify-content-space-between">
      <b-button label="Back" @click="backPreviousComponnent" />
      <b-button
        label="Finish"
        :disabled="finishButtonDisable"
        @click="addConnection"
        type="is-primary"
      />
    </div>
  </div>
</template>

<script>
import { validationFunction } from "../Plugins/formIdentityMixins";
import { v4 as uuidv4 } from "uuid";

export default {
  mixins: [validationFunction],
  props: ["dataHost"],
  data() {
    return {
      selectedHostname: "",
      duplicated: true,
      finishButtonDisable: true,
      indexProviders: "",
      indexProduct: "",
      danger: false,
      username: "",
      password: "",
      tags: [],
      headers: "[]",
      message: `Optional array of e-mail headers to include in all messages sent from this server. eg: [{"X-Custom": "value"}, {"X-Custom2": "value"}] `
    };
  },
  computed: {
    dataProviders() {
      this.setSelectedHostname(this.dataHost.settings.hostname);
      return this.dataHost.settings.hostname;
    },

    providerDatabase() {
      return this.$store.getters["providers/dataForm"];
    },
    getDataForm: {
      get() {
        let form = this.$store.getters["providers/dataForm"];
        let id = this.getIndexByMessenger();
        let dataForm = form.providers[id].product[this.getIndexByProductName()].connection;

        return dataForm;
      },
      set(host) {
        //To dispact with new host
        let id = this.getIndexByMessenger();
        let productId = this.getIndexByProductName();
        // console.log(id);
        this.$store.dispatch("providers/setNewHost", {
          id,
          productId,
          host
        });
      }
    }
  },

  methods: {
    setSelectedHostname(hostName) {
      this.selectedHostname = Object.values(hostName)[0];
    },

    getIndexByMessenger() {
      return this.providerDatabase.providers.findIndex(
        element => element.messenger.toLowerCase() === this.dataHost.messenger.toLowerCase()
      );
    },
    getIndexByProductName() {
      return this.providerDatabase.providers[this.getIndexByMessenger()].product.findIndex(
        element => element.name.toLowerCase() === this.dataHost.productCode.toLowerCase()
      );
    },

    getDataHost(value) {
      console.log(value);
    },
    addConnection() {
      let newData = {
        uuid: uuidv4(),
        enabled: true,
        host: this.selectedHostname,
        hello_hostname: "",
        port: 587,
        auth_protocol: "plain",
        username: this.username,
        password: this.password,
        email_headers: JSON.parse(this.headers),
        max_conns: 1000,
        max_msg_retries: 2,
        idle_timeout: "15s",
        wait_timeout: "5s",
        tls_enabled: true,
        tls_skip_verify: true,
        tag: this.tags,
        details: [
          {
            uuid: uuidv4(),
            name: this.dataHost.name,
            summary: this.dataHost.summary,
            category: this.dataHost.category,
            status: this.dataHost.status,
            icon: this.dataHost.icon,
            product_code: this.dataHost.productCode,
            messenger: this.dataHost.messenger
          }
        ]
      };

      this.getDataForm.push(newData);

      this.$emit("addNewProvider");
    }
  },

  watch: {
    headers(e) {
      //Conditional At 0 value Because it just Optional, So if user make empty value no problem and then button finish activated
      if (e.length > 0) {
        if (e === "[]") {
          this.hiddenHeaderError();
        } else {
          //Checking Parse json Error or not
          try {
            let parse = JSON.parse(e);
            this.headerProccess(parse);
          } catch (error) {
            if (error instanceof SyntaxError) {
              this.headerError();
            }
          }
        }
      } else if (e.length === 0) {
        this.hiddenHeaderError();
      }
    },
    selectedHostname(newHost) {
      this.getDataForm = newHost;
    }
  }
};
</script>

<style>
.warningColor {
  color: #f14668;
  font-size: 12px;
  padding: 5px 0px 5px 0px;
}
</style>
