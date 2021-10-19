import { ToastProgrammatic as Toast, DialogProgrammatic as Dialog } from "buefy";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";

dayjs.extend(relativeTime);

const reEmail = /(.+?)@(.+?)/gi;

const htmlEntities = {
  "&": "&amp;",
  "<": "&lt;",
  ">": "&gt;",
  '"': "&quot;",
  "'": "&#39;",
  "/": "&#x2F;",
  "`": "&#x60;",
  "=": "&#x3D;"
};

export default class Utils {
  constructor(i18n) {
    this.i18n = i18n;
  }

  // Parses an ISO timestamp to a simpler form.
  niceDate = (stamp, showTime) => {
    if (!stamp) {
      return "";
    }

    const d = new Date(stamp);
    const day = this.i18n.t(`globals.days.${d.getDay()}`);
    const month = this.i18n.t(`globals.months.${d.getMonth() + 1}`);
    let out = `${day}, ${d.getDate()}`;
    out += ` ${month} ${d.getFullYear()}`;
    if (showTime) {
      out += ` ${d.getHours()}:${d.getMinutes()}`;
    }

    return out;
  };

  duration = (start, end) => dayjs(end).from(dayjs(start), true);

  // Simple, naive, e-mail address check.
  validateEmail = e => e.match(reEmail);

  niceNumber = n => {
    if (n === null || n === undefined) {
      return 0;
    }

    let pfx = "";
    let div = 1;

    if (n >= 1.0e9) {
      pfx = "b";
      div = 1.0e9;
    } else if (n >= 1.0e6) {
      pfx = "m";
      div = 1.0e6;
    } else if (n >= 1.0e4) {
      pfx = "k";
      div = 1.0e3;
    } else {
      return n;
    }

    // Whole number without decimals.
    const out = n / div;
    if (Math.floor(out) === n) {
      return out + pfx;
    }

    return out.toFixed(2) + pfx;
  };

  // https://stackoverflow.com/a/12034334
  escapeHTML = html => html.replace(/[&<>"'`=/]/g, s => htmlEntities[s]);

  // UI shortcuts.
  confirm = (msg, onConfirm, onCancel) => {
    Dialog.confirm({
      scroll: "clip",
      message: !msg ? this.i18n.t("globals.messages.confirm") : msg,
      confirmText: this.i18n.t("globals.buttons.ok"),
      cancelText: this.i18n.t("globals.buttons.cancel"),
      onConfirm,
      onCancel
    });
  };

  prompt = (msg, inputAttrs, onConfirm, onCancel) => {
    Dialog.prompt({
      scroll: "clip",
      message: msg,
      confirmText: this.i18n.t("globals.buttons.ok"),
      cancelText: this.i18n.t("globals.buttons.cancel"),
      inputAttrs: {
        type: "string",
        maxlength: 200,
        ...inputAttrs
      },
      trapFocus: true,
      onConfirm,
      onCancel
    });
  };

  toast = (msg, typ, duration) => {
    Toast.open({
      message: this.escapeHTML(msg),
      type: !typ ? "is-success" : typ,
      queue: false,
      duration: duration || 2000
    });
  };

  // Takes a props.row from a Buefy b-column <td> template and
  // returns a `data-id` attribute which Buefy then applies to the td.
  tdID = row => ({ "data-id": row.id.toString() });
}
