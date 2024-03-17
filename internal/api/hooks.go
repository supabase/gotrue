package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/supabase/auth/internal/observability"

	"github.com/supabase/auth/internal/conf"
	"github.com/supabase/auth/internal/crypto"

	"github.com/sirupsen/logrus"
	"github.com/supabase/auth/internal/hooks"

	"github.com/supabase/auth/internal/storage"
)

const (
	DefaultHTTPHookTimeout  = 5 * time.Second
	DefaultHTTPHookRetries  = 3
	HTTPHookBackoffDuration = 2 * time.Second
)

func (a *API) runPostgresHook(ctx context.Context, tx *storage.Connection, name string, input, output any) ([]byte, error) {
	db := a.db.WithContext(ctx)

	request, err := json.Marshal(input)
	if err != nil {
		panic(err)
	}

	var response []byte
	invokeHookFunc := func(tx *storage.Connection) error {
		// We rely on Postgres timeouts to ensure the function doesn't overrun
		if terr := tx.RawQuery(fmt.Sprintf("set local statement_timeout TO '%d';", hooks.DefaultTimeout)).Exec(); terr != nil {
			return terr
		}

		if terr := tx.RawQuery(fmt.Sprintf("select %s(?);", name), request).First(&response); terr != nil {
			return terr
		}

		// reset the timeout
		if terr := tx.RawQuery("set local statement_timeout TO default;").Exec(); terr != nil {
			return terr
		}

		return nil
	}

	if tx != nil {
		if err := invokeHookFunc(tx); err != nil {
			return nil, err
		}
	} else {
		if err := db.Transaction(invokeHookFunc); err != nil {
			return nil, err
		}
	}

	if err := json.Unmarshal(response, output); err != nil {
		return response, err
	}

	return response, nil
}

func readBodyWithLimit(rsp *http.Response) ([]byte, error) {
	defer rsp.Body.Close()

	const limit = 20 * 1024 // 20KB
	limitedReader := io.LimitedReader{R: rsp.Body, N: limit}

	body, err := io.ReadAll(&limitedReader)
	if err != nil {
		return nil, err
	}

	if limitedReader.N <= 0 {
		// Attempt to read one more byte to check if we're exactly at the limit or over
		_, err := rsp.Body.Read(make([]byte, 1))
		if err == nil {
			// If we could read more, then the payload was too large
			return nil, fmt.Errorf("payload too large")
		}
	}

	return body, nil
}

func (a *API) runHTTPHook(r *http.Request, hookConfig conf.ExtensibilityPointConfiguration, input, output any) ([]byte, error) {
	client := http.Client{
		Timeout: DefaultHTTPHookTimeout,
	}
	log := observability.GetLogEntry(r)
	requestURL := hookConfig.URI
	hookLog := log.WithFields(logrus.Fields{
		"component": "auth_hook",
		"url":       requestURL,
	})

	inputPayload, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	for i := 0; i < DefaultHTTPHookRetries; i++ {
		hookLog.Infof("invocation attempt: %d", i)
		if time.Since(start) > time.Duration(i+1)*DefaultHTTPHookTimeout {
			return []byte{}, gatewayTimeoutError(ErrorHookTimeout, "failed to reach hook within timeout")
		}
		msgID := uuid.Must(uuid.NewV4())
		currentTime := time.Now()
		signatureList, err := crypto.GenerateSignatures(hookConfig.HTTPHookSecrets, msgID, currentTime, inputPayload)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewBuffer(inputPayload))
		if err != nil {
			return nil, internalServerError("Failed to make request object").WithInternalError(err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("webhook-id", msgID.String())
		req.Header.Set("webhook-timestamp", fmt.Sprintf("%d", currentTime.Unix()))
		req.Header.Set("webhook-signature", strings.Join(signatureList, ", "))

		watcher, req := watchForConnection(req)
		rsp, err := client.Do(req)

		if err != nil {
			if terr, ok := err.(net.Error); ok && terr.Timeout() {
				hookLog.Errorf("Request timed out for attempt %d with err %s", i, err)
				time.Sleep(HTTPHookBackoffDuration)
				continue
			} else if !watcher.gotConn && i < DefaultHTTPHookRetries-1 {
				hookLog.Errorf("Failed to establish a connection on attempt %d with err %s", i, err)
				time.Sleep(HTTPHookBackoffDuration)
				continue
			} else if i == DefaultHTTPHookRetries-1 {
				return nil, gatewayTimeoutError(ErrorHookTimeout, "Failed to reach hook within allotted interval")

			} else {
				return nil, internalServerError("Failed to trigger auth hook, error making HTTP request").WithInternalError(err)
			}
		}

		switch rsp.StatusCode {
		case http.StatusOK, http.StatusNoContent, http.StatusAccepted:
			if rsp.Body == nil {
				return nil, nil
			}
			body, err := readBodyWithLimit(rsp)
			if err != nil {
				return nil, err
			}
			return body, nil
		case http.StatusTooManyRequests, http.StatusServiceUnavailable:
			retryAfterHeader := rsp.Header.Get("retry-after")
			// Check for truthy values to allow for flexibility to swtich to time duration
			if retryAfterHeader != "" {
				continue
			}
			return []byte{}, internalServerError("Service currently unavailable")
		case http.StatusBadRequest:
			return nil, badRequestError(ErrorCodeValidationFailed, "Invalid payload sent to hook")
		case http.StatusUnauthorized:
			return []byte{}, httpError(http.StatusUnauthorized, ErrorCodeNoAuthorization, "Hook requires authorizaition token")
		default:
			return []byte{}, internalServerError("Error executing Hook")
		}
	}
	return nil, internalServerError("error executing hook")
}

func watchForConnection(req *http.Request) (*connectionWatcher, *http.Request) {
	w := new(connectionWatcher)
	t := &httptrace.ClientTrace{
		GotConn: w.GotConn,
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), t))
	return w, req
}

