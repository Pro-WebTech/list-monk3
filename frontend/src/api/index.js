import { ToastProgrammatic as Toast } from "buefy";
import axios from "axios";
import humps from "humps";
import qs from "qs";
import VueAxios from "vue-axios";
import Vue from "vue";
import store from "../store";
import { models } from "../constants";

axios.defaults.baseURL = process.env.BASE_URL;
// eslint-disable-next-line
Vue.use(VueAxios, axios);
// eslint-disable-next-line
export const httpApi = axios;

const http = axios.create({
  baseURL: process.env.BASE_URL,
  withCredentials: false,
  responseType: "json",

  headers: {
    Authorization: `Bearer ${localStorage.getItem("JWT")}`,
    "Access-Control-Allow-Origin": "*",
    "Access-Control-Allow-Methods": "GET,PUT,POST,DELETE,PATCH,OPTIONS",
    "Access-Control-Allow-Headers": "Content-type,Accept,X-Access-Token,X-Key"
  },

  // Override the default serializer to switch params from becoming []id=a&[]id=b ...
  // in GET and DELETE requests to id=a&id=b.
  paramsSerializer: params => qs.stringify(params, { arrayFormat: "repeat" })
});

// Intercept requests to set the 'loading' state of a model.
http.interceptors.request.use(
  config => {
    if ("loading" in config) {
      store.commit("setLoading", { model: config.loading, status: true });
    }
    return config;
  },
  error => Promise.reject(error)
);

// Intercept responses to set them to store.
http.interceptors.response.use(
  resp => {
    // Clear the loading state for a model.
    if ("loading" in resp.config) {
      store.commit("setLoading", { model: resp.config.loading, status: false });
    }

    let data = {};
    if (typeof resp.data.data === "object") {
      data = { ...resp.data.data };
      if (!resp.config.preserveCase) {
        // Transform field case.
        data = humps.camelizeKeys(resp.data.data);
      }
    } else {
      data = resp.data.data;
    }

    // Store the API response for a model.
    if ("store" in resp.config) {
      store.commit("setModelResponse", { model: resp.config.store, data });
    }

    return data;
  },
  err => {
    // Clear the loading state for a model.
    if ("loading" in err.config) {
      store.commit("setLoading", { model: err.config.loading, status: false });
    }

    let msg = "";
    if (err.response.data && err.response.data.message) {
      msg = err.response.data.message;
    } else {
      msg = err.toString();
    }

    if (!err.config.disableToast) {
      Toast.open({
        message: msg,
        type: "is-danger",
        queue: false
      });
    }

    return Promise.reject(err);
  }
);

// API calls accept the following config keys.
// loading: modelName (set's the loading status in the global store: eg: store.loading.lists = true)
// store: modelName (set's the API response in the global store. eg: store.lists: { ... } )

// Health check endpoint that does not throw a toast.
export const getHealth = () => http.get("/api/health", { disableToast: true });

export const reloadApp = () => http.post("/v1/api/admin/reload");

// Dashboard
export const getDashboardCounts = () =>
  http.get("/v1/api/dashboard/counts", { loading: models.dashboard });

export const getDashboardCharts = () =>
  http.get("/v1/api/dashboard/charts", { loading: models.dashboard });

// Lists.
export const initLists = params =>
  http.get("/v1/api/initlists", {
    params: !params ? { per_page: "all" } : params,
    loading: models.lists,
    store: models.lists
  });

export const getLists = params =>
  http.get("/v1/api/lists", {
    params: !params ? { per_page: "all" } : params,
    loading: models.lists,
    store: models.lists
  });

export const createList = data => http.post("/v1/api/lists", data, { loading: models.lists });

export const updateList = data =>
  http.put(`/v1/api/lists/${data.id}`, data, { loading: models.lists });

export const deleteList = id => http.delete(`/v1/api/lists/${id}`, { loading: models.lists });

// Subscribers.
export const getSubscribers = async params =>
  http.get("/v1/api/subscribers", {
    params,
    loading: models.subscribers,
    store: models.subscribers
  });

export const filterSubscribers = async params =>
  http.get(
    `/v1/api/subscribers/filter?maxsubscribers=${params.maxsubscribers}&selection=${params.selection}&type=${params.type}&timerange=${params.timerange}&list=${params.listrange}&campaignlist=${params.listCampaignIDRange}`
  );

export const createSubscriber = data =>
  http.post("/v1/api/subscribers", data, { loading: models.subscribers });

export const updateSubscriber = data =>
  http.put(`/v1/api/subscribers/${data.id}`, data, { loading: models.subscribers });

