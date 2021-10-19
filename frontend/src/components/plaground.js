let providers = [
  {
    messenger: "email_api",
    name: "email (API)",
    product: [
      {
        name: "AWS",
        connection: [
          {
            uuid: "8d5de38b-5c8c-4beb-b869-4985cf336a19",
            enabled: true,
            host: "email-smtp.ap-northeast-1.amazonaws.com",
            hello_hostname: "",
            port: 587,
            auth_protocol: "plain",
            username: "AKIAQOPR2NBMZZ4R3HHK",
            password: "OE7MIjYazNC+NsTQ6RjfGmwbnMC4n/69izdxJT7o",
            email_headers: [],
            max_conns: 1000,
            max_msg_retries: 2,
            idle_timeout: "15s",
            wait_timeout: "5s",
            tls_enabled: true,
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
            uuid: "8d5de38b-5c8c-4beb-b869-4985cf336a18",
            enabled: true,
            host: "smtp.postmarkapp.com",
            hello_hostname: "",
            port: 587,
            auth_protocol: "plain",
            username: "",
            email_headers: [],
            max_conns: 100,
            max_msg_retries: 2,
            idle_timeout: "15s",
            wait_timeout: "5s",
            tls_enabled: true,
            tls_skip_verify: true
          }
        ]
      },
      {
        name: "Sendinblue",
        connection: [
          {
            uuid: "8d5de38b-5c8c-4beb-b869-4985cf336a17",
            enabled: true,
            host: "smtp-relay.sendinblue.com",
            hello_hostname: "",
            port: 587,
            auth_protocol: "plain",
            username: "trial1",
            password: "123",
            email_headers: [],
            max_conns: 100,
            max_msg_retries: 2,
            idle_timeout: "15s",
            wait_timeout: "5s",
            tls_enabled: true,
            tls_skip_verify: true
          }
        ]
      }
    ]
  }
];