type connectionWatcher struct {
	gotConn bool
}

func (c *connectionWatcher) GotConn(_ httptrace.GotConnInfo) {
	c.gotConn = true
}

func (a *API) invokeHTTPHook(r *http.Request, input, output any, hookURI string) error {
	switch input.(type) {
	case *hooks.CustomSMSProviderInput:
		hookOutput, ok := output.(*hooks.CustomSMSProviderOutput)
		if !ok {
			panic("output should be *hooks.CustomSMSProviderOutput")
		}
		var response []byte
		var err error

		if response, err = a.runHTTPHook(r, a.config.Hook.CustomSMSProvider, input, output); err != nil {
			return internalServerError("Error invoking custom SMS provider hook.").WithInternalError(err)
		}
		if err != nil {
			return err
		}

		if err := json.Unmarshal(response, hookOutput); err != nil {
			return internalServerError("Error unmarshaling custom SMS provider hook output.").WithInternalError(err)
		}
		fmt.Printf("%v", hookOutput)

	default:
		panic("unknown HTTP hook type")
	}
	return nil
}

// invokePostgresHook invokes the hook code. tx can be nil, in which case a new
// transaction is opened. If calling invokeHook within a transaction, always
// pass the current transaction, as pool-exhaustion deadlocks are very easy to
// trigger.
func (a *API) invokePostgresHook(ctx context.Context, conn *storage.Connection, input, output any, hookURI string) error {
	config := a.config
	// Switch based on hook type
	switch input.(type) {
	case *hooks.MFAVerificationAttemptInput:
		hookOutput, ok := output.(*hooks.MFAVerificationAttemptOutput)
		if !ok {
			panic("output should be *hooks.MFAVerificationAttemptOutput")
		}

		if _, err := a.runPostgresHook(ctx, conn, config.Hook.MFAVerificationAttempt.HookName, input, output); err != nil {
			return internalServerError("Error invoking MFA verification hook.").WithInternalError(err)
		}

		if hookOutput.IsError() {
			httpCode := hookOutput.HookError.HTTPCode

			if httpCode == 0 {
				httpCode = http.StatusInternalServerError
			}

			httpError := &HTTPError{
				HTTPStatus: httpCode,
				Message:    hookOutput.HookError.Message,
			}

			return httpError.WithInternalError(&hookOutput.HookError)
		}

		return nil
	case *hooks.PasswordVerificationAttemptInput:
		hookOutput, ok := output.(*hooks.PasswordVerificationAttemptOutput)
		if !ok {
			panic("output should be *hooks.PasswordVerificationAttemptOutput")
		}

		if _, err := a.runPostgresHook(ctx, conn, config.Hook.PasswordVerificationAttempt.HookName, input, output); err != nil {
			return internalServerError("Error invoking password verification hook.").WithInternalError(err)
		}

		if hookOutput.IsError() {
			httpCode := hookOutput.HookError.HTTPCode

			if httpCode == 0 {
				httpCode = http.StatusInternalServerError
			}

			httpError := &HTTPError{
				HTTPStatus: httpCode,
				Message:    hookOutput.HookError.Message,
			}

			return httpError.WithInternalError(&hookOutput.HookError)
		}

		return nil
	case *hooks.CustomAccessTokenInput:
		hookOutput, ok := output.(*hooks.CustomAccessTokenOutput)
		if !ok {
			panic("output should be *hooks.CustomAccessTokenOutput")
		}

		if _, err := a.runPostgresHook(ctx, conn, config.Hook.CustomAccessToken.HookName, input, output); err != nil {
			return internalServerError("Error invoking access token hook.").WithInternalError(err)
		}

		if hookOutput.IsError() {
			httpCode := hookOutput.HookError.HTTPCode

			if httpCode == 0 {
				httpCode = http.StatusInternalServerError
			}

			httpError := &HTTPError{
				HTTPStatus: httpCode,
				Message:    hookOutput.HookError.Message,
			}

			return httpError.WithInternalError(&hookOutput.HookError)
		}
		if err := validateTokenClaims(hookOutput.Claims); err != nil {
			httpCode := hookOutput.HookError.HTTPCode

			if httpCode == 0 {
				httpCode = http.StatusInternalServerError
			}

			httpError := &HTTPError{
				HTTPStatus: httpCode,
				Message:    err.Error(),
			}

			return httpError
		}
		return nil

	default:
		panic("unknown Postgres hook input type")
	}
}
