<template>
  <Layout>
    <section class="settings">
      <b-loading :is-full-page="true" v-if="loading.settings || isLoading" active />
      <header class="columns">
        <div class="column is-half">
          <h1 class="title is-4">{{ $t("settings.title") }}</h1>
        </div>
        <div class="column has-text-right">
          <b-button
            :disabled="!hasFormChanged"
            type="is-primary"
            icon-left="content-save-outline"
            @click="onSubmit"
            class="isSaveEnabled"
            data-cy="btn-save"
          >
            {{ $t("globals.buttons.save") }}
          </b-button>
        </div>
      </header>
      <hr />

      <section class="wrap-small">
        <form @submit.prevent="onSubmit">
          <b-tabs type="is-boxed" :animated="false">
            <b-tab-item :label="$t('settings.general.name')" label-position="on-border">
              <div class="items">
                <b-field
                  :label="$t('settings.general.rootURL')"
                  label-position="on-border"
                  :message="$t('settings.general.rootURLHelp')"
                >
                  <b-input
                    v-model="form['app.root_url']"
                    name="app.root_url"
                    placeholder="https://listmonk.yoursite.com"
                    :maxlength="300"
                  />
                </b-field>

                <b-field
                  :label="$t('settings.general.logoURL')"
                  label-position="on-border"
                  :message="$t('settings.general.logoURLHelp')"
                >
                  <b-input
                    v-model="form['app.logo_url']"
                    name="app.logo_url"
                    placeholder="https://listmonk.yoursite.com/logo.png"
                    :maxlength="300"
                  />
                </b-field>

                <b-field
                  :label="$t('settings.general.faviconURL')"
                  label-position="on-border"
                  :message="$t('settings.general.faviconURLHelp')"
                >
                  <b-input
                    v-model="form['app.favicon_url']"
                    name="app.favicon_url"
                    placeholder="https://listmonk.yoursite.com/favicon.png"
                    :maxlength="300"
                  />
                </b-field>

                <hr />
                <b-field
                  :label="$t('settings.general.fromEmail')"
                  label-position="on-border"
                  :message="$t('settings.general.fromEmailHelp')"
                >
                  <b-input
                    v-model="form['app.from_email']"
                    name="app.from_email"
                    placeholder="Listmonk <noreply@listmonk.yoursite.com>"
                    pattern="(.+?)\s<(.+?)@(.+?)>"
                    :maxlength="300"
                  />
                </b-field>

                <b-field
                  :label="$t('settings.general.adminNotifEmails')"
                  label-position="on-border"
                  :message="$t('settings.general.adminNotifEmailsHelp')"
                >
                  <b-taginput
                    v-model="form['app.notify_emails']"
                    name="app.notify_emails"
                    :before-adding="v => v.match(/(.+?)@(.+?)/)"
                    placeholder="you@yoursite.com"
                  />
                </b-field>

                <b-field
                  :label="$t('settings.general.enablePublicSubPage')"
                  :message="$t('settings.general.enablePublicSubPageHelp')"
                >
                  <b-switch
                    v-model="form['app.enable_public_subscription_page']"
                    name="app.enable_public_subscription_page"
                  />
                </b-field>

                <b-field
                  :label="$t('settings.general.checkUpdates')"
                  :message="$t('settings.general.checkUpdatesHelp')"
                >
                  <b-switch v-model="form['app.check_updates']" name="app.check_updates" />
                </b-field>

                <hr />
                <b-field :label="$t('settings.general.language')" label-position="on-border">
                  <b-select v-model="form['app.lang']" name="app.lang">
                    <option v-for="l in serverConfig.langs" :key="l.code" :value="l.code">
                      {{ l.name }}
                    </option>
                  </b-select>
                </b-field>
              </div> </b-tab-item
            ><!-- general -->

            <b-tab-item label="Billing" label-position="on-border">
              <div class="items">
                <Billing />
              </div>
            </b-tab-item>
            <!-- Billing  -->

            <b-tab-item :label="$t('settings.performance.name')">
              <div class="items">
                <b-field
                  :label="$t('settings.performance.concurrency')"
                  label-position="on-border"
                  :message="$t('settings.performance.concurrencyHelp')"
                >
                  <b-numberinput
                    v-model="form['app.concurrency']"
                    name="app.concurrency"
                    type="is-light"
                    placeholder="5"
                    min="1"
                    max="10000"
                  />
                </b-field>

                <b-field
                  :label="$t('settings.performance.messageRate')"
                  label-position="on-border"
                  :message="$t('settings.performance.messageRateHelp')"
                >
                  <b-numberinput
                    v-model="form['app.message_rate']"
                    name="app.message_rate"
                    type="is-light"
                    placeholder="5"
                    min="1"
                    max="100000"
                  />
                </b-field>

                <b-field
                  :label="$t('settings.performance.batchSize')"
                  label-position="on-border"
                  :message="$t('settings.performance.batchSizeHelp')"
                >
                  <b-numberinput
                    v-model="form['app.batch_size']"
                    name="app.batch_size"
                    type="is-light"
                    placeholder="1000"
                    min="1"
                    max="100000"
                  />
                </b-field>

                <b-field
                  :label="$t('settings.performance.maxErrThreshold')"
                  label-position="on-border"
                  :message="$t('settings.performance.maxErrThresholdHelp')"
                >
                  <b-numberinput
                    v-model="form['app.max_send_errors']"
                    name="app.max_send_errors"
                    type="is-light"
                    placeholder="1999"
                    min="0"
                    max="100000"
                  />
                </b-field>

                <div>
                  <div class="columns">
                    <div class="column is-6">
                      <b-field
                        :label="$t('settings.performance.slidingWindow')"
                        :message="$t('settings.performance.slidingWindowHelp')"
                      >
                        <b-switch
                          v-model="form['app.message_sliding_window']"
                          name="app.message_sliding_window"
                        />
                      </b-field>
                    </div>

                    <div
                      class="column is-3"
                      :class="{ disabled: !form['app.message_sliding_window'] }"
                    >
                      <b-field
                        :label="$t('settings.performance.slidingWindowRate')"
                        label-position="on-border"
                        :message="$t('settings.performance.slidingWindowRateHelp')"
                      >
                        <b-numberinput
                          v-model="form['app.message_sliding_window_rate']"
                          name="sliding_window_rate"
                          type="is-light"
                          controls-position="compact"
                          :disabled="!form['app.message_sliding_window']"
                          placeholder="25"
                          min="1"
                          max="10000000"
                        />
                      </b-field>
                    </div>

                    <div
                      class="column is-3"
                      :class="{ disabled: !form['app.message_sliding_window'] }"
                    >
                      <b-field
                        :label="$t('settings.performance.slidingWindowDuration')"
                        label-position="on-border"
                        :message="$t('settings.performance.slidingWindowDurationHelp')"
                      >
                        <b-input
                          v-model="form['app.message_sliding_window_duration']"
                          name="sliding_window_duration"
                          :disabled="!form['app.message_sliding_window']"
                          placeholder="1h"
                          :pattern="regDuration"
                          :maxlength="10"
                        />
                      </b-field>
                    </div>
                  </div>
                </div>
                <!-- sliding window -->
              </div>
            </b-tab-item>
            <!-- performance -->

            <b-tab-item :label="$t('settings.privacy.name')">
              <div class="items">
                <b-field
                  :label="$t('settings.privacy.individualSubTracking')"
                  :message="$t('settings.privacy.individualSubTrackingHelp')"
                >
                  <b-switch
                    v-model="form['privacy.individual_tracking']"
                    name="privacy.individual_tracking"
                  />
                </b-field>

                <b-field
                  :label="$t('settings.privacy.listUnsubHeader')"
                  :message="$t('settings.privacy.listUnsubHeaderHelp')"
                >
                  <b-switch
                    v-model="form['privacy.unsubscribe_header']"
                    name="privacy.unsubscribe_header"
                  />
                </b-field>

                <b-field
                  :label="$t('settings.privacy.allowBlocklist')"
                  :message="$t('settings.privacy.allowBlocklistHelp')"
                >
                  <b-switch
                    v-model="form['privacy.allow_blocklist']"
                    name="privacy.allow_blocklist"
                  />
                </b-field>

                <b-field
                  :label="$t('settings.privacy.allowExport')"
                  :message="$t('settings.privacy.allowExportHelp')"
                >
                  <b-switch v-model="form['privacy.allow_export']" name="privacy.allow_export" />
                </b-field>

                <b-field
                  :label="$t('settings.privacy.allowWipe')"
                  :message="$t('settings.privacy.allowWipeHelp')"
                >
                  <b-switch v-model="form['privacy.allow_wipe']" name="privacy.allow_wipe" />
                </b-field>
              </div> </b-tab-item
            ><!-- privacy -->

            <b-tab-item :label="$t('settings.media.title')">
              <div class="items">
                <b-field :label="$t('settings.media.provider')" label-position="on-border">
                  <b-select v-model="form['upload.provider']" name="upload.provider">
                    <option value="filesystem">filesystem</option>
                    <option value="s3">s3</option>
                  </b-select>
                </b-field>

                <div class="block" v-if="form['upload.provider'] === 'filesystem'">
                  <b-field
                    :label="$t('settings.media.upload.path')"
                    label-position="on-border"
                    :message="$t('settings.media.upload.pathHelp')"
                  >
                    <b-input
                      v-model="form['upload.filesystem.upload_path']"
                      name="app.upload_path"
                      placeholder="/home/listmonk/uploads"
                      :maxlength="200"
                    />
                  </b-field>

                  <b-field
                    :label="$t('settings.media.upload.uri')"
                    label-position="on-border"
                    :message="$t('settings.media.upload.uriHelp')"
                  >
                    <b-input
                      v-model="form['upload.filesystem.upload_uri']"
                      name="app.upload_uri"
                      placeholder="/uploads"
                      :maxlength="200"
                    />
                  </b-field>
                </div>
                <!-- filesystem -->

                <div class="block" v-if="form['upload.provider'] === 's3'">
                  <div class="columns">
                    <div class="column is-3">
                      <b-field
                        :label="$t('settings.media.s3.region')"
                        label-position="on-border"
                        expanded
                      >
                        <b-input
                          v-model="form['upload.s3.aws_default_region']"
                          name="upload.s3.aws_default_region"
                          :maxlength="200"
                          placeholder="ap-south-1"
                        />
                      </b-field>
                    </div>
                    <div class="column">
                      <b-field grouped>
                        <b-field
                          :label="$t('settings.media.s3.key')"
                          label-position="on-border"
                          expanded
                        >
                          <b-input
                            v-model="form['upload.s3.aws_access_key_id']"
                            name="upload.s3.aws_access_key_id"
                            :maxlength="200"
                          />
                        </b-field>
                        <b-field
                          :label="$t('settings.media.s3.secret')"
                          label-position="on-border"
                          expanded
                          message="Enter a value to change."
                        >
                          <b-input
                            v-model="form['upload.s3.aws_secret_access_key']"
                            name="upload.s3.aws_secret_access_key"
                            type="password"
                            :maxlength="200"
                          />
                        </b-field>
                      </b-field>
                    </div>
                  </div>

                  <div class="columns">
                    <div class="column is-3">
                      <b-field
                        :label="$t('settings.media.s3.bucketType')"
                        label-position="on-border"
                      >
                        <b-select
                          v-model="form['upload.s3.bucket_type']"
                          name="upload.s3.bucket_type"
                          expanded
                        >
                          <option value="private">
                            {{ $t("settings.media.s3.bucketTypePrivate") }}
                          </option>
                          <option value="public">
                            {{ $t("settings.media.s3.bucketTypePublic") }}
                          </option>
                        </b-select>
                      </b-field>
                    </div>
                    <div class="column">
                      <b-field grouped>
                        <b-field
                          :label="$t('settings.media.s3.bucket')"
                          label-position="on-border"
                          expanded
                        >
                          <b-input
                            v-model="form['upload.s3.bucket']"
                            name="upload.s3.bucket"
                            :maxlength="200"
                            placeholder=""
                          />
                        </b-field>
                        <b-field
                          :label="$t('settings.media.s3.bucketPath')"
                          label-position="on-border"
                          :message="$t('settings.media.s3.bucketPathHelp')"
                          expanded
                        >
                          <b-input
                            v-model="form['upload.s3.bucket_path']"
                            name="upload.s3.bucket_path"
                            :maxlength="200"
                            placeholder="/"
                          />
                        </b-field>
                      </b-field>
                    </div>
                  </div>
                  <div class="columns">
                    <div class="column is-3">
                      <b-field
                        :label="$t('settings.media.s3.uploadExpiry')"
                        label-position="on-border"
                        :message="$t('settings.media.s3.uploadExpiryHelp')"
                        expanded
                      >
                        <b-input
                          v-model="form['upload.s3.expiry']"
                          name="upload.s3.expiry"
                          placeholder="14d"
                          :pattern="regDuration"
                          :maxlength="10"
                        />
                      </b-field>
                    </div>
                  </div>
                </div>
                <!-- s3 -->
              </div>
            </b-tab-item>
            <!-- media -->

            <b-tab-item label="Providers" label-position="on-border">
              <div class="items">
                <Steps @submitProvider="onSubmit" @removeConnection="removeConnection" />
              </div>
            </b-tab-item>
            <!-- Providers  -->

            <b-tab-item :label="$t('settings.smtp.name')">
              <SmtpService
                :dataSmtp="form.smtp"
                :loadingSetting="loading.settings"
                :loading="isLoading"
                :duration="regDuration"
                @newSmtp="addSMTP"
                @removeSmtp="removeSMTP"
                @showHeader="showSMTPHeaders"
              />

              <!-- mail-servers -->
            </b-tab-item>

            <!-- mail servers -->

            <b-tab-item :label="$t('settings.messengers.name')">
              <div class="items messengers">
                <div class="block box" v-for="(item, n) in form.messengers" :key="n">
                  <div class="columns">
                    <div class="column is-2">
                      <b-field :label="$t('globals.buttons.enabled')">
                        <b-switch v-model="item.enabled" name="enabled" :native-value="true" />
                      </b-field>
                      <b-field>
                        <a
                          @click.prevent="$utils.confirm(null, () => removeMessenger(n))"
                          href="#"
                          class="is-size-7"
                        >
                          <b-icon icon="trash-can-outline" size="is-small" />
                          {{ $t("globals.buttons.delete") }}
                        </a>
                      </b-field>
                    </div>
                    <!-- first column -->

                    <div class="column" :class="{ disabled: !item.enabled }">
                      <div class="columns">
                        <div class="column is-4">
                          <b-field
                            :label="$t('globals.fields.name')"
                            label-position="on-border"
                            :message="$t('settings.messengers.nameHelp')"
                          >
                            <b-input
                              v-model="item.name"
                              name="name"
                              placeholder="mymessenger"
                              :maxlength="200"
                            />
                          </b-field>
                        </div>
                        <div class="column is-8">
                          <b-field
                            :label="$t('settings.messengers.url')"
                            label-position="on-border"
                            :message="$t('settings.messengers.urlHelp')"
                          >
                            <b-input
                              v-model="item.root_url"
                              name="root_url"
                              placeholder="https://postback.messenger.net/path"
                              :maxlength="200"
                            />
                          </b-field>
                        </div>
                      </div>
                      <!-- host -->

                      <div class="columns">
                        <div class="column">
                          <b-field grouped>
                            <b-field
                              :label="$t('settings.messengers.username')"
                              label-position="on-border"
                              expanded
                            >
                              <b-input v-model="item.username" name="username" :maxlength="200" />
                            </b-field>
                            <b-field
                              :label="$t('settings.messengers.password')"
                              label-position="on-border"
                              expanded
                              :message="$t('globals.messages.passwordChange')"
                            >
                              <b-input
                                v-model="item.password"
                                name="password"
                                type="password"
                                :placeholder="$t('globals.messages.passwordChange')"
                                :maxlength="200"
                              />
                            </b-field>
                          </b-field>
                        </div>
                      </div>
                      <!-- auth -->
                      <hr />

                      <div class="columns">
                        <div class="column is-4">
                          <b-field
                            :label="$t('settings.messengers.maxConns')"
                            label-position="on-border"
                            :message="$t('settings.messengers.maxConnsHelp')"
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
                        <div class="column is-4">
                          <b-field
                            :label="$t('settings.messengers.retries')"
                            label-position="on-border"
                            :message="$t('settings.messengers.retriesHelp')"
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
                        <div class="column is-4">
                          <b-field
                            :label="$t('settings.messengers.timeout')"
                            label-position="on-border"
                            :message="$t('settings.messengers.timeoutHelp')"
                          >
                            <b-input
                              v-model="item.timeout"
                              name="timeout"
                              placeholder="5s"
                              :pattern="regDuration"
                              :maxlength="10"
                            />
                          </b-field>
                        </div>
                      </div>
                      <hr />
                    </div>
                  </div>
                  <!-- second container column -->
                </div>
                <!-- block -->
              </div>
              <!-- mail-servers -->

              <b-button @click="addMessenger" icon-left="plus" type="is-primary">
                {{ $t("globals.buttons.addNew") }}
              </b-button> </b-tab-item
            ><!-- messengers -->
          </b-tabs>
        </form>
      </section>
    </section>
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
      submitButton: false,

      // formCopy is a stringified copy of the original settings against which
      // form is compared to detect changes.
      formCopy: "",
      form: {}
    };
  },

  methods: {
    getIndexByMessenger(value) {
      return this.form.providers.findIndex(
        element => element.messenger.toLowerCase() === value.toLowerCase()
      );
    },

    matchToProductName(id, value) {
      return this.form.providers[id].product.findIndex(
        element => element.name.toLowerCase() === value.toLowerCase()
      );
    },

    inputProvider() {},
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

    addProvider(value) {
      this.form.providers.push(value);
    },

    removeConnection() {
      let index = this.$store.getters["providers/dataRemove"];

      this.form.providers[index.messengerIndex].product[index.productIndex].connection.splice(
        index.conectionIndex,
        1
      );

      this.$utils.confirm("Are you Sure Delete This Connection.. ??", () => {
        this.onSubmit();
      });
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

        for (let i = 0; i < d.providers.length; i++) {
          d.providers[i].product.forEach(element => {
            element.connection.forEach(element => {
              element.strEmailHeaders = JSON.stringify(element.email_headers);
            });
          });
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
        this.$store.dispatch("providers/setDataForm", this.form);
        this.formCopy = JSON.stringify(d); // Copy from
        this.isLoading = false;
      });
    }
  },

  computed: {
    ...mapState(["serverConfig", "loading"])

    // hasFormChanged() {
    //   if (!this.formCopy) {
    //     // console.log(this.formCopy);  // Duplicate data from form
    //     return false;
    //   }
    //   return JSON.stringify(this.form) !== this.formCopy;
    // }
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