export const deleteSubscriber = id =>
  http.delete(`/v1/api/subscribers/${id}`, { loading: models.subscribers });

export const addSubscribersToLists = data =>
  http.put("/v1/api/subscribers/lists", data, { loading: models.subscribers });

export const addSubscribersToListsByQuery = data =>
  http.put("/v1/api/subscribers/query/lists", data, { loading: models.subscribers });

export const blocklistSubscribers = data =>
  http.put("/v1/api/subscribers/blocklist", data, { loading: models.subscribers });

export const blocklistSubscribersByQuery = data =>
  http.put("/v1/api/subscribers/query/blocklist", data, { loading: models.subscribers });

export const deleteSubscribers = params =>
  http.delete("/v1/api/subscribers", { params, loading: models.subscribers });

export const deleteSubscribersByQuery = data =>
  http.post("/v1/api/subscribers/query/delete", data, { loading: models.subscribers });

// Subscriber import.
export const importSubscribers = data => http.post("/v1/api/import/subscribers", data);

export const getImportStatus = () => http.get("/v1/api/import/subscribers");

export const getImportLogs = async () =>
  http.get("/v1/api/import/subscribers/logs", { preserveCase: true });

export const stopImport = () => http.delete("/v1/api/import/subscribers");

// Campaigns.
export const getCampaigns = async params =>
  http.get("/v1/api/campaigns", { params, loading: models.campaigns, store: models.campaigns });

export const getCampaign = async id =>
  http.get(`/v1/api/campaigns/${id}`, { loading: models.campaigns });

export const getCampaignStats = async () => http.get("/v1/api/campaigns/running/stats", {});

export const createCampaign = async data =>
  http.post("/v1/api/campaigns", data, { loading: models.campaigns });

export const convertCampaignContent = async data =>
  http.post(`/v1/api/campaigns/${data.id}/content`, data, { loading: models.campaigns });

export const testCampaign = async data =>
  http.post(`/v1/api/campaigns/${data.id}/test`, data, { loading: models.campaigns });

export const updateCampaign = async (id, data) =>
  http.put(`/v1/api/campaigns/${id}`, data, { loading: models.campaigns });

export const changeCampaignStatus = async (id, status) =>
  http.put(`/v1/api/campaigns/${id}/status`, { status }, { loading: models.campaigns });

export const deleteCampaign = async id =>
  http.delete(`/v1/api/campaigns/${id}`, { loading: models.campaigns });

// Media.
export const getMedia = async () =>
  http.get("/v1/api/media", { loading: models.media, store: models.media });

export const uploadMedia = data => http.post("/v1/api/media", data, { loading: models.media });

export const deleteMedia = id => http.delete(`/v1/api/media/${id}`, { loading: models.media });

//Checkout plan
export const checkout = async data => http.post("/v1/api/checkout/email/plan", data);

// Grahql

export const getGraphql = async dataQuery =>
  http.post("https://testmonk.emailitapp.com/v1/api/settings/proxy/graphql", {
    url: process.env.VUE_APP_GRAPHQL_URL,
    header: [
      {
        key: "content-type",
        value: "application/json"
      },
      {
        key: "x-hasura-role",
        value: "anonymous"
      }
    ],
    query: dataQuery
  });

// Templates.
export const createTemplate = async data =>
  http.post("/v1/api/templates", data, { loading: models.templates });

export const getTemplates = async () =>
  http.get("/v1/api/templates", { loading: models.templates, store: models.templates });

export const updateTemplate = async data =>
  http.put(`/v1/api/templates/${data.id}`, data, { loading: models.templates });

export const makeTemplateDefault = async id =>
  http.put(`/v1/api/templates/${id}/default`, {}, { loading: models.templates });

export const deleteTemplate = async id =>
  http.delete(`/v1/api/templates/${id}`, { loading: models.templates });

// Settings.
export const getServerConfig = async () =>
  http.get("/api/config", {
    loading: models.serverConfig,
    store: models.serverConfig,
    preserveCase: true
  });

export const initSetting = async () =>
  http.get("v1/api/initsettings", {
    loading: models.settings,
    store: models.settings,
    preserveCase: true
  });

export const getSettings = async () =>
  http.get("/v1/api/settings", {
    loading: models.settings,
    store: models.settings,
    preserveCase: true
  });

export const updateSettings = async data =>
  http.put("/v1/api/settings", data, { loading: models.settings });

export const getLogs = async () => http.get("/v1/api/logs", { loading: models.logs });

export const getLang = async lang =>
  http.get(`/api/lang/${lang}`, { loading: models.lang, preserveCase: true });
