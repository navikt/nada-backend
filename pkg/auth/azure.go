package auth

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/navikt/nada-backend/pkg/openapi"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

type Azure struct {
	clientID     string
	clientSecret string
	tenantID     string
	hostname     string

	// teamUUIDs map[string]string
}

func NewAzure(clientID, clientSecret, tenantID, hostname string) *Azure {
	return &Azure{
		clientID:     clientID,
		clientSecret: clientSecret,
		tenantID:     tenantID,
		hostname:     hostname,
	}
}

func (a *Azure) KeyDiscoveryURL() string {
	return fmt.Sprintf("https://login.microsoftonline.com/%s/discovery/v2.0/keys", a.tenantID)
}

func (a *Azure) OAuth2Config() oauth2.Config {
	var callbackURL string
	if a.hostname == "localhost" {
		callbackURL = "http://localhost:8080/oauth2/callback"
	} else {
		callbackURL = fmt.Sprintf("https://%v/oauth2/callback", a.hostname)
	}

	return oauth2.Config{
		ClientID:     a.clientID,
		ClientSecret: a.clientSecret,
		Endpoint:     endpoints.AzureAD(a.tenantID),
		RedirectURL:  callbackURL,
		Scopes:       []string{"openid", fmt.Sprintf("%s/.default", a.clientID)},
	}
}

func (a *Azure) Groups(client *http.Client) *AzureGroups {
	return NewAzureGroups(client, a.clientID, a.clientSecret, a.tenantID)
}

func (a *Azure) Middleware(teamsCache teamsCache) openapi.MiddlewareFunc {
	return JWTValidatorMiddleware(a.KeyDiscoveryURL(), a.clientID, a.Groups(http.DefaultClient), teamsCache)
}

type CertificateList []*x509.Certificate

func FetchCertificates(discoveryURL string) (map[string]CertificateList, error) {
	log.Infof("Discover Microsoft signing certificates from %s", discoveryURL)
	azureKeyDiscovery, err := DiscoverURL(discoveryURL)
	if err != nil {
		return nil, err
	}

	log.Infof("Decoding certificates for %d keys", len(azureKeyDiscovery.Keys))
	azureCertificates, err := azureKeyDiscovery.Map()
	if err != nil {
		return nil, err
	}
	return azureCertificates, nil
}

// Map transform a KeyDiscovery object into a dictionary with "kid" as key
// and lists of decoded X509 certificates as values.
//
// Returns an error if any certificate does not decode.
func (k *KeyDiscovery) Map() (result map[string]CertificateList, err error) {
	result = make(map[string]CertificateList)

	for _, key := range k.Keys {
		certList := make(CertificateList, 0)
		for _, encodedCertificate := range key.X5c {
			certificate, err := encodedCertificate.Decode()
			if err != nil {
				return nil, err
			}
			certList = append(certList, certificate)
		}
		result[key.Kid] = certList
	}

	return
}

type KeyDiscovery struct {
	Keys []Key `json:"keys"`
}

func DiscoverURL(url string) (*KeyDiscovery, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	return Discover(response.Body)
}

// Decode a base64 encoded certificate into a X509 structure.
func (c EncodedCertificate) Decode() (*x509.Certificate, error) {
	stream := strings.NewReader(string(c))
	decoder := base64.NewDecoder(base64.StdEncoding, stream)
	key, err := ioutil.ReadAll(decoder)
	if err != nil {
		return nil, err
	}

	return x509.ParseCertificate(key)
}

func Discover(reader io.Reader) (*KeyDiscovery, error) {
	document, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	keyDiscovery := &KeyDiscovery{}
	err = json.Unmarshal(document, keyDiscovery)

	return keyDiscovery, err
}

type EncodedCertificate string

type Key struct {
	Kid string               `json:"kid"`
	X5c []EncodedCertificate `json:"x5c"`
}
