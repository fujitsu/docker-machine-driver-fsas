package keycloak

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	slog "github.com/fujitsu/docker-machine-driver-fsas/logger"

	"github.com/fujitsu/docker-machine-driver-fsas/httputils"
	"github.com/golang-jwt/jwt/v5"
)

const (
	masterRealm = "master"
	accessTokenType  = "access_token"
	refreshTokenType = "refresh_token"
)

type KeycloakHttpError struct {
	Message string
	Code    int
}

// Implement the Error() method for the error interface
func (e *KeycloakHttpError) Error() string {
	return fmt.Sprintf("Error %d: %s", e.Code, e.Message)
}

var (
	ErrNoneOfConstructorArgsCanBeEmpty             = errors.New("none of the arguments can be empty; neither 'Realm', 'User', 'Password', 'BaseURI' or 'Port'")
	ErrResponseBodyMapNotContainKeyAccessToken     = errors.New("response body map does not contain key 'access_token'")
	ErrResponseBodyMapNotContainKeyRefreshToken    = errors.New("response body map does not contain key 'refresh_token'")
	ErrResponseBodyMapNotContainKeyPgcdiPrivileges = errors.New("response body map does not contain key 'pgcdi_privileges'")

	TemplateForErrorCannotCreateClusterInvalidRole    = "User '%s' is not allowed to create cluster. Only users assigned to role '%s' can create cluster. User roles: '%s'"
	TemplateForErrorCannotCreateClusterRealmsDiffer   = "Cannot create cluster. User realm and tenant differ. Realm: '%s', tenant: '%s'"
	TemplateForErrorCannotCreateClusterInDefaultRealm = "User '%s' is not allowed to create cluster in master realm '%s'"

	isInit = false
)

type Keycloak interface {
	IsInit() bool
	InitConnection() error
	UserIsAllowedToCreateCluster() error
	GetToken() string
}

type KeycloakClient struct {
	Realm         string
	UserName      string
	UserPassword  string
	clientId      string
	clientSecret  string
	AccessToken   string
	RefreshToken  string

	userRoles    []string
	userClusters []string
	userTenant   string
	cdiClient    httputils.CdiHTTPClient
}

// This makes KeycloakClient implement the Keycloak interface
var _ Keycloak = (*KeycloakClient)(nil)

// NewKeycloak Creates and returns a new instance of the Keycloak
func NewKeycloak(realm, userName, userPassword, baseURI, endpoint string) (*KeycloakClient, error) {
	if realm == "" || userName == "" || userPassword == "" || baseURI == "" || endpoint == "" {
		slog.Debug("Keycloak constructor: ", "realm", realm, "userName", userName,
			"userPassword", userPassword, "baseURI", baseURI, "endpoint", endpoint)
		slog.Error(ErrNoneOfConstructorArgsCanBeEmpty.Error())
		return nil, ErrNoneOfConstructorArgsCanBeEmpty
	}

	logAllEnvVars := func() {
		envVars := os.Environ()
		slog.Debug("All environment variables:")
		for i, envVar := range envVars {
			if strings.HasPrefix(envVar, "CLIENT_SECRET") {
				slog.Debug("env var: ", "i", i, "envVar", "CLIENT_SECRET=<hidden-for-security-reasons>")
				continue
			}
			slog.Debug("env var: ", "i", i, "envVar", envVar)
		}
	}

	const (
		envVarForClientId      = "CLIENT_ID"
		envVarForClientSecret  = "CLIENT_SECRET"
	)

	clientId := os.Getenv(envVarForClientId)
	if clientId == "" {
		logAllEnvVars()
		return nil, fmt.Errorf("environment variable '%s' for Keycloak client id is empty", envVarForClientId)
	}

	clientSecret := os.Getenv(envVarForClientSecret)
	if clientSecret == "" {
		logAllEnvVars()
		return nil, fmt.Errorf("environment variable '%s' for Keycloak client secret is empty", envVarForClientSecret)
	}

	serverURI := httputils.UrlBuilder(baseURI, endpoint)
	isInit = true

	return &KeycloakClient{
		Realm:         realm,
		UserName:      userName,
		UserPassword:  userPassword,
		clientId:      clientId,
		clientSecret:  clientSecret,
		AccessToken:   "",
		RefreshToken:  "",
		cdiClient:     httputils.NewStandardCdiHTTPClient(serverURI),
	}, nil
}

