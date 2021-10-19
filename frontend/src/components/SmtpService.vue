<template>
  <div>
    <b-loading :is-full-page="true" v-if="loadingSetting || loading" active />
    <div class="items mail-servers">
      <div class="block box" v-for="(item, n) in smtp" :key="n">
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
            <b-field v-if="smtp.length > 1">
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
                    :pattern="duration"
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
                    :pattern="duration"
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

    <b-button @click="addSMTP" icon-left="plus" type="is-primary">
      {{ $t("globals.buttons.addNew") }}
    </b-button>
  </div>
</template>

<script>
export default {
  props: ["dataSmtp", "loadingSetting", "loading", "duration"],
  data() {
    return {
      selectedHostname: "hostname1",
      hostname: [
        {
          key: "hostname1",
          value: "Hostname1"
        },
        {
          key: "hostname2",
          value: "Hostname2"
        },
        {
          key: "hostname3",
          value: "Hostname3"
        }
      ]
    };
  },

  methods: {
    inputText(event, index) {
      this.$emit("newHost", event, index, "host");
    },
    dataHost(value) {
      /// Create new object inside array with map
      console.log(value[0].settings);
      let getHostname = value[0].settings.hostname;

      return Object.keys(getHostname);
    },
    dataHostValue() {},
    addSMTP() {
      this.$emit("newSmtp");
    },

    removeSMTP(i) {
      this.$emit("removeSmtp", i);
    },

    showSMTPHeaders(i) {
      this.$emit("showHeader", i);
    }
  },

  computed: {
    smtp() {
      return this.dataSmtp;
    },
    dataProviders() {
      return this.$store.getters["providers/allProviders"];
    }
  },

  mounted() {
    this.smtp ? (this.isLoading = true) : (this.isLoading = false);
  }
};
</script>

<style></style>
