package plugins

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/sqlstore"
	"github.com/grafana/grafana/pkg/tests/testinfra"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	usernameAdmin    = "admin"
	usernameNonAdmin = "nonAdmin"
	defaultPassword  = "password"
)

func TestPluginInstallAccess(t *testing.T) {
	dir, cfgPath := testinfra.CreateGrafDir(t, testinfra.GrafanaOpts{
		MarketplaceAppEnabled: true,
	})
	store := testinfra.SetUpDatabase(t, dir)
	store.Bus = bus.GetBus() // in order to allow successful user auth
	grafanaListedAddr := testinfra.StartGrafana(t, dir, cfgPath, store)

	createUser(t, store, usernameNonAdmin, defaultPassword, false)
	createUser(t, store, usernameAdmin, defaultPassword, true)

	t.Run("Request is forbidden if not from an admin", func(t *testing.T) {
		statusCode, body := makePostRequest(t, grafanaAPIURL(usernameNonAdmin, grafanaListedAddr, "plugins/grafana-plugin/install"))
		assert.Equal(t, 403, statusCode)
		assert.JSONEq(t, "{\"message\": \"Permission denied\"}", body)

		statusCode, body = makePostRequest(t, grafanaAPIURL(usernameNonAdmin, grafanaListedAddr, "plugins/grafana-plugin/uninstall"))
		assert.Equal(t, 403, statusCode)
		assert.JSONEq(t, "{\"message\": \"Permission denied\"}", body)
	})

	t.Run("Request is not forbidden if from an admin", func(t *testing.T) {
		statusCode, body := makePostRequest(t, grafanaAPIURL(usernameAdmin, grafanaListedAddr, "plugins/test/install"))
		assert.Equal(t, 404, statusCode)
		assert.JSONEq(t, "{\"error\":\"plugin not found\", \"message\":\"Plugin not found\"}", body)

		statusCode, body = makePostRequest(t, grafanaAPIURL(usernameAdmin, grafanaListedAddr, "plugins/test/uninstall"))
		assert.Equal(t, 404, statusCode)
		assert.JSONEq(t, "{\"error\":\"plugin is not installed\", \"message\":\"Plugin not installed\"}", body)
	})
}

func createUser(t *testing.T, store *sqlstore.SQLStore, username, password string, isAdmin bool) {
	t.Helper()

	cmd := models.CreateUserCommand{
		Login:    username,
		Password: password,
		IsAdmin:  isAdmin,
	}
	_, err := store.CreateUser(context.Background(), cmd)
	require.NoError(t, err)
}

func makePostRequest(t *testing.T, URL string) (int, string) {
	t.Helper()

	// nolint:gosec
	resp, err := http.Post(URL, "application/json", bytes.NewBufferString(""))
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = resp.Body.Close()
		log.Warn("Failed to close response body", "err", err)
	})
	b, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp.StatusCode, string(b)
}

func grafanaAPIURL(username string, grafanaListedAddr string, path string) string {
	return fmt.Sprintf("http://%s:%s@%s/api/%s", username, defaultPassword, grafanaListedAddr, path)
}