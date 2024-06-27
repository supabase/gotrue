# Auth - Authentication and User Management by Supabase

[![Coverage Status](https://coveralls.io/repos/github/supabase/auth/badge.svg?branch=master)](https://coveralls.io/github/supabase/gotrue?branch=master)

Auth is a user management and authentication server written in Go that powers
[Supabase](https://supabase.com)'s features such as:

- Issuing JWTs
- Row Level Security with PostgREST
- User management
- Sign in with email, password, magic link, phone number
- Sign in with external providers (Google, Apple, Facebook, Discord, ...)

It is originally based on the excellent
[GoTrue codebase by Netlify](https://github.com/netlify/gotrue), however both have diverged significantly in features and capabilities.

If you wish to contribute to the project, please refer to the [contributing guide](/CONTRIBUTING.md).

## Table Of Contents

- [Quick Start](#quick-start)
- [Running in Production](#running-in-production)
- [Configuration](#configuration)
- [Endpoints](#endpoints)

## Quick Start

Create a `.env` file to store your own custom env vars. See [`example.env`](example.env)

1. Start the local postgres database in a postgres container: `docker-compose -f docker-compose-dev.yml up postgres`
2. Build the auth binary: `make build` . You should see an output like this:

```
go build -ldflags "-X github.com/supabase/auth/cmd.Version=`git rev-parse HEAD`"
GOOS=linux GOARCH=arm64 go build -ldflags "-X github.com/supabase/auth/cmd.Version=`git rev-parse HEAD`" -o gotrue-arm64
```

3. Execute the auth binary: `./gotrue`

### If you have docker installed

Create a `.env.docker` file to store your own custom env vars. See [`example.docker.env`](example.docker.env)

1. `make build`
2. `make dev`
3. `docker ps` should show 2 docker containers (`auth_postgresql` and `gotrue_gotrue`)
4. That's it! Visit the [health checkendpoint](http://localhost:9999/health) to confirm that auth is running.

## Running in production

Running an authentication server in production is not an easy feat. We
recommend using [Supabase Auth](https://supabase.com/auth) which gets regular
security updates.

Otherwise, please make sure you setup a process to promptly update to the
latest version. You can do that by following this repository, specifically the
[Releases](https://github.com/supabase/auth/releases) and [Security
Advisories](https://github.com/supabase/auth/security/advisories) sections.

### Backward compatibility

Auth uses the [Semantic Versioning](https://semver.org) scheme. Here are some
further clarifications on backward compatibility guarantees:

**Go API compatibility**

Auth is not meant to be used as a Go library. There are no guarantees on
backward API compatibility when used this way regardless which version number
changes.

**Patch**

Changes to the patch version guarantees backward compatibility with:

- Database objects (tables, columns, indexes, functions).
- REST API
- JWT structure
- Configuration

Guaranteed examples:

- A column won't change its type.
- A table won't change its primary key.
- An index will not be removed.
- A uniqueness constraint will not be removed.
- A REST API will not be removed.
- Parameters to REST APIs will work equivalently as before (or better, if a bug
  has been fixed).
- Configuration will not change.

Not guaranteed examples:

- A table may add new columns.
- Columns in a table may be reordered.
- Non-unique constraints may be removed (database level checks, null, default
  values).
- JWT may add new properties.

**Minor**

Changes to minor version guarantees backward compatibility with:

- REST API
- JWT structure
- Configuration

Exceptions to these guarantees will be made only when serious security issues
are found that can't be remedied in any other way.

Guaranteed examples:

- Existing APIs may be deprecated but continue working for the next few minor
  version releases.
- Configuration changes may become deprecated but continue working for the next
  few minor version releases.
- Already issued JWTs will be accepted, but new JWTs may be with a different
  structure (but usually similar).

Not guaranteed examples:

- Removal of JWT fields after a deprecation notice.
- Removal of certain APIs after a deprecation notice.
- Removal of sign-in with external providers, after a deprecation notice.
- Deletion, truncation, significant schema changes to tables, indexes, views,
  functions.

We aim to provide a deprecation notice in execution logs for at least two major
version releases or two weeks if multiple releases go out. Compatibility will
be guaranteed while the notice is live.

**Major**

Changes to the major version do not guarantee any backward compatibility with
previous versions.

### Inherited features

Certain inherited features from the Netlify codebase are not supported by
Supabase and they may be removed without prior notice in the future. This is a
comprehensive list of those features:

1. Multi-tenancy via the `instances` table i.e. `GOTRUE_MULTI_INSTANCE_MODE`
   configuration parameter.
2. System user (zero UUID user).
3. Super admin via the `is_super_admin` column.
4. Group information in JWTs via `GOTRUE_JWT_ADMIN_GROUP_NAME` and other
   configuration fields.
5. Symmetrics JWTs. In the future it is very likely that Auth will begin
   issuing asymmetric JWTs (subject to configuration), so do not rely on the
   assumption that only HS256 signed JWTs will be issued long term.

Note that this is not an exhaustive list and it may change.

### Best practices when self-hosting

These are some best practices to follow when self-hosting to ensure backward
compatibility with Auth:

1. Do not modify the schema managed by Auth. You can see all of the
   migrations in the `migrations` directory.
2. Do not rely on schema and structure of data in the database. Always use
   Auth APIs and JWTs to infer information about users.
3. Always run Auth behind a TLS-capable proxy such as a load balancer, CDN,
   nginx or other similar software.

## Configuration

You may configure Auth using either a configuration file named `.env`,
environment variables, or a combination of both. Environment variables are prefixed with `GOTRUE_`, and will always have precedence over values provided via file.

### Top-Level

```properties
GOTRUE_SITE_URL=https://example.netlify.com/
```

`SITE_URL` - `string` **required**

The base URL your site is located at. Currently used in combination with other settings to construct URLs used in emails. Any URI that shares a host with `SITE_URL` is a permitted value for `redirect_to` params (see `/authorize` etc.).

`URI_ALLOW_LIST` - `string`

A comma separated list of URIs (e.g. `"https://foo.example.com,https://*.foo.example.com,https://bar.example.com"`) which are permitted as valid `redirect_to` destinations. Defaults to []. Supports wildcard matching through globbing. e.g. `https://*.foo.example.com` will allow `https://a.foo.example.com` and `https://b.foo.example.com` to be accepted. Globbing is also supported on subdomains. e.g. `https://foo.example.com/*` will allow `https://foo.example.com/page1` and `https://foo.example.com/page2` to be accepted.

For more common glob patterns, check out the [following link](https://pkg.go.dev/github.com/gobwas/glob#Compile).

`OPERATOR_TOKEN` - `string` _Multi-instance mode only_

The shared secret with an operator (usually Netlify) for this microservice. Used to verify requests have been proxied through the operator and
the payload values can be trusted.

`DISABLE_SIGNUP` - `bool`

When signup is disabled the only way to create new users is through invites. Defaults to `false`, all signups enabled.

`GOTRUE_EXTERNAL_EMAIL_ENABLED` - `bool`

Use this to disable email signups (users can still use external oauth providers to sign up / sign in)

`GOTRUE_EXTERNAL_PHONE_ENABLED` - `bool`

Use this to disable phone signups (users can still use external oauth providers to sign up / sign in)

`GOTRUE_RATE_LIMIT_HEADER` - `string`

Header on which to rate limit the `/token` endpoint.

`GOTRUE_RATE_LIMIT_EMAIL_SENT` - `string`

Rate limit the number of emails sent per hr on the following endpoints: `/signup`, `/invite`, `/magiclink`, `/recover`, `/otp`, & `/user`.

`GOTRUE_PASSWORD_MIN_LENGTH` - `int`

Minimum password length, defaults to 6.

`GOTRUE_PASSWORD_REQUIRED_CHARACTERS` - a string of character sets separated by `:`. A password must contain at least one character of each set to be accepted. To use the `:` character escape it with `\`.

`GOTRUE_SECURITY_REFRESH_TOKEN_ROTATION_ENABLED` - `bool`

If refresh token rotation is enabled, auth will automatically detect malicious attempts to reuse a revoked refresh token. When a malicious attempt is detected, gotrue immediately revokes all tokens that descended from the offending token.

`GOTRUE_SECURITY_REFRESH_TOKEN_REUSE_INTERVAL` - `string`

This setting is only applicable if `GOTRUE_SECURITY_REFRESH_TOKEN_ROTATION_ENABLED` is enabled. The reuse interval for a refresh token allows for exchanging the refresh token multiple times during the interval to support concurrency or offline issues. During the reuse interval, auth will not consider using a revoked token as a malicious attempt and will simply return the child refresh token.

Only the previous revoked token can be reused. Using an old refresh token way before the current valid refresh token will trigger the reuse detection.

### API

```properties
GOTRUE_API_HOST=localhost
PORT=9999
API_EXTERNAL_URL=http://localhost:9999
```

`API_HOST` - `string`

Hostname to listen on.

`PORT` (no prefix) / `API_PORT` - `number`

Port number to listen on. Defaults to `8081`.

`API_ENDPOINT` - `string` _Multi-instance mode only_

Controls what endpoint Netlify can access this API on.

`API_EXTERNAL_URL` - `string` **required**

The URL on which Gotrue might be accessed at.

`REQUEST_ID_HEADER` - `string`

If you wish to inherit a request ID from the incoming request, specify the name in this value.

### Database

```properties
GOTRUE_DB_DRIVER=postgres
DATABASE_URL=root@localhost/auth
```

`DB_DRIVER` - `string` **required**

Chooses what dialect of database you want. Must be `postgres`.

`DATABASE_URL` (no prefix) / `DB_DATABASE_URL` - `string` **required**

Connection string for the database.

`GOTRUE_DB_MAX_POOL_SIZE` - `int`

Sets the maximum number of open connections to the database. Defaults to 0 which is equivalent to an "unlimited" number of connections.

`DB_NAMESPACE` - `string`

Adds a prefix to all table names.

**Migrations Note**

Migrations are applied automatically when you run `./auth`. However, you also have the option to rerun the migrations via the following methods:

- If built locally: `./auth migrate`
- Using Docker: `docker run --rm auth gotrue migrate`

### Logging

```properties
LOG_LEVEL=debug # available without GOTRUE prefix (exception)
GOTRUE_LOG_FILE=/var/log/go/auth.log
```

`LOG_LEVEL` - `string`

Controls what log levels are output. Choose from `panic`, `fatal`, `error`, `warn`, `info`, or `debug`. Defaults to `info`.

`LOG_FILE` - `string`

If you wish logs to be written to a file, set `log_file` to a valid file path.

### Observability

Auth has basic observability built in. It is able to export
[OpenTelemetry](https://opentelemetry.io) metrics and traces to a collector.

#### Tracing

To enable tracing configure these variables:

`GOTRUE_TRACING_ENABLED` - `boolean`

`GOTRUE_TRACING_EXPORTER` - `string` only `opentelemetry` supported

Make sure you also configure the [OpenTelemetry
Exporter](https://opentelemetry.io/docs/reference/specification/protocol/exporter/)
configuration for your collector or service.

For example, if you use
[Honeycomb.io](https://docs.honeycomb.io/getting-data-in/opentelemetry/go-distro/#using-opentelemetry-without-the-honeycomb-distribution)
you should set these standard OpenTelemetry OTLP variables:

```
OTEL_SERVICE_NAME=auth
OTEL_EXPORTER_OTLP_PROTOCOL=grpc
OTEL_EXPORTER_OTLP_ENDPOINT=https://api.honeycomb.io:443
OTEL_EXPORTER_OTLP_HEADERS="x-honeycomb-team=<API-KEY>,x-honeycomb-dataset=auth"
```

#### Metrics

To enable metrics configure these variables:

`GOTRUE_METRICS_ENABLED` - `boolean`

`GOTRUE_METRICS_EXPORTER` - `string` only `opentelemetry` and `prometheus`
supported

Make sure you also configure the [OpenTelemetry
Exporter](https://opentelemetry.io/docs/reference/specification/protocol/exporter/)
configuration for your collector or service.

If you use the `prometheus` exporter, the server host and port can be
configured using these standard OpenTelemetry variables:

`OTEL_EXPORTER_PROMETHEUS_HOST` - IP address, default `0.0.0.0`

`OTEL_EXPORTER_PROMETHEUS_PORT` - port number, default `9100`

The metrics are exported on the `/` path on the server.

If you use the `opentelemetry` exporter, the metrics are pushed to the
collector.

For example, if you use
[Honeycomb.io](https://docs.honeycomb.io/getting-data-in/opentelemetry/go-distro/#using-opentelemetry-without-the-honeycomb-distribution)
you should set these standard OpenTelemetry OTLP variables:

```
OTEL_SERVICE_NAME=auth
OTEL_EXPORTER_OTLP_PROTOCOL=grpc
OTEL_EXPORTER_OTLP_ENDPOINT=https://api.honeycomb.io:443
OTEL_EXPORTER_OTLP_HEADERS="x-honeycomb-team=<API-KEY>,x-honeycomb-dataset=auth"
```

Note that Honeycomb.io requires a paid plan to ingest metrics.

If you need to debug an issue with traces or metrics not being pushed, you can
set `DEBUG=true` to get more insights from the OpenTelemetry SDK.

#### Custom resource attributes

When using the OpenTelemetry tracing or metrics exporter you can define custom
resource attributes using the [standard `OTEL_RESOURCE_ATTRIBUTES` environment
variable](https://opentelemetry.io/docs/reference/specification/resource/sdk/#specifying-resource-information-via-an-environment-variable).

A default attribute `auth.version` is provided containing the build version.

#### Tracing HTTP routes

All HTTP calls to the Auth API are traced. Routes use the parametrized
version of the route, and the values for the route parameters can be found as
the `http.route.params.<route-key>` span attribute.

For example, the following request:

```
GET /admin/users/4acde936-82dc-4552-b851-831fb8ce0927/
```

will be traced as:

```
http.method = GET
http.route = /admin/users/{user_id}
http.route.params.user_id = 4acde936-82dc-4552-b851-831fb8ce0927
```

#### Go runtime and HTTP metrics

All of the Go runtime metrics are exposed. Some HTTP metrics are also collected
by default.

### JSON Web Tokens (JWT)

```properties
GOTRUE_JWT_SECRET=supersecretvalue
GOTRUE_JWT_EXP=3600
GOTRUE_JWT_AUD=netlify
```

`JWT_SECRET` - `string` **required**

The secret used to sign JWT tokens with.

`JWT_EXP` - `number`

How long tokens are valid for, in seconds. Defaults to 3600 (1 hour).

`JWT_AUD` - `string`

The default JWT audience. Use audiences to group users.

`JWT_ADMIN_GROUP_NAME` - `string`

The name of the admin group (if enabled). Defaults to `admin`.

`JWT_DEFAULT_GROUP_NAME` - `string`

The default group to assign all new users to.

### External Authentication Providers

We support `apple`, `azure`, `bitbucket`, `discord`, `facebook`, `figma`, `github`, `gitlab`, `google`, `keycloak`, `linkedin`, `notion`, `spotify`, `slack`, `twitch`, `twitter` and `workos` for external authentication.

Use the names as the keys underneath `external` to configure each separately.

```properties
GOTRUE_EXTERNAL_GITHUB_ENABLED=true
GOTRUE_EXTERNAL_GITHUB_CLIENT_ID=myappclientid
GOTRUE_EXTERNAL_GITHUB_SECRET=clientsecretvaluessssh
GOTRUE_EXTERNAL_GITHUB_REDIRECT_URI=http://localhost:3000/callback
```

No external providers are required, but you must provide the required values if you choose to enable any.

`EXTERNAL_X_ENABLED` - `bool`

Whether this external provider is enabled or not

`EXTERNAL_X_CLIENT_ID` - `string` **required**

The OAuth2 Client ID registered with the external provider.

`EXTERNAL_X_SECRET` - `string` **required**

The OAuth2 Client Secret provided by the external provider when you registered.

`EXTERNAL_X_REDIRECT_URI` - `string` **required**

The URI a OAuth2 provider will redirect to with the `code` and `state` values.

`EXTERNAL_X_URL` - `string`

The base URL used for constructing the URLs to request authorization and access tokens. Used by `gitlab` and `keycloak`. For `gitlab` it defaults to `https://gitlab.com`. For `keycloak` you need to set this to your instance, for example: `https://keycloak.example.com/realms/myrealm`

#### Apple OAuth

To try out external authentication with Apple locally, you will need to do the following:

1. Remap localhost to \<my_custom_dns \> in your `/etc/hosts` config.
2. Configure auth to serve HTTPS traffic over localhost by replacing `ListenAndServe` in [api.go](api/api.go) with:

   ```
      func (a *API) ListenAndServe(hostAndPort string) {
        log := logrus.WithField("component", "api")
        path, err := os.Getwd()
        if err != nil {
          log.Println(err)
        }
        server := &http.Server{
          Addr:    hostAndPort,
          Handler: a.handler,
        }
        done := make(chan struct{})
        defer close(done)
        go func() {
          waitForTermination(log, done)
          ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
          defer cancel()
          server.Shutdown(ctx)
        }()
        if err := server.ListenAndServeTLS("PATH_TO_CRT_FILE", "PATH_TO_KEY_FILE"); err != http.ErrServerClosed {
          log.WithError(err).Fatal("http server listen failed")
        }
    }
   ```

3. Generate the crt and key file. See [here](https://www.freecodecamp.org/news/how-to-get-https-working-on-your-local-development-environment-in-5-minutes-7af615770eec/) for more information.
4. Generate the `GOTRUE_EXTERNAL_APPLE_SECRET` by following this [post](https://medium.com/identity-beyond-borders/how-to-configure-sign-in-with-apple-77c61e336003)!

### E-Mail

Sending email is not required, but highly recommended for password recovery.
If enabled, you must provide the required values below.

```properties
GOTRUE_SMTP_HOST=smtp.mandrillapp.com
GOTRUE_SMTP_PORT=587
GOTRUE_SMTP_USER=smtp-delivery@example.com
GOTRUE_SMTP_PASS=correcthorsebatterystaple
GOTRUE_SMTP_ADMIN_EMAIL=support@example.com
GOTRUE_MAILER_SUBJECTS_CONFIRMATION="Please confirm"
```

`SMTP_ADMIN_EMAIL` - `string` **required**

The `From` email address for all emails sent.

`SMTP_HOST` - `string` **required**

The mail server hostname to send emails through.

`SMTP_PORT` - `number` **required**

The port number to connect to the mail server on.

`SMTP_USER` - `string`

If the mail server requires authentication, the username to use.

`SMTP_PASS` - `string`

If the mail server requires authentication, the password to use.

`SMTP_MAX_FREQUENCY` - `number`

Controls the minimum amount of time that must pass before sending another signup confirmation or password reset email. The value is the number of seconds. Defaults to 900 (15 minutes).

`SMTP_SENDER_NAME` - `string`

Sets the name of the sender. Defaults to the `SMTP_ADMIN_EMAIL` if not used.

`MAILER_AUTOCONFIRM` - `bool`

If you do not require email confirmation, you may set this to `true`. Defaults to `false`.

`MAILER_OTP_EXP` - `number`

Controls the duration an email link or otp is valid for.

`MAILER_URLPATHS_INVITE` - `string`

URL path to use in the user invite email. Defaults to `/verify`.

`MAILER_URLPATHS_CONFIRMATION` - `string`

URL path to use in the signup confirmation email. Defaults to `/verify`.

`MAILER_URLPATHS_RECOVERY` - `string`

URL path to use in the password reset email. Defaults to `/verify`.

`MAILER_URLPATHS_EMAIL_CHANGE` - `string`

URL path to use in the email change confirmation email. Defaults to `/verify`.

`MAILER_SUBJECTS_INVITE` - `string`

Email subject to use for user invite. Defaults to `You have been invited`.

`MAILER_SUBJECTS_CONFIRMATION` - `string`

Email subject to use for signup confirmation. Defaults to `Confirm Your Signup`.

`MAILER_SUBJECTS_RECOVERY` - `string`

Email subject to use for password reset. Defaults to `Reset Your Password`.

`MAILER_SUBJECTS_MAGIC_LINK` - `string`

Email subject to use for magic link email. Defaults to `Your Magic Link`.

`MAILER_SUBJECTS_EMAIL_CHANGE` - `string`

Email subject to use for email change confirmation. Defaults to `Confirm Email Change`.

`MAILER_TEMPLATES_INVITE` - `string`

URL path to an email template to use when inviting a user. (e.g. `https://www.example.com/path-to-email-template.html`)
`SiteURL`, `Email`, `MailType`, and `ConfirmationURL` variables are available.

Default Content (if template is unavailable):

```html
<h2>You have been invited</h2>

<p>
  You have been invited to create a user on {{ .SiteURL }}. Follow this link to
  accept the invite:
</p>
<p><a href="{{ .ConfirmationURL }}">Accept the invite</a></p>
```

`MAILER_TEMPLATES_CONFIRMATION` - `string`

URL path to an email template to use when confirming a signup. (e.g. `https://www.example.com/path-to-email-template.html`)
`SiteURL`, `Email`, `MailType`, and `ConfirmationURL` variables are available.

Default Content (if template is unavailable):

```html
<h2>Confirm your signup</h2>

<p>Follow this link to confirm your user:</p>
<p><a href="{{ .ConfirmationURL }}">Confirm your mail</a></p>
```

`MAILER_TEMPLATES_RECOVERY` - `string`

URL path to an email template to use when resetting a password. (e.g. `https://www.example.com/path-to-email-template.html`)
`SiteURL`, `Email`, `MailType`, and `ConfirmationURL` variables are available.

Default Content (if template is unavailable):

```html
<h2>Reset Password</h2>

<p>Follow this link to reset the password for your user:</p>
<p><a href="{{ .ConfirmationURL }}">Reset Password</a></p>
```

`MAILER_TEMPLATES_MAGIC_LINK` - `string`

URL path to an email template to use when sending magic link. (e.g. `https://www.example.com/path-to-email-template.html`)
`SiteURL`, `Email`, `MailType`, and `ConfirmationURL` variables are available.

Default Content (if template is unavailable):

```html
<h2>Magic Link</h2>

<p>Follow this link to login:</p>
<p><a href="{{ .ConfirmationURL }}">Log In</a></p>
```

`MAILER_TEMPLATES_EMAIL_CHANGE` - `string`

URL path to an email template to use when confirming the change of an email address. (e.g. `https://www.example.com/path-to-email-template.html`)
`SiteURL`, `Email`, `NewEmail`, `MailType`, and `ConfirmationURL` variables are available.

Default Content (if template is unavailable):

```html
<h2>Confirm Change of Email</h2>

<p>
  Follow this link to confirm the update of your email from {{ .Email }} to {{
  .NewEmail }}:
</p>
<p><a href="{{ .ConfirmationURL }}">Change Email</a></p>
```

### Phone Auth

`SMS_AUTOCONFIRM` - `bool`

If you do not require phone confirmation, you may set this to `true`. Defaults to `false`.

`SMS_MAX_FREQUENCY` - `number`

Controls the minimum amount of time that must pass before sending another sms otp. The value is the number of seconds. Defaults to 60 (1 minute)).

`SMS_OTP_EXP` - `number`

Controls the duration an sms otp is valid for.

`SMS_OTP_LENGTH` - `number`

Controls the number of digits of the sms otp sent.

`SMS_PROVIDER` - `string`

Available options are: `twilio`, `messagebird`, `textlocal`, and `vonage`

Then you can use your [twilio credentials](https://www.twilio.com/docs/usage/requests-to-twilio#credentials):

- `SMS_TWILIO_ACCOUNT_SID`
- `SMS_TWILIO_AUTH_TOKEN`
- `SMS_TWILIO_MESSAGE_SERVICE_SID` - can be set to your twilio sender mobile number

Or Messagebird credentials, which can be obtained in the [Dashboard](https://dashboard.messagebird.com/en/developers/access):

- `SMS_MESSAGEBIRD_ACCESS_KEY` - your Messagebird access key
- `SMS_MESSAGEBIRD_ORIGINATOR` - SMS sender (your Messagebird phone number with + or company name)

### CAPTCHA

- If enabled, CAPTCHA will check the request body for the `captcha_token` field and make a verification request to the CAPTCHA provider.

`SECURITY_CAPTCHA_ENABLED` - `string`

Whether captcha middleware is enabled

`SECURITY_CAPTCHA_PROVIDER` - `string`

for now the only options supported are: `hcaptcha` and `turnstile`

- `SECURITY_CAPTCHA_SECRET` - `string`
- `SECURITY_CAPTCHA_TIMEOUT` - `string`

Retrieve from hcaptcha or turnstile account

### Reauthentication

`SECURITY_UPDATE_PASSWORD_REQUIRE_REAUTHENTICATION` - `bool`

Enforce reauthentication on password update.

### Anonymous Sign-Ins

`GOTRUE_EXTERNAL_ANONYMOUS_USERS_ENABLED` - `bool`

Use this to enable/disable anonymous sign-ins.

## Endpoints

Auth exposes the following endpoints:

### **GET /settings**

Returns the publicly available settings for this auth instance.

```json
{
  "external": {
    "apple": true,
    "azure": true,
    "bitbucket": true,
    "discord": true,
    "facebook": true,
    "figma": true,
    "github": true,
    "gitlab": true,
    "google": true,
    "keycloak": true,
    "linkedin": true,
    "notion": true,
    "slack": true,
    "spotify": true,
    "twitch": true,
    "twitter": true,
    "workos": true
  },
  "disable_signup": false,
  "autoconfirm": false
}
```

### **POST, PUT /admin/users/<user_id>**

Creates (POST) or Updates (PUT) the user based on the `user_id` specified. The `ban_duration` field accepts the following time units: "ns", "us", "ms", "s", "m", "h". See [`time.ParseDuration`](https://pkg.go.dev/time#ParseDuration) for more details on the format used.

```js
headers:
{
  "Authorization": "Bearer eyJhbGciOiJI...M3A90LCkxxtX9oNP9KZO" // requires a role claim that can be set in the GOTRUE_JWT_ADMIN_ROLES env var
}

body:
{
  "role": "test-user",
  "email": "email@example.com",
  "phone": "12345678",
  "password": "secret", // only if type = signup
  "email_confirm": true,
  "phone_confirm": true,
  "user_metadata": {},
  "app_metadata": {},
  "ban_duration": "24h" or "none" // to unban a user
}
```

### **POST /admin/generate_link**

Returns the corresponding email action link based on the type specified. Among other things, the response also contains the query params of the action link as separate JSON fields for convenience (along with the email OTP from which the corresponding token is generated).

```js
headers:
{
  "Authorization": "Bearer eyJhbGciOiJI...M3A90LCkxxtX9oNP9KZO" // admin role required
}

body:
{
  "type": "signup" or "magiclink" or "recovery" or "invite",
  "email": "email@example.com",
  "password": "secret", // only if type = signup
  "data": {
    ...
  }, // only if type = signup
  "redirect_to": "https://supabase.io" // Redirect URL to send the user to after an email action. Defaults to SITE_URL.

}
```

Returns

```js
{
  "action_link": "http://localhost:9999/verify?token=TOKEN&type=TYPE&redirect_to=REDIRECT_URL",
  "email_otp": "EMAIL_OTP",
  "hashed_token": "TOKEN",
  "verification_type": "TYPE",
  "redirect_to": "REDIRECT_URL",
  ...
}
```

### **POST /signup**

Register a new user with an email and password.

```js
{
  "email": "email@example.com",
  "password": "secret"
}
```

returns:

```json
{
  "id": "11111111-2222-3333-4444-5555555555555",
  "email": "email@example.com",
  "confirmation_sent_at": "2016-05-15T20:49:40.882805774-07:00",
  "created_at": "2016-05-15T19:53:12.368652374-07:00",
  "updated_at": "2016-05-15T19:53:12.368652374-07:00"
}

// if sign up is a duplicate then faux data will be returned
// as to not leak information about whether a given email
// has an account with your service or not
```

Register a new user with a phone number and password.

```js
{
  "phone": "12345678", // follows the E.164 format
  "password": "secret"
}
```

Returns:

```json
{
  "id": "11111111-2222-3333-4444-5555555555555", // if duplicate sign up, this ID will be faux
  "phone": "12345678",
  "confirmation_sent_at": "2016-05-15T20:49:40.882805774-07:00",
  "created_at": "2016-05-15T19:53:12.368652374-07:00",
  "updated_at": "2016-05-15T19:53:12.368652374-07:00"
}
```

if AUTOCONFIRM is enabled and the sign up is a duplicate, then the endpoint will return:

```
{
  "code":400,
  "msg":"User already registered"
}
```

### **POST /invite**

Invites a new user with an email.
This endpoint requires the `service_role` or `supabase_admin` JWT set as an Auth Bearer header:

e.g.

```json
headers: {
  "Authorization" : "Bearer eyJhbGciOiJI...M3A90LCkxxtX9oNP9KZO"
}
```

```json
{
  "email": "email@example.com"
}
```

Returns:

```json
{
  "id": "11111111-2222-3333-4444-5555555555555",
  "email": "email@example.com",
  "confirmation_sent_at": "2016-05-15T20:49:40.882805774-07:00",
  "created_at": "2016-05-15T19:53:12.368652374-07:00",
  "updated_at": "2016-05-15T19:53:12.368652374-07:00",
  "invited_at": "2016-05-15T19:53:12.368652374-07:00"
}
```

### **POST /verify**

Verify a registration or a password recovery. Type can be `signup` or `recovery` or `invite`
and the `token` is a token returned from either `/signup` or `/recover`.

```json
{
  "type": "signup",
  "token": "confirmation-code-delivered-in-email"
}
```

`password` is required for signup verification if no existing password exists.

Returns:

```json
{
  "access_token": "jwt-token-representing-the-user",
  "token_type": "bearer",
  "expires_in": 3600,
  "refresh_token": "a-refresh-token",
  "type": "signup | recovery | invite"
}
```

Verify a phone signup or sms otp. Type should be set to `sms`.

```json
{
  "type": "sms",
  "token": "confirmation-otp-delivered-in-sms",
  "redirect_to": "https://supabase.io",
  "phone": "phone-number-sms-otp-was-delivered-to"
}
```

Returns:

```json
{
  "access_token": "jwt-token-representing-the-user",
  "token_type": "bearer",
  "expires_in": 3600,
  "refresh_token": "a-refresh-token"
}
```

### **GET /verify**

Verify a registration or a password recovery. Type can be `signup` or `recovery` or `magiclink` or `invite`
and the `token` is a token returned from either `/signup` or `/recover` or `/magiclink`.

query params:

```json
{
  "type": "signup",
  "token": "confirmation-code-delivered-in-email",
  "redirect_to": "https://supabase.io"
}
```

User will be logged in and redirected to:

```json
SITE_URL/#access_token=jwt-token-representing-the-user&token_type=bearer&expires_in=3600&refresh_token=a-refresh-token&type=invite
```

Your app should detect the query params in the fragment and use them to set the session (supabase-js does this automatically)

You can use the `type` param to redirect the user to a password set form in the case of `invite` or `recovery`,
or show an account confirmed/welcome message in the case of `signup`, or direct them to some additional onboarding flow

### **POST /otp**

One-Time-Password. Will deliver a magiclink or sms otp to the user depending on whether the request body contains an "email" or "phone" key.

If `"create_user": true`, user will not be automatically signed up if the user doesn't exist.

```js
{
  "phone": "12345678" // follows the E.164 format
  "create_user": true
}

OR

// exactly the same as /magiclink
{
  "email": "email@example.com"
  "create_user": true
}
```

Returns:

```
{}
```

### **POST /magiclink** (recommended to use /otp instead. See above.)

Magic Link. Will deliver a link (e.g. `/verify?type=magiclink&token=fgtyuf68ddqdaDd`) to the user based on
email address which they can use to redeem an access_token.

By default Magic Links can only be sent once every 60 seconds

```json
{
  "email": "email@example.com"
}
```

Returns:

```json
{}
```

when clicked the magic link will redirect the user to `<SITE_URL>#access_token=x&refresh_token=y&expires_in=z&token_type=bearer&type=magiclink` (see `/verify` above)

### **POST /recover**

Password recovery. Will deliver a password recovery mail to the user based on
email address.

By default recovery links can only be sent once every 60 seconds

```json
{
  "email": "email@example.com"
}
```

Returns:

```json
{}
```

### **POST /token**

This is an OAuth2 endpoint that currently implements
the password and refresh_token grant types

query params:

```
?grant_type=password
```

body:

```json
// Email login
{
  "email": "name@domain.com",
  "password": "somepassword"
}

// Phone login
{
  "phone": "12345678",
  "password": "somepassword"
}
```

or

query params:

```
grant_type=refresh_token
```

body:

```json
{
  "refresh_token": "a-refresh-token"
}
```

Once you have an access token, you can access the methods requiring authentication
by settings the `Authorization: Bearer YOUR_ACCESS_TOKEN_HERE` header.

Returns:

```json
{
  "access_token": "jwt-token-representing-the-user",
  "token_type": "bearer",
  "expires_in": 3600,
  "refresh_token": "a-refresh-token"
}
```

### **GET /user**

Get the JSON object for the logged in user (requires authentication)

Returns:

```json
{
  "id": "11111111-2222-3333-4444-5555555555555",
  "email": "email@example.com",
  "confirmation_sent_at": "2016-05-15T20:49:40.882805774-07:00",
  "created_at": "2016-05-15T19:53:12.368652374-07:00",
  "updated_at": "2016-05-15T19:53:12.368652374-07:00"
}
```

### **PUT /user**

Update a user (Requires authentication). Apart from changing email/password, this
method can be used to set custom user data. Changing the email will result in a magiclink being sent out.

```json
{
  "email": "new-email@example.com",
  "password": "new-password",
  "phone": "+123456789",
  "data": {
    "key": "value",
    "number": 10,
    "admin": false
  }
}
```

Returns:

```json
{
  "id": "11111111-2222-3333-4444-5555555555555",
  "email": "email@example.com",
  "email_change_sent_at": "2016-05-15T20:49:40.882805774-07:00",
  "phone": "+123456789",
  "phone_change_sent_at": "2016-05-15T20:49:40.882805774-07:00",
  "created_at": "2016-05-15T19:53:12.368652374-07:00",
  "updated_at": "2016-05-15T19:53:12.368652374-07:00"
}
```

If `GOTRUE_SECURITY_UPDATE_PASSWORD_REQUIRE_REAUTHENTICATION` is enabled, the user will need to reauthenticate first.

```json
{
  "password": "new-password",
  "nonce": "123456"
}
```

### **GET /reauthenticate**

Sends a nonce to the user's email (preferred) or phone. This endpoint requires the user to be logged in / authenticated first. The user needs to have either an email or phone number for the nonce to be sent successfully.

```json
headers: {
  "Authorization" : "Bearer eyJhbGciOiJI...M3A90LCkxxtX9oNP9KZO"
}
```

### **POST /logout**

Logout a user (Requires authentication).

This will revoke all refresh tokens for the user. Remember that the JWT tokens
will still be valid for stateless auth until they expires.

### **GET /authorize**

Get access_token from external oauth provider

query params:

```
provider=apple | azure | bitbucket | discord | facebook | figma | github | gitlab | google | keycloak | linkedin | notion | slack | spotify | twitch | twitter | workos

scopes=<optional additional scopes depending on the provider (email and name are requested by default)>
```

Redirects to provider and then to `/callback`

For apple specific setup see: <https://github.com/supabase/auth#apple-oauth>

### **GET /callback**

External provider should redirect to here

Redirects to `<GOTRUE_SITE_URL>#access_token=<access_token>&refresh_token=<refresh_token>&provider_token=<provider_oauth_token>&expires_in=3600&provider=<provider_name>`
If additional scopes were requested then `provider_token` will be populated, you can use this to fetch additional data from the provider or interact with their services
