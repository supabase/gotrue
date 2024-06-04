package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/supabase/auth/internal/models"
	"github.com/supabase/auth/internal/observability"
	"github.com/supabase/auth/internal/security"
	"go.opentelemetry.io/otel/attribute"

	"github.com/didip/tollbooth/v5"
	"github.com/didip/tollbooth/v5/limiter"
	jwt "github.com/golang-jwt/jwt"
)

type FunctionHooks map[string][]string

type AuthMicroserviceClaims struct {
	jwt.StandardClaims
	SiteURL       string        `json:"site_url"`
	InstanceID    string        `json:"id"`
	FunctionHooks FunctionHooks `json:"function_hooks"`
}

func (f *FunctionHooks) UnmarshalJSON(b []byte) error {
	var raw map[string][]string
	err := json.Unmarshal(b, &raw)
	if err == nil {
		*f = FunctionHooks(raw)
		return nil
	}
	// If unmarshaling into map[string][]string fails, try legacy format.
	var legacy map[string]string
	err = json.Unmarshal(b, &legacy)
	if err != nil {
		return err
	}
	if *f == nil {
		*f = make(FunctionHooks)
	}
	for event, hook := range legacy {
		(*f)[event] = []string{hook}
	}
	return nil
}

var emailRateLimitCounter = observability.ObtainMetricCounter("gotrue_email_rate_limit_counter", "Number of times an email rate limit has been triggered")

func (a *API) limitHandler(lmt *limiter.Limiter) middlewareHandler {
	return func(w http.ResponseWriter, req *http.Request) (context.Context, error) {
		c := req.Context()

		if limitHeader := a.config.RateLimitHeader; limitHeader != "" {
			key := req.Header.Get(limitHeader)

			if key == "" {
				log := observability.GetLogEntry(req).Entry
				log.WithField("header", limitHeader).Warn("request does not have a value for the rate limiting header, rate limiting is not applied")
				return c, nil
			} else {
				err := tollbooth.LimitByKeys(lmt, []string{key})
				if err != nil {
					return c, tooManyRequestsError(ErrorCodeOverRequestRateLimit, "Request rate limit reached")
				}
			}
		}
		return c, nil
	}
}

func (a *API) limitEmailOrPhoneSentHandler() middlewareHandler {
	// limit per hour
	emailFreq := a.config.RateLimitEmailSent / (60 * 60)
	smsFreq := a.config.RateLimitSmsSent / (60 * 60)

	emailLimiter := tollbooth.NewLimiter(emailFreq, &limiter.ExpirableOptions{
		DefaultExpirationTTL: time.Hour,
	}).SetBurst(int(a.config.RateLimitEmailSent)).SetMethods([]string{"PUT", "POST"})

	phoneLimiter := tollbooth.NewLimiter(smsFreq, &limiter.ExpirableOptions{
		DefaultExpirationTTL: time.Hour,
	}).SetBurst(int(a.config.RateLimitSmsSent)).SetMethods([]string{"PUT", "POST"})

	return func(w http.ResponseWriter, req *http.Request) (context.Context, error) {
		c := req.Context()
		config := a.config
		shouldRateLimitEmail := config.External.Email.Enabled && !config.Mailer.Autoconfirm
		shouldRateLimitPhone := config.External.Phone.Enabled && !config.Sms.Autoconfirm

		if shouldRateLimitEmail || shouldRateLimitPhone {
			if req.Method == "PUT" || req.Method == "POST" {
				var requestBody struct {
					Email string `json:"email"`
					Phone string `json:"phone"`
				}

				if err := retrieveRequestParams(req, &requestBody); err != nil {
					return c, err
				}

				if shouldRateLimitEmail {
					if requestBody.Email != "" {
						if err := tollbooth.LimitByKeys(emailLimiter, []string{"email_functions"}); err != nil {
							emailRateLimitCounter.Add(
								req.Context(),
								1,
								attribute.String("path", req.URL.Path),
							)
							return c, tooManyRequestsError(ErrorCodeOverEmailSendRateLimit, "Email rate limit exceeded")
						}
					}
				}

				if shouldRateLimitPhone {
					if requestBody.Phone != "" {
						if err := tollbooth.LimitByKeys(phoneLimiter, []string{"phone_functions"}); err != nil {
							return c, tooManyRequestsError(ErrorCodeOverSMSSendRateLimit, "SMS rate limit exceeded")
						}
					}
				}
			}
		}

		return c, nil
	}
}

