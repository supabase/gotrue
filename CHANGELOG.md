# Changelog

## [2.153.1](https://github.com/nickmitchko/auth/compare/v2.153.0...v2.153.1) (2024-05-30)


### Bug Fixes

* call write header in write if not written ([#1598](https://github.com/nickmitchko/auth/issues/1598)) ([0ef7eb3](https://github.com/nickmitchko/auth/commit/0ef7eb30619d4c365e06a94a79b9cb0333d792da))
* deadlock issue with timeout middleware write ([#1595](https://github.com/nickmitchko/auth/issues/1595)) ([6c9fbd4](https://github.com/nickmitchko/auth/commit/6c9fbd4bd5623c729906fca7857ab508166a3056))

## [2.153.0](https://github.com/nickmitchko/auth/compare/v2.152.0...v2.153.0) (2024-05-29)


### Features

* add `actor_via_sso` to audit log ([#1002](https://github.com/nickmitchko/auth/issues/1002)) ([c52de4a](https://github.com/nickmitchko/auth/commit/c52de4a68753e5404325d385a02a816b4f91ad78))
* add `GOTRUE_&lt;PROVIDER&gt;_SKIP_NONCE_CHECK` to skip nonce checks in ODIC flow ([#1264](https://github.com/nickmitchko/auth/issues/1264)) ([4291959](https://github.com/nickmitchko/auth/commit/4291959c4057332633265745073d26dc0e548898))
* add `is_sso_user` column to `users` which allows duplicate emails to exist on those rows ([#828](https://github.com/nickmitchko/auth/issues/828)) ([0e2cd70](https://github.com/nickmitchko/auth/commit/0e2cd70bbfe1becdabf1f207d2ba062a4e0a3c04))
* add `kid`, `iss`, `iat` claims to the JWT ([#1148](https://github.com/nickmitchko/auth/issues/1148)) ([3446197](https://github.com/nickmitchko/auth/commit/34461975d04ddfa1ba3b6534600b9b6eb55a7832))
* add `provider` claim to `amr` when the method is `sso/saml` ([#837](https://github.com/nickmitchko/auth/issues/837)) ([68acb95](https://github.com/nickmitchko/auth/commit/68acb95a5aab03951e9109166dea0715007e76fe))
* add array attribute mapping for SAML ([#1526](https://github.com/nickmitchko/auth/issues/1526)) ([7326285](https://github.com/nickmitchko/auth/commit/7326285c8af5c42e5c0c2d729ab224cf33ac3a1f))
* add cleanup for session timebox and inactivity timeout ([#1298](https://github.com/nickmitchko/auth/issues/1298)) ([9226979](https://github.com/nickmitchko/auth/commit/92269796c7cb515f4c1e905220c1a1fd8c6764d5))
* add cleanup of unverified factors in 24 hour window ([#1379](https://github.com/nickmitchko/auth/issues/1379)) ([0100a80](https://github.com/nickmitchko/auth/commit/0100a80aa2a0e22a4ce0f281079a900cd06c1df0))
* add configuration for custom sms sender hook ([#1428](https://github.com/nickmitchko/auth/issues/1428)) ([1ea56b6](https://github.com/nickmitchko/auth/commit/1ea56b62d47edb0766d9e445406ecb43d387d920))
* add CORS allowed headers config ([#1197](https://github.com/nickmitchko/auth/issues/1197)) ([7134000](https://github.com/nickmitchko/auth/commit/71340009d9ed9cdc102f57bf7a6e1d96bea8d70c))
* add custom access token hook ([#1332](https://github.com/nickmitchko/auth/issues/1332)) ([312f871](https://github.com/nickmitchko/auth/commit/312f871614438245aa11ca8cde34b2500611de52))
* add custom sms hook ([#1474](https://github.com/nickmitchko/auth/issues/1474)) ([0f6b29a](https://github.com/nickmitchko/auth/commit/0f6b29a46f1dcbf92aa1f7cb702f42e7640f5f93))
* add database cleanup logic, runs after each request ([#875](https://github.com/nickmitchko/auth/issues/875)) ([aaad5bd](https://github.com/nickmitchko/auth/commit/aaad5bd813487062ca5e64de08c55a666b62219d))
* add different logout scopes ([#1112](https://github.com/nickmitchko/auth/issues/1112)) ([df07540](https://github.com/nickmitchko/auth/commit/df075408fbbde2179fd449d98841b4329b3798f3))
* add email rate limit breach metric ([#1208](https://github.com/nickmitchko/auth/issues/1208)) ([4ff1fe0](https://github.com/nickmitchko/auth/commit/4ff1fe058cfab418c445808004091e89dcf87124))
* add endpoint to resend email confirmation ([#912](https://github.com/nickmitchko/auth/issues/912)) ([a50b5a7](https://github.com/nickmitchko/auth/commit/a50b5a711b9c1df902a85edc71cc10314dcf8100))
* add endpoint to unlink identity from user ([#1315](https://github.com/nickmitchko/auth/issues/1315)) ([af83b34](https://github.com/nickmitchko/auth/commit/af83b34850dfe7d983a41a8fb5d02d325ee72985))
* add error codes ([#1377](https://github.com/nickmitchko/auth/issues/1377)) ([e4beea1](https://github.com/nickmitchko/auth/commit/e4beea1cdb80544b0581f1882696a698fdf64938))
* add Figma provider ([#1139](https://github.com/nickmitchko/auth/issues/1139)) ([007324c](https://github.com/nickmitchko/auth/commit/007324cb9607095eadd09a45fd52a37035959bb0))
* add fly oauth provider ([#1261](https://github.com/nickmitchko/auth/issues/1261)) ([0fe4285](https://github.com/nickmitchko/auth/commit/0fe4285873cea1f6815170a2da1589838b66d8af))
* add friendly name to enroll factor response ([#1277](https://github.com/nickmitchko/auth/issues/1277)) ([3c72faf](https://github.com/nickmitchko/auth/commit/3c72faf2b6c83d16f5a438d106c07fe40ec5f49e))
* add generated admin client ([#924](https://github.com/nickmitchko/auth/issues/924)) ([3ee3f34](https://github.com/nickmitchko/auth/commit/3ee3f34509f433123db2cac845dde55988b43843))
* add haveibeenpwned.org password strength check ([#1324](https://github.com/nickmitchko/auth/issues/1324)) ([c3acfe7](https://github.com/nickmitchko/auth/commit/c3acfe7cf4d17a3fdf98bd6376cd9a4ae645564b))
* add idempotent refresh token algorithm ([#1278](https://github.com/nickmitchko/auth/issues/1278)) ([b0426c6](https://github.com/nickmitchko/auth/commit/b0426c6b7cfc3060ff0efa72e9d70a574e1f3ab6))
* add idle db connection options (duration, count, healthcheck period) ([#811](https://github.com/nickmitchko/auth/issues/811)) ([e187280](https://github.com/nickmitchko/auth/commit/e187280ed8496d86a99aef4b458a229a6ab5f785))
* add inactivity-timeout to sessions ([#1288](https://github.com/nickmitchko/auth/issues/1288)) ([6c8a96e](https://github.com/nickmitchko/auth/commit/6c8a96e39190e1d499ab667fcd24bed2b2fa01c8))
* add index on user_id of mfa_factors ([#1247](https://github.com/nickmitchko/auth/issues/1247)) ([6ea135a](https://github.com/nickmitchko/auth/commit/6ea135aa5e6745ff7cbd2e2df242218a66025ca8))
* add kakao OIDC ([#1381](https://github.com/nickmitchko/auth/issues/1381)) ([b5566e7](https://github.com/nickmitchko/auth/commit/b5566e7ac001cc9f2bac128de0fcb908caf3a5ed))
* add log entries for pkce ([#1068](https://github.com/nickmitchko/auth/issues/1068)) ([9c3ba87](https://github.com/nickmitchko/auth/commit/9c3ba87d43fb0a7c77caa1d4decfa39b076d5d2c))
* add manual linking APIs ([#1317](https://github.com/nickmitchko/auth/issues/1317)) ([80172a1](https://github.com/nickmitchko/auth/commit/80172a1ff4921b9c1b81d7cef8edd23a065c2469))
* add mfa cleanup ([#1105](https://github.com/nickmitchko/auth/issues/1105)) ([f5c9afb](https://github.com/nickmitchko/auth/commit/f5c9afb81fa1dd039e4bb285ed744beda5dbae05))
* add mfa indexes ([#746](https://github.com/nickmitchko/auth/issues/746)) ([cb6a879](https://github.com/nickmitchko/auth/commit/cb6a879c4b7e18cb318ee102755ae12b4a31f603))
* add MFA support (disabled by default) ([#736](https://github.com/nickmitchko/auth/issues/736)) ([940f582](https://github.com/nickmitchko/auth/commit/940f5820b311a72cc4a4346527b4b72278fcd286))
* add mfa verification postgres hook ([#1314](https://github.com/nickmitchko/auth/issues/1314)) ([db344d5](https://github.com/nickmitchko/auth/commit/db344d54a5d433dd240a731b0b3974da37cfbe9b))
* Add new Kakao Provider ([#834](https://github.com/nickmitchko/auth/issues/834)) ([bafb89b](https://github.com/nickmitchko/auth/commit/bafb89b657253bca600d2cc4ad99bb992bc69298))
* add new Linkedin OIDC due to deprecated scopes for new linkedin applications ([#1248](https://github.com/nickmitchko/auth/issues/1248)) ([f40acfe](https://github.com/nickmitchko/auth/commit/f40acfe3d1f852dd806797c00997a4e04949ff51))
* add opentelemetry tracer and metrics ([#679](https://github.com/nickmitchko/auth/issues/679)) ([650fa3b](https://github.com/nickmitchko/auth/commit/650fa3bf41161926610eb8919a8e8f1998bfdee7))
* add password hashing metrics ([#769](https://github.com/nickmitchko/auth/issues/769)) ([47adfef](https://github.com/nickmitchko/auth/commit/47adfef0b6bce6e9a181c72a03522c6363f085e3))
* add PKCE (OAuth) ([#891](https://github.com/nickmitchko/auth/issues/891)) ([cf47ec2](https://github.com/nickmitchko/auth/commit/cf47ec2d356e63d976b2b9338f3bc5ae29db3a0e))
* add pkce recovery  ([#1022](https://github.com/nickmitchko/auth/issues/1022)) ([1954560](https://github.com/nickmitchko/auth/commit/19545601676675a923bb9850499717e89cf91a0b))
* add pkce to email_change routes ([#1082](https://github.com/nickmitchko/auth/issues/1082)) ([0f8548f](https://github.com/nickmitchko/auth/commit/0f8548fee9471fcfada0744f41ffa151b7e1731d))
* add required characters password strength check ([#1323](https://github.com/nickmitchko/auth/issues/1323)) ([3991bdb](https://github.com/nickmitchko/auth/commit/3991bdb269f72756becfacee5bd9e540c5b71250))
* add safe deferred closing ([#945](https://github.com/nickmitchko/auth/issues/945)) ([29c431f](https://github.com/nickmitchko/auth/commit/29c431f703f5b741e9efa1b57a4bd2a700d65007))
* add SAML config (disabled by default) ([#759](https://github.com/nickmitchko/auth/issues/759)) ([91fa9bd](https://github.com/nickmitchko/auth/commit/91fa9bd3935a563f717c91acce956d789162e127))
* add saml metadata force update every 24 hours ([#1020](https://github.com/nickmitchko/auth/issues/1020)) ([965feb9](https://github.com/nickmitchko/auth/commit/965feb943060d35f8817616288609833d7ad6129))
* add send email Hook ([#1512](https://github.com/nickmitchko/auth/issues/1512)) ([cf42e02](https://github.com/nickmitchko/auth/commit/cf42e02ec63779f52b1652a7413f64994964c82d))
* add session id to required claim for output of custom access token hook ([#1360](https://github.com/nickmitchko/auth/issues/1360)) ([31222d5](https://github.com/nickmitchko/auth/commit/31222d5ad997bad1ea4e6509480c15eab8f4745a))
* add single session per user with tags support ([#1297](https://github.com/nickmitchko/auth/issues/1297)) ([69feebc](https://github.com/nickmitchko/auth/commit/69feebc43358f3462c527c2b5b777235e1e804bd))
* add soft delete option to admin delete endpoint ([#489](https://github.com/nickmitchko/auth/issues/489)) ([2a2f425](https://github.com/nickmitchko/auth/commit/2a2f425554e8eca701722f4aab794ccf3647b27b))
* add sso pkce ([#1137](https://github.com/nickmitchko/auth/issues/1137)) ([2c0e0a1](https://github.com/nickmitchko/auth/commit/2c0e0a1e44b073770e02b101a357e80a11ba5b6e))
* add support for Azure CIAM login ([#1541](https://github.com/nickmitchko/auth/issues/1541)) ([1cb4f96](https://github.com/nickmitchko/auth/commit/1cb4f96bdc7ef3ef995781b4cf3c4364663a2bf3))
* add support for Twilio Verify ([#1124](https://github.com/nickmitchko/auth/issues/1124)) ([7e240f8](https://github.com/nickmitchko/auth/commit/7e240f8b4112f7bf736e94b7cd8b6439f24af49b))
* add test OTP support for mobile app reviews ([#1166](https://github.com/nickmitchko/auth/issues/1166)) ([2fb0cf5](https://github.com/nickmitchko/auth/commit/2fb0cf54d3e390abd23dcf19fdc6db2b46f43adb))
* add time-boxed sessions ([#1286](https://github.com/nickmitchko/auth/issues/1286)) ([9a1f461](https://github.com/nickmitchko/auth/commit/9a1f4613eb3e6dd2af3ce76b07256fd80ddbc708))
* add timeout middleware ([#1529](https://github.com/nickmitchko/auth/issues/1529)) ([f96ff31](https://github.com/nickmitchko/auth/commit/f96ff31040b28e3a7373b4fd41b7334eda1b413e))
* add turnstile support ([#1094](https://github.com/nickmitchko/auth/issues/1094)) ([b1d2f1c](https://github.com/nickmitchko/auth/commit/b1d2f1c75fb1c38d1e3fa42a8b716d4c593226a2))
* add weak password check on sign in ([#1346](https://github.com/nickmitchko/auth/issues/1346)) ([8785527](https://github.com/nickmitchko/auth/commit/8785527a166fb2614abf99d3f2f911b1579721c2))
* allow `POST /verify` to accept a token hash ([#1165](https://github.com/nickmitchko/auth/issues/1165)) ([e9ab555](https://github.com/nickmitchko/auth/commit/e9ab55559f7e5e62f7a56005347993e4e9da527b))
* allow `whatsapp` channels with Twilio Verify ([#1207](https://github.com/nickmitchko/auth/issues/1207)) ([ff98d2f](https://github.com/nickmitchko/auth/commit/ff98d2fc43f4069b911a3037b338d283048ab92e))
* allow for postgres and http functions on each extensibility point ([#1528](https://github.com/nickmitchko/auth/issues/1528)) ([348a1da](https://github.com/nickmitchko/auth/commit/348a1daee24f6e44b14c018830b748e46d34b4c2))
* allow more than one verified factor per user ([#856](https://github.com/nickmitchko/auth/issues/856)) ([47e4afc](https://github.com/nickmitchko/auth/commit/47e4afc3330f97b704b5a8a2feca6a50d1578533))
* allow unverified email signins ([#1301](https://github.com/nickmitchko/auth/issues/1301)) ([94293b7](https://github.com/nickmitchko/auth/commit/94293b72b829436308050be5399069976b810cdb))
* allow updating saml providers `metadata_xml` ([#1096](https://github.com/nickmitchko/auth/issues/1096)) ([20e503e](https://github.com/nickmitchko/auth/commit/20e503e3b41f8a7a699c83ff4fca6cb78c4c314f))
* alter tag to use raw ([#1427](https://github.com/nickmitchko/auth/issues/1427)) ([53cfe5d](https://github.com/nickmitchko/auth/commit/53cfe5de57d4b5ab6e8e2915493856ecd96f4ede))
* anonymous sign-ins  ([#1460](https://github.com/nickmitchko/auth/issues/1460)) ([130df16](https://github.com/nickmitchko/auth/commit/130df165270c69c8e28aaa1b9421342f997c1ff3))
* azure oidc fix ([#1349](https://github.com/nickmitchko/auth/issues/1349)) ([97b3595](https://github.com/nickmitchko/auth/commit/97b359522d7bf5a314fe615a1abadbf493a4fc98))
* calculate aal without transaction ([#1437](https://github.com/nickmitchko/auth/issues/1437)) ([8dae661](https://github.com/nickmitchko/auth/commit/8dae6614f1a2b58819f94894cef01e9f99117769))
* clean up expired factors ([#1371](https://github.com/nickmitchko/auth/issues/1371)) ([5c94207](https://github.com/nickmitchko/auth/commit/5c9420743a9aef0675f823c30aa4525b4933836e))
* clean up test setup in MFA tests ([#1452](https://github.com/nickmitchko/auth/issues/1452)) ([7185af8](https://github.com/nickmitchko/auth/commit/7185af8de4a269cdde2629054d222333d3522ebe))
* complete OIDC support for Apple and Google providers ([#1108](https://github.com/nickmitchko/auth/issues/1108)) ([aab7c34](https://github.com/nickmitchko/auth/commit/aab7c3481219f136729d80d37731aa64fb8c380a))
* configurable NameID format for SAML provider ([#1481](https://github.com/nickmitchko/auth/issues/1481)) ([ef405d8](https://github.com/nickmitchko/auth/commit/ef405d89e69e008640f275bc37f8ec02ad32da40))
* deprecate and explicitly allow freeform ID token issuers ([#934](https://github.com/nickmitchko/auth/issues/934)) ([99df661](https://github.com/nickmitchko/auth/commit/99df661e4a367543c4d1a8aeee18724b023f7c13))
* deprecate existing webhook implementation ([#1417](https://github.com/nickmitchko/auth/issues/1417)) ([5301e48](https://github.com/nickmitchko/auth/commit/5301e481b0c7278c18b4578a5b1aa8d2256c2f5d))
* drop restriction that PKCE cannot be used with autoconfirm ([#1176](https://github.com/nickmitchko/auth/issues/1176)) ([0a6f218](https://github.com/nickmitchko/auth/commit/0a6f2189a6e1f297ce152d04c3faca57d6900a6e))
* drop SAML RelayState IP address check ([#1376](https://github.com/nickmitchko/auth/issues/1376)) ([6284d99](https://github.com/nickmitchko/auth/commit/6284d99e38f2ea9920ff92406a9b17d8eae767ce))
* drop sha hash tag ([#1422](https://github.com/nickmitchko/auth/issues/1422)) ([76853ce](https://github.com/nickmitchko/auth/commit/76853ce6d45064de5608acc8100c67a8337ba791))
* expose email address being sent to for email change flow ([#1231](https://github.com/nickmitchko/auth/issues/1231)) ([f7308ad](https://github.com/nickmitchko/auth/commit/f7308ad9355db7526a30798b8aa17dabff9f543b))
* fix account linking ([#1098](https://github.com/nickmitchko/auth/issues/1098)) ([93d12d9](https://github.com/nickmitchko/auth/commit/93d12d904820ea6acc20386ef313e35fb28a5a40))
* fix empty string parsing for `GOTRUE_SMS_TEST_OTP_VALID_UNTIL` ([#1234](https://github.com/nickmitchko/auth/issues/1234)) ([25f2dcb](https://github.com/nickmitchko/auth/commit/25f2dcbc97bac18266f1d3583614656182154f85))
* fix refresh token reuse revocation ([#1312](https://github.com/nickmitchko/auth/issues/1312)) ([6e313f8](https://github.com/nickmitchko/auth/commit/6e313f813fc14337a3cd0bdd898f76fe02c9be40))
* fix SAML metadata XML update on fetched metadata ([#1135](https://github.com/nickmitchko/auth/issues/1135)) ([aba0e24](https://github.com/nickmitchko/auth/commit/aba0e241b56bd13b0a24e5064a52824fcc1ff208))
* forbid generating an access token without a session ([#1504](https://github.com/nickmitchko/auth/issues/1504)) ([795e93d](https://github.com/nickmitchko/auth/commit/795e93d0afbe94bcd78489a3319a970b7bf8e8bc))
* HTTP Hook - Add custom envconfig decoding for HTTP Hook Secrets ([#1467](https://github.com/nickmitchko/auth/issues/1467)) ([5b24c4e](https://github.com/nickmitchko/auth/commit/5b24c4eb05b2b52c4177d5f41cba30cb68495c8c))
* ignore common Azure issuer for ID tokens ([#1272](https://github.com/nickmitchko/auth/issues/1272)) ([4c50357](https://github.com/nickmitchko/auth/commit/4c50357841c51c2da0eff4d7f8920aed5e640df2))
* infer `Mail` in SAML assertion and allow deleting SSO user ([#1132](https://github.com/nickmitchko/auth/issues/1132)) ([47ad9de](https://github.com/nickmitchko/auth/commit/47ad9de4285a7f7a112f50e27a9634444e29e276))
* initial fix for invite followed by signup. ([#1262](https://github.com/nickmitchko/auth/issues/1262)) ([76c8eeb](https://github.com/nickmitchko/auth/commit/76c8eeb7275d47f2c7a4029219fa7b3ca4c26da8))
* internalize implementation ([#925](https://github.com/nickmitchko/auth/issues/925)) ([1a52eb6](https://github.com/nickmitchko/auth/commit/1a52eb6e3aaa5a2331bab2cf233856fa6f6015d8))
* make dropping `users_email_key` backward compatible ([#995](https://github.com/nickmitchko/auth/issues/995)) ([aff2fe6](https://github.com/nickmitchko/auth/commit/aff2fe6ba215ce9619bda02ef776713aaef8f52d))
* make error message in factor creation more obvious ([#1374](https://github.com/nickmitchko/auth/issues/1374)) ([74af993](https://github.com/nickmitchko/auth/commit/74af9934c8e919ddf22a98996736d30c829fe01e))
* make phone data type alter backward compatible ([#994](https://github.com/nickmitchko/auth/issues/994)) ([551793e](https://github.com/nickmitchko/auth/commit/551793e9c6dc2a6b7785f973a7e761fe66335dce))
* merge provider metadata on link account ([#1552](https://github.com/nickmitchko/auth/issues/1552)) ([bd8b5c4](https://github.com/nickmitchko/auth/commit/bd8b5c41dd544575e1a52ccf1ef3f0fdee67458c))
* modify email duplicate lookup to use `identities` ([#826](https://github.com/nickmitchko/auth/issues/826)) ([a31545f](https://github.com/nickmitchko/auth/commit/a31545ffaba8367b71b9a000bcab2bab8c44a689))
* new timeout writer implementation ([#1584](https://github.com/nickmitchko/auth/issues/1584)) ([72614a1](https://github.com/nickmitchko/auth/commit/72614a1fce27888f294772b512f8e31c55a36d87))
* no email password resets for users with no email identity ([#793](https://github.com/nickmitchko/auth/issues/793)) ([21c37ed](https://github.com/nickmitchko/auth/commit/21c37edad3e7cc19b4c4bf4cf08a7a5e78e5b31e))
* pass transaction to `invokeHook`, fixing pool exhaustion ([#1465](https://github.com/nickmitchko/auth/issues/1465)) ([b536d36](https://github.com/nickmitchko/auth/commit/b536d368f35adb31f937169e3f093d28352fa7be))
* password sign-up no longer blocks the db connection ([#1319](https://github.com/nickmitchko/auth/issues/1319)) ([84d4b75](https://github.com/nickmitchko/auth/commit/84d4b751ae71c9e5a7c8f61a6692486ff09d86a3))
* PKCE magic link ([#1016](https://github.com/nickmitchko/auth/issues/1016)) ([6fdad13](https://github.com/nickmitchko/auth/commit/6fdad133078b42ab45275e15d43e198e50b85ae1))
* prefix release with v ([#1424](https://github.com/nickmitchko/auth/issues/1424)) ([9d398cd](https://github.com/nickmitchko/auth/commit/9d398cd75fca01fb848aa88b4f545552e8b5751a))
* properly return hook error ([#1355](https://github.com/nickmitchko/auth/issues/1355)) ([890663f](https://github.com/nickmitchko/auth/commit/890663f6cdf21bc2889aa8e646f659c530837d57))
* refactor for central password strength check ([#1321](https://github.com/nickmitchko/auth/issues/1321)) ([5524653](https://github.com/nickmitchko/auth/commit/5524653b0c375872ab49694f0dc99a2093886187))
* refactor generate accesss token to take in request ([#1531](https://github.com/nickmitchko/auth/issues/1531)) ([e4f2b59](https://github.com/nickmitchko/auth/commit/e4f2b59e8e1f8158b6461a384349f1a32cc1bf9a))
* refactor hook error handling ([#1329](https://github.com/nickmitchko/auth/issues/1329)) ([72fdb16](https://github.com/nickmitchko/auth/commit/72fdb160119c4611cfd3eb276f19f2fa21e8eaeb))
* refactor one-time tokens for performance ([#1558](https://github.com/nickmitchko/auth/issues/1558)) ([d1cf8d9](https://github.com/nickmitchko/auth/commit/d1cf8d9096e9183d7772b73031de8ecbd66e912b))
* refactor password changes and logout ([#1162](https://github.com/nickmitchko/auth/issues/1162)) ([b079c35](https://github.com/nickmitchko/auth/commit/b079c3561c4e8166e3a562732c03c237e17abb82))
* refactor PKCE FlowState to reduce duplicate code ([#1446](https://github.com/nickmitchko/auth/issues/1446)) ([b8d0337](https://github.com/nickmitchko/auth/commit/b8d0337922c6712380f6dc74f7eac9fb71b1ae48))
* refactor resource owner password grant ([#1443](https://github.com/nickmitchko/auth/issues/1443)) ([e63ad6f](https://github.com/nickmitchko/auth/commit/e63ad6ff0f67d9a83456918a972ecb5109125628))
* reinstate upgrade whatsapp support on Twilio Programmable Messaging to support Content API ([#1266](https://github.com/nickmitchko/auth/issues/1266)) ([00ee75c](https://github.com/nickmitchko/auth/commit/00ee75c5509facc668295a57ba9130064c267b31))
* remove `id_token` flow with freeform provider ([#927](https://github.com/nickmitchko/auth/issues/927)) ([2646967](https://github.com/nickmitchko/auth/commit/264696734fc922c8bbf563d44fcc59b26e7e6537))
* remove `SafeRoundTripper` and allow private-IP HTTP connections ([#1152](https://github.com/nickmitchko/auth/issues/1152)) ([773e45e](https://github.com/nickmitchko/auth/commit/773e45e1abb9e6ba3b72001f928c7ca75754f70b))
* remove duplicate `add_identities_email_column` migrations, reorder others ([#863](https://github.com/nickmitchko/auth/issues/863)) ([ed08260](https://github.com/nickmitchko/auth/commit/ed08260df22811b5b78b3eb6a2af9f1f2b368de7))
* remove flow state expiry on Magic Links (PKCE) ([#1179](https://github.com/nickmitchko/auth/issues/1179)) ([caa9393](https://github.com/nickmitchko/auth/commit/caa939382a33f7b6ab47f66f0bb60aca631dd061))
* remove legacy lookup in users for one_time_tokens (phase II) ([#1569](https://github.com/nickmitchko/auth/issues/1569)) ([39ca026](https://github.com/nickmitchko/auth/commit/39ca026035f6c61d206d31772c661b326c2a424c))
* remove non-SSO restriction for MFA ([#1378](https://github.com/nickmitchko/auth/issues/1378)) ([9ca6970](https://github.com/nickmitchko/auth/commit/9ca6970baeed8cfa3ec4fc17c32a41562b8db6c9))
* remove opentracing ([#1307](https://github.com/nickmitchko/auth/issues/1307)) ([93e5f82](https://github.com/nickmitchko/auth/commit/93e5f82ced83c08799ce99020be9dea82fc56d24))
* remove saml beta warning ([#1003](https://github.com/nickmitchko/auth/issues/1003)) ([794dab0](https://github.com/nickmitchko/auth/commit/794dab007b886ef9488d279832e3f7b2fe413d38))
* remove unused API `NewAPIFromConfigFile` ([#909](https://github.com/nickmitchko/auth/issues/909)) ([f91a450](https://github.com/nickmitchko/auth/commit/f91a45014b4a9472aaab5fb6a1b5a27b6e4b825c))
* rename `gotrue` to `auth` ([#1340](https://github.com/nickmitchko/auth/issues/1340)) ([8430113](https://github.com/nickmitchko/auth/commit/843011384ebe8a73306a31f74dc80311dd9b5d5f))
* rename package to supabase from netlify ([#947](https://github.com/nickmitchko/auth/issues/947)) ([4f5c2f6](https://github.com/nickmitchko/auth/commit/4f5c2f6801c4fddd37cf6abbbd67eaa6897eb5cc))
* require different passwords on update ([#1163](https://github.com/nickmitchko/auth/issues/1163)) ([154dd91](https://github.com/nickmitchko/auth/commit/154dd9106f073a12935d4e49b95cf1fe783e91df))
* retry concurrent refresh token attempts ([#1202](https://github.com/nickmitchko/auth/issues/1202)) ([d894012](https://github.com/nickmitchko/auth/commit/d894012490582cab50283a4ad9407a1224194f6d))
* return `expires_at` in addition to `expires_in` ([#1183](https://github.com/nickmitchko/auth/issues/1183)) ([3cd4bd5](https://github.com/nickmitchko/auth/commit/3cd4bd5a077240655e6f881cfd7d3afb04dc7ab4))
* return bad request error when factor with duplicate friendly name is registered ([#1375](https://github.com/nickmitchko/auth/issues/1375)) ([55febd2](https://github.com/nickmitchko/auth/commit/55febd290d26b0abf5c2de2e1bdaf6923736831f))
* return SMS ID when possible ([#1145](https://github.com/nickmitchko/auth/issues/1145)) ([02cb927](https://github.com/nickmitchko/auth/commit/02cb9273ec759b2ff55bea1f6eedb7b8db7a2880))
* revert "remove `id_token` flow with freeform provider" ([#933](https://github.com/nickmitchko/auth/issues/933)) ([4d98e30](https://github.com/nickmitchko/auth/commit/4d98e30a131f6ac67ee35ab4a5aaa4af330d813e))
* **saml:** add `not_after` column to `sessions` table (not used) ([#810](https://github.com/nickmitchko/auth/issues/810)) ([8d7477a](https://github.com/nickmitchko/auth/commit/8d7477a23b10b309ab26b2d70f2e880b4a3b94ae))
* **saml:** add SAML ACS handler (disabled by default) ([#779](https://github.com/nickmitchko/auth/issues/779)) ([ae83dce](https://github.com/nickmitchko/auth/commit/ae83dce1b2a652e163c407cae7c0377075f4aeb1))
* **saml:** add SAML metadata endpoint (disabled by default) ([#775](https://github.com/nickmitchko/auth/issues/775)) ([41668b7](https://github.com/nickmitchko/auth/commit/41668b77c23003296ef642f7565fb97375b4fef2))
* **saml:** add session expiration (not after timestamp) support (disabled by default) ([#812](https://github.com/nickmitchko/auth/issues/812)) ([6c6d3ad](https://github.com/nickmitchko/auth/commit/6c6d3ad930140b29714ae3330f2bf229c8a9fdfd))
* **saml:** add SSO authorization API (disabled by default) ([#786](https://github.com/nickmitchko/auth/issues/786)) ([fc6f58d](https://github.com/nickmitchko/auth/commit/fc6f58d1437f896e893c8c93d122cefa8c3546ac))
* **saml:** add SSO/SAML admin endpoints (disabled by default) ([#771](https://github.com/nickmitchko/auth/issues/771)) ([273b41f](https://github.com/nickmitchko/auth/commit/273b41fc574d29a3389adea479b37ecc97b2be58))
* **saml:** add SSO/SAML migrations ([#762](https://github.com/nickmitchko/auth/issues/762)) ([437e683](https://github.com/nickmitchko/auth/commit/437e6835511b190b3da8320928880ba104793eb8))
* **saml:** add X.509 Distinguished Name to generated certificate ([#801](https://github.com/nickmitchko/auth/issues/801)) ([8d85788](https://github.com/nickmitchko/auth/commit/8d85788a90f6df3ec9b82dcc3743d6dea34d518a))
* **saml:** remove unused features, small refactors ([#846](https://github.com/nickmitchko/auth/issues/846)) ([61c8eb8](https://github.com/nickmitchko/auth/commit/61c8eb8053cf4b086603c9791092c693526d0b56))
* **saml:** return JSON response on `POST /sso` with optional JSON response ([#800](https://github.com/nickmitchko/auth/issues/800)) ([dfe9143](https://github.com/nickmitchko/auth/commit/dfe9143ca91822ce74a00c2ecd18e2a1c930858c))
* send over user in SendSMS Hook instead of UserID ([#1551](https://github.com/nickmitchko/auth/issues/1551)) ([d4d743c](https://github.com/nickmitchko/auth/commit/d4d743c2ae9490e1b3249387e3b0d60df6913c68))
* serialized access to session in `refresh_token` grant ([#1190](https://github.com/nickmitchko/auth/issues/1190)) ([a8f1712](https://github.com/nickmitchko/auth/commit/a8f171257c4517eab8a47925c0f4815a9f5bd0a4))
* set `updated_at` on `refresh_tokens` when revoking family ([#1167](https://github.com/nickmitchko/auth/issues/1167)) ([bebd27a](https://github.com/nickmitchko/auth/commit/bebd27ab6a679c9a27441e806af7af55af7dff18))
* simplify token reuse algorithm ([#1072](https://github.com/nickmitchko/auth/issues/1072)) ([9ee3ab6](https://github.com/nickmitchko/auth/commit/9ee3ab66f2f4b843fb37b1e559e1e63c8976e4f1))
* split validation and population of hook name ([#1337](https://github.com/nickmitchko/auth/issues/1337)) ([c03ae09](https://github.com/nickmitchko/auth/commit/c03ae091ab69c20afcd98577fb96a59719777c1b))
* spotify oauth ([#1296](https://github.com/nickmitchko/auth/issues/1296)) ([cc07b4a](https://github.com/nickmitchko/auth/commit/cc07b4aa2ace75d9c8e46ae5107dbabadf944e87))
* strip user-agent from otel tracing ([#1309](https://github.com/nickmitchko/auth/issues/1309)) ([d76f439](https://github.com/nickmitchko/auth/commit/d76f439b65413803ccf37cf8a217a932addfb477))
* support for whatsapp as a channel for sending OTPs  ([#981](https://github.com/nickmitchko/auth/issues/981)) ([d0d079f](https://github.com/nickmitchko/auth/commit/d0d079fb44c99fd1953e9bd8105c6350a4d9c330))
* switch to github.com/supabase/mailme package ([#1159](https://github.com/nickmitchko/auth/issues/1159)) ([dbb9cf7](https://github.com/nickmitchko/auth/commit/dbb9cf706985e15b72f28dd61cb29e13565b0d15)), closes [#870](https://github.com/nickmitchko/auth/issues/870)
* unlinking primary identity should update email ([#1326](https://github.com/nickmitchko/auth/issues/1326)) ([bdc3300](https://github.com/nickmitchko/auth/commit/bdc33008d2af9e4e49b9efd4ff905cc14694faba))
* update `github.com/lestrrat-go/jwx/jwk` to 1.2.25 ([#926](https://github.com/nickmitchko/auth/issues/926)) ([ff8ee5a](https://github.com/nickmitchko/auth/commit/ff8ee5ae0594335c17d3f683464cb19d60bb36b3))
* update chi version ([#1581](https://github.com/nickmitchko/auth/issues/1581)) ([c64ae3d](https://github.com/nickmitchko/auth/commit/c64ae3dd775e8fb3022239252c31b4ee73893237))
* update github.com/coreos/go-oidc/v3@v3.6.0 ([#1115](https://github.com/nickmitchko/auth/issues/1115)) ([23c8b45](https://github.com/nickmitchko/auth/commit/23c8b453cff181f29adab764baacec8362df11f0))
* update github.com/rs/cors to v1.9.0 ([#1198](https://github.com/nickmitchko/auth/issues/1198)) ([27d3a7f](https://github.com/nickmitchko/auth/commit/27d3a7f4d1d43e0cb7eec71573aeb1ce3cf60279))
* update oauth1.a flow ([#1382](https://github.com/nickmitchko/auth/issues/1382)) ([4f39d2e](https://github.com/nickmitchko/auth/commit/4f39d2e42fcaf77c201039e7bc60b0d663e62428))
* update openapi spec with identity and is_anonymous fields ([#1573](https://github.com/nickmitchko/auth/issues/1573)) ([86a79df](https://github.com/nickmitchko/auth/commit/86a79df9ecfcf09fda0b8e07afbc41154fbb7d9d))
* update primary key for identities table ([#1311](https://github.com/nickmitchko/auth/issues/1311)) ([d8ec801](https://github.com/nickmitchko/auth/commit/d8ec8015e50f6199786a9e5f05589888fa8862be))
* update publish.yml checkout repository so there is access to Dockerfile ([#1419](https://github.com/nickmitchko/auth/issues/1419)) ([7cce351](https://github.com/nickmitchko/auth/commit/7cce3518e8c9f1f3f93e4f6a0658ee08771c4f1c))
* update README.md to trigger release ([#1425](https://github.com/nickmitchko/auth/issues/1425)) ([91e0e24](https://github.com/nickmitchko/auth/commit/91e0e245f5957ebce13370f79fd4a6be8108ed80))
* update to Go 1.19 ([#770](https://github.com/nickmitchko/auth/issues/770)) ([e6525ab](https://github.com/nickmitchko/auth/commit/e6525abd6733c2e86e0b7913951e0edf5f24e018))
* upgrade whatsapp support on Twilio Programmable Messaging ([#1249](https://github.com/nickmitchko/auth/issues/1249)) ([c58febe](https://github.com/nickmitchko/auth/commit/c58febed896c7152a03634ac32b7f596b7b65d6f))
* use `DO` blocks around SQL statements in migrations ([#1335](https://github.com/nickmitchko/auth/issues/1335)) ([061391a](https://github.com/nickmitchko/auth/commit/061391aceed64b2cac56e8a82b6a3da3e83cbb14))
* use `otherMails` with Azure ([#1130](https://github.com/nickmitchko/auth/issues/1130)) ([fba1988](https://github.com/nickmitchko/auth/commit/fba19885daa1e8d93c12bd2931383a5899b154e0))
* use `template/text` instead of `strings.Replace` for phone OTP messages ([#1188](https://github.com/nickmitchko/auth/issues/1188)) ([5caacc1](https://github.com/nickmitchko/auth/commit/5caacc1f81ff20f8f09f4598b510c767e589a1c5))
* use account linking algorithm ([#829](https://github.com/nickmitchko/auth/issues/829)) ([c709ed5](https://github.com/nickmitchko/auth/commit/c709ed5f1bb21b2c971bd08f402932423291477a))
* use dummy instance id to improve performance on refresh token queries ([#1454](https://github.com/nickmitchko/auth/issues/1454)) ([656474e](https://github.com/nickmitchko/auth/commit/656474e1b9ff3d5129190943e8c48e456625afe5))
* use OIDC ID token for Azure ([#1269](https://github.com/nickmitchko/auth/issues/1269)) ([57e336e](https://github.com/nickmitchko/auth/commit/57e336e9c0d8f8fc27e5efeecf06bff5507fef54))
* use unique message IDs for emails to prevent grouping ([#986](https://github.com/nickmitchko/auth/issues/986)) ([aaf2765](https://github.com/nickmitchko/auth/commit/aaf2765d84ec4f56582f2ba550f8015c79c2bf22))


### Bug Fixes

* [#1218](https://github.com/nickmitchko/auth/issues/1218) fixes existing migrations to allow namespaces!="auth" ([#1279](https://github.com/nickmitchko/auth/issues/1279)) ([206fc09](https://github.com/nickmitchko/auth/commit/206fc0908992e6c22a6343a7a7517f66322764b5))
* `createNewIdentity` uses provided transaction ([#776](https://github.com/nickmitchko/auth/issues/776)) ([3f61950](https://github.com/nickmitchko/auth/commit/3f61950ba93ff4c6500630bf48f5afa1ddcb0f7f))
* account linking logic ([#990](https://github.com/nickmitchko/auth/issues/990)) ([17162c9](https://github.com/nickmitchko/auth/commit/17162c991980566cef3330ef08565ce4fda4adac))
* add `email` as verification type for email OTPs ([#885](https://github.com/nickmitchko/auth/issues/885)) ([8d21cbc](https://github.com/nickmitchko/auth/commit/8d21cbc55cc2613ed0aa47cd0e1383f5575d2e5d))
* add check for max password length ([#1368](https://github.com/nickmitchko/auth/issues/1368)) ([41aac69](https://github.com/nickmitchko/auth/commit/41aac695029a8e8ae6aeed87e71abea63030c799))
* add checks for ownership for unenroll and verify ([#835](https://github.com/nickmitchko/auth/issues/835)) ([bdd9947](https://github.com/nickmitchko/auth/commit/bdd99479732f3c14e95030b0a96bc6b86b962416))
* add cleanup statement for anonymous users ([#1497](https://github.com/nickmitchko/auth/issues/1497)) ([cf2372a](https://github.com/nickmitchko/auth/commit/cf2372a177796b829b72454e7491ce768bf5a42f))
* add db conn max idle time setting ([#1555](https://github.com/nickmitchko/auth/issues/1555)) ([2caa7b4](https://github.com/nickmitchko/auth/commit/2caa7b4d75d2ff54af20f3e7a30a8eeec8cbcda9))
* add discord `global_name` to custom_claims ([#1171](https://github.com/nickmitchko/auth/issues/1171)) ([3b1a5b9](https://github.com/nickmitchko/auth/commit/3b1a5b980ed49eb17b93f8fb43346cc5b1525b97))
* add error handling for hook ([#1339](https://github.com/nickmitchko/auth/issues/1339)) ([7ac7586](https://github.com/nickmitchko/auth/commit/7ac7586c114f581722f07eb54ff4ca193c34ddd9))
* add guard check in case factor, session, or user are missing ([#1099](https://github.com/nickmitchko/auth/issues/1099)) ([b4a3fec](https://github.com/nickmitchko/auth/commit/b4a3fec6d00566becc51f001b828187f736fb383))
* add http support for https hooks on localhost ([#1484](https://github.com/nickmitchko/auth/issues/1484)) ([5c04104](https://github.com/nickmitchko/auth/commit/5c04104bf77a9c2db46d009764ec3ec3e484fc09))
* add improved HTTP metrics ([#768](https://github.com/nickmitchko/auth/issues/768)) ([2f78644](https://github.com/nickmitchko/auth/commit/2f786449287e1b113b503d4bdb9d8548b999fdaf))
* add index on `(session_id, revoked)` in `refresh_tokens` ([#765](https://github.com/nickmitchko/auth/issues/765)) ([5ba3aca](https://github.com/nickmitchko/auth/commit/5ba3aca0e4cbbe1af64a9d786c60093e34340105))
* add index on `identities.user_id` ([#781](https://github.com/nickmitchko/auth/issues/781)) ([6c2c734](https://github.com/nickmitchko/auth/commit/6c2c734e912e8224b0efc6557795ddfc13621864))
* add mfa migrations ([#722](https://github.com/nickmitchko/auth/issues/722)) ([afdb223](https://github.com/nickmitchko/auth/commit/afdb2235087e2b769bbc8196d69d49aa1be2e50d))
* add migration to backfill email identities ([#823](https://github.com/nickmitchko/auth/issues/823)) ([b54d60a](https://github.com/nickmitchko/auth/commit/b54d60a497ce4786c391bee5da978af76baa1253))
* add missing index on `user_id` under `sessions` ([#763](https://github.com/nickmitchko/auth/issues/763)) ([3332072](https://github.com/nickmitchko/auth/commit/3332072aebec8efceae6d5b52d44dd7ef35551a3))
* add profiler server ([#1158](https://github.com/nickmitchko/auth/issues/1158)) ([58552d6](https://github.com/nickmitchko/auth/commit/58552d6090a57367be92e32198ca1cf712d745af))
* add redirectTo to email templates ([#1276](https://github.com/nickmitchko/auth/issues/1276)) ([40aed62](https://github.com/nickmitchko/auth/commit/40aed622f24066b2718e4509a001026fe7d4b76d))
* add separate config for sms rate limits ([#860](https://github.com/nickmitchko/auth/issues/860)) ([1ff475c](https://github.com/nickmitchko/auth/commit/1ff475c3597e6711e78fb6d988d4809a754ec0b4))
* add swagger docs ([#695](https://github.com/nickmitchko/auth/issues/695)) ([8eefabb](https://github.com/nickmitchko/auth/commit/8eefabbe7bd932217ada398bc9baf6072c7fe8a9))
* add test for all sms providers ([#676](https://github.com/nickmitchko/auth/issues/676)) ([de6cd79](https://github.com/nickmitchko/auth/commit/de6cd793280c43dc580afea7877f1430aba964eb))
* add validation and proper decoding on send email hook ([#1520](https://github.com/nickmitchko/auth/issues/1520)) ([e19e762](https://github.com/nickmitchko/auth/commit/e19e762e3e29729a1d1164c65461427822cc87f1))
* add validation to admin update user ([#717](https://github.com/nickmitchko/auth/issues/717)) ([497ce10](https://github.com/nickmitchko/auth/commit/497ce10a9438bc79c1e8c297d3dfb49869ae4590))
* admin delete factor should be allowed to delete unverified factors ([#854](https://github.com/nickmitchko/auth/issues/854)) ([4c2bac3](https://github.com/nickmitchko/auth/commit/4c2bac32cb79a27db67f2b061469c4ab493309e1))
* admin user create & update ([#929](https://github.com/nickmitchko/auth/issues/929)) ([5526627](https://github.com/nickmitchko/auth/commit/55266277f9f643910f7426e9e0f0309e1a9c9d96))
* allow all URL forms in redirects ([#711](https://github.com/nickmitchko/auth/issues/711)) ([4ece9e3](https://github.com/nickmitchko/auth/commit/4ece9e3d01b265dbeb60231b9d2e7a55109457a0))
* allow any oauth providers to pass query params ([#757](https://github.com/nickmitchko/auth/issues/757)) ([ac2e7ae](https://github.com/nickmitchko/auth/commit/ac2e7ae539578e86f3a60d0672cd0a224427050b))
* allow gotrue to work with multiple custom domains ([#999](https://github.com/nickmitchko/auth/issues/999)) ([91a82ed](https://github.com/nickmitchko/auth/commit/91a82ed468ec0f1e6edb4b4bbc560815ff0d8167))
* allow transactions to be committed while returning a custom error ([#1310](https://github.com/nickmitchko/auth/issues/1310)) ([8565d26](https://github.com/nickmitchko/auth/commit/8565d264014557f721b6d12afa3171a25a38b905))
* backfill email identities for invited users ([#914](https://github.com/nickmitchko/auth/issues/914)) ([f7286dd](https://github.com/nickmitchko/auth/commit/f7286dddbd5c42de97dfbb25b93ce46b97353a27))
* bypass captcha for certain routes ([#693](https://github.com/nickmitchko/auth/issues/693)) ([70a6070](https://github.com/nickmitchko/auth/commit/70a607099fffe3214e5b689810cf4ca6dcf3bb87))
* Change Dockerfile.dev target from netlify to Supabase ([#973](https://github.com/nickmitchko/auth/issues/973)) ([ee74d52](https://github.com/nickmitchko/auth/commit/ee74d5222d8d92efbcf9c53fe08dd481d6f1acdf))
* change email update flow to return both ? messages and # messages ([#1129](https://github.com/nickmitchko/auth/issues/1129)) ([77afd28](https://github.com/nickmitchko/auth/commit/77afd2834201e50672502a48bdc365b4ba7a095b))
* check err before using user ([#1154](https://github.com/nickmitchko/auth/issues/1154)) ([53e1b3a](https://github.com/nickmitchko/auth/commit/53e1b3aa31dc4ac87d0815491fd4c752a9f8e03d))
* check for pkce prefix ([#1291](https://github.com/nickmitchko/auth/issues/1291)) ([05c629b](https://github.com/nickmitchko/auth/commit/05c629b1b521e950e8951e9e8d328c9813ebe6bd))
* check freq on email change ([#1090](https://github.com/nickmitchko/auth/issues/1090)) ([659ca66](https://github.com/nickmitchko/auth/commit/659ca66386f818d707995dbcca2eaeebc4d0bfd7))
* check linking domain prefix ([#1336](https://github.com/nickmitchko/auth/issues/1336)) ([9194ffc](https://github.com/nickmitchko/auth/commit/9194ffc72d68ca45dfb18dc1b0eb7ce64e62592c))
* cleanup panics due to bad inactivity timeout code ([#1471](https://github.com/nickmitchko/auth/issues/1471)) ([548edf8](https://github.com/nickmitchko/auth/commit/548edf898161c9ba9a136fc99ec2d52a8ba1f856))
* confirm email on email change ([#1084](https://github.com/nickmitchko/auth/issues/1084)) ([0624655](https://github.com/nickmitchko/auth/commit/0624655649c8de483fe8caa4a69bb3895fd967be))
* convert `string` -&gt; `*string` for AAL and AMR ([#785](https://github.com/nickmitchko/auth/issues/785)) ([d887d18](https://github.com/nickmitchko/auth/commit/d887d18253614eef3f8f5fcb0a43d28ae20ea061))
* correct pkce redirect generation ([#1097](https://github.com/nickmitchko/auth/issues/1097)) ([bdf93b4](https://github.com/nickmitchko/auth/commit/bdf93b41b198a9d09813359ec285af1b3c47b4e3))
* create identity for invited user ([#895](https://github.com/nickmitchko/auth/issues/895)) ([8ddf54b](https://github.com/nickmitchko/auth/commit/8ddf54b21263cc88d05ad637a2d614f1389a8482))
* deprecate hooks  ([#1421](https://github.com/nickmitchko/auth/issues/1421)) ([effef1b](https://github.com/nickmitchko/auth/commit/effef1b6ecc448b7927eff23df8d5b509cf16b5c))
* disable allow unverified email sign ins if autoconfirm enabled ([#1313](https://github.com/nickmitchko/auth/issues/1313)) ([9b93ac1](https://github.com/nickmitchko/auth/commit/9b93ac1d988519354a59ecff573ba22718896971))
* do call send sms hook when SMS autoconfirm is enabled ([#1562](https://github.com/nickmitchko/auth/issues/1562)) ([bfe4d98](https://github.com/nickmitchko/auth/commit/bfe4d988f3768b0407526bcc7979fb21d8cbebb3))
* **docs:** remove bracket on file name for broken link ([#1493](https://github.com/nickmitchko/auth/issues/1493)) ([96f7a68](https://github.com/nickmitchko/auth/commit/96f7a68a5479825e31106c2f55f82d5b2c007c0f))
* don't encode query fragment ([#1153](https://github.com/nickmitchko/auth/issues/1153)) ([e414cb3](https://github.com/nickmitchko/auth/commit/e414cb3a98cff8598f4aa2c96a1ca63c78bd65a6))
* don't update user metadata on subsequent signups ([#825](https://github.com/nickmitchko/auth/issues/825)) ([9e97a32](https://github.com/nickmitchko/auth/commit/9e97a32743eae25737c530f551bf81a0a4b54261))
* drop mfa flag ([#831](https://github.com/nickmitchko/auth/issues/831)) ([f0642c0](https://github.com/nickmitchko/auth/commit/f0642c08ea8c01c3134ca8efe87db0213842c369))
* duplicate identity error on update user ([#1141](https://github.com/nickmitchko/auth/issues/1141)) ([39ca89c](https://github.com/nickmitchko/auth/commit/39ca89c67c2aee10b656d2c056039ee1eb9e99be))
* enforce code challenge validity across endpoints ([#1026](https://github.com/nickmitchko/auth/issues/1026)) ([be7c082](https://github.com/nickmitchko/auth/commit/be7c082cad77176dd1bc987e90fdd1502eafdd67))
* error should be an IsNotFoundError ([#1432](https://github.com/nickmitchko/auth/issues/1432)) ([7f40047](https://github.com/nickmitchko/auth/commit/7f40047aec3577d876602444b1d88078b2237d66))
* expose `provider` under `amr` in access token ([#1456](https://github.com/nickmitchko/auth/issues/1456)) ([e9f38e7](https://github.com/nickmitchko/auth/commit/e9f38e76d8a7b93c5c2bb0de918a9b156155f018))
* expose x-total-count and link ([#991](https://github.com/nickmitchko/auth/issues/991)) ([e6dac54](https://github.com/nickmitchko/auth/commit/e6dac5450590d5a5f86ededb551353ec28098e1e))
* fetch new IDP metadata if stale ([#833](https://github.com/nickmitchko/auth/issues/833)) ([be3766d](https://github.com/nickmitchko/auth/commit/be3766d544d4c16a1caa0bf77750db7270dd16a1))
* fill `last_sign_in_at` with a non-null value on backfilled email identities ([#850](https://github.com/nickmitchko/auth/issues/850)) ([ef1a51f](https://github.com/nickmitchko/auth/commit/ef1a51ffa0c16853b8c7752b29a16ad05cf56126))
* fix flow state expiry check ([#1088](https://github.com/nickmitchko/auth/issues/1088)) ([6000e70](https://github.com/nickmitchko/auth/commit/6000e70e9e1d6929f9d8c90c36fab2bf94bb6d85))
* format test otps ([#1567](https://github.com/nickmitchko/auth/issues/1567)) ([434a59a](https://github.com/nickmitchko/auth/commit/434a59ae387c35fd6629ec7c674d439537e344e5))
* garbled text in sms message when message contains unicode ([#971](https://github.com/nickmitchko/auth/issues/971)) ([55544e2](https://github.com/nickmitchko/auth/commit/55544e2b515c8999c59355829a642ae2cb5c93e1))
* generate signup link should not error ([#1514](https://github.com/nickmitchko/auth/issues/1514)) ([4fc3881](https://github.com/nickmitchko/auth/commit/4fc388186ac7e7a9a32ca9b963a83d6ac2eb7603))
* generateLink should create identity for invite & signup ([#774](https://github.com/nickmitchko/auth/issues/774)) ([0032b65](https://github.com/nickmitchko/auth/commit/0032b654df588318fb3b4b5094a5f292d909462f))
* handle error properly for redirects ([#887](https://github.com/nickmitchko/auth/issues/887)) ([30c55e8](https://github.com/nickmitchko/auth/commit/30c55e857c96f9d7e40792a421790a674284e690))
* handle oauth email check separately ([#1348](https://github.com/nickmitchko/auth/issues/1348)) ([757989c](https://github.com/nickmitchko/auth/commit/757989c1d3856a1dc450c2e0a5cb1c8e0172a6a6))
* ignore exchangeCodeForSession when captcha is enabled ([#1121](https://github.com/nickmitchko/auth/issues/1121)) ([4970bbc](https://github.com/nickmitchko/auth/commit/4970bbcba91a435cc0bfa8a75b4899a79d8d4dea))
* impose expiry on auth code instead of magic link ([#1440](https://github.com/nickmitchko/auth/issues/1440)) ([35aeaf1](https://github.com/nickmitchko/auth/commit/35aeaf1b60dd27a22662a6d1955d60cc907b55dd))
* improve default settings used  ([4745451](https://github.com/nickmitchko/auth/commit/4745451a931c2be5d36c07b37bd0eb3ab7780587))
* improve logging structure ([#1583](https://github.com/nickmitchko/auth/issues/1583)) ([c22fc15](https://github.com/nickmitchko/auth/commit/c22fc15d2a8383e95a2364f383dfa7dce5f5df88))
* improve MFA QR Code resilience so as to support providers like 1Password ([#1455](https://github.com/nickmitchko/auth/issues/1455)) ([6522780](https://github.com/nickmitchko/auth/commit/652278046c9dd92f5cecd778735b058ef3fb41c7))
* improve perf in account linking ([#1394](https://github.com/nickmitchko/auth/issues/1394)) ([8eedb95](https://github.com/nickmitchko/auth/commit/8eedb95dbaa310aac464645ec91d6a374813ab89))
* include `/organizations` in expected issuer exemption ([#1275](https://github.com/nickmitchko/auth/issues/1275)) ([47cbe6e](https://github.com/nickmitchko/auth/commit/47cbe6e481ccec9d7f533c7fdba0328c8f6227e5))
* include email claim in identityData ([#796](https://github.com/nickmitchko/auth/issues/796)) ([930f5af](https://github.com/nickmitchko/auth/commit/930f5affdab112db81e17ef799418206bead9092))
* include symbols in generated password ([#1364](https://github.com/nickmitchko/auth/issues/1364)) ([f81a748](https://github.com/nickmitchko/auth/commit/f81a748b10f26c11c9940ee864c3fb58e19a98a1))
* invalidate email, phone OTPs on password change ([#1489](https://github.com/nickmitchko/auth/issues/1489)) ([960a4f9](https://github.com/nickmitchko/auth/commit/960a4f94f5500e33a0ec2f6afe0380bbc9562500))
* IsDuplicatedEmail should filter out identities for the currentUser ([#1092](https://github.com/nickmitchko/auth/issues/1092)) ([dd2b688](https://github.com/nickmitchko/auth/commit/dd2b6883d666e9a714f9f05c65a44b293f76a6a6))
* linkedin provider issue with missing avatar url ([#847](https://github.com/nickmitchko/auth/issues/847)) ([895fc2a](https://github.com/nickmitchko/auth/commit/895fc2a4d4d2f02a8568c23d3086f735ac67ed28))
* linkedin_oidc provider error ([#1534](https://github.com/nickmitchko/auth/issues/1534)) ([4f5e8e5](https://github.com/nickmitchko/auth/commit/4f5e8e5120531e5a103fbdda91b51cabcb4e1a8c))
* load user after sign-up to pull data from triggers ([#712](https://github.com/nickmitchko/auth/issues/712)) ([e553477](https://github.com/nickmitchko/auth/commit/e5534770d013ea8473a91a92471d6c3746668bbe))
* log clearer internal error messages for verify ([#1292](https://github.com/nickmitchko/auth/issues/1292)) ([aafad5c](https://github.com/nickmitchko/auth/commit/aafad5c2b073f0f56239109eef2cf5f2ee5cfd70))
* log correct referer value ([#1178](https://github.com/nickmitchko/auth/issues/1178)) ([a6950a0](https://github.com/nickmitchko/auth/commit/a6950a0e606ed47cf9580eb4c35ad07b58afbd36))
* log final writer error instead of handling ([#1564](https://github.com/nickmitchko/auth/issues/1564)) ([170bd66](https://github.com/nickmitchko/auth/commit/170bd6615405afc852c7107f7358dfc837bad737))
* logout cookies not cleared ([#830](https://github.com/nickmitchko/auth/issues/830)) ([596dd70](https://github.com/nickmitchko/auth/commit/596dd705f9148ab17b6abee2c953524744096233))
* lowercase emails ([#714](https://github.com/nickmitchko/auth/issues/714)) ([d65ba60](https://github.com/nickmitchko/auth/commit/d65ba60d8d88e1be0c472664a59511e8ed35f9bf))
* lowercase oauth emails for account linking ([#1125](https://github.com/nickmitchko/auth/issues/1125)) ([df22915](https://github.com/nickmitchko/auth/commit/df229158ac6d41daf5e17e9ccd9a9f4b9a1c5f32))
* maintain query params order ([#1161](https://github.com/nickmitchko/auth/issues/1161)) ([c925065](https://github.com/nickmitchko/auth/commit/c925065059b69b86f64d9cd1509e4ad24bc37904))
* make add_mfa_indexes re-runnable ([#827](https://github.com/nickmitchko/auth/issues/827)) ([00c21d8](https://github.com/nickmitchko/auth/commit/00c21d84b6dc3b7d4283359b626966f527997f13))
* make flow_state migrations idempotent, add index ([#1086](https://github.com/nickmitchko/auth/issues/1086)) ([7ca755a](https://github.com/nickmitchko/auth/commit/7ca755a2da24967a7fff56d37ee9d9ece24a5b69))
* make migration idempotent ([#1079](https://github.com/nickmitchko/auth/issues/1079)) ([2be90c7](https://github.com/nickmitchko/auth/commit/2be90c7ca08871576827c7e039c81ce0ae13b7b8))
* make migration idempotent ([#923](https://github.com/nickmitchko/auth/issues/923)) ([c792443](https://github.com/nickmitchko/auth/commit/c7924433fe712e66330dfdda2187b43aadb131f1))
* move all EmailActionTypes to mailer package ([#1510](https://github.com/nickmitchko/auth/issues/1510)) ([765db08](https://github.com/nickmitchko/auth/commit/765db08582669a1b7f054217fa8f0ed45804c0b5))
* move creation of flow state into function ([#1470](https://github.com/nickmitchko/auth/issues/1470)) ([4392a08](https://github.com/nickmitchko/auth/commit/4392a08d68d18828005d11382730117a7b143635))
* nil pointer dereference in stale SAML metadata check ([#977](https://github.com/nickmitchko/auth/issues/977)) ([bb21c93](https://github.com/nickmitchko/auth/commit/bb21c93f915e9364bb5744e3bf635cfff7428088))
* OIDC provider validation log message ([#1380](https://github.com/nickmitchko/auth/issues/1380)) ([27e6b1f](https://github.com/nickmitchko/auth/commit/27e6b1f9a4394c5c4f8dff9a8b5529db1fc67af9))
* only apply rate limit if autoconfirm is false ([#1184](https://github.com/nickmitchko/auth/issues/1184)) ([46932da](https://github.com/nickmitchko/auth/commit/46932da6baa95306df6c72f411b5e485f695c98e))
* only create or update the email / phone identity after it's been verified ([#1403](https://github.com/nickmitchko/auth/issues/1403)) ([2d20729](https://github.com/nickmitchko/auth/commit/2d207296ec22dd6c003c89626d255e35441fd52d))
* only create or update the email / phone identity after it's been verified (again) ([#1409](https://github.com/nickmitchko/auth/issues/1409)) ([bc6a5b8](https://github.com/nickmitchko/auth/commit/bc6a5b884b43fe6b8cb924d3f79999fe5bfe7c5f))
* pass through redirect query parameters ([#1224](https://github.com/nickmitchko/auth/issues/1224)) ([577e320](https://github.com/nickmitchko/auth/commit/577e3207aab8ee4c4661f5a8148f02296210f1d8))
* patch secure email change (double confirm) response format. ([#1241](https://github.com/nickmitchko/auth/issues/1241)) ([064e8a1](https://github.com/nickmitchko/auth/commit/064e8a1a1a71163d81f6c549b31148b88c3ef7be))
* pkce bug with magiclink ([#1074](https://github.com/nickmitchko/auth/issues/1074)) ([4b84129](https://github.com/nickmitchko/auth/commit/4b84129e668e9f3ab4fc8d768c73edc50106d2d5))
* pkce issues ([#1083](https://github.com/nickmitchko/auth/issues/1083)) ([eb50ba1](https://github.com/nickmitchko/auth/commit/eb50ba1de139de7a244637190cb1071c8d50bf9e))
* populate password verification attempt hook ([#1436](https://github.com/nickmitchko/auth/issues/1436)) ([f974bdb](https://github.com/nickmitchko/auth/commit/f974bdb58340395955ca27bdd26d57062433ece9))
* POST /verify should check pkce case ([#1085](https://github.com/nickmitchko/auth/issues/1085)) ([7f42eaa](https://github.com/nickmitchko/auth/commit/7f42eaa582b497859ba07d77e6db0eb18026117d))
* potential panics on error ([#1389](https://github.com/nickmitchko/auth/issues/1389)) ([5ad703b](https://github.com/nickmitchko/auth/commit/5ad703bddc6ec74f076cbe6ce1f942663343d47a))
* preserve backward compatibility with Twilio Existing API ([#1260](https://github.com/nickmitchko/auth/issues/1260)) ([71fb156](https://github.com/nickmitchko/auth/commit/71fb1569c9daff8ac99ae6b9626e098606b2934f))
* prevent user email side-channel leak on verify ([#1472](https://github.com/nickmitchko/auth/issues/1472)) ([311cde8](https://github.com/nickmitchko/auth/commit/311cde8d1e82f823ae26a341e068034d60273864))
* properly escape `redirectTo` URL for magic links ([#750](https://github.com/nickmitchko/auth/issues/750)) ([cc1d49d](https://github.com/nickmitchko/auth/commit/cc1d49db0bf08bbc1f6a07426aaaf05f3336c2d4))
* rate limiting not applied on phone OTP ([#788](https://github.com/nickmitchko/auth/issues/788)) ([6a129f3](https://github.com/nickmitchko/auth/commit/6a129f3d80c87bdd093622bfe9b3001943665fb8))
* refactor email sending functions ([#1495](https://github.com/nickmitchko/auth/issues/1495)) ([285c290](https://github.com/nickmitchko/auth/commit/285c290adf231fea7ca1dff954491dc427cf18e2))
* refactor factor_test to centralize setup ([#1473](https://github.com/nickmitchko/auth/issues/1473)) ([c86007e](https://github.com/nickmitchko/auth/commit/c86007e59684334b5e8c2285c36094b6eec89442))
* refactor mfa and aal update methods ([#1503](https://github.com/nickmitchko/auth/issues/1503)) ([31a5854](https://github.com/nickmitchko/auth/commit/31a585429bf248aa919d94c82c7c9e0c1c695461))
* refactor mfa challenge and tests ([#1469](https://github.com/nickmitchko/auth/issues/1469)) ([6c76f21](https://github.com/nickmitchko/auth/commit/6c76f21cee5dbef0562c37df6a546939affb2f8d))
* refactor request params to use generics ([#1464](https://github.com/nickmitchko/auth/issues/1464)) ([e1cdf5c](https://github.com/nickmitchko/auth/commit/e1cdf5c4b5c1bf467094f4bdcaa2e42a5cc51c20))
* releaserc ([#680](https://github.com/nickmitchko/auth/issues/680)) ([3f7f39e](https://github.com/nickmitchko/auth/commit/3f7f39e79e9a44257f9266d421c51cc4c20bbf09))
* remove captcha on id_token grant ([#1175](https://github.com/nickmitchko/auth/issues/1175)) ([910079c](https://github.com/nickmitchko/auth/commit/910079c4e48f9fc0d82f7956f974bb25b4c3a154))
* remove deprecated LogoutAllRefreshTokens ([#1519](https://github.com/nickmitchko/auth/issues/1519)) ([35533ea](https://github.com/nickmitchko/auth/commit/35533ea100669559e1209ecc7b091db3657234d9))
* remove duplicated index on refresh_tokens table ([#1058](https://github.com/nickmitchko/auth/issues/1058)) ([1aa8447](https://github.com/nickmitchko/auth/commit/1aa84478eb8a4cdd30510de5467fd2f78a451c8e))
* remove foreign key constraint on `refresh_tokens`.`parent` ([af00058](https://github.com/nickmitchko/auth/commit/af000589a53358c91842952f894ef56a1debbd17))
* remove organizations from fly provider ([#1267](https://github.com/nickmitchko/auth/issues/1267)) ([c79fc6e](https://github.com/nickmitchko/auth/commit/c79fc6e41988e2854e3e30c7c3f96b1374bdf983))
* remove redundant queries to get session ([#1204](https://github.com/nickmitchko/auth/issues/1204)) ([669ce97](https://github.com/nickmitchko/auth/commit/669ce9706656b157b4e0026ec143827cbe0692b4))
* rename from CustomSMSProvider to SendSMS ([#1513](https://github.com/nickmitchko/auth/issues/1513)) ([c0bc37b](https://github.com/nickmitchko/auth/commit/c0bc37b44effaebb62ba85102f072db07fe57e48))
* rename metadata to data ([#764](https://github.com/nickmitchko/auth/issues/764)) ([70e354d](https://github.com/nickmitchko/auth/commit/70e354d54fdf32a8a982cc255922ff1dc436716e))
* resend email change ([#1151](https://github.com/nickmitchko/auth/issues/1151)) ([ddad10f](https://github.com/nickmitchko/auth/commit/ddad10fa69e41fb161469f07b3f24483b9c980cf))
* resend email change & phone change issues ([#1100](https://github.com/nickmitchko/auth/issues/1100)) ([184fa38](https://github.com/nickmitchko/auth/commit/184fa38f0f90b7a7c6d7c9c4c8a1a087f5e9b453))
* Resend SMS when duplicate SMS sign ups are made ([#1490](https://github.com/nickmitchko/auth/issues/1490)) ([73240a0](https://github.com/nickmitchko/auth/commit/73240a0b096977703e3c7d24a224b5641ce47c81))
* resolve nil pointer dereference issue ([#813](https://github.com/nickmitchko/auth/issues/813)) ([4d78d5f](https://github.com/nickmitchko/auth/commit/4d78d5f58fe5415010c7318c03d9aac138d03928))
* respect last_sign_in_at on secure password update ([#1164](https://github.com/nickmitchko/auth/issues/1164)) ([963df37](https://github.com/nickmitchko/auth/commit/963df37946445af7c762d12d57d95492d3952ec6))
* restrict mfa enrollment to aal2 if verified factors are present ([#1439](https://github.com/nickmitchko/auth/issues/1439)) ([7e10d45](https://github.com/nickmitchko/auth/commit/7e10d45e54010d38677f4c3f2f224127688eb9a2))
* return 404 instead of 500 in maybeLoadUserOrSession ([#783](https://github.com/nickmitchko/auth/issues/783)) ([92ddade](https://github.com/nickmitchko/auth/commit/92ddadeee50270bb1a77888e47edc15ebd10997c))
* return correct sms otp error ([#1351](https://github.com/nickmitchko/auth/issues/1351)) ([5b06680](https://github.com/nickmitchko/auth/commit/5b06680601b4129f34e5fe571ab01dae435c853e))
* return error if session id does not exist ([#1538](https://github.com/nickmitchko/auth/issues/1538)) ([91e9eca](https://github.com/nickmitchko/auth/commit/91e9ecabe33a1c022f8e82a6050c22a7ca42de48))
* return error if user not found but identity exists ([#1200](https://github.com/nickmitchko/auth/issues/1200)) ([1802ff3](https://github.com/nickmitchko/auth/commit/1802ff39c90cf61dc48c0d6ecebc4e4ed707e70d))
* return signup confirmation if signup is incomplete for magiclink / otp ([#889](https://github.com/nickmitchko/auth/issues/889)) ([8137dd8](https://github.com/nickmitchko/auth/commit/8137dd8c9352b4881ae42079898a6611947985d1))
* return the latest flow state ([#1076](https://github.com/nickmitchko/auth/issues/1076)) ([00c9a11](https://github.com/nickmitchko/auth/commit/00c9a11bcbde4a3d3d8856e7aa797f0142995895))
* return unauthorized error for invalid jwt ([#744](https://github.com/nickmitchko/auth/issues/744)) ([85cff37](https://github.com/nickmitchko/auth/commit/85cff37bac7fdd22a87235890caca7d6ace6c244))
* Revert "feat: no email password resets for users with no email identi… ([#822](https://github.com/nickmitchko/auth/issues/822)) ([1129482](https://github.com/nickmitchko/auth/commit/11294825fbeb0a47d3b66fc6d62aedc4a4599295))
* Revert "fix: remove organizations from fly provider" ([#1287](https://github.com/nickmitchko/auth/issues/1287)) ([84e16ed](https://github.com/nickmitchko/auth/commit/84e16ed362a38610deee94b9dbea48a855a1fbbe))
* revert patch for linkedin_oidc provider error ([#1535](https://github.com/nickmitchko/auth/issues/1535)) ([58ef4af](https://github.com/nickmitchko/auth/commit/58ef4af0b4224b78cd9e59428788d16a8d31e562))
* revert refactor resource owner password grant ([#1466](https://github.com/nickmitchko/auth/issues/1466)) ([fa21244](https://github.com/nickmitchko/auth/commit/fa21244fa929709470c2e1fc4092a9ce947399e7))
* **saml:** access DB with context for SSO admin functions ([#805](https://github.com/nickmitchko/auth/issues/805)) ([ca9ad7a](https://github.com/nickmitchko/auth/commit/ca9ad7a8d1ddd6090b19c0b3886be79fe97195e6))
* **saml:** always request persistent NameID in authn requests ([#840](https://github.com/nickmitchko/auth/issues/840)) ([3c2b56e](https://github.com/nickmitchko/auth/commit/3c2b56ede73c1a61e0dcfce21441e591404bb1bf))
* **saml:** correct SSO domain, SAML attribute mapping update logic ([#816](https://github.com/nickmitchko/auth/issues/816)) ([9dbdd61](https://github.com/nickmitchko/auth/commit/9dbdd61e693819f30a608d5c82293f7d81406230))
* **saml:** not specifying `domains` should not delete all domains ([#851](https://github.com/nickmitchko/auth/issues/851)) ([c1ad911](https://github.com/nickmitchko/auth/commit/c1ad911440ee1f9ece7ae91b68d8fb337b43ad06))
* **saml:** persist attribute mappings on provider create and update ([#802](https://github.com/nickmitchko/auth/issues/802)) ([af7c8ba](https://github.com/nickmitchko/auth/commit/af7c8ba1d879134bd38458035cfd9f4143ac9854))
* **saml:** saml user accounts not being set as `is_sso_user` ([#841](https://github.com/nickmitchko/auth/issues/841)) ([e290983](https://github.com/nickmitchko/auth/commit/e290983c05b8e71c8cf87cb80e4cab27b94b2f64))
* **saml:** use `SessionNotOnOrAfter` from the authn. statement instead of conditions ([#838](https://github.com/nickmitchko/auth/issues/838)) ([35acc4c](https://github.com/nickmitchko/auth/commit/35acc4c72f4bb572fe7c0bfa6ced544f22492c1c))
* sanitizeUser leaks user role ([#1366](https://github.com/nickmitchko/auth/issues/1366)) ([8ce9d3f](https://github.com/nickmitchko/auth/commit/8ce9d3f7d93afb056b7ecd151545270a46002ae6))
* set emailChange to email ([#920](https://github.com/nickmitchko/auth/issues/920)) ([c23b6ce](https://github.com/nickmitchko/auth/commit/c23b6cedaac50f0dad251c6ea42e15dfbda4ba33))
* set the otp if it's not a test otp ([#1223](https://github.com/nickmitchko/auth/issues/1223)) ([3afc8a9](https://github.com/nickmitchko/auth/commit/3afc8a9a309d20a7f582f83f94a2776b1d3e13f7))
* show proper error message on textlocal ([#1338](https://github.com/nickmitchko/auth/issues/1338)) ([44e2466](https://github.com/nickmitchko/auth/commit/44e2466da22bc639e08bbaac1ce73bb169eca225))
* skip captcha on `POST /verify` ([#795](https://github.com/nickmitchko/auth/issues/795)) ([eef1bb7](https://github.com/nickmitchko/auth/commit/eef1bb77d45590152ce158976d4c755e18982133))
* skip rate limit if header not present ([#706](https://github.com/nickmitchko/auth/issues/706)) ([8fb0c1e](https://github.com/nickmitchko/auth/commit/8fb0c1e948bbbbccbad54d2ee2a5ba8adac3a9b1))
* sms verify should update is_anonymous field ([#1580](https://github.com/nickmitchko/auth/issues/1580)) ([e5f98cb](https://github.com/nickmitchko/auth/commit/e5f98cb9e24ecebb0b7dc88c495fd456cc73fcba))
* support email verification type on token hash verification ([#1177](https://github.com/nickmitchko/auth/issues/1177)) ([ffa5efa](https://github.com/nickmitchko/auth/commit/ffa5efa4da8c19841e2ab2abe2709c249f427271))
* support message IDs for Twilio Whatsapp ([#1203](https://github.com/nickmitchko/auth/issues/1203)) ([77e85c8](https://github.com/nickmitchko/auth/commit/77e85c87f7f53245dd2792f3e885791063a1201f))
* switch to aws roles ([#893](https://github.com/nickmitchko/auth/issues/893)) ([76c8710](https://github.com/nickmitchko/auth/commit/76c8710aeef06ff75c766ed5097ed0a3763e8d86))
* take into account test otp for twilio verify ([#1255](https://github.com/nickmitchko/auth/issues/1255)) ([18b4291](https://github.com/nickmitchko/auth/commit/18b4291ea00eb0f95229f5dbe5d6474c1e563b4d))
* test otp with twilio verify ([#1259](https://github.com/nickmitchko/auth/issues/1259)) ([ab2aba6](https://github.com/nickmitchko/auth/commit/ab2aba69ae0261454eaef1dc9dacf8717f0bbe15))
* unenroll should remove totp amr claim ([#758](https://github.com/nickmitchko/auth/issues/758)) ([c7a62de](https://github.com/nickmitchko/auth/commit/c7a62dea09621a11f2d8c2f8369e873b0c4c819b))
* unlink identity bugs ([#1475](https://github.com/nickmitchko/auth/issues/1475)) ([73e8d87](https://github.com/nickmitchko/auth/commit/73e8d8742de3575b3165a707b5d2f486b2598d9d))
* unmarshal is_private_email correctly ([#1402](https://github.com/nickmitchko/auth/issues/1402)) ([47df151](https://github.com/nickmitchko/auth/commit/47df15113ce8d86666c0aba3854954c24fe39f7f))
* update .yml to mfa ([#731](https://github.com/nickmitchko/auth/issues/731)) ([e034ca0](https://github.com/nickmitchko/auth/commit/e034ca037d146ea93a08fae2009e76a161c51b2e))
* update dependencies (1/2) ([#1304](https://github.com/nickmitchko/auth/issues/1304)) ([accccee](https://github.com/nickmitchko/auth/commit/accccee91650880e530a7d9b2cd62bb5cc4a7266))
* update email, phone identities on change ([#824](https://github.com/nickmitchko/auth/issues/824)) ([390e34d](https://github.com/nickmitchko/auth/commit/390e34d8160490f5f1cf60d60af9263fae793ed6))
* update file name so migration to Drop IP Address is applied ([#1447](https://github.com/nickmitchko/auth/issues/1447)) ([f29e89d](https://github.com/nickmitchko/auth/commit/f29e89d7d2c48ee8fd5bf8279a7fa3db0ad4d842))
* update from oauth_pkce to pkce ([#1017](https://github.com/nickmitchko/auth/issues/1017)) ([63bc007](https://github.com/nickmitchko/auth/commit/63bc0077d38124d5846c093e0a2e02224eaba806))
* update github.com/crewjam/saml from 0.4.8 to 0.4.9 ([#839](https://github.com/nickmitchko/auth/issues/839)) ([7a10a05](https://github.com/nickmitchko/auth/commit/7a10a05cbe3013826a1f31573b575218cca77237))
* update gobuffalo to v5.3.4 ([#814](https://github.com/nickmitchko/auth/issues/814)) ([aa1ff23](https://github.com/nickmitchko/auth/commit/aa1ff23c506adae8378b536d8204191e0f6e6cfb))
* update linkedin issuer url ([#1536](https://github.com/nickmitchko/auth/issues/1536)) ([10d6d8b](https://github.com/nickmitchko/auth/commit/10d6d8b1eafa504da2b2a351d1f64a3a832ab1b9))
* update password should logout all other sessions ([#806](https://github.com/nickmitchko/auth/issues/806)) ([4b4ca39](https://github.com/nickmitchko/auth/commit/4b4ca39f61e3e4a5bcfecebc1512ea5e9dd1f5d8))
* update phone if autoconfirm is enabled ([#1431](https://github.com/nickmitchko/auth/issues/1431)) ([95db770](https://github.com/nickmitchko/auth/commit/95db770c5d2ecca4a1e960a8cb28ded37cccc100))
* update settings & route for SAML ([#1009](https://github.com/nickmitchko/auth/issues/1009)) ([f405615](https://github.com/nickmitchko/auth/commit/f4056153633066eca0b29e1cb3d8e1a54a5a1442))
* update soft deletion ([#894](https://github.com/nickmitchko/auth/issues/894)) ([6581728](https://github.com/nickmitchko/auth/commit/65817282f2ed05bae19b57f85d4c09cf20b7780c))
* update suggested Go version for contributors to 1.21 ([#1331](https://github.com/nickmitchko/auth/issues/1331)) ([9feeec4](https://github.com/nickmitchko/auth/commit/9feeec48ef85539ad5e818a45e53e262748479e5))
* upgrade pop version ([#1069](https://github.com/nickmitchko/auth/issues/1069)) ([969691f](https://github.com/nickmitchko/auth/commit/969691ffed3282cd09d954bf350c60d2d5d5f261))
* use `pattern` for semver docker image tags ([#1411](https://github.com/nickmitchko/auth/issues/1411)) ([14a3aeb](https://github.com/nickmitchko/auth/commit/14a3aeb6c3f46c8d38d98cc840112dfd0278eeda))
* use api_external_url domain as localname ([#1575](https://github.com/nickmitchko/auth/issues/1575)) ([ed2b490](https://github.com/nickmitchko/auth/commit/ed2b4907244281e4c54aaef74b1f4c8a8e3d97c9))
* use clear hCaptcha error messages ([#789](https://github.com/nickmitchko/auth/issues/789)) ([2906976](https://github.com/nickmitchko/auth/commit/290697630c51199d4d442e6aee35402824ae8ae2))
* use configured redirect URL for external providers ([#1114](https://github.com/nickmitchko/auth/issues/1114)) ([42bb1e0](https://github.com/nickmitchko/auth/commit/42bb1e0310cd4407a49913ac392e4da1be6f4ccd))
* use email change email in identity ([#1429](https://github.com/nickmitchko/auth/issues/1429)) ([4d3b9b8](https://github.com/nickmitchko/auth/commit/4d3b9b8841b1a5fa8f3244825153cc81a73ba300))
* use linkedin oidc endpoint ([#1254](https://github.com/nickmitchko/auth/issues/1254)) ([6d5c8eb](https://github.com/nickmitchko/auth/commit/6d5c8ebb4c894d028e90493a35f43eae1d6c5e7d))
* use proper index name in `20221215195500_modify_users_email_unique_index` ([9eda0ab](https://github.com/nickmitchko/auth/commit/9eda0ab2668578aa63d332514b1a8a4b088aafd5))
* use started transaction, not a new one ([#1196](https://github.com/nickmitchko/auth/issues/1196)) ([0b5b656](https://github.com/nickmitchko/auth/commit/0b5b656d1ed0870fcf9fc4b09273055d5a5b8edc))


### Reverts

* "fix: only create or update the email / phone identity after i… ([#1407](https://github.com/nickmitchko/auth/issues/1407)) ([ff86849](https://github.com/nickmitchko/auth/commit/ff868493169a0d9ac18b66058a735197b1df5b9b))

## [2.152.0](https://github.com/supabase/auth/compare/v2.151.0...v2.152.0) (2024-05-22)


### Features

* new timeout writer implementation ([#1584](https://github.com/supabase/auth/issues/1584)) ([72614a1](https://github.com/supabase/auth/commit/72614a1fce27888f294772b512f8e31c55a36d87))
* remove legacy lookup in users for one_time_tokens (phase II) ([#1569](https://github.com/supabase/auth/issues/1569)) ([39ca026](https://github.com/supabase/auth/commit/39ca026035f6c61d206d31772c661b326c2a424c))
* update chi version ([#1581](https://github.com/supabase/auth/issues/1581)) ([c64ae3d](https://github.com/supabase/auth/commit/c64ae3dd775e8fb3022239252c31b4ee73893237))
* update openapi spec with identity and is_anonymous fields ([#1573](https://github.com/supabase/auth/issues/1573)) ([86a79df](https://github.com/supabase/auth/commit/86a79df9ecfcf09fda0b8e07afbc41154fbb7d9d))


### Bug Fixes

* improve logging structure ([#1583](https://github.com/supabase/auth/issues/1583)) ([c22fc15](https://github.com/supabase/auth/commit/c22fc15d2a8383e95a2364f383dfa7dce5f5df88))
* sms verify should update is_anonymous field ([#1580](https://github.com/supabase/auth/issues/1580)) ([e5f98cb](https://github.com/supabase/auth/commit/e5f98cb9e24ecebb0b7dc88c495fd456cc73fcba))
* use api_external_url domain as localname ([#1575](https://github.com/supabase/auth/issues/1575)) ([ed2b490](https://github.com/supabase/auth/commit/ed2b4907244281e4c54aaef74b1f4c8a8e3d97c9))

## [2.151.0](https://github.com/supabase/auth/compare/v2.150.1...v2.151.0) (2024-05-06)


### Features

* refactor one-time tokens for performance ([#1558](https://github.com/supabase/auth/issues/1558)) ([d1cf8d9](https://github.com/supabase/auth/commit/d1cf8d9096e9183d7772b73031de8ecbd66e912b))


### Bug Fixes

* do call send sms hook when SMS autoconfirm is enabled ([#1562](https://github.com/supabase/auth/issues/1562)) ([bfe4d98](https://github.com/supabase/auth/commit/bfe4d988f3768b0407526bcc7979fb21d8cbebb3))
* format test otps ([#1567](https://github.com/supabase/auth/issues/1567)) ([434a59a](https://github.com/supabase/auth/commit/434a59ae387c35fd6629ec7c674d439537e344e5))
* log final writer error instead of handling ([#1564](https://github.com/supabase/auth/issues/1564)) ([170bd66](https://github.com/supabase/auth/commit/170bd6615405afc852c7107f7358dfc837bad737))

## [2.150.1](https://github.com/supabase/auth/compare/v2.150.0...v2.150.1) (2024-04-28)


### Bug Fixes

* add db conn max idle time setting ([#1555](https://github.com/supabase/auth/issues/1555)) ([2caa7b4](https://github.com/supabase/auth/commit/2caa7b4d75d2ff54af20f3e7a30a8eeec8cbcda9))

## [2.150.0](https://github.com/supabase/auth/compare/v2.149.0...v2.150.0) (2024-04-25)


### Features

* add support for Azure CIAM login ([#1541](https://github.com/supabase/auth/issues/1541)) ([1cb4f96](https://github.com/supabase/auth/commit/1cb4f96bdc7ef3ef995781b4cf3c4364663a2bf3))
* add timeout middleware ([#1529](https://github.com/supabase/auth/issues/1529)) ([f96ff31](https://github.com/supabase/auth/commit/f96ff31040b28e3a7373b4fd41b7334eda1b413e))
* allow for postgres and http functions on each extensibility point ([#1528](https://github.com/supabase/auth/issues/1528)) ([348a1da](https://github.com/supabase/auth/commit/348a1daee24f6e44b14c018830b748e46d34b4c2))
* merge provider metadata on link account ([#1552](https://github.com/supabase/auth/issues/1552)) ([bd8b5c4](https://github.com/supabase/auth/commit/bd8b5c41dd544575e1a52ccf1ef3f0fdee67458c))
* send over user in SendSMS Hook instead of UserID ([#1551](https://github.com/supabase/auth/issues/1551)) ([d4d743c](https://github.com/supabase/auth/commit/d4d743c2ae9490e1b3249387e3b0d60df6913c68))


### Bug Fixes

* return error if session id does not exist ([#1538](https://github.com/supabase/auth/issues/1538)) ([91e9eca](https://github.com/supabase/auth/commit/91e9ecabe33a1c022f8e82a6050c22a7ca42de48))

## [2.149.0](https://github.com/supabase/auth/compare/v2.148.0...v2.149.0) (2024-04-15)


### Features

* refactor generate accesss token to take in request ([#1531](https://github.com/supabase/auth/issues/1531)) ([e4f2b59](https://github.com/supabase/auth/commit/e4f2b59e8e1f8158b6461a384349f1a32cc1bf9a))


### Bug Fixes

* linkedin_oidc provider error ([#1534](https://github.com/supabase/auth/issues/1534)) ([4f5e8e5](https://github.com/supabase/auth/commit/4f5e8e5120531e5a103fbdda91b51cabcb4e1a8c))
* revert patch for linkedin_oidc provider error ([#1535](https://github.com/supabase/auth/issues/1535)) ([58ef4af](https://github.com/supabase/auth/commit/58ef4af0b4224b78cd9e59428788d16a8d31e562))
* update linkedin issuer url ([#1536](https://github.com/supabase/auth/issues/1536)) ([10d6d8b](https://github.com/supabase/auth/commit/10d6d8b1eafa504da2b2a351d1f64a3a832ab1b9))

## [2.148.0](https://github.com/supabase/auth/compare/v2.147.1...v2.148.0) (2024-04-10)


### Features

* add array attribute mapping for SAML ([#1526](https://github.com/supabase/auth/issues/1526)) ([7326285](https://github.com/supabase/auth/commit/7326285c8af5c42e5c0c2d729ab224cf33ac3a1f))

## [2.147.1](https://github.com/supabase/auth/compare/v2.147.0...v2.147.1) (2024-04-09)


### Bug Fixes

* add validation and proper decoding on send email hook ([#1520](https://github.com/supabase/auth/issues/1520)) ([e19e762](https://github.com/supabase/auth/commit/e19e762e3e29729a1d1164c65461427822cc87f1))
* remove deprecated LogoutAllRefreshTokens ([#1519](https://github.com/supabase/auth/issues/1519)) ([35533ea](https://github.com/supabase/auth/commit/35533ea100669559e1209ecc7b091db3657234d9))

## [2.147.0](https://github.com/supabase/auth/compare/v2.146.0...v2.147.0) (2024-04-05)


### Features

* add send email Hook ([#1512](https://github.com/supabase/auth/issues/1512)) ([cf42e02](https://github.com/supabase/auth/commit/cf42e02ec63779f52b1652a7413f64994964c82d))

## [2.146.0](https://github.com/supabase/auth/compare/v2.145.0...v2.146.0) (2024-04-03)


### Features

* add custom sms hook ([#1474](https://github.com/supabase/auth/issues/1474)) ([0f6b29a](https://github.com/supabase/auth/commit/0f6b29a46f1dcbf92aa1f7cb702f42e7640f5f93))
* forbid generating an access token without a session ([#1504](https://github.com/supabase/auth/issues/1504)) ([795e93d](https://github.com/supabase/auth/commit/795e93d0afbe94bcd78489a3319a970b7bf8e8bc))


### Bug Fixes

* add cleanup statement for anonymous users ([#1497](https://github.com/supabase/auth/issues/1497)) ([cf2372a](https://github.com/supabase/auth/commit/cf2372a177796b829b72454e7491ce768bf5a42f))
* generate signup link should not error ([#1514](https://github.com/supabase/auth/issues/1514)) ([4fc3881](https://github.com/supabase/auth/commit/4fc388186ac7e7a9a32ca9b963a83d6ac2eb7603))
* move all EmailActionTypes to mailer package ([#1510](https://github.com/supabase/auth/issues/1510)) ([765db08](https://github.com/supabase/auth/commit/765db08582669a1b7f054217fa8f0ed45804c0b5))
* refactor mfa and aal update methods ([#1503](https://github.com/supabase/auth/issues/1503)) ([31a5854](https://github.com/supabase/auth/commit/31a585429bf248aa919d94c82c7c9e0c1c695461))
* rename from CustomSMSProvider to SendSMS ([#1513](https://github.com/supabase/auth/issues/1513)) ([c0bc37b](https://github.com/supabase/auth/commit/c0bc37b44effaebb62ba85102f072db07fe57e48))

## [2.145.0](https://github.com/supabase/gotrue/compare/v2.144.0...v2.145.0) (2024-03-26)


### Features

* add error codes ([#1377](https://github.com/supabase/gotrue/issues/1377)) ([e4beea1](https://github.com/supabase/gotrue/commit/e4beea1cdb80544b0581f1882696a698fdf64938))
* add kakao OIDC ([#1381](https://github.com/supabase/gotrue/issues/1381)) ([b5566e7](https://github.com/supabase/gotrue/commit/b5566e7ac001cc9f2bac128de0fcb908caf3a5ed))
* clean up expired factors ([#1371](https://github.com/supabase/gotrue/issues/1371)) ([5c94207](https://github.com/supabase/gotrue/commit/5c9420743a9aef0675f823c30aa4525b4933836e))
* configurable NameID format for SAML provider ([#1481](https://github.com/supabase/gotrue/issues/1481)) ([ef405d8](https://github.com/supabase/gotrue/commit/ef405d89e69e008640f275bc37f8ec02ad32da40))
* HTTP Hook - Add custom envconfig decoding for HTTP Hook Secrets ([#1467](https://github.com/supabase/gotrue/issues/1467)) ([5b24c4e](https://github.com/supabase/gotrue/commit/5b24c4eb05b2b52c4177d5f41cba30cb68495c8c))
* refactor PKCE FlowState to reduce duplicate code ([#1446](https://github.com/supabase/gotrue/issues/1446)) ([b8d0337](https://github.com/supabase/gotrue/commit/b8d0337922c6712380f6dc74f7eac9fb71b1ae48))


### Bug Fixes

* add http support for https hooks on localhost ([#1484](https://github.com/supabase/gotrue/issues/1484)) ([5c04104](https://github.com/supabase/gotrue/commit/5c04104bf77a9c2db46d009764ec3ec3e484fc09))
* cleanup panics due to bad inactivity timeout code ([#1471](https://github.com/supabase/gotrue/issues/1471)) ([548edf8](https://github.com/supabase/gotrue/commit/548edf898161c9ba9a136fc99ec2d52a8ba1f856))
* **docs:** remove bracket on file name for broken link ([#1493](https://github.com/supabase/gotrue/issues/1493)) ([96f7a68](https://github.com/supabase/gotrue/commit/96f7a68a5479825e31106c2f55f82d5b2c007c0f))
* impose expiry on auth code instead of magic link ([#1440](https://github.com/supabase/gotrue/issues/1440)) ([35aeaf1](https://github.com/supabase/gotrue/commit/35aeaf1b60dd27a22662a6d1955d60cc907b55dd))
* invalidate email, phone OTPs on password change ([#1489](https://github.com/supabase/gotrue/issues/1489)) ([960a4f9](https://github.com/supabase/gotrue/commit/960a4f94f5500e33a0ec2f6afe0380bbc9562500))
* move creation of flow state into function ([#1470](https://github.com/supabase/gotrue/issues/1470)) ([4392a08](https://github.com/supabase/gotrue/commit/4392a08d68d18828005d11382730117a7b143635))
* prevent user email side-channel leak on verify ([#1472](https://github.com/supabase/gotrue/issues/1472)) ([311cde8](https://github.com/supabase/gotrue/commit/311cde8d1e82f823ae26a341e068034d60273864))
* refactor email sending functions ([#1495](https://github.com/supabase/gotrue/issues/1495)) ([285c290](https://github.com/supabase/gotrue/commit/285c290adf231fea7ca1dff954491dc427cf18e2))
* refactor factor_test to centralize setup ([#1473](https://github.com/supabase/gotrue/issues/1473)) ([c86007e](https://github.com/supabase/gotrue/commit/c86007e59684334b5e8c2285c36094b6eec89442))
* refactor mfa challenge and tests ([#1469](https://github.com/supabase/gotrue/issues/1469)) ([6c76f21](https://github.com/supabase/gotrue/commit/6c76f21cee5dbef0562c37df6a546939affb2f8d))
* Resend SMS when duplicate SMS sign ups are made ([#1490](https://github.com/supabase/gotrue/issues/1490)) ([73240a0](https://github.com/supabase/gotrue/commit/73240a0b096977703e3c7d24a224b5641ce47c81))
* unlink identity bugs ([#1475](https://github.com/supabase/gotrue/issues/1475)) ([73e8d87](https://github.com/supabase/gotrue/commit/73e8d8742de3575b3165a707b5d2f486b2598d9d))

## [2.144.0](https://github.com/supabase/gotrue/compare/v2.143.0...v2.144.0) (2024-03-04)


### Features

* add configuration for custom sms sender hook ([#1428](https://github.com/supabase/gotrue/issues/1428)) ([1ea56b6](https://github.com/supabase/gotrue/commit/1ea56b62d47edb0766d9e445406ecb43d387d920))
* anonymous sign-ins  ([#1460](https://github.com/supabase/gotrue/issues/1460)) ([130df16](https://github.com/supabase/gotrue/commit/130df165270c69c8e28aaa1b9421342f997c1ff3))
* clean up test setup in MFA tests ([#1452](https://github.com/supabase/gotrue/issues/1452)) ([7185af8](https://github.com/supabase/gotrue/commit/7185af8de4a269cdde2629054d222333d3522ebe))
* pass transaction to `invokeHook`, fixing pool exhaustion ([#1465](https://github.com/supabase/gotrue/issues/1465)) ([b536d36](https://github.com/supabase/gotrue/commit/b536d368f35adb31f937169e3f093d28352fa7be))
* refactor resource owner password grant ([#1443](https://github.com/supabase/gotrue/issues/1443)) ([e63ad6f](https://github.com/supabase/gotrue/commit/e63ad6ff0f67d9a83456918a972ecb5109125628))
* use dummy instance id to improve performance on refresh token queries ([#1454](https://github.com/supabase/gotrue/issues/1454)) ([656474e](https://github.com/supabase/gotrue/commit/656474e1b9ff3d5129190943e8c48e456625afe5))


### Bug Fixes

* expose `provider` under `amr` in access token ([#1456](https://github.com/supabase/gotrue/issues/1456)) ([e9f38e7](https://github.com/supabase/gotrue/commit/e9f38e76d8a7b93c5c2bb0de918a9b156155f018))
* improve MFA QR Code resilience so as to support providers like 1Password ([#1455](https://github.com/supabase/gotrue/issues/1455)) ([6522780](https://github.com/supabase/gotrue/commit/652278046c9dd92f5cecd778735b058ef3fb41c7))
* refactor request params to use generics ([#1464](https://github.com/supabase/gotrue/issues/1464)) ([e1cdf5c](https://github.com/supabase/gotrue/commit/e1cdf5c4b5c1bf467094f4bdcaa2e42a5cc51c20))
* revert refactor resource owner password grant ([#1466](https://github.com/supabase/gotrue/issues/1466)) ([fa21244](https://github.com/supabase/gotrue/commit/fa21244fa929709470c2e1fc4092a9ce947399e7))
* update file name so migration to Drop IP Address is applied ([#1447](https://github.com/supabase/gotrue/issues/1447)) ([f29e89d](https://github.com/supabase/gotrue/commit/f29e89d7d2c48ee8fd5bf8279a7fa3db0ad4d842))

## [2.143.0](https://github.com/supabase/gotrue/compare/v2.142.0...v2.143.0) (2024-02-19)


### Features

* calculate aal without transaction ([#1437](https://github.com/supabase/gotrue/issues/1437)) ([8dae661](https://github.com/supabase/gotrue/commit/8dae6614f1a2b58819f94894cef01e9f99117769))


### Bug Fixes

* deprecate hooks  ([#1421](https://github.com/supabase/gotrue/issues/1421)) ([effef1b](https://github.com/supabase/gotrue/commit/effef1b6ecc448b7927eff23df8d5b509cf16b5c))
* error should be an IsNotFoundError ([#1432](https://github.com/supabase/gotrue/issues/1432)) ([7f40047](https://github.com/supabase/gotrue/commit/7f40047aec3577d876602444b1d88078b2237d66))
* populate password verification attempt hook ([#1436](https://github.com/supabase/gotrue/issues/1436)) ([f974bdb](https://github.com/supabase/gotrue/commit/f974bdb58340395955ca27bdd26d57062433ece9))
* restrict mfa enrollment to aal2 if verified factors are present ([#1439](https://github.com/supabase/gotrue/issues/1439)) ([7e10d45](https://github.com/supabase/gotrue/commit/7e10d45e54010d38677f4c3f2f224127688eb9a2))
* update phone if autoconfirm is enabled ([#1431](https://github.com/supabase/gotrue/issues/1431)) ([95db770](https://github.com/supabase/gotrue/commit/95db770c5d2ecca4a1e960a8cb28ded37cccc100))
* use email change email in identity ([#1429](https://github.com/supabase/gotrue/issues/1429)) ([4d3b9b8](https://github.com/supabase/gotrue/commit/4d3b9b8841b1a5fa8f3244825153cc81a73ba300))

## [2.142.0](https://github.com/supabase/gotrue/compare/v2.141.0...v2.142.0) (2024-02-14)


### Features

* alter tag to use raw ([#1427](https://github.com/supabase/gotrue/issues/1427)) ([53cfe5d](https://github.com/supabase/gotrue/commit/53cfe5de57d4b5ab6e8e2915493856ecd96f4ede))
* update README.md to trigger release ([#1425](https://github.com/supabase/gotrue/issues/1425)) ([91e0e24](https://github.com/supabase/gotrue/commit/91e0e245f5957ebce13370f79fd4a6be8108ed80))

## [2.141.0](https://github.com/supabase/gotrue/compare/v2.140.0...v2.141.0) (2024-02-13)


### Features

* drop sha hash tag ([#1422](https://github.com/supabase/gotrue/issues/1422)) ([76853ce](https://github.com/supabase/gotrue/commit/76853ce6d45064de5608acc8100c67a8337ba791))
* prefix release with v ([#1424](https://github.com/supabase/gotrue/issues/1424)) ([9d398cd](https://github.com/supabase/gotrue/commit/9d398cd75fca01fb848aa88b4f545552e8b5751a))

## [2.140.0](https://github.com/supabase/gotrue/compare/v2.139.2...v2.140.0) (2024-02-13)


### Features

* deprecate existing webhook implementation ([#1417](https://github.com/supabase/gotrue/issues/1417)) ([5301e48](https://github.com/supabase/gotrue/commit/5301e481b0c7278c18b4578a5b1aa8d2256c2f5d))
* update publish.yml checkout repository so there is access to Dockerfile ([#1419](https://github.com/supabase/gotrue/issues/1419)) ([7cce351](https://github.com/supabase/gotrue/commit/7cce3518e8c9f1f3f93e4f6a0658ee08771c4f1c))

## [2.139.2](https://github.com/supabase/gotrue/compare/v2.139.1...v2.139.2) (2024-02-08)


### Bug Fixes

* improve perf in account linking ([#1394](https://github.com/supabase/gotrue/issues/1394)) ([8eedb95](https://github.com/supabase/gotrue/commit/8eedb95dbaa310aac464645ec91d6a374813ab89))
* OIDC provider validation log message ([#1380](https://github.com/supabase/gotrue/issues/1380)) ([27e6b1f](https://github.com/supabase/gotrue/commit/27e6b1f9a4394c5c4f8dff9a8b5529db1fc67af9))
* only create or update the email / phone identity after it's been verified ([#1403](https://github.com/supabase/gotrue/issues/1403)) ([2d20729](https://github.com/supabase/gotrue/commit/2d207296ec22dd6c003c89626d255e35441fd52d))
* only create or update the email / phone identity after it's been verified (again) ([#1409](https://github.com/supabase/gotrue/issues/1409)) ([bc6a5b8](https://github.com/supabase/gotrue/commit/bc6a5b884b43fe6b8cb924d3f79999fe5bfe7c5f))
* unmarshal is_private_email correctly ([#1402](https://github.com/supabase/gotrue/issues/1402)) ([47df151](https://github.com/supabase/gotrue/commit/47df15113ce8d86666c0aba3854954c24fe39f7f))
* use `pattern` for semver docker image tags ([#1411](https://github.com/supabase/gotrue/issues/1411)) ([14a3aeb](https://github.com/supabase/gotrue/commit/14a3aeb6c3f46c8d38d98cc840112dfd0278eeda))


### Reverts

* "fix: only create or update the email / phone identity after i… ([#1407](https://github.com/supabase/gotrue/issues/1407)) ([ff86849](https://github.com/supabase/gotrue/commit/ff868493169a0d9ac18b66058a735197b1df5b9b))
