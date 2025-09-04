package keycloak

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"testing"

	httputils "github.com/fujitsu/docker-machine-driver-fsas/httputils/mock"
	keycloakMock "github.com/fujitsu/docker-machine-driver-fsas/keycloak/mock"
	"github.com/fujitsu/docker-machine-driver-fsas/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	mockClientSecret = "mockedClientSecret"
	mockClientId     = "mockClientId"
)

func TestNewKeycloakAuthService_Success(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil))) // Suppress slog output in test
	realm := "test-realm"
	userName := "test-user"
	userPassword := "test-password"
	baseURI := "http://192.168.122.1"
	endpoint := "/id_manager/"

	expectedClientId := "cdi"
	expectedClientSecret := "secret"
	t.Setenv("CLIENT_ID", expectedClientId)
	t.Setenv("CLIENT_SECRET", expectedClientSecret)

	authService, err := NewKeycloak(realm, userName, userPassword, baseURI, endpoint)

	assert.NoError(t, err)
	assert.NotNil(t, authService)
	assert.Equal(t, realm, authService.Realm)
	assert.Equal(t, userName, authService.UserName)
	assert.Equal(t, userPassword, authService.UserPassword)
	assert.Equal(t, expectedClientId, authService.clientId)
	assert.Equal(t, expectedClientSecret, authService.clientSecret)
	assert.Equal(t, "", authService.AccessToken)
	assert.Equal(t, "", authService.RefreshToken)
}

func TestNewKeycloakAuthService_MissingArgs(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil))) // Suppress slog output in test
	testCases := []struct {
		realm        string
		userName     string
		userPassword string
		baseURI      string
		endpoint     string
	}{
		{"", "user", "password", "uri", "endpoint"},
		{"realm", "", "password", "uri", "endpoint"},
		{"realm", "user", "", "uri", "endpoint"},
		{"realm", "user", "password", "", "endpoint"},
		{"realm", "user", "password", "uri", ""},
	}

	for _, tc := range testCases {
		authService, err := NewKeycloak(tc.realm, tc.userName, tc.userPassword, tc.baseURI, tc.endpoint)
		assert.ErrorIs(t, err, ErrNoneOfConstructorArgsCanBeEmpty)
		assert.Nil(t, authService)
	}
}

func Test_getTokenFromResponse_Success(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil))) // Suppress slog output in test
	data := make(map[string]any)
	err := json.Unmarshal([]byte(models.TestBearerTokenResponse), &data)
	if err != nil {
		t.Fatal(err)
	}

	accessToken, err := getTokenFromResponse("access_token", data)
	assert.NoError(t, err)
	assert.Equal(t, models.TestAccessTokenExpected, accessToken)

	refreshToken, err := getTokenFromResponse("refresh_token", data)
	assert.NoError(t, err)
	assert.Equal(t, models.TestRefreshTokenExpected, refreshToken)
}

func Test_getTokenFromResponse_invalidToken(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil))) // Suppress slog output in test
	data := make(map[string]any)
	err := json.Unmarshal([]byte(models.TestBearerTokenResponseWithoutTokenKeys), &data)
	if err != nil {
		t.Fatal(err)
	}

	accessToken, err := getTokenFromResponse("access_token", data)
	assert.Error(t, err, ErrResponseBodyMapNotContainKeyAccessToken)
	assert.Equal(t, accessToken, "")

	refreshToken, err := getTokenFromResponse("refresh_token", data)
	assert.Error(t, err, ErrResponseBodyMapNotContainKeyRefreshToken)
	assert.Equal(t, refreshToken, "")
}

func Test_getTokenFromResponse_invalidTokenType(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil))) // Suppress slog output in test
	data := make(map[string]any)
	err := json.Unmarshal([]byte(models.TestBearerTokenResponseWithoutTokenKeys), &data)
	if err != nil {
		t.Fatal(err)
	}

	invalidTokenType := "invalid-token-type"
	token, err := getTokenFromResponse(invalidTokenType, data)
	assert.ErrorContains(t, err, fmt.Sprintf("unknown token type: '%s'", invalidTokenType))
	assert.Equal(t, token, "")

}