func (a *API) requireAdminCredentials(w http.ResponseWriter, req *http.Request) (context.Context, error) {
	t, err := a.extractBearerToken(req)
	if err != nil || t == "" {
		return nil, err
	}

	ctx, err := a.parseJWTClaims(t, req)
	if err != nil {
		a.clearCookieTokens(a.config, w)
		return nil, err
	}

	return a.requireAdmin(ctx)
}

func (a *API) requireEmailProvider(w http.ResponseWriter, req *http.Request) (context.Context, error) {
	ctx := req.Context()
	config := a.config

	if !config.External.Email.Enabled {
		return nil, badRequestError(ErrorCodeEmailProviderDisabled, "Email logins are disabled")
	}

	return ctx, nil
}

func (a *API) verifyCaptcha(w http.ResponseWriter, req *http.Request) (context.Context, error) {
	ctx := req.Context()
	config := a.config

	if !config.Security.Captcha.Enabled {
		return ctx, nil
	}
	if _, err := a.requireAdminCredentials(w, req); err == nil {
		// skip captcha validation if authorization header contains an admin role
		return ctx, nil
	}
	if shouldIgnore := isIgnoreCaptchaRoute(req); shouldIgnore {
		return ctx, nil
	}

	verificationResult, err := security.VerifyRequest(req, strings.TrimSpace(config.Security.Captcha.Secret), config.Security.Captcha.Provider)
	if err != nil {
		return nil, internalServerError("captcha verification process failed").WithInternalError(err)
	}

	if !verificationResult.Success {
		return nil, badRequestError(ErrorCodeCaptchaFailed, "captcha protection: request disallowed (%s)", strings.Join(verificationResult.ErrorCodes, ", "))
	}

	return ctx, nil
}

func isIgnoreCaptchaRoute(req *http.Request) bool {
	// captcha shouldn't be enabled on the following grant_types
	// id_token, refresh_token, pkce
	if req.URL.Path == "/token" && req.FormValue("grant_type") != "password" {
		return true
	}
	return false
}

func (a *API) isValidExternalHost(w http.ResponseWriter, req *http.Request) (context.Context, error) {
	ctx := req.Context()
	config := a.config

	var u *url.URL
	var err error

	baseUrl := config.API.ExternalURL
	xForwardedHost := req.Header.Get("X-Forwarded-Host")
	xForwardedProto := req.Header.Get("X-Forwarded-Proto")
	if xForwardedHost != "" && xForwardedProto != "" && !config.API.ForceExternalURL {
		baseUrl = fmt.Sprintf("%s://%s", xForwardedProto, xForwardedHost)
	} else if req.URL.Scheme != "" && req.URL.Hostname() != "" && !config.API.ForceExternalURL {
		baseUrl = fmt.Sprintf("%s://%s", req.URL.Scheme, req.URL.Hostname())
		// Restores enforced external URLs by adding in an envionment variable. API_FORCE_EXTERNAL_URL
	}
	if u, err = url.ParseRequestURI(baseUrl); err != nil {
		// fallback to the default hostname
		log := observability.GetLogEntry(req).Entry
		log.WithField("request_url", baseUrl).Warn(err)
		if u, err = url.ParseRequestURI(config.API.ExternalURL); err != nil {
			return ctx, err
		}
	}
	return withExternalHost(ctx, u), nil
}

func (a *API) requireSAMLEnabled(w http.ResponseWriter, req *http.Request) (context.Context, error) {
	ctx := req.Context()
	if !a.config.SAML.Enabled {
		return nil, notFoundError(ErrorCodeSAMLProviderDisabled, "SAML 2.0 is disabled")
	}
	return ctx, nil
}