func (k *KeycloakClient) String() string {
	return "{" +
		fmt.Sprintf("Realm: %s, ", k.Realm) +
		fmt.Sprintf("UserName: %s, ", k.UserName) +
		fmt.Sprintf("clientId: %s, ", k.clientId) +
		fmt.Sprintf("userRoles: %s, ", k.userRoles) +
		fmt.Sprintf("userClusters: %s, ", k.userClusters) +
		fmt.Sprintf("userTenant: %s", k.userTenant) +
		"}"
}

func (k *KeycloakClient) IsInit() bool {
	return isInit
}

// InitConnection Init connection with Keycloak service.
// In this method bearer token is taken
func (k *KeycloakClient) InitConnection() error {
	slog.Debug(fmt.Sprintf("keycloak authorization service structure: %+v", k))
	accessToken, refreshToken, err := k.getTokens()
	if err != nil {
		slog.Error("Error while getting tokens ", "err", err)
		return err
	}

	k.AccessToken = accessToken
	k.RefreshToken = refreshToken
	slog.Debug("The access and refresh tokens successfully initialized: ")

	if err := k.updateUserPgcdiPrivileges(); err != nil {
		return err
	}

	return nil
}

// accessTokenIsValid Check if access token is still valid or in other words if it has not expired yet
func (k *KeycloakClient) accessTokenIsValid() bool {
	claims := jwt.MapClaims{}
	_, _, err := jwt.NewParser().ParseUnverified(k.AccessToken, claims)
	if err != nil {
		slog.Error("Error while parsing jwt token ", "err", err)
		return false
	}

	expirationTime, err := claims.GetExpirationTime()
	if err != nil {
		slog.Error("Error while getting expiration time from claims ", "err", err)
		return false
	}
	slog.Debug(fmt.Sprintf("access token is valid from now %+v and next %+v",
		time.Now().Format(time.RFC3339),
		expirationTime.Time.Sub(time.Now())))
	slog.Debug(fmt.Sprintf("access token is valid till %+v", expirationTime.Format(time.RFC3339)))

	return time.Now().Before(expirationTime.Time)
}

// getTokenEndpoint Returns endpoint for requesting token
func (k *KeycloakClient) getTokenEndpoint() string {
	return fmt.Sprintf("/realms/%s/protocol/openid-connect/token", k.Realm)
}

