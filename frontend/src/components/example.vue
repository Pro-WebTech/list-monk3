<template>
  <div>
    <div class="columns">
      <!-- <div class="column">
        <b-field
          :label="$t('settings.smtp.host')"
          class="is-size-7"
          label-position="on-border"
          :message="$t('settings.smtp.hostHelp')"
        >
          <b-select
            v-model="selectedHostname"
            expanded
            @input="inputText($event, n)"
            class="is-size-6"
            placeholder="Select a Hostname"
          >
            <option
              v-for="(data, index) in hostname"
              :key="index"
              :value="data.key"
              v-text="data.value"
            ></option>
          </b-select>
        </b-field>
      </div>
      <div class="column">
        <b-field
          :label="$t('settings.smtp.port')"
          label-position="on-border"
          :message="$t('settings.smtp.portHelp')"
        >
          <b-numberinput
            name="port"
            type="is-light"
            controls-position="compact"
            placeholder="25"
            min="1"
            max="65535"
          />
        </b-field>
      </div> -->

      <div class="columns">
        <div class="column is-8">
          <b-field
            :label="$t('settings.smtp.host')"
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
                v-for="(data, index) in hostname"
                :key="index"
                :value="data.key"
                v-text="data"
              ></option>
            </b-select>
          </b-field>
        </div>
        <div class="column">
          <b-field
            :label="$t('settings.smtp.port')"
            label-position="on-border"
            :message="$t('settings.smtp.portHelp')"
          >
            <b-numberinput
              value="12"
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
            <b-select name="auth_protocol">
              <option value="none">none</option>
              <option value="cram">cram</option>
              <option value="plain">plain</option>
              <option value="login">login</option>
            </b-select>
          </b-field>
        </div>
        <div class="column">
          <b-field grouped>
            <b-field :label="$t('settings.smtp.username')" label-position="on-border" expanded>
              <b-input name="username" placeholder="mysmtp" :maxlength="200" />
            </b-field>
            <b-field
              :label="$t('settings.smtp.password')"
              label-position="on-border"
              expanded
              :message="$t('settings.smtp.passwordHelp')"
            >
              <b-input
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
            <b-input name="hello_hostname" placeholder="" :maxlength="200" />
          </b-field>
        </div>
        <div class="column">
          <b-field grouped>
            <b-field
              :label="$t('settings.smtp.tls')"
              expanded
              :message="$t('settings.smtp.tlsHelp')"
            >
              <b-switch name="item.tls_enabled" />
            </b-field>
            <b-field
              :label="$t('settings.smtp.skipTLS')"
              expanded
              :message="$t('settings.smtp.skipTLSHelp')"
            >
              <b-switch name="item.tls_skip_verify" />
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
            <b-input name="idle_timeout" placeholder="15s" :maxlength="10" />
          </b-field>
        </div>
        <div class="column is-3">
          <b-field
            :label="$t('settings.smtp.waitTimeout')"
            label-position="on-border"
            :message="$t('settings.smtp.waitTimeoutHelp')"
          >
            <b-input name="wait_timeout" placeholder="5s" :pattern="duration" :maxlength="10" />
          </b-field>
        </div>
      </div>
      <hr />
    </div>
  </div>
</template>

<script>
export default {};
</script>

<style></style>
