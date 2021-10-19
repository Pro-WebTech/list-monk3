export const validationFunction = {
  watch: {
    tags(value) {
      if (value.length == 0) this.tagDangerActive("Tags Is Required");

      this.validate();
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

    checkHeaders(e) {
      if (e.length > 0) {
        if (e === "[]") {
          this.hiddenHeaderError();
        } else {
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
    headerError() {
      let getErrorMessage = document.querySelector(".fieldHeader").nextElementSibling;

      this.finishButtonDisable = true;

      if (getErrorMessage.classList.contains("is-hidden")) {
        getErrorMessage.classList.remove("is-hidden", "help");
        getErrorMessage.classList.add("warningColor");
      } else {
        getErrorMessage.classList.replace("help", "warningColor");
      }
    },
    hiddenHeaderError() {
      let getErrorHeader = document.querySelector(".fieldHeader");
      getErrorHeader.nextElementSibling.classList.add("is-hidden");
      //Also checking required field (username ,  password and tags) there are value or not because are required
      if (this.username.length && this.password.length && this.tags.length > 0) {
        this.finishButtonDisable = false;
      }
    },
    headerProccess(parsing) {
      //check Array Or not
      if (Array.isArray(parsing)) {
        //Checking array inside there are element with type Object or not
        parsing.forEach(element => {
          if (typeof element == "object") {
            let getAllKeys = Object.keys(element);

            if (getAllKeys.length > 0) {
              this.hiddenHeaderError();
            }
          }
        });
      }
    },

    backPreviousComponnent() {
      this.$emit("changeComponent", "ProvidersMarket");
    },

    requiredField() {
      return this.username.length && this.password.length > 0;
    },
    usernameCheck() {
      return this.$refs.usernameProvider.validate();
    },
    passwordCheck() {
      return this.$refs.passwordProvider.validate();
    },
    checkTagsLength() {
      return this.tags.length === 0;
    },

    async validationUsernamePassword() {
      /// Running Validation From Validate Per tag <ValidationProvider>
      const [username, password] = await Promise.all([this.usernameCheck(), this.passwordCheck()]);
      /// Conditional When at All Field True Or False

      // username.valid && password.valid === true
      //   ? (this.finishButtonDisable = false)
      //   : (this.finishButtonDisable = true);

      if (username.valid && password.valid === true) {
        if (this.tags.length === 0) {
          console.log(this.tags.length);
          this.finishButtonDisable = true;
          this.tagDangerActive("Tags Is Required");
        } else {
          this.finishButtonDisable = false;
        }
      } else {
        this.finishButtonDisable = true;
      }
    },

    validate() {
      // Check Required
      this.requiredField() ? this.validationUsernamePassword() : (this.finishButtonDisable = true);
    },

    checkErrorTag() {
      let error = document.querySelector(".getTags p.warningColor");

      if (this.tags.length == 0) {
        this.tagDangerActive("Tags Is Required");
      } else if (error !== null) {
        this.tagDangerRemove();
      }

      //if (error !== null) this.tagDangerRemove();
    },

    checkTag(value) {
      let rule = /[ `!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?~]/;

      //Checking One of all input tags with regex
      if (rule.test(value)) {
        this.tagDangerActive("Just Text and Number , For Special Character Not Allowed");
      } else if (this.checkDuplicatedTag(value)) {
        this.tagDangerActive("Duplicated not allowed");
      } else {
        // If tags Success add / Not Error And then running Required
        this.tagDangerRemove();
        this.validate();
      }

      return rule.test(value) ? false : true;
    },
    checkDuplicatedTag(value) {
      let status;
      if (this.tags.length > 0) {
        this.tags.forEach(element => {
          if (element == value) {
            status = true;
            this.duplicated = false;
          }
        });
      }
      return status;
    },
    tagDangerActive(message) {
      let getTagsField = document.querySelector(".getTags");
      let tagSpan = document.createElement("p");
      let error = document.querySelector(".getTags p.warningColor");

      this.danger = true;
      // Checking Element tag error existing or not
      if (error == null) {
        this.finishButtonDisable = true;

        tagSpan.className = "warningColor";

        tagSpan.innerText = message;

        getTagsField.appendChild(tagSpan);
      }
    },
    tagDangerRemove() {
      let getTag = document.querySelector(".getTags p.warningColor");
      this.danger = false;
      if (getTag !== null) {
        this.requiredField() ? (this.finishButtonDisable = false) : false;

        getTag.remove();
      }
    }
  }
};