func (a *API) requireManualLinkingEnabled(w http.ResponseWriter, req *http.Request) (context.Context, error) {
	ctx := req.Context()
	if !a.config.Security.ManualLinkingEnabled {
		return nil, notFoundError(ErrorCodeManualLinkingDisabled, "Manual linking is disabled")
	}
	return ctx, nil
}

func (a *API) databaseCleanup(cleanup *models.Cleanup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)

			switch r.Method {
			case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
				// continue

			default:
				return
			}

			db := a.db.WithContext(r.Context())
			log := observability.GetLogEntry(r).Entry

			affectedRows, err := cleanup.Clean(db)
			if err != nil {
				log.WithError(err).WithField("affected_rows", affectedRows).Warn("database cleanup failed")
			} else if affectedRows > 0 {
				log.WithField("affected_rows", affectedRows).Debug("cleaned up expired or stale rows")
			}
		})
	}
}

// timeoutResponseWriter is a http.ResponseWriter that queues up a response
// body to be sent if the serving completes before the context has exceeded its
// deadline.
type timeoutResponseWriter struct {
	sync.Mutex

	header      http.Header
	wroteHeader bool
	snapHeader  http.Header // snapshot of the header at the time WriteHeader was called
	statusCode  int
	buf         bytes.Buffer
}

func (t *timeoutResponseWriter) Header() http.Header {
	t.Lock()
	defer t.Unlock()

	return t.header
}

func (t *timeoutResponseWriter) Write(bytes []byte) (int, error) {
	t.Lock()
	defer t.Unlock()

	if !t.wroteHeader {
		t.writeHeaderLocked(http.StatusOK)
	}

	return t.buf.Write(bytes)
}

func (t *timeoutResponseWriter) WriteHeader(statusCode int) {
	t.Lock()
	defer t.Unlock()

	t.writeHeaderLocked(statusCode)
}

func (t *timeoutResponseWriter) writeHeaderLocked(statusCode int) {
	if t.wroteHeader {
		// ignore multiple calls to WriteHeader
		// once WriteHeader has been called once, a snapshot of the header map is taken
		// and saved in snapHeader to be used in finallyWrite
		return
	}

	t.statusCode = statusCode
	t.wroteHeader = true
	t.snapHeader = t.header.Clone()
}

func (t *timeoutResponseWriter) finallyWrite(w http.ResponseWriter) {
	t.Lock()
	defer t.Unlock()

	dst := w.Header()
	for k, vv := range t.snapHeader {
		dst[k] = vv
	}

	if !t.wroteHeader {
		t.statusCode = http.StatusOK
	}

	w.WriteHeader(t.statusCode)
	if _, err := w.Write(t.buf.Bytes()); err != nil {
		logrus.WithError(err).Warn("Write failed")
	}
}

func timeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			timeoutWriter := &timeoutResponseWriter{
				header: make(http.Header),
			}

			panicChan := make(chan any, 1)
			serverDone := make(chan struct{})
			go func() {
				defer func() {
					if p := recover(); p != nil {
						panicChan <- p
					}
				}()

				next.ServeHTTP(timeoutWriter, r.WithContext(ctx))
				close(serverDone)
			}()

			select {
			case p := <-panicChan:
				panic(p)

			case <-serverDone:
				timeoutWriter.finallyWrite(w)

			case <-ctx.Done():
				err := ctx.Err()

				if err == context.DeadlineExceeded {
					httpError := &HTTPError{
						HTTPStatus: http.StatusGatewayTimeout,
						ErrorCode:  ErrorCodeRequestTimeout,
						Message:    "Processing this request timed out, please retry after a moment.",
					}

					httpError = httpError.WithInternalError(err)

					HandleResponseError(httpError, w, r)
				} else {
					// unrecognized context error, so we should wait for the server to finish
					// and write out the response
					<-serverDone

					timeoutWriter.finallyWrite(w)
				}
			}
		})
	}
}
