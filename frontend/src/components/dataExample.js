[
  {
    messenger: "email_api",
    name: "email (API)",
    product: [
      {
        name: "AWS",
        connection: [
          {
            host: "email.us-east-1.amazonaws.com",
            port: 587,
            uuid: "8d5de38b-5c8c-4beb-b869-4985cf336a19",
            enabled: true,
            password: "OE7MIjYazNC+NsTQ6RjfGmwbnMC4n/69izdxJT7o",
            username: "AKIAQOPR2NBMZZ4R3HHK",
            max_conns: 1000,
            tls_enabled: true,
            idle_timeout: "15s",
            wait_timeout: "5s",
            auth_protocol: "plain",
            email_headers: [],
            hello_hostname: "",
            max_msg_retries: 2,
            tls_skip_verify: true
          }
        ]
      },
      {
        name: "Google",
        connection: [
          {
            host: "email.us-east-1.googlecloud.com",
            port: 587,
            uuid: "8d5de38b-5c8c-4beb-b869-4985cf336a19",
            enabled: true,
            password: "OE7MIjYazNC+NsTQ6RjfGmwbnMC4n/69izdxJT7o",
            username: "AKIAQOPR2NBMZZ4R3HHK",
            max_conns: 1000,
            tls_enabled: true,
            idle_timeout: "15s",
            wait_timeout: "5s",
            auth_protocol: "plain",
            email_headers: [],
            hello_hostname: "",
            max_msg_retries: 2,
            tls_skip_verify: true
          }
        ]
      }
    ]
  },
  {
    messenger: "email_smtp",
    name: "email (SMTP)",
    product: [
      {
        name: "Postmark",
        connection: [
          {
            host: "smtp.postmarkapp.com",
            port: 587,
            uuid: "8d5de38b-5c8c-4beb-b869-4985cf336a19",
            enabled: true,
            password: "",
            username: "",
            max_conns: 100,
            tls_enabled: true,
            idle_timeout: "15s",
            wait_timeout: "5s",
            auth_protocol: "plain",
            email_headers: [],
            hello_hostname: "",
            max_msg_retries: 2,
            tls_skip_verify: true
          }
        ]
      },
      {
        name: "Sendinblue",
        connection: [
          {
            host: "smtp-relay.sendinblue.com",
            port: 587,
            uuid: "8d5de38b-5c8c-4beb-b869-4985cf336a19",
            enabled: true,
            password: "",
            username: "",
            max_conns: 100,
            tls_enabled: true,
            idle_timeout: "15s",
            wait_timeout: "5s",
            auth_protocol: "plain",
            email_headers: [],
            hello_hostname: "",
            max_msg_retries: 2,
            tls_skip_verify: true
          }
        ]
      }
    ]
  }
];