// getTokens Returns access token and refresh token as strings and error
func (k *KeycloakClient) getTokens() (accessToken, refreshToken string, er error) {
	endpoint := k.getTokenEndpoint()
	queryParams := map[string]string{}
	headers := getHeadersForPostRequest()
	payload := []byte(k.getRequestBodyForAccessToken().Encode())
	var data map[string]any
	statusCode, err := k.cdiClient.Post(payload, endpoint, queryParams, &data, headers)
	if err != nil {
		slog.Error(fmt.Sprintf("Request POST '%s' failed: ", endpoint), "err", err)
		return accessToken, refreshToken, &KeycloakHttpError{
			Message: err.Error(),
			Code:    statusCode,
		}
	}

	accessToken, err = getTokenFromResponse(accessTokenType, data)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = getTokenFromResponse(refreshTokenType, data)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// getTokenFromResponse Returns token of given type from response and error
func getTokenFromResponse(tokenType string, responseData map[string]any) (string, error) {
	var err error

	switch tokenType {
	case accessTokenType:
		err = ErrResponseBodyMapNotContainKeyAccessToken
	case refreshTokenType:
		err = ErrResponseBodyMapNotContainKeyRefreshToken
	default:
		slog.Error("Unknown token type: ", "tokenType", tokenType)
		return "", fmt.Errorf("unknown token type: '%s'", tokenType)
	}

	token, ok := responseData[tokenType].(string)
	if !ok {
		slog.Error("Response data does not contain token: ", "token", token)
		return "", err
	}
	return token, nil
}

// GetToken Returns bearer token.
func (k *KeycloakClient) GetToken() string {
	if !k.accessTokenIsValid() {
		slog.Info("The access token expired and needs to be refreshed")
		if err := k.refreshToken(); err != nil {
			k.InitConnection()
		}
	}

	return k.AccessToken
}

func (k *KeycloakClient) refreshToken() error {
	endpoint := k.getTokenEndpoint()
	queryParams := map[string]string{}
	headers := getHeadersForPostRequest()
	payload := []byte(k.getRequestBodyForRefreshToken().Encode())
	var data map[string]any

	statusCode, err := k.cdiClient.Post(payload, endpoint, queryParams, &data, headers)
	if err != nil || statusCode != http.StatusOK {
		slog.Error(fmt.Sprintf("Error requesting refresh token; request POST '%s' failed: ", endpoint),
			"statusCode", statusCode, "err", err)
		return err
	}
	accessToken, err := getTokenFromResponse(accessTokenType, data)
	if err != nil {
		slog.Error("Error while getting access token from response: ", "err", err)
		return err
	}
	k.AccessToken = accessToken

	refreshToken, err := getTokenFromResponse(refreshTokenType, data)
	if err != nil {
		slog.Error("Error while getting refresh token from response: ", "err", err)
		return err
	}
	k.RefreshToken = refreshToken
	slog.Debug("Both tokens successfully refreshed.")

	return nil
}

// getRequestBodyForAccessToken Returns request body for getting access token
func (k *KeycloakClient) getRequestBodyForAccessToken() url.Values {
	requestBody := k.getBasicRequestBody()
	requestBody.Set("username", k.UserName)
	requestBody.Set("password", k.UserPassword)
	requestBody.Set("scope", "openid")
	requestBody.Set("response", "id_token token")
	requestBody.Set("grant_type", "password")
	return requestBody
}

// getRequestBodyForRefreshToken Returns request body for getting refresh token
func (k *KeycloakClient) getRequestBodyForRefreshToken() url.Values {
	requestBody := k.getBasicRequestBody()
	requestBody.Set("refresh_token", k.RefreshToken)
	requestBody.Set("grant_type", "refresh_token")
	return requestBody
}

// getRequestBodyWithAccessToken Returns request body with access token
func (k *KeycloakClient) getRequestBodyWithAccessToken() url.Values {
	requestBody := k.getBasicRequestBody()
	requestBody.Set("token", k.AccessToken)
	return requestBody
}

// getBasicRequestBody Returns basic request body
func (k *KeycloakClient) getBasicRequestBody() url.Values {
	return url.Values{
		"client_id":     {k.clientId},
		"client_secret": {k.clientSecret},
	}
}

// updateUserPgcdiPrivileges Update KeycloakAuthService struct with info about PG CDI privileges like:
// user roles, clusters, tenant name.
func (k *KeycloakClient) updateUserPgcdiPrivileges() error {
	endpoint := fmt.Sprintf("/realms/%s/protocol/openid-connect/token/introspect", k.Realm)
	queryParams := map[string]string{}
	headers := getHeadersForPostRequest()
	payload := []byte(k.getRequestBodyWithAccessToken().Encode())
	var data map[string]any
	statusCode, err := k.cdiClient.Post(payload, endpoint, queryParams, &data, headers)
	if err != nil {
		slog.Error(fmt.Sprintf("Request POST failed '%s'", endpoint), "err", err)
		return &KeycloakHttpError{
			Message: err.Error(),
			Code:    statusCode,
		}
	}

	k.userTenant, k.userRoles, k.userClusters, err = getPgcdiPrivilegesFromResponse(data)
	if err != nil {
		return err
	}

	slog.Debug("PG CDI privileges taken from keycloak: ",
		"user", k.UserName, "roles", k.userRoles, "clusters",
		k.userClusters, "tenant", k.userTenant)

	return nil
}

// getPgcdiPrivilegesFromResponse Return tuple with PG CDI privileges like:
// user roles, clusters, tenant name.
/* Response taken from endpoint '/realms/some-realm/protocol/openid-connect/token/introspect' may looks like below.
   Part of response that store PG CDI related data is written in key 'pgcdi_privileges'.
{
	...
    "name": "Alice Liddel",
    "pgcdi_privileges": {
        "roles": [
            "cluster_manager"
        ],
        "clusters": [
            "cluster_a",
            "cluster_b"
        ],
        "tenant": "cdi-test"
    },
    "preferred_username": "alice",
    ...
}
*/
func getPgcdiPrivilegesFromResponse(response map[string]any) (tenant string, roles, clusters []string, err error) {

	if _, ok := response["pgcdi_privileges"]; !ok {
		return "", nil, nil, ErrResponseBodyMapNotContainKeyPgcdiPrivileges
	}

	type PgcdiPrivileges struct {
		Roles    []string `json:"roles"`
		Clusters []string `json:"clusters"`
		Tenant   string   `json:"tenant"`
	}

	type User struct {
		PgcdiPrivileges PgcdiPrivileges `json:"pgcdi_privileges"`
	}

	var user User
	data, err := json.Marshal(response)
	if err != nil {
		fmt.Println(err)
		return "", nil, nil, err
	}
	err = json.Unmarshal(data, &user)
	if err != nil {

		return "", nil, nil, err
	}

	return user.PgcdiPrivileges.Tenant, user.PgcdiPrivileges.Roles, user.PgcdiPrivileges.Clusters, nil
}

// UserIsAllowedToCreateCluster Verify if user is allowed to create cluster
func (k *KeycloakClient) UserIsAllowedToCreateCluster() error {

	if !k.userBelongsToClusterCreatorRoles() {
		message := fmt.Sprintf(TemplateForErrorCannotCreateClusterInvalidRole,
			k.UserName, strings.Join(k.getClusterCreatorRoles(), ","), k.userRoles)
		slog.Error(message)
		return errors.New(message)
	}

	if k.Realm != k.userTenant {
		message := fmt.Sprintf(TemplateForErrorCannotCreateClusterRealmsDiffer,
			k.Realm, k.userTenant)
		slog.Error(message)
		return errors.New(message)
	}

	if k.Realm == masterRealm {
		message := fmt.Sprintf(TemplateForErrorCannotCreateClusterInDefaultRealm,
			k.UserName, masterRealm)
		slog.Error(message)
		return errors.New(message)
	}

	slog.Info(fmt.Sprintf("User '%s' is allowed to create a cluster in realm '%s'",
		k.UserName, k.Realm))
	return nil
}

func (k *KeycloakClient) userBelongsToClusterCreatorRoles() bool {
	return isAnyItemInCollection(k.getClusterCreatorRoles(), k.userRoles)
}

// getClusterCreatorRoles Return roles that are allowed to create cluster
func (k *KeycloakClient) getClusterCreatorRoles() []string {
	return []string{"system_manager", "tenant_manager"}
}

// isAnyItemInCollection Check if any item from the expected ones belongs to the collection.
// If yes then return true else false
func isAnyItemInCollection(collection, itemsExpectedInCollection []string) bool {
	for _, item := range itemsExpectedInCollection {
		if slices.Contains(collection, item) {
			return true
		}
	}
	return false
}

// getHeadersForPostRequest Returns headers for POST request
func getHeadersForPostRequest() map[string]string {
	return map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
}