func Test_refreshToken_Success(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil))) // Suppress slog output in test
	mockClient := httputils.NewMockCdiHTTPClient(t)
	keycloakClient := &KeycloakClient{
		cdiClient: mockClient,
	}

	expectedData := make(map[string]any)
	err := json.Unmarshal([]byte(models.TestBearerTokenResponse), &expectedData)
	if err != nil {
		t.Fatal(err) // Fail the test immediately if unmarshaling fails
	}

	helperSetResponse := func(payload []byte, endpoint string, queryParams map[string]string, response any, headers map[string]string) {
		resp := response.(*map[string]any)
		*resp = expectedData
	}

	mockClient.EXPECT().Post(
		mock.AnythingOfType("[]uint8"), // Match any byte slice (payload)
		fmt.Sprintf("/realms/%s/protocol/openid-connect/token", keycloakClient.Realm),
		map[string]string{},
		mock.AnythingOfType("*map[string]interface {}"), // Match any map[string]interface{} (response for keycloak)
		map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
	).Run(helperSetResponse).Return(http.StatusOK, nil)

	err = keycloakClient.refreshToken()
	assert.NoError(t, err)
	assert.Equal(t, models.TestAccessTokenExpected, keycloakClient.AccessToken)
	assert.Equal(t, models.TestRefreshTokenExpected, keycloakClient.RefreshToken)

}

func Test_refreshToken_Fail404(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil))) // Suppress slog output in test
	mockClient := httputils.NewMockCdiHTTPClient(t)
	authService := &KeycloakClient{
		Realm:         "test-realm",
		UserName:      "test-user",
		UserPassword:  "test-password",
		clientId:      mockClientId,
		clientSecret:  mockClientSecret,
		cdiClient:     mockClient,
	}

	simulatedError := fmt.Errorf("request failed: %s", models.TestBearerTokenResponse)

	mockClient.EXPECT().Post(
		mock.AnythingOfType("[]uint8"), // Match any byte slice (payload)
		fmt.Sprintf("/realms/%s/protocol/openid-connect/token", authService.Realm),
		map[string]string{},
		mock.AnythingOfType("*map[string]interface {}"), // Match any map[string]interface{} (response for keycloak)
		map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
	).Return(http.StatusBadRequest, simulatedError)

	accessToken, refreshToken, err := authService.getTokens()

	assert.Error(t, err)
	assert.Equal(t, "", accessToken)
	assert.Equal(t, "", refreshToken)
	assert.Equal(t, http.StatusBadRequest, err.(*KeycloakHttpError).Code)
	assert.Equal(t, simulatedError.Error(), err.(*KeycloakHttpError).Message)

}

func Test_getTokens_Success(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil))) // Suppress slog output in test
	mockClient := httputils.NewMockCdiHTTPClient(t)
	authService := &KeycloakClient{
		Realm:         "test-realm",
		UserName:      "test-user",
		UserPassword:  "test-password",
		clientId:      mockClientId,
		clientSecret:  mockClientSecret,
		cdiClient:     mockClient,
	}

	expectedData := make(map[string]any)
	err := json.Unmarshal([]byte(models.TestBearerTokenResponse), &expectedData)
	if err != nil {
		t.Fatal(err) // Fail the test immediately if unmarshaling fails
	}

	helperSetResponse := func(payload []byte, endpoint string, queryParams map[string]string, response any, headers map[string]string) {
		resp := response.(*map[string]any)
		*resp = expectedData
	}

	mockClient.EXPECT().Post(
		mock.AnythingOfType("[]uint8"), // Match any byte slice (payload)
		fmt.Sprintf("/realms/%s/protocol/openid-connect/token", authService.Realm),
		map[string]string{},
		mock.AnythingOfType("*map[string]interface {}"), // Match any map[string]interface{} (response for keycloak)
		map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
	).Run(helperSetResponse).Return(http.StatusOK, nil)

	accessToken, refreshToken, err := authService.getTokens()

	assert.NoError(t, err)
	assert.Equal(t, models.TestAccessTokenExpected, accessToken)
	assert.Equal(t, models.TestRefreshTokenExpected, refreshToken)
}

