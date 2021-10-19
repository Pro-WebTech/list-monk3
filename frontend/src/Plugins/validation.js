import Vue from "vue";
import { messages } from "vee-validate/dist/locale/en.json";
import {
  ValidationProvider,
  ValidationObserver,
  extend,
  Rules,
  setInteractionMode
} from "vee-validate/dist/vee-validate.full.esm";

const Validate = {
  install(Vue) {
    Vue.component("ValidationProvider", ValidationProvider);
    Vue.component("ValidationObserver", ValidationObserver);
    setInteractionMode("eager");
    Object.keys(Rules).forEach(rule => {
      extend(rule, {
        ...Rules[rule],
        message: messages[rule]
      });
    });
  }
};

export default Validate;
