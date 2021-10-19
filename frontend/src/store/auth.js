export default {
  namespaced: true,

  state: {
    user: []
  },
  mutations: {
    toStoreDataUser: (state, payload) => {
      state.user.push(payload);
    }
  },

  actions: {
    login: ({ commit }, payload) => {
      commit("toStoreDataUser", payload);
    }
  },

  getters: {
    user() {
      return Vue.auth.user();
    },

    impersonating() {
      return Vue.auth.impersonating();
    }
  }
};