func TestGetBearerToken_InvalidResponse(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil))) // Suppress slog output in test
	mockClient := httputils.NewMockCdiHTTPClient(t)
	authService := &KeycloakClient{
		Realm:         "test-realm",
		UserName:      "test-user",
		UserPassword:  "test-password",
		clientId:      mockClientId,
		clientSecret:  mockClientSecret,
		cdiClient:     mockClient,
	}

	simulatedError := fmt.Errorf("request failed: %s", models.TestBearerTokenResponse)

	mockClient.EXPECT().Post(
		mock.AnythingOfType("[]uint8"), // Match any byte slice (payload)
		fmt.Sprintf("/realms/%s/protocol/openid-connect/token", authService.Realm),
		map[string]string{},
		mock.AnythingOfType("*map[string]interface {}"), // Match any map[string]interface{} (response for keycloak)
		map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
	).Return(http.StatusBadRequest, simulatedError)

	accessToken, refreshToken, err := authService.getTokens()

	assert.Error(t, err)
	assert.Equal(t, "", accessToken)
	assert.Equal(t, "", refreshToken)
	assert.Equal(t, http.StatusBadRequest, err.(*KeycloakHttpError).Code)
	assert.Equal(t, simulatedError.Error(), err.(*KeycloakHttpError).Message)
}

func TestGetBearerToken_NoAccessToken(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil))) // Suppress slog output in test
	mockClient := httputils.NewMockCdiHTTPClient(t)
	authService := &KeycloakClient{
		Realm:         "test-realm",
		UserName:      "test-user",
		UserPassword:  "test-password",
		clientId:      mockClientId,
		clientSecret:  mockClientSecret,
		cdiClient:     mockClient,
	}

	expectedData := make(map[string]any)
	err := json.Unmarshal([]byte(models.TestBearerTokenResponseWithoutKeyAccessToken), &expectedData)
	if err != nil {
		t.Fatal(err) // Fail the test immediately if unmarshaling fails
	}

	helperSetResponse := func(payload []byte, endpoint string, queryParams map[string]string, response any, headers map[string]string) {
		resp := response.(*map[string]any)
		*resp = expectedData
	}

	mockClient.EXPECT().Post(
		mock.AnythingOfType("[]uint8"), // Match any byte slice (payload)
		fmt.Sprintf("/realms/%s/protocol/openid-connect/token", authService.Realm),
		map[string]string{},
		mock.AnythingOfType("*map[string]interface {}"), // Match any map[string]interface{} (response for keycloak)
		map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
	).Run(helperSetResponse).Return(http.StatusOK, nil)

	accessToken, refreshToken, err := authService.getTokens()

	assert.ErrorIs(t, err, ErrResponseBodyMapNotContainKeyAccessToken)
	assert.Equal(t, "", accessToken)
	assert.Equal(t, "", refreshToken)
}

func TestUpdateUserPgcdiPrivileges_Success(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil))) // Suppress slog output in test
	mockClient := httputils.NewMockCdiHTTPClient(t)
	authService := &KeycloakClient{
		Realm:        models.TestRealm,
		UserName:     models.TestUserNameAllowedToCreateCluster,
		UserPassword: models.TestUserPasswordAllowedToCreateCluster,
		AccessToken:  "test-token",
		cdiClient:    mockClient,
	}
	expectedData := make(map[string]any)
	err := json.Unmarshal([]byte(models.TestPgcdiPrivilegesResponseValidRole), &expectedData)
	if err != nil {
		t.Fatal(err)
	}

	helperSetResponse := func(payload []byte, endpoint string, queryParams map[string]string, response any, headers map[string]string) {
		resp := response.(*map[string]any)
		*resp = expectedData
	}

	mockClient.EXPECT().Post(
		mock.AnythingOfType("[]uint8"), // Match any byte slice (payload)
		fmt.Sprintf("/realms/%s/protocol/openid-connect/token/introspect", authService.Realm),
		map[string]string{},
		mock.AnythingOfType("*map[string]interface {}"), // Match any map[string]interface{} (response for keycloak)
		map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
	).Run(helperSetResponse).Return(http.StatusOK, nil)

	err = authService.updateUserPgcdiPrivileges()

	assert.NoError(t, err)
	assert.Equal(t, models.TestPgcdiPrivilegesTenant, authService.userTenant)
	assert.Equal(t, []string{"system_manager"}, authService.userRoles)
	assert.Equal(t, []string{"cluster_b", "cluster_c"}, authService.userClusters)
}

