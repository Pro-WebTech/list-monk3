<template>
  <Layout>
    <b-tabs type="is-boxed" :animated="false">
      <b-tab-item :label="$t('settings.smtp.name')">
        <!-- <SmtpService /> -->
        <!-- Default VErsion -->
        <div class="items mail-servers">
          <div class="block box" v-for="(item, n) in form.smtp" :key="n">
            <div class="columns">
              <div class="column is-2">
                <b-field :label="$t('globals.buttons.enabled')">
                  <b-switch
                    v-model="item.enabled"
                    name="enabled"
                    :native-value="true"
                    data-cy="btn-enable-smtp"
                  />
                </b-field>
                <b-field v-if="form.smtp.length > 1">
                  <a
                    @click.prevent="$utils.confirm(null, () => removeSMTP(n))"
                    href="#"
                    class="is-size-7"
                    data-cy="btn-delete-smtp"
                  >
                    <b-icon icon="trash-can-outline" size="is-small" />
                    {{ $t("globals.buttons.delete") }}
                  </a>
                </b-field>
              </div>
              <!-- first column -->

              <div class="column" :class="{ disabled: !item.enabled }">
                <div class="columns">
                  <div class="column is-8">
                    <b-field
                      :label="$t('settings.smtp.host')"
                      label-position="on-border"
                      :message="$t('settings.smtp.hostHelp')"
                    >
                      <b-input
                        v-model="item.host"
                        name="host"
                        placeholder="smtp.yourmailserver.net"
                        :maxlength="200"
                      />
                    </b-field>
                  </div>
                  <div class="column">
                    <b-field
                      :label="$t('settings.smtp.port')"
                      label-position="on-border"
                      :message="$t('settings.smtp.portHelp')"
                    >
                      <b-numberinput
                        v-model="item.port"
                        name="port"
                        type="is-light"
                        controls-position="compact"
                        placeholder="25"
                        min="1"
                        max="65535"
                      />
                    </b-field>
                  </div>
                </div>
                <!-- host -->

                <div class="columns">
                  <div class="column is-2">
                    <b-field :label="$t('settings.smtp.authProtocol')" label-position="on-border">
                      <b-select v-model="item.auth_protocol" name="auth_protocol">
                        <option value="none">none</option>
                        <option value="cram">cram</option>
                        <option value="plain">plain</option>
                        <option value="login">login</option>
                      </b-select>
                    </b-field>
                  </div>
                  <div class="column">
                    <b-field grouped>
                      <b-field
                        :label="$t('settings.smtp.username')"
                        label-position="on-border"
                        expanded
                      >
                        <b-input
                          v-model="item.username"
                          :disabled="item.auth_protocol === 'none'"
                          name="username"
                          placeholder="mysmtp"
                          :maxlength="200"
                        />
                      </b-field>
                      <b-field
                        :label="$t('settings.smtp.password')"
                        label-position="on-border"
                        expanded
                        :message="$t('settings.smtp.passwordHelp')"
                      >
                        <b-input
                          v-model="item.password"
                          :disabled="item.auth_protocol === 'none'"
                          name="password"
                          type="password"
                          :placeholder="$t('settings.smtp.passwordHelp')"
                          :maxlength="200"
                        />
                      </b-field>
                    </b-field>
                  </div>
                </div>
                <!-- auth -->
                <hr />

                <div class="columns">
                  <div class="column is-6">
                    <b-field
                      :label="$t('settings.smtp.heloHost')"
                      label-position="on-border"
                      :message="$t('settings.smtp.heloHostHelp')"
                    >
                      <b-input
                        v-model="item.hello_hostname"
                        name="hello_hostname"
                        placeholder=""
                        :maxlength="200"
                      />
                    </b-field>
                  </div>
                  <div class="column">
                    <b-field grouped>
                      <b-field
                        :label="$t('settings.smtp.tls')"
                        expanded
                        :message="$t('settings.smtp.tlsHelp')"
                      >
                        <b-switch v-model="item.tls_enabled" name="item.tls_enabled" />
                      </b-field>
                      <b-field
                        :label="$t('settings.smtp.skipTLS')"
                        expanded
                        :message="$t('settings.smtp.skipTLSHelp')"
                      >
                        <b-switch
                          v-model="item.tls_skip_verify"
                          :disabled="!item.tls_enabled"
                          name="item.tls_skip_verify"
                        />
                      </b-field>
                    </b-field>
                  </div>
                </div>
                <!-- TLS -->
                <hr />

                <div class="columns">
                  <div class="column is-3">
                    <b-field
                      :label="$t('settings.smtp.maxConns')"
                      label-position="on-border"
                      :message="$t('settings.smtp.maxConnsHelp')"
                    >
                      <b-numberinput
                        v-model="item.max_conns"
                        name="max_conns"
                        type="is-light"
                        controls-position="compact"
                        placeholder="25"
                        min="1"
                        max="65535"
                      />
                    </b-field>
                  </div>
                  <div class="column is-3">
                    <b-field
                      :label="$t('settings.smtp.retries')"
                      label-position="on-border"
                      :message="$t('settings.smtp.retriesHelp')"
                    >
                      <b-numberinput
                        v-model="item.max_msg_retries"
                        name="max_msg_retries"
                        type="is-light"
                        controls-position="compact"
                        placeholder="2"
                        min="1"
                        max="1000"
                      />
                    </b-field>
                  </div>
                  <div class="column is-3">
                    <b-field
                      :label="$t('settings.smtp.idleTimeout')"
                      label-position="on-border"
                      :message="$t('settings.smtp.idleTimeoutHelp')"
                    >
                      <b-input
                        v-model="item.idle_timeout"
                        name="idle_timeout"
                        placeholder="15s"
                        :pattern="regDuration"
                        :maxlength="10"
                      />
                    </b-field>
                  </div>
                  <div class="column is-3">
                    <b-field
                      :label="$t('settings.smtp.waitTimeout')"
                      label-position="on-border"
                      :message="$t('settings.smtp.waitTimeoutHelp')"
                    >
                      <b-input
                        v-model="item.wait_timeout"
                        name="wait_timeout"
                        placeholder="5s"
                        :pattern="regDuration"
                        :maxlength="10"
                      />
                    </b-field>
                  </div>
                </div>
                <hr />

                <div>
                  <p v-if="item.email_headers.length === 0 && !item.showHeaders">
                    <a href="#" class="is-size-7" @click.prevent="() => showSMTPHeaders(n)">
                      <b-icon icon="plus" />{{ $t("settings.smtp.setCustomHeaders") }}</a
                    >
                  </p>
                  <b-field
                    v-if="item.email_headers.length > 0 || item.showHeaders"
                    :label="$t('')"
                    label-position="on-border"
                    :message="$t('settings.smtp.customHeadersHelp')"
                  >
                    <b-input
                      v-model="item.strEmailHeaders"
                      name="email_headers"
                      type="textarea"
                      placeholder='[{"X-Custom": "value"}, {"X-Custom2": "value"}]'
                    />
                  </b-field>
                </div>
              </div>
            </div>
            <!-- second container column -->
          </div>
          <!-- block -->
        </div>
        <!-- mail-servers -->

        <b-button @click="addSMTP" icon-left="plus" type="is-primary">
          {{ $t("globals.buttons.addNew") }}
        </b-button>
      </b-tab-item>
    </b-tabs>
  </Layout>
