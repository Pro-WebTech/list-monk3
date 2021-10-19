<template>
  <div class="is-fullheight">
    <div class="hero-body">
      <div class="container">
        <div class="column is-4 is-offset-4">
          <div class="is-flex is-justify-content-center py-5 my-2">
            <img ref="image" alt="" srcset="" class="" />
          </div>

          <div class="box" v-show="formShow">
            <ValidationObserver ref="form">
              <ValidationProvider v-slot="{ errors }" name="Username" rules="required">
                <div class="field p-3">
                  <label class="label py-2">Username</label>
                  <div class="control">
                    <input
                      class="input"
                      type="text"
                      @keyup.enter="validate"
                      :class="{ 'is-danger': errors[0] }"
                      placeholder="Username input"
                      v-model="dataLogin.username"
                    />
                  </div>
                  <span v-if="errors[0]" class="warningColor">{{ errors[0] }}</span>
                </div>
              </ValidationProvider>
              <ValidationProvider v-slot="{ errors }" name="Password" rules="required">
                <div class="field p-3">
                  <label class="label py-2">Password</label>
                  <div class="control">
                    <input
                      class="input"
                      @keyup.enter="validate"
                      :class="{ 'is-danger': errors[0] }"
                      type="password"
                      placeholder="Password input"
                      v-model="dataLogin.code"
                    />
                  </div>
                  <span v-if="errors[0]" class="warningColor">{{ errors[0] }}</span>
                </div>
              </ValidationProvider>
            </ValidationObserver>
            <div class="field p-3">
              <b-button @click="validate" type="is-primary">Login</b-button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  data() {
    return {
      image: "",
      formShow: true,
      hasError: "",
      dataLogin: {
        username: "",
        code: ""
      }
    };
  },

  methods: {
    validate() {
      this.$refs.form.validate().then(success => {
        success ? this.submit() : false;
      });
    },

    submit() {
      this.axios({
        method: "POST",
        url: "/login",
        headers: {
          "Content-Type": "application/json"
        },
        data: {
          username: this.dataLogin.username.trim(),
          code: this.dataLogin.code.trim()
        }
      })
        .then(result => {
          let { data } = result;

          data.code != 200 ? this.errorLogin(data.message) : this.successLogin(data.data.token);
        })
        .catch(err => {
          console.log(err);
        });
    },

    successLogin(token) {
      localStorage.setItem("JWT", token);
      window.location.reload();
      this.$router.push({ name: "dashboard" });
    },
    errorLogin(message) {
      this.$buefy.toast.open({
        duration: 5000,
        message: message,
        position: "is-top",
        type: "is-danger"
      });
    },

    getImageLink() {
      this.axios.get("/public/asset/logo").then(res => {
        let { value } = res.data.data;

        localStorage.setItem("logo", value);

        this.showFormAndImage(value);
      });
    },
    getImageFromLocal() {
      let getlocalImage = localStorage.getItem("logo");
      this.showFormAndImage(getlocalImage);
    },

    showFormAndImage(image) {
      this.$refs.image.src = image;
      this.$refs.image.width = 300;
      this.$refs.image.height = 300;
      this.formShow = true;
    }
  },
  mounted() {
    let getImage = localStorage.getItem("logo");

    getImage == null ? this.getImageLink() : this.getImageFromLocal();
  }
};
</script>

<style scoped>
.warningColor {
  color: #f14668;
  font-size: 12px;
  padding: 5px 0px 5px 0px;
}
</style>