func TestUpdateUserPgcdiPrivileges_ErrorResponseFromServer(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil))) // Suppress slog output in test
	mockClient := httputils.NewMockCdiHTTPClient(t)
	authService := &KeycloakClient{
		Realm:        models.TestRealm,
		UserName:     models.TestUserNameAllowedToCreateCluster,
		UserPassword: models.TestUserPasswordAllowedToCreateCluster,
		AccessToken:  "test-token",
		cdiClient:    mockClient,
	}
	expectedData := make(map[string]any)
	err := json.Unmarshal([]byte(models.TestPgcdiPrivilegesResponseInvalidRole), &expectedData)
	if err != nil {
		t.Fatal(err)
	}

	simulatedError := fmt.Errorf("request failed: %s", models.TestPgcdiPrivilegesResponseInvalidRole)

	mockClient.EXPECT().Post(
		mock.AnythingOfType("[]uint8"), // Match any byte slice (payload)
		fmt.Sprintf("/realms/%s/protocol/openid-connect/token/introspect", authService.Realm),
		map[string]string{},
		mock.AnythingOfType("*map[string]interface {}"), // Match any map[string]interface{} (response for keycloak)
		map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
	).Return(http.StatusBadRequest, simulatedError)

	err = authService.updateUserPgcdiPrivileges()

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, err.(*KeycloakHttpError).Code)
	assert.Equal(t, simulatedError.Error(), err.(*KeycloakHttpError).Message)
}

func TestUpdateUserPgcdiPrivileges_NoPriviledges(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil))) // Suppress slog output in test
	mockClient := httputils.NewMockCdiHTTPClient(t)
	authService := &KeycloakClient{
		Realm:        models.TestRealm,
		UserName:     models.TestUserNameAllowedToCreateCluster,
		UserPassword: models.TestUserPasswordAllowedToCreateCluster,
		AccessToken:  "test-token",
		cdiClient:    mockClient,
	}
	expectedData := make(map[string]any)
	err := json.Unmarshal([]byte(models.TestPgcdiPrivilegesResponseValidRoleWithoutPriviledges), &expectedData)
	if err != nil {
		t.Fatal(err)
	}

	helperSetResponse := func(payload []byte, endpoint string, queryParams map[string]string, response any, headers map[string]string) {
		resp := response.(*map[string]any)
		*resp = expectedData
	}

	mockClient.EXPECT().Post(
		mock.AnythingOfType("[]uint8"), // Match any byte slice (payload)
		fmt.Sprintf("/realms/%s/protocol/openid-connect/token/introspect", authService.Realm),
		map[string]string{},
		mock.AnythingOfType("*map[string]interface {}"), // Match any map[string]interface{} (response for keycloak)
		map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
	).Run(helperSetResponse).Return(http.StatusOK, nil)

	err = authService.updateUserPgcdiPrivileges()

	assert.ErrorIs(t, err, ErrResponseBodyMapNotContainKeyPgcdiPrivileges)
}

func TestUserIsAllowedToCreateCluster_Success(t *testing.T) {
	authService := &KeycloakClient{
		Realm:        models.TestRealm,
		UserName:     models.TestUserNameAllowedToCreateCluster,
		UserPassword: models.TestUserPasswordAllowedToCreateCluster,
		userRoles:    []string{"system_manager"},
		userTenant:   models.TestRealm,
	}

	err := authService.UserIsAllowedToCreateCluster()

	assert.NoError(t, err)
}

func TestUserIsAllowedToCreateCluster_MasterRealm(t *testing.T) {
	authService := &KeycloakClient{
		Realm:        masterRealm,
		UserName:     models.TestUserNameAllowedToCreateCluster,
		UserPassword: models.TestUserPasswordAllowedToCreateCluster,
		userRoles:    []string{"system_manager"},
		userTenant:   masterRealm,
	}

	err := authService.UserIsAllowedToCreateCluster()

	assert.Error(t, err)
	expectedError := fmt.Sprintf(TemplateForErrorCannotCreateClusterInDefaultRealm, authService.UserName, masterRealm)
	assert.EqualError(t, err, expectedError)
}

