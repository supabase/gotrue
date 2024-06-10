package api

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/supabase/auth/internal/conf"
	"github.com/supabase/auth/internal/crypto"
	"github.com/supabase/auth/internal/observability"
	"github.com/supabase/auth/internal/storage"
	"github.com/supabase/auth/internal/storage/test"
)

const (
	apiTestVersion = "1"
	apiTestConfig  = "../../hack/test.env"
)

func init() {
	crypto.PasswordHashCost = crypto.QuickHashCost
}

// setupAPIForTest creates a new API to run tests with.
// Using this function allows us to keep track of the database connection
// and cleaning up data between tests.
func setupAPIForTest() (*API, *conf.GlobalConfiguration, error) {
	return setupAPIForTestWithCallback(nil)
}

func setupAPIForTestWithCallback(cb func(*conf.GlobalConfiguration, *storage.Connection)) (*API, *conf.GlobalConfiguration, error) {
	config, err := conf.LoadGlobal(apiTestConfig)
	if err != nil {
		return nil, nil, err
	}

	if err := observability.ConfigureLogging(&config.Logging); err != nil {
		return nil, nil, err
	}

	if cb != nil {
		cb(config, nil)
	}

	conn, err := test.SetupDBConnection(config)
	if err != nil {
		return nil, nil, err
	}

	if cb != nil {
		cb(nil, conn)
	}

	return NewAPIWithVersion(config, conn, apiTestVersion), config, nil
}

func TestEmailEnabledByDefault(t *testing.T) {
	api, _, err := setupAPIForTest()
	require.NoError(t, err)

	require.True(t, api.config.External.Email.Enabled)
}
