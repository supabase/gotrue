package api

import (
	"net/http"
	"regexp"
	"time"

	"github.com/didip/tollbooth/v5"
	"github.com/didip/tollbooth/v5/limiter"
	"github.com/rs/cors"
	"github.com/sebest/xff"
	"github.com/sirupsen/logrus"
	"github.com/supabase/auth/internal/conf"
	"github.com/supabase/auth/internal/mailer"
	"github.com/supabase/auth/internal/models"
	"github.com/supabase/auth/internal/observability"
	"github.com/supabase/auth/internal/storage"
	"github.com/supabase/auth/internal/utilities"
	"github.com/supabase/hibp"
)

const (
	audHeaderName  = "X-JWT-AUD"
	defaultVersion = "unknown version"
)

var bearerRegexp = regexp.MustCompile(`^(?:B|b)earer (\S+$)`)

// API is the main REST API
type API struct {
	handler http.Handler
	db      *storage.Connection
	config  *conf.GlobalConfiguration
	version string

	hibpClient *hibp.PwnedClient

	// overrideTime can be used to override the clock used by handlers. Should only be used in tests!
	overrideTime func() time.Time
}

func (a *API) Now() time.Time {
	if a.overrideTime != nil {
		return a.overrideTime()
	}

	return time.Now()
}

// NewAPI instantiates a new REST API
func NewAPI(globalConfig *conf.GlobalConfiguration, db *storage.Connection) *API {
	return NewAPIWithVersion(globalConfig, db, defaultVersion)
}

func (a *API) deprecationNotices() {
	config := a.config

	log := logrus.WithField("component", "api")

	if config.JWT.AdminGroupName != "" {
		log.Warn("DEPRECATION NOTICE: GOTRUE_JWT_ADMIN_GROUP_NAME not supported by Supabase's GoTrue, will be removed soon")
	}

	if config.JWT.DefaultGroupName != "" {
		log.Warn("DEPRECATION NOTICE: GOTRUE_JWT_DEFAULT_GROUP_NAME not supported by Supabase's GoTrue, will be removed soon")
	}
}

