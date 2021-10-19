import Vue from "vue";
import VueRouter from "vue-router";
import { ToastProgrammatic as Toast } from "buefy";

Vue.use(VueRouter);

// The meta.group param is used in App.vue to expand menu group by name.
const routes = [
  {
    path: "/",
    name: "login",
    meta: { title: "Login" },
    component: () => import(/* webpackChunkName: "main" */ "../views/Login.vue")
  },
  {
    path: "/dashboard",
    name: "dashboard",
    meta: { title: "Dashboard", auth: true },
    component: () => import(/* webpackChunkName: "main" */ "../views/Dashboard.vue")
  },
  {
    path: "/lists",
    name: "lists",
    meta: { title: "Lists", group: "lists", auth: true },
    component: () => import(/* webpackChunkName: "main" */ "../views/Lists.vue")
  },
  {
    path: "/lists/forms",
    name: "forms",
    meta: { title: "Forms", group: "lists", auth: true },
    component: () => import(/* webpackChunkName: "main" */ "../views/Forms.vue")
  },
  {
    path: "/subscribers",
    name: "subscribers",
    meta: { title: "Subscribers", group: "subscribers", auth: true },
    component: () => import(/* webpackChunkName: "main" */ "../views/Subscribers.vue")
  },
  {
    path: "/subscribers/import",
    name: "import",
    meta: { title: "Import subscribers", group: "subscribers", auth: true },
    component: () => import(/* webpackChunkName: "main" */ "../views/Import.vue")
  },
  {
    path: "/subscribers/lists/:listID",
    name: "subscribers_list",
    meta: { title: "Subscribers", group: "subscribers", auth: true },
    component: () => import(/* webpackChunkName: "main" */ "../views/Subscribers.vue")
  },
  {
    path: "/subscribers/:id",
    name: "subscriber",
    meta: { title: "Subscribers", group: "subscribers", auth: true },
    component: () => import(/* webpackChunkName: "main" */ "../views/Subscribers.vue")
  },
  {
    path: "/campaigns",
    name: "campaigns",
    meta: { title: "Campaigns", group: "campaigns", auth: true },
    component: () => import(/* webpackChunkName: "main" */ "../views/Campaigns.vue")
  },
  {
    path: "/campaigns/media",
    name: "media",
    meta: { title: "Media", group: "campaigns", auth: true },
    component: () => import(/* webpackChunkName: "main" */ "../views/Media.vue")
  },
  {
    path: "/campaigns/templates",
    name: "templates",
    meta: { title: "Templates", group: "campaigns", auth: true },
    component: () => import(/* webpackChunkName: "main" */ "../views/Templates.vue")
  },
  {
    path: "/campaigns/:id",
    name: "campaign",
    meta: { title: "Campaign", group: "campaigns", auth: true },
    component: () => import(/* webpackChunkName: "main" */ "../views/Campaign.vue")
  },
  {
    path: "/settings",
    name: "settings",
    meta: { title: "Settings", group: "settings", auth: true },
    component: () => import(/* webpackChunkName: "main" */ "../views/Settings.vue")
  },
  {
    path: "/settings/logs",
    name: "logs",
    meta: { title: "Logs", group: "settings", auth: true },
    component: () => import(/* webpackChunkName: "main" */ "../views/Logs.vue")
  },
  {
    path: "/success",
    name: "success",
    meta: { title: "Success", auth: true },
    component: () => import(/* webpackChunkName: "main" */ "../views/SuccessCheckout.vue")
  },
  {
    path: "/integration",
    name: "integration",
    meta: { title: "Integration", auth: true },
    component: () => import(/* webpackChunkName: "main" */ "../views/Integration.vue")
  },
  {
    path: "*",
    meta: { title: "Error" },
    component: () => import(/* webpackChunkName: "main" */ "../views/Error.vue")
  }
];

Vue.router = new VueRouter({
  mode: "history",
  base: process.env.BASE_URL,
  routes,

  scrollBehavior(to) {
    if (to.hash) {
      return { selector: to.hash };
    }
    return { x: 0, y: 0 };
  }
});

Vue.router.beforeEach((to, from, next) => {
  if (to.matched.some(record => record.meta.auth)) {
    if (localStorage.getItem("JWT")) {
      next();
    } else {
      next("/");
      Toast.open({
        message: "You are not logged in yet",
        type: "is-danger",
        queue: false
      });
    }
  } else if ((to.path, "/")) {
    if (localStorage.getItem("JWT")) {
      next({ name: "dashboard" });
    } else {
      next();
    }
  } else {
    next();
  }
});

Vue.router.afterEach(to => {
  Vue.nextTick(() => {
    document.title = `${to.meta.title} / listmonk`;
  });
});

export default Vue.router;