func TestUserIsAllowedToCreateCluster_InvalidRole(t *testing.T) {
	testCases := []struct {
		name        string
		userRoles   []string
		expectedErr string
	}{
		{
			name:      "Empty roles",
			userRoles: []string{},
			expectedErr: fmt.Sprintf(TemplateForErrorCannotCreateClusterInvalidRole, "test-user",
				"system_manager,tenant_manager", []string{}),
		},
		{
			name:      "Cluster manager role only",
			userRoles: []string{"cluster_manager"},
			expectedErr: fmt.Sprintf(TemplateForErrorCannotCreateClusterInvalidRole, "test-user",
				"system_manager,tenant_manager", []string{"cluster_manager"}),
		},
		{
			name:      "Other important role only",
			userRoles: []string{"other_important_role"},
			expectedErr: fmt.Sprintf(TemplateForErrorCannotCreateClusterInvalidRole, "test-user",
				"system_manager,tenant_manager", []string{"other_important_role"}),
		},
		{
			name:      "Double role cluster manager and other important role",
			userRoles: []string{"cluster_manager", "other_important_role"},
			expectedErr: fmt.Sprintf(TemplateForErrorCannotCreateClusterInvalidRole, "test-user",
				"system_manager,tenant_manager", []string{"cluster_manager", "other_important_role"}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			authService := &KeycloakClient{
				Realm:      models.TestRealm,
				UserName:   "test-user",
				userRoles:  tc.userRoles,
				userTenant: models.TestRealm, // Keep tenant the same as realm for this test
			}

			err := authService.UserIsAllowedToCreateCluster()

			assert.EqualError(t, err, tc.expectedErr)
		})
	}
}

func TestUserIsAllowedToCreateCluster_RealmsDiffer(t *testing.T) {
	testCases := []struct {
		name        string
		userTenant  string
		expectedErr string
	}{
		{
			name:        "Empty tenant",
			userTenant:  "",
			expectedErr: fmt.Sprintf(TemplateForErrorCannotCreateClusterRealmsDiffer, "test-realm", ""),
		},
		{
			name:        "Master tenant",
			userTenant:  "master",
			expectedErr: fmt.Sprintf(TemplateForErrorCannotCreateClusterRealmsDiffer, "test-realm", "master"),
		},
		{
			name:        "Non-existing tenant",
			userTenant:  "non-existing-tenant",
			expectedErr: fmt.Sprintf(TemplateForErrorCannotCreateClusterRealmsDiffer, "test-realm", "non-existing-tenant"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			authService := &KeycloakClient{
				Realm:      "test-realm",
				userRoles:  []string{"system_manager"}, // Keep role valid for this test
				userTenant: tc.userTenant,
			}
			err := authService.UserIsAllowedToCreateCluster()
			assert.EqualError(t, err, tc.expectedErr)
		})
	}
}

