export default {
  namespaced: true,

  state: {
    allProviders: [],
    dataForm: "",
    dataHost: "",
    dataUser: "",
    finalChoiceHost: "",
    indexProviders: "",
    dataRemove: ""
  },
  mutations: {
    toStoreAllProvider: (state, payload) => {
      state.dataHost = payload;

      function getIndexByMessenger(messenger) {
        return state.dataForm.findIndex(
          element => element.messenger.toLowerCase() === messenger.toLowerCase()
        );
      }

      console.log(getIndexByMessenger(payload.messenger));

      function matchToProductName(id, productCode) {
        let status = state.dataForm[id].product.findIndex(
          element => element.name.toLowerCase() === productCode.toLowerCase()
        );

        // console.log(state.dataForm[1].product[status]); //caranya sudah benar
      }

      matchToProductName(getIndexByMessenger(payload.messenger), payload.productCode);

      console.log(payload);
    },
    toCollectDataUser: (state, payload) => {
      state.dataUser = payload;
    },
    toCollectDataForm: (state, payload) => {
      state.dataForm = payload;
    },
    toCollectedChoiceData: (state, payload) => {
      state.finalChoiceHost = payload;
    },
    newHost: (state, payload) => {
      state.dataForm.providers[payload.id].product[payload.productId].connection.find(
        element => (element.host = payload.host)
      );
    },
    toRemoveConnection: (state, payload) => {
      // console.log(payload);
      state.dataRemove = payload;
    }
  },

  actions: {
    setProvider: ({ commit }, payload) => {
      commit("toStoreAllProvider", payload);
    },
    setDataUser: ({ commit }, payload) => {
      commit("toCollectDataUser", payload);
    },
    setDataForm: ({ commit }, payload) => {
      commit("toCollectDataForm", payload);
    },
    setChoiceData: ({ commit }, payload) => {
      commit("toCollectedChoiceData", payload);
    },
    setNewHost: ({ commit }, payload) => {
      commit("newHost", payload);
    },
    setRemoveConnection: ({ commit }, payload) => {
      commit("toRemoveConnection", payload);
    }
  },

  getters: {
    dataHost: state => state.dataHost.settings.hostname, // get all data host
    hostInformation: state => state.dataHost, // All information host like usernam & password , etc
    finalChoiceHost: state => state.finalChoiceHost,
    dataForm: state => state.dataForm,
    dataRemove: state => state.dataRemove
  }
};