// NewAPIWithVersion creates a new REST API using the specified version
func NewAPIWithVersion(globalConfig *conf.GlobalConfiguration, db *storage.Connection, version string) *API {
	api := &API{config: globalConfig, db: db, version: version}

	if api.config.Password.HIBP.Enabled {
		httpClient := &http.Client{
			// all HIBP API requests should finish quickly to avoid
			// unnecessary slowdowns
			Timeout: 5 * time.Second,
		}

		api.hibpClient = &hibp.PwnedClient{
			UserAgent: api.config.Password.HIBP.UserAgent,
			HTTP:      httpClient,
		}

		if api.config.Password.HIBP.Bloom.Enabled {
			cache := utilities.NewHIBPBloomCache(api.config.Password.HIBP.Bloom.Items, api.config.Password.HIBP.Bloom.FalsePositives)
			api.hibpClient.Cache = cache

			logrus.Infof("Pwned passwords cache is %.2f KB", float64(cache.Cap())/(8*1024.0))
		}
	}

	api.deprecationNotices()

	xffmw, _ := xff.Default()
	logger := observability.NewStructuredLogger(logrus.StandardLogger(), globalConfig)

	r := newRouter()
	r.UseBypass(observability.AddRequestID(globalConfig))
	r.UseBypass(logger)
	r.UseBypass(xffmw.Handler)
	r.UseBypass(recoverer)

	if globalConfig.API.MaxRequestDuration > 0 {
		r.UseBypass(timeoutMiddleware(globalConfig.API.MaxRequestDuration))
	}

	// request tracing should be added only when tracing or metrics is enabled
	if globalConfig.Tracing.Enabled || globalConfig.Metrics.Enabled {
		r.UseBypass(observability.RequestTracing())
	}

	if globalConfig.DB.CleanupEnabled {
		cleanup := models.NewCleanup(globalConfig)
		r.UseBypass(api.databaseCleanup(cleanup))
	}

	r.Get("/health", api.HealthCheck)

	r.Route("/callback", func(r *router) {
		r.Use(api.isValidExternalHost)
		r.Use(api.loadFlowState)

		r.Get("/", api.ExternalProviderCallback)
		r.Post("/", api.ExternalProviderCallback)
	})

	r.Route("/", func(r *router) {
		r.Use(api.isValidExternalHost)

		r.Get("/settings", api.Settings)

		r.Get("/authorize", api.ExternalProviderRedirect)

		sharedLimiter := api.limitEmailOrPhoneSentHandler()
		r.With(sharedLimiter).With(api.requireAdminCredentials).Post("/invite", api.Invite)
		r.With(sharedLimiter).With(api.verifyCaptcha).Route("/signup", func(r *router) {
			// rate limit per hour
			limitAnonymousSignIns := tollbooth.NewLimiter(api.config.RateLimitAnonymousUsers/(60*60), &limiter.ExpirableOptions{
				DefaultExpirationTTL: time.Hour,
			}).SetBurst(int(api.config.RateLimitAnonymousUsers)).SetMethods([]string{"POST"})

			limitSignups := tollbooth.NewLimiter(api.config.RateLimitOtp/(60*5), &limiter.ExpirableOptions{
				DefaultExpirationTTL: time.Hour,
			}).SetBurst(30)

			r.Post("/", func(w http.ResponseWriter, r *http.Request) error {
				params := &SignupParams{}
				if err := retrieveRequestParams(r, params); err != nil {
					return err
				}
				if params.Email == "" && params.Phone == "" {
					if !api.config.External.AnonymousUsers.Enabled {
						return unprocessableEntityError(ErrorCodeAnonymousProviderDisabled, "Anonymous sign-ins are disabled")
					}
					if _, err := api.limitHandler(limitAnonymousSignIns)(w, r); err != nil {
						return err
					}
					return api.SignupAnonymously(w, r)
				}

				// apply ip-based rate limiting on otps
				if _, err := api.limitHandler(limitSignups)(w, r); err != nil {
					return err
				}
				// apply shared rate limiting on email / phone
				if _, err := sharedLimiter(w, r); err != nil {
					return err
				}
				return api.Signup(w, r)
			})
		})
		r.With(api.limitHandler(
			// Allow requests at the specified rate per 5 minutes
			tollbooth.NewLimiter(api.config.RateLimitOtp/(60*5), &limiter.ExpirableOptions{
				DefaultExpirationTTL: time.Hour,
			}).SetBurst(30),
		)).With(sharedLimiter).With(api.verifyCaptcha).With(api.requireEmailProvider).Post("/recover", api.Recover)

		r.With(api.limitHandler(
			// Allow requests at the specified rate per 5 minutes
			tollbooth.NewLimiter(api.config.RateLimitOtp/(60*5), &limiter.ExpirableOptions{
				DefaultExpirationTTL: time.Hour,
			}).SetBurst(30),
		)).With(sharedLimiter).With(api.verifyCaptcha).Post("/resend", api.Resend)

		r.With(api.limitHandler(
			// Allow requests at the specified rate per 5 minutes
			tollbooth.NewLimiter(api.config.RateLimitOtp/(60*5), &limiter.ExpirableOptions{
				DefaultExpirationTTL: time.Hour,
			}).SetBurst(30),
		)).With(sharedLimiter).With(api.verifyCaptcha).Post("/magiclink", api.MagicLink)

		r.With(api.limitHandler(
			// Allow requests at the specified rate per 5 minutes
			tollbooth.NewLimiter(api.config.RateLimitOtp/(60*5), &limiter.ExpirableOptions{
				DefaultExpirationTTL: time.Hour,
			}).SetBurst(30),
		)).With(sharedLimiter).With(api.verifyCaptcha).Post("/otp", api.Otp)

		r.With(api.limitHandler(
			// Allow requests at the specified rate per 5 minutes.
			tollbooth.NewLimiter(api.config.RateLimitTokenRefresh/(60*5), &limiter.ExpirableOptions{
				DefaultExpirationTTL: time.Hour,
			}).SetBurst(30),
		)).With(api.verifyCaptcha).Post("/token", api.Token)

		r.With(api.limitHandler(
			// Allow requests at the specified rate per 5 minutes.
			tollbooth.NewLimiter(api.config.RateLimitVerify/(60*5), &limiter.ExpirableOptions{
				DefaultExpirationTTL: time.Hour,
			}).SetBurst(30),
		)).Route("/verify", func(r *router) {
			r.Get("/", api.Verify)
			r.Post("/", api.Verify)
		})

		r.With(api.requireAuthentication).Post("/logout", api.Logout)

		r.With(api.requireAuthentication).Route("/reauthenticate", func(r *router) {
			r.Get("/", api.Reauthenticate)
		})

		r.With(api.requireAuthentication).Route("/user", func(r *router) {
			r.Get("/", api.UserGet)
			r.With(api.limitHandler(
				// Allow requests at the specified rate per 5 minutes
				tollbooth.NewLimiter(api.config.RateLimitOtp/(60*5), &limiter.ExpirableOptions{
					DefaultExpirationTTL: time.Hour,
				}).SetBurst(30),
			)).With(sharedLimiter).Put("/", api.UserUpdate)

			r.Route("/identities", func(r *router) {
				r.Use(api.requireManualLinkingEnabled)
				r.Get("/authorize", api.LinkIdentity)
				r.Delete("/{identity_id}", api.DeleteIdentity)
			})
		})

		r.With(api.requireAuthentication).Route("/factors", func(r *router) {
			r.Use(api.requireNotAnonymous)
			r.Post("/", api.EnrollFactor)
			r.Route("/{factor_id}", func(r *router) {
				r.Use(api.loadFactor)

				r.With(api.limitHandler(
					tollbooth.NewLimiter(api.config.MFA.RateLimitChallengeAndVerify/60, &limiter.ExpirableOptions{
						DefaultExpirationTTL: time.Minute,
					}).SetBurst(30))).Post("/verify", api.VerifyFactor)
				r.With(api.limitHandler(
					tollbooth.NewLimiter(api.config.MFA.RateLimitChallengeAndVerify/60, &limiter.ExpirableOptions{
						DefaultExpirationTTL: time.Minute,
					}).SetBurst(30))).Post("/challenge", api.ChallengeFactor)
				r.Delete("/", api.UnenrollFactor)

			})
		})

		r.Route("/sso", func(r *router) {
			r.Use(api.requireSAMLEnabled)
			r.With(api.limitHandler(
				// Allow requests at the specified rate per 5 minutes.
				tollbooth.NewLimiter(api.config.RateLimitSso/(60*5), &limiter.ExpirableOptions{
					DefaultExpirationTTL: time.Hour,
				}).SetBurst(30),
			)).With(api.verifyCaptcha).Post("/", api.SingleSignOn)

			r.Route("/saml", func(r *router) {
				r.Get("/metadata", api.SAMLMetadata)

				r.With(api.limitHandler(
					// Allow requests at the specified rate per 5 minutes.
					tollbooth.NewLimiter(api.config.SAML.RateLimitAssertion/(60*5), &limiter.ExpirableOptions{
						DefaultExpirationTTL: time.Hour,
					}).SetBurst(30),
				)).Post("/acs", api.SAMLACS)
			})
		})

		r.Route("/admin", func(r *router) {
			r.Use(api.requireAdminCredentials)

			r.Route("/audit", func(r *router) {
				r.Get("/", api.adminAuditLog)
			})

			r.Route("/users", func(r *router) {
				r.Get("/", api.adminUsers)
				r.Post("/", api.adminUserCreate)

				r.Route("/{user_id}", func(r *router) {
					r.Use(api.loadUser)
					r.Route("/factors", func(r *router) {
						r.Get("/", api.adminUserGetFactors)
						r.Route("/{factor_id}", func(r *router) {
							r.Use(api.loadFactor)
							r.Delete("/", api.adminUserDeleteFactor)
							r.Put("/", api.adminUserUpdateFactor)
						})
					})

					r.Get("/", api.adminUserGet)
					r.Put("/", api.adminUserUpdate)
					r.Delete("/", api.adminUserDelete)
				})
			})

			r.Post("/generate_link", api.adminGenerateLink)

			r.Route("/sso", func(r *router) {
				r.Route("/providers", func(r *router) {
					r.Get("/", api.adminSSOProvidersList)
					r.Post("/", api.adminSSOProvidersCreate)

					r.Route("/{idp_id}", func(r *router) {
						r.Use(api.loadSSOProvider)

						r.Get("/", api.adminSSOProvidersGet)
						r.Put("/", api.adminSSOProvidersUpdate)
						r.Delete("/", api.adminSSOProvidersDelete)
					})
				})
			})

		})
	})

	corsHandler := cors.New(cors.Options{
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowedHeaders:   globalConfig.CORS.AllAllowedHeaders([]string{"Accept", "Authorization", "Content-Type", "X-Client-IP", "X-Client-Info", audHeaderName, useCookieHeader, "X-Supabase-Api-Version"}),
		ExposedHeaders:   []string{"X-Total-Count", "Link", "X-Supabase-Api-Version"},
		AllowCredentials: true,
	})

	api.handler = corsHandler.Handler(r)
	return api
}

type HealthCheckResponse struct {
	Version     string `json:"version"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// HealthCheck endpoint indicates if the gotrue api service is available
func (a *API) HealthCheck(w http.ResponseWriter, r *http.Request) error {
	return sendJSON(w, http.StatusOK, HealthCheckResponse{
		Version:     a.version,
		Name:        "GoTrue",
		Description: "GoTrue is a user registration and authentication API",
	})
}

// Mailer returns NewMailer with the current tenant config
func (a *API) Mailer() mailer.Mailer {
	config := a.config
	return mailer.NewMailer(config)
}