</template>

<script>
import Vue from "vue";
import { mapState } from "vuex";

const dummyPassword = " ".repeat(8);

export default Vue.extend({
  components: {
    Billing: () => import("../components/Billing.vue"),
    ProvidersMarket: () => import("../components/ProvidersMarket.vue"),
    SmtpService: () => import("../components/SmtpService.vue"),
    Steps: () => import("../components/Steps.vue")
  },
  data() {
    return {
      regDuration: "[0-9]+(ms|s|m|h|d)",
      isLoading: false,

      // formCopy is a stringified copy of the original settings against which
      // form is compared to detect changes.
      formCopy: "",
      form: {}
    };
  },

  methods: {
    addSMTP() {
      this.form.smtp.push({
        enabled: true,
        host: "",
        hello_hostname: "",
        port: 587,
        auth_protocol: "none",
        username: "",
        password: "",
        email_headers: [],
        max_conns: 10,
        max_msg_retries: 2,
        idle_timeout: "15s",
        wait_timeout: "5s",
        tls_enabled: true,
        tls_skip_verify: false
      });

      this.$nextTick(() => {
        const items = document.querySelectorAll('.mail-servers input[name="host"]');
        items[items.length - 1].focus();
      });
    },

    removeSMTP(i) {
      this.form.smtp.splice(i, 1);
    },

    showSMTPHeaders(i) {
      const s = this.form.smtp[i];
      s.showHeaders = true;
      this.form.smtp.splice(i, 1, s);
    },

    addMessenger() {
      this.form.messengers.push({
        enabled: true,
        root_url: "",
        name: "",
        username: "",
        password: "",
        max_conns: 25,
        max_msg_retries: 2,
        timeout: "5s"
      });

      this.$nextTick(() => {
        const items = document.querySelectorAll('.messengers input[name="name"]');
        items[items.length - 1].focus();
      });
    },

    removeMessenger(i) {
      this.form.messengers.splice(i, 1);
    },

    onSubmit() {
      const form = JSON.parse(JSON.stringify(this.form));

      // De-serialize custom e-mail headers.
      for (let i = 0; i < form.smtp.length; i += 1) {
        // If it's the dummy UI password placeholder, ignore it.
        if (form.smtp[i].password === dummyPassword) {
          form.smtp[i].password = "";
        }

        if (form.smtp[i].strEmailHeaders && form.smtp[i].strEmailHeaders !== "[]") {
          form.smtp[i].email_headers = JSON.parse(form.smtp[i].strEmailHeaders);
        } else {
          form.smtp[i].email_headers = [];
        }
      }

      if (form["upload.s3.aws_secret_access_key"] === dummyPassword) {
        form["upload.s3.aws_secret_access_key"] = "";
      }

      for (let i = 0; i < form.messengers.length; i += 1) {
        // If it's the dummy UI password placeholder, ignore it.
        if (form.messengers[i].password === dummyPassword) {
          form.messengers[i].password = "";
        }
      }

      this.isLoading = true;
      this.$api.updateSettings(form).then(
        data => {
          if (data.needsRestart) {
            // There are running campaigns and the app didn't auto restart.
            // The UI will show a warning.
            this.$root.loadConfig();
            this.getSettings();
            this.isLoading = false;
            return;
          }

          this.$utils.toast(this.$t("settings.messengers.messageSaved"));

          // Poll until there's a 200 response, waiting for the app
          // to restart and come back up.
          const pollId = setInterval(() => {
            this.$api.getHealth().then(() => {
              clearInterval(pollId);
              this.$root.loadConfig();
              this.getSettings();
            });
          }, 500);
        },
        () => {
          this.isLoading = false;
        }
      );
    },

    getSettings() {
      this.$api.getSettings().then(data => {
        const d = JSON.parse(JSON.stringify(data));

        // Serialize the `email_headers` array map to display on the form.
        for (let i = 0; i < d.smtp.length; i += 1) {
          d.smtp[i].strEmailHeaders = JSON.stringify(d.smtp[i].email_headers, null, 4);

          // The backend doesn't send passwords, so add a dummy so that
          // the password looks filled on the UI.
          d.smtp[i].password = dummyPassword;
        }

        for (let i = 0; i < d.messengers.length; i += 1) {
          // The backend doesn't send passwords, so add a dummy so that it
          // the password looks filled on the UI.
          d.messengers[i].password = dummyPassword;
        }

        if (d["upload.provider"] === "s3") {
          d["upload.s3.aws_secret_access_key"] = dummyPassword;
        }

        this.form = d;
        this.formCopy = JSON.stringify(d);
        this.isLoading = false;
      });
    }
  },

  computed: {
    ...mapState(["serverConfig", "loading"]),

    hasFormChanged() {
      if (!this.formCopy) {
        return false;
      }
      return JSON.stringify(this.form) !== this.formCopy;
    }
  },

  beforeRouteLeave(to, from, next) {
    if (this.hasFormChanged) {
      this.$utils.confirm(this.$t("settings.messengers.messageDiscard"), () => next(true));
      return;
    }
    next(true);
  },

  mounted() {
    this.getSettings();
  }
});
</script>