func TestIsAnyItemInCollection(t *testing.T) {
	clusterCreatorRoles := []string{"system_manager", "tenant_manager"}

	testCases := []struct {
		name      string
		userRoles []string
		expected  bool
	}{
		{
			name:      "All three roles",
			userRoles: []string{"system_manager", "tenant_manager", "cluster_manager"},
			expected:  true,
		},
		{
			name:      "System manager and tenant manager",
			userRoles: []string{"system_manager", "tenant_manager"},
			expected:  true,
		},
		{
			name:      "System manager and cluster manager",
			userRoles: []string{"system_manager", "cluster_manager"},
			expected:  true,
		},
		{
			name:      "Tenant manager and cluster manager",
			userRoles: []string{"tenant_manager", "cluster_manager"},
			expected:  true,
		},
		{
			name:      "Cluster manager only",
			userRoles: []string{"cluster_manager"},
			expected:  false,
		},
		{
			name:      "Double role but invalid",
			userRoles: []string{"cluster_manager", "other_role"},
			expected:  false,
		},
		{
			name:      "No role",
			userRoles: []string{},
			expected:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isAnyItemInCollection(clusterCreatorRoles, tc.userRoles)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetClusterCreatorRoles(t *testing.T) {
	authService := &KeycloakClient{}
	expectedRoles := []string{"system_manager", "tenant_manager"}
	assert.Equal(t, expectedRoles, authService.getClusterCreatorRoles())
}

func TestUserBelongsToClusterCreatorRoles(t *testing.T) {
	testCases := []struct {
		name      string
		userRoles []string
		expected  bool
	}{
		{
			name:      "User has system_manager role",
			userRoles: []string{"system_manager", "other_role"},
			expected:  true,
		},
		{
			name:      "User has tenant_manager role",
			userRoles: []string{"other_role", "tenant_manager"},
			expected:  true,
		},
		{
			name:      "User does not have required roles",
			userRoles: []string{"cluster_manager", "other_role"},
			expected:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			authService := &KeycloakClient{userRoles: tc.userRoles}
			result := authService.userBelongsToClusterCreatorRoles()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetBasicRequestBody(t *testing.T) {
	authService := &KeycloakClient{clientId: "test-client-id", clientSecret: "test-client-secret"}
	requestBody := authService.getBasicRequestBody()

	assert.Equal(t, url.Values{"client_id": []string{"test-client-id"},
		"client_secret": []string{"test-client-secret"}}, requestBody)
}

func TestGetRequestBodyWithToken(t *testing.T) {
	authService := &KeycloakClient{clientId: "test-client-id", clientSecret: "test-client-secret", AccessToken: "test-token"}
	requestBody := authService.getRequestBodyWithAccessToken()
	assert.Equal(t, url.Values{"client_id": []string{"test-client-id"}, "client_secret": []string{"test-client-secret"},
		"token": []string{"test-token"}}, requestBody)
}

func TestGetRequestBodyForBearerToken(t *testing.T) {
	authService := &KeycloakClient{
		clientId:      "test-client-id",
		clientSecret:  "test-client-secret",
		UserName:      "test-username",
		UserPassword:  "test-password",
	}
	requestBody := authService.getRequestBodyForAccessToken()

	expectedBody := url.Values{
		"client_id":     []string{"test-client-id"},
		"client_secret": []string{"test-client-secret"},
		"username":      []string{"test-username"},
		"password":      []string{"test-password"},
		"scope":         []string{"openid"},
		"response":      []string{"id_token token"},
		"grant_type":    []string{"password"},
	}
	assert.Equal(t, expectedBody, requestBody)
}

func TestGetPgcdiPrivilegesFromResponse_Success(t *testing.T) {
	response := map[string]any{
		"pgcdi_privileges": map[string]any{
			"roles":    []any{"cluster_manager"},
			"clusters": []any{"cluster_a", "cluster_b"},
			"tenant":   "cdi-test",
		},
	}

	tenant, roles, clusters, err := getPgcdiPrivilegesFromResponse(response)

	assert.NoError(t, err)
	assert.Equal(t, "cdi-test", tenant)
	assert.Equal(t, []string{"cluster_manager"}, roles)
	assert.Equal(t, []string{"cluster_a", "cluster_b"}, clusters)
}

func TestGetPgcdiPrivilegesFromResponse_MissingKey(t *testing.T) {
	response := map[string]any{}

	_, _, _, err := getPgcdiPrivilegesFromResponse(response)

	assert.ErrorIs(t, err, ErrResponseBodyMapNotContainKeyPgcdiPrivileges)
}

func TestGetPgcdiPrivilegesFromResponse_InvalidJSONMarshal(t *testing.T) {
	response := map[string]any{
		"pgcdi_privileges": func() {}, // Not marshallable
	}

	_, _, _, err := getPgcdiPrivilegesFromResponse(response)

	assert.ErrorContains(t, err, "json: unsupported type")
}

func TestGetPgcdiPrivilegesFromResponse_InvalidJSONUnmarshal(t *testing.T) {
	response := map[string]any{
		"pgcdi_privileges": "invalid-structure", // Cannot be unmarshalled to PgcdiPrivileges
	}

	_, _, _, err := getPgcdiPrivilegesFromResponse(response)

	assert.ErrorContains(t, err, "json: cannot unmarshal string into Go struct")
}

func TestGetToken_Success(t *testing.T) {

	mockKeycloak := keycloakMock.NewMockKeycloak(t)
	expectedToken := "123"
	mockKeycloak.On("GetToken").Return(expectedToken)
	observedToken := mockKeycloak.GetToken()
	assert.Equal(t, expectedToken, observedToken)
}
