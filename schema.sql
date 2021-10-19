DROP TYPE IF EXISTS list_type CASCADE; CREATE TYPE list_type AS ENUM ('public', 'private', 'temporary');
DROP TYPE IF EXISTS list_optin CASCADE; CREATE TYPE list_optin AS ENUM ('single', 'double');
DROP TYPE IF EXISTS subscriber_status CASCADE; CREATE TYPE subscriber_status AS ENUM ('enabled', 'disabled', 'blocklisted');
DROP TYPE IF EXISTS subscription_status CASCADE; CREATE TYPE subscription_status AS ENUM ('unconfirmed', 'confirmed', 'unsubscribed');
DROP TYPE IF EXISTS campaign_status CASCADE; CREATE TYPE campaign_status AS ENUM ('draft', 'running', 'scheduled', 'paused', 'cancelled', 'finished');
DROP TYPE IF EXISTS campaign_type CASCADE; CREATE TYPE campaign_type AS ENUM ('regular', 'optin');
DROP TYPE IF EXISTS content_type CASCADE; CREATE TYPE content_type AS ENUM ('richtext', 'html', 'plain', 'markdown');

-- subscribers
DROP TABLE IF EXISTS subscribers CASCADE;
CREATE TABLE subscribers (
    id              SERIAL PRIMARY KEY,
    uuid uuid       NOT NULL UNIQUE,
    email           TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    attribs         JSONB NOT NULL DEFAULT '{}',
    status          subscriber_status NOT NULL DEFAULT 'enabled',
    campaigns       INTEGER[],

    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_email_sent TIMESTAMP WITH TIME ZONE DEFAULT NOW() - '3 years'::interval,
    last_email_open TIMESTAMP WITH TIME ZONE DEFAULT NOW() - '3 years'::interval,
    last_email_clicked TIMESTAMP WITH TIME ZONE DEFAULT NOW() - '3 years'::interval
);
DROP INDEX IF EXISTS idx_subs_email; CREATE UNIQUE INDEX idx_subs_email ON subscribers(LOWER(email));
DROP INDEX IF EXISTS idx_subs_status; CREATE INDEX idx_subs_status ON subscribers(status);

-- lists
DROP TABLE IF EXISTS lists CASCADE;
CREATE TABLE lists (
    id              SERIAL PRIMARY KEY,
    uuid            uuid NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    type            list_type NOT NULL,
    optin           list_optin NOT NULL DEFAULT 'single',
    tags            VARCHAR(100)[],

    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
INSERT INTO public.lists
(uuid, "name", "type", optin, tags)
VALUES('5b2082e8-d500-44c2-ab66-6e7c7880ee0d', 'SMART-YAHOO', 'private', 'single', NULL);
INSERT INTO public.lists
(uuid, "name", "type", optin, tags)
VALUES('ce95b46c-b248-4f89-8c23-7abb66f1d40b', 'SMART-OPENED', 'private', 'single', NULL);
INSERT INTO public.lists
(uuid, "name", "type", optin, tags)
VALUES('9dffde4f-32bf-46a2-8427-95eae46cc187', 'SMART-CLICKED', 'private', 'single', NULL);
INSERT INTO public.lists
(uuid, "name", "type", optin, tags)
VALUES('274a51b0-1e39-448f-b1bd-75f3c56e8031', 'SMART-AOL', 'private', 'single', NULL);
INSERT INTO public.lists
(uuid, "name", "type", optin, tags)
VALUES('86dd083d-85c6-46d3-b01b-deb981d41327', 'SMART-HOTMAIL', 'private', 'single', NULL);
INSERT INTO public.lists
(uuid, "name", "type", optin, tags)
VALUES('8ac70d78-2269-4207-9c42-b0c4fb5e3454', 'SMART-GMAIL', 'private', 'single', NULL);


DROP TABLE IF EXISTS subscriber_lists CASCADE;
CREATE TABLE subscriber_lists (
    subscriber_id      INTEGER REFERENCES subscribers(id) ON DELETE CASCADE ON UPDATE CASCADE,
    list_id            INTEGER NULL REFERENCES lists(id) ON DELETE CASCADE ON UPDATE CASCADE,
    status             subscription_status NOT NULL DEFAULT 'unconfirmed',

    created_at         TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at         TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    PRIMARY KEY(subscriber_id, list_id)
);
DROP INDEX IF EXISTS idx_sub_lists_sub_id; CREATE INDEX idx_sub_lists_sub_id ON subscriber_lists(subscriber_id);
DROP INDEX IF EXISTS idx_sub_lists_list_id; CREATE INDEX idx_sub_lists_list_id ON subscriber_lists(list_id);
DROP INDEX IF EXISTS idx_sub_lists_status; CREATE INDEX idx_sub_lists_status ON subscriber_lists(status);

-- templates
DROP TABLE IF EXISTS templates CASCADE;
CREATE TABLE templates (
    id              SERIAL PRIMARY KEY,
    name            TEXT NOT NULL,
    body            TEXT NOT NULL,
    is_default      BOOLEAN NOT NULL DEFAULT false,

    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE UNIQUE INDEX ON templates (is_default) WHERE is_default = true;


-- campaigns
DROP TABLE IF EXISTS campaigns CASCADE;
CREATE TABLE campaigns (
    id               SERIAL PRIMARY KEY,
    uuid uuid        NOT NULL UNIQUE,
    name             TEXT NOT NULL,
    subject          TEXT NOT NULL,
    from_email       TEXT NOT NULL,
    body             TEXT NOT NULL,
    altbody          TEXT NULL,
    content_type     content_type NOT NULL DEFAULT 'richtext',
    send_at          TIMESTAMP WITH TIME ZONE,
    status           campaign_status NOT NULL DEFAULT 'draft',
    tags             VARCHAR(100)[],

    -- The subscription statuses of subscribers to which a campaign will be sent.
    -- For opt-in campaigns, this will be 'unsubscribed'.
    type campaign_type DEFAULT 'regular',

    -- The ID of the messenger backend used to send this campaign. 
    messenger        TEXT NOT NULL,
    template_id      INTEGER REFERENCES templates(id) ON DELETE SET DEFAULT DEFAULT 1,

    -- Progress and stats.
    to_send            INT NOT NULL DEFAULT 0,
    sent               INT NOT NULL DEFAULT 0,
    max_subscriber_id  INT NOT NULL DEFAULT 0,
    last_subscriber_id INT NOT NULL DEFAULT 0,

    started_at       TIMESTAMP WITH TIME ZONE,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

DROP TABLE IF EXISTS campaign_lists CASCADE;
CREATE TABLE campaign_lists (
    campaign_id  INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE ON UPDATE CASCADE,

    -- Lists may be deleted, so list_id is nullable
    -- and a copy of the original list name is maintained here.
    list_id      INTEGER NULL REFERENCES lists(id) ON DELETE SET NULL ON UPDATE CASCADE,
    list_name    TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX ON campaign_lists (campaign_id, list_id);
DROP INDEX IF EXISTS idx_camp_lists_camp_id; CREATE INDEX idx_camp_lists_camp_id ON campaign_lists(campaign_id);
DROP INDEX IF EXISTS idx_camp_lists_list_id; CREATE INDEX idx_camp_lists_list_id ON campaign_lists(list_id);

DROP TABLE IF EXISTS campaign_views CASCADE;
CREATE TABLE campaign_views (
    campaign_id      INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE ON UPDATE CASCADE,

    -- Subscribers may be deleted, but the view counts should remain.
    subscriber_id    INTEGER NULL REFERENCES subscribers(id) ON DELETE SET NULL ON UPDATE CASCADE,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
DROP INDEX IF EXISTS idx_views_camp_id; CREATE INDEX idx_views_camp_id ON campaign_views(campaign_id);
DROP INDEX IF EXISTS idx_views_subscriber_id; CREATE INDEX idx_views_subscriber_id ON campaign_views(subscriber_id);

-- media
DROP TABLE IF EXISTS media CASCADE;
CREATE TABLE media (
    id               SERIAL PRIMARY KEY,
    uuid uuid        NOT NULL UNIQUE,
    provider         TEXT NOT NULL DEFAULT '',
    filename         TEXT NOT NULL,
    thumb            TEXT NOT NULL,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- links
DROP TABLE IF EXISTS links CASCADE;
CREATE TABLE links (
    id               SERIAL PRIMARY KEY,
    uuid uuid        NOT NULL UNIQUE,
    url              TEXT NOT NULL UNIQUE,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

DROP TABLE IF EXISTS link_clicks CASCADE;
CREATE TABLE link_clicks (
    campaign_id      INTEGER NULL REFERENCES campaigns(id) ON DELETE CASCADE ON UPDATE CASCADE,
    link_id          INTEGER NOT NULL REFERENCES links(id) ON DELETE CASCADE ON UPDATE CASCADE,

    -- Subscribers may be deleted, but the link counts should remain.
    subscriber_id    INTEGER NULL REFERENCES subscribers(id) ON DELETE SET NULL ON UPDATE CASCADE,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
DROP INDEX IF EXISTS idx_clicks_camp_id; CREATE INDEX idx_clicks_camp_id ON link_clicks(campaign_id);
DROP INDEX IF EXISTS idx_clicks_link_id; CREATE INDEX idx_clicks_link_id ON link_clicks(link_id);
DROP INDEX IF EXISTS idx_clicks_sub_id; CREATE INDEX idx_clicks_sub_id ON link_clicks(subscriber_id);

-- settings
DROP TABLE IF EXISTS settings CASCADE;
CREATE TABLE settings (
    key             TEXT NOT NULL UNIQUE,
    value           JSONB NOT NULL DEFAULT '{}',
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
DROP INDEX IF EXISTS idx_settings_key; CREATE INDEX idx_settings_key ON settings(key);
INSERT INTO settings (key, value) VALUES
    ('app.root_url', '"http://localhost:9000"'),
    ('app.favicon_url', '""'),
    ('app.from_email', '"listmonk <noreply@listmonk.yoursite.com>"'),
    ('app.logo_url', '"http://localhost:9000/public/static/logo.png"'),
    ('app.concurrency', '10'),
    ('app.message_rate', '10'),
    ('app.batch_size', '1000'),
    ('app.max_send_errors', '1000'),
    ('app.message_sliding_window', 'false'),
    ('app.message_sliding_window_duration', '"1h"'),
    ('app.message_sliding_window_rate', '10000'),
    ('app.enable_public_subscription_page', 'true'),
    ('app.check_updates', 'true'),
    ('app.notify_emails', '["admin1@mysite.com", "admin2@mysite.com"]'),
    ('app.lang', '"en"'),
    ('emailsent.total', '0'),
    ('emailsent.allowed', '0'),
    ('smssent.total', '0'),
    ('smssent.allowed', '0'),
    ('pushsent.total', '0'),
    ('pushsent.allowed', '0'),
    ('validations.total', '0'),
    ('validations.allowed', '0'),
    ('privacy.individual_tracking', 'false'),
    ('privacy.unsubscribe_header', 'true'),
    ('privacy.allow_blocklist', 'true'),
    ('privacy.allow_export', 'true'),
    ('privacy.allow_wipe', 'true'),
    ('privacy.exportable', '["profile", "subscriptions", "campaign_views", "link_clicks"]'),
    ('upload.provider', '"filesystem"'),
    ('upload.filesystem.upload_path', '"uploads"'),
    ('upload.filesystem.upload_uri', '"/uploads"'),
    ('upload.s3.aws_access_key_id', '""'),
    ('upload.s3.aws_secret_access_key', '""'),
    ('upload.s3.aws_default_region', '"ap-south-b"'),
    ('upload.s3.bucket', '""'),
    ('upload.s3.bucket_domain', '""'),
    ('upload.s3.bucket_path', '"/"'),
    ('upload.s3.bucket_type', '"public"'),
    ('upload.s3.expiry', '"14d"'),
    ('smtp',
        '[{"enabled":true, "host":"smtp.yoursite.com","port":25,"auth_protocol":"cram","username":"username","password":"password","hello_hostname":"","max_conns":10,"idle_timeout":"15s","wait_timeout":"5s","max_msg_retries":2,"tls_enabled":true,"tls_skip_verify":false,"email_headers":[]},
          {"enabled":false, "host":"smtp2.yoursite.com","port":587,"auth_protocol":"plain","username":"username","password":"password","hello_hostname":"","max_conns":10,"idle_timeout":"15s","wait_timeout":"5s","max_msg_retries":2,"tls_enabled":false,"tls_skip_verify":false,"email_headers":[]}]'),
    ('providers', '[{"messenger":"email_api","name":"email (API)","product":[{"name":"AWS","connection":[{"host":"email.us-east-1.amazonaws.com","port":587,"uuid":"8d5de38b-5c8c-4beb-b869-4985cf336a19","enabled":true,"password":"OE7MIjYazNC+NsTQ6RjfGmwbnMC4n/69izdxJT7o","username":"AKIAQOPR2NBMZZ4R3HHK","max_conns":1000,"tls_enabled":true,"idle_timeout":"15s","wait_timeout":"5s","auth_protocol":"plain","email_headers":[],"hello_hostname":"","max_msg_retries":2,"tls_skip_verify":true}]}]},{"messenger":"email_smtp","name":"email (SMTP)","product":[{"name":"Postmark","connection":[{"host":"smtp.postmarkapp.com","port":587,"uuid":"8d5de38b-5c8c-4beb-b869-4985cf336a18","enabled":true,"password":"","username":"","max_conns":100,"tls_enabled":true,"idle_timeout":"15s","wait_timeout":"5s","auth_protocol":"plain","email_headers":[],"hello_hostname":"","max_msg_retries":2,"tls_skip_verify":true}]},{"name":"Sendinblue","connection":[{"host":"smtp-relay.sendinblue.com","port":587,"uuid":"8d5de38b-5c8c-4beb-b869-4985cf336a17","enabled":true,"password":"","username":"","max_conns":100,"tls_enabled":true,"idle_timeout":"15s","wait_timeout":"5s","auth_protocol":"plain","email_headers":[],"hello_hostname":"","max_msg_retries":2,"tls_skip_verify":true}]}]}]');
    ('messengers', '[]');

CREATE TABLE events (
    id              SERIAL PRIMARY KEY,
    subscriber_id   INTEGER NOT NULL,
    event_type      varchar(100) NOT NULL,
    event_reason    text NOT NULL,
    event_timestamp TIMESTAMP,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    flag_platform int4 NOT NULL DEFAULT 0,
    CONSTRAINT events_pkey PRIMARY KEY (id)
);
DROP INDEX IF EXISTS idx_events_sub_id; CREATE INDEX idx_events_sub_id ON events(subscriber_id);

DROP TABLE IF EXISTS users CASCADE;
CREATE TABLE users (
    id              SERIAL PRIMARY KEY,
    email           varchar(100) NOT NULL UNIQUE,
    pass            varchar(100) NOT NULL,
    username        varchar(100) NOT NULL,
    role_id 		int4 NOT null default 0,
    active          int4 NOT null default 1,
    token_jwt       varchar(200) NOT null default '',
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
DROP INDEX IF EXISTS idx_users_email; CREATE UNIQUE INDEX idx_users_email ON users(LOWER(email));
DROP INDEX IF EXISTS idx_users_active; CREATE INDEX idx_users_active ON users(active);

INSERT INTO users
(id, email, pass, username, role_id, active, token_jwt, created_at, updated_at)
VALUES(1, 'emailitco1@yahoo.com', '$argon2id$v=19$m=65536,t=1,p=4$oGQodCFOfHA0BkJpDMxuOQ$RafSMECVBPC1z+UKztAvHbzb3VcFnr/TddNlBanWBzk', 'listmonk', 1, 1, '', '2021-04-26 12:32:53.817', '2021-04-26 12:32:53.817');


CREATE TABLE "role" (
    id serial NOT NULL,
    "name" varchar(255) NULL,
    description varchar(255) NULL,
    status int4 NULL,
    parent_role_id int4 NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT role_pkey PRIMARY KEY (id)
);

INSERT INTO role
("name", description, status, parent_role_id)
VALUES('ADMIN', 'administrator', 1, 0);

CREATE TABLE menu (
    id serial NOT NULL,
    "name" varchar(255) NULL DEFAULT ''::character varying,
    description varchar(255) NULL DEFAULT ''::character varying,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT menu_pkey PRIMARY KEY (id)
);

INSERT INTO menu
("name",  description)
VALUES('subscribers', 'access subscribers details');

CREATE TABLE menu_access_control (
    id serial NOT NULL,
    menu_id int4 NULL,
    "access" varchar(255) NULL DEFAULT ''::character varying,
    "control" varchar(255) NULL DEFAULT ''::character varying,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT menu_access_control_pkey PRIMARY KEY (id),
    CONSTRAINT menu_access_control_menu_id_menu_id_foreign FOREIGN KEY (menu_id) REFERENCES menu(id) ON UPDATE CASCADE ON DELETE CASCADE
);

INSERT INTO menu_access_control
(menu_id, "access", "control")
VALUES(1, '/v1/api/subscribers*', 'read');

CREATE TABLE privilege (
    id serial NOT NULL,
    role_id int4 NULL,
    menu_id int4 NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT privilege_pkey PRIMARY KEY (id),
    CONSTRAINT privilege_menu_id_fkey FOREIGN KEY (menu_id) REFERENCES menu(id) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT privilege_role_id_fkey FOREIGN KEY (role_id) REFERENCES role(id) ON UPDATE CASCADE ON DELETE CASCADE
);

INSERT INTO privilege
(role_id, menu_id)
VALUES(1, 1);

CREATE TABLE privilege_access_control (
    id serial NOT NULL,
    role_menu_id int4 NULL,
    menu_access_control int4 NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT privilege_access_control_pkey PRIMARY KEY (id),
    CONSTRAINT privilege_access_control_role_menu_access_control_id_foreign FOREIGN KEY (menu_access_control) REFERENCES menu_access_control(id) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT privilege_access_control_role_menu_id_privilege_id_foreign FOREIGN KEY (role_menu_id) REFERENCES privilege(id) ON UPDATE CASCADE ON DELETE CASCADE
);
INSERT INTO privilege_access_control
(role_menu_id, menu_access_control)
VALUES(1, 1);

CREATE TABLE stripe_payment_history (
    id SERIAL PRIMARY KEY,
    product varchar(255) NULL DEFAULT ''::character varying,
    plan_name varchar(255) NULL DEFAULT ''::character varying,
    plan_qty int4 not null default 0,
    event_type varchar(255) NULL DEFAULT ''::character varying,
    status varchar(255) NULL DEFAULT ''::character varying,
    invoice varchar(255) NULL DEFAULT ''::character varying,
    platform varchar(255) NULL DEFAULT ''::character varying,
    email varchar(255) NULL DEFAULT ''::character varying,
    amount int4 not null default 0,
    currency varchar(255) NULL DEFAULT ''::character varying,
    mode varchar(255) NULL DEFAULT ''::character varying,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    raw TEXT NOT null
);
