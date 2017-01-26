package token

import (
	"crypto/rsa"

	"github.com/almighty/almighty-core/account"
	jwt "github.com/dgrijalva/jwt-go"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/net/context"
)

// Manager generate and find auth token information
type Manager interface {
	Generate(account.Identity) (string, error)
	Extract(string) (*account.Identity, error)
	Locate(ctx context.Context) (uuid.UUID, error)
	PublicKey() *rsa.PublicKey
}

type tokenManager struct {
	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
}

// NewManager returns a new token Manager for handling creation of tokens
func NewManager(publicKey *rsa.PublicKey, privateKey *rsa.PrivateKey) Manager {
	return &tokenManager{
		publicKey:  publicKey,
		privateKey: privateKey,
	}
}

// Generate generates a new JWT token. For tests only.
func (mgm tokenManager) Generate(ident account.Identity) (string, error) {
	token := jwt.New(jwt.SigningMethodRS256)
	token.Claims.(jwt.MapClaims)["uuid"] = ident.ID.String()
	token.Claims.(jwt.MapClaims)["fullName"] = ident.FullName
	token.Claims.(jwt.MapClaims)["imageURL"] = ident.ImageURL

	tokenStr, err := token.SignedString(mgm.privateKey)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return tokenStr, nil
}

func (mgm tokenManager) Extract(tokenString string) (*account.Identity, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return mgm.publicKey, nil
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if !token.Valid {
		return nil, errors.New("Token not valid")
	}

	claimedUUID := token.Claims.(jwt.MapClaims)["sub"]
	if claimedUUID == nil {
		return nil, errors.New("Subject can not be nil")
	}
	// in case of nil UUID, below type casting will fail hence we need above check
	id, err := uuid.FromString(token.Claims.(jwt.MapClaims)["sub"].(string))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	ident := account.Identity{
		ID:       id,
		FullName: token.Claims.(jwt.MapClaims)["name"].(string),
		ImageURL: "",
	}

	return &ident, nil
}

func (mgm tokenManager) Locate(ctx context.Context) (uuid.UUID, error) {
	token := goajwt.ContextJWT(ctx)
	if token == nil {
		return uuid.UUID{}, errors.New("Missing token") // TODO, make specific tokenErrors
	}
	id := token.Claims.(jwt.MapClaims)["sub"]
	if id == nil {
		return uuid.UUID{}, errors.New("Missing sub")
	}
	idTyped, err := uuid.FromString(id.(string))
	if err != nil {
		return uuid.UUID{}, errors.New("uuid not of type string")
	}
	return idTyped, nil
}

func (mgm tokenManager) PublicKey() *rsa.PublicKey {
	return mgm.publicKey
}

// ParsePublicKey parses a []byte representation of a public key into a rsa.PublicKey instance
func ParsePublicKey(key []byte) (*rsa.PublicKey, error) {
	return jwt.ParseRSAPublicKeyFromPEM(key)
}

// ParsePrivateKey parses a []byte representation of a private key into a rsa.PrivateKey instance
func ParsePrivateKey(key []byte) (*rsa.PrivateKey, error) {
	return jwt.ParseRSAPrivateKeyFromPEM(key)
}

// RSAPrivateKey for signing JWT Tokens
// ssh-keygen -f alm_rsa
var RSAPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAnwrjH5iTSErw9xUptp6QSFoUfpHUXZ+PaslYSUrpLjw1q27O
DSFwmhV4+dAaTMO5chFv/kM36H3ZOyA146nwxBobS723okFaIkshRrf6qgtD6coT
HlVUSBTAcwKEjNn4C9jtEpyOl+eSgxhMzRH3bwTIFlLlVMiZf7XVE7P3yuOCpqkk
2rdYVSpQWQWKU+ZRywJkYcLwjEYjc70AoNpjO5QnY+Exx98E30iEdPHZpsfNhsjh
9Z7IX5TrMYgz7zBTw8+niO/uq3RBaHyIhDbvenbR9Q59d88lbnEeHKgSMe2RQpFR
3rxFRkc/64Rn/bMuL/ptNowPqh1P+9GjYzWmPwIDAQABAoIBAQCBCl5ZpnvprhRx
BVTA/Upnyd7TCxNZmzrME+10Gjmz79pD7DV25ejsu/taBYUxP6TZbliF3pggJOv6
UxomTB4znlMDUz0JgyjUpkyril7xVQ6XRAPbGrS1f1Def+54MepWAn3oGeqASb3Q
bAj0Yl12UFTf+AZmkhQpUKk/wUeN718EIY4GRHHQ6ykMSqCKvdnVbMyb9sIzbSTl
v+l1nQFnB/neyJq6P0Q7cxlhVj03IhYj/AxveNlKqZd2Ih3m/CJo0Abtwhx+qHZp
cCBrYj7VelEaGARTmfoIVoGxFGKZNCcNzn7R2ic7safxXqeEnxugsAYX/UmMoq1b
vMYLcaLRAoGBAMqMbbgejbD8Cy6wa5yg7XquqOP5gPdIYYS88TkQTp+razDqKPIU
hPKetnTDJ7PZleOLE6eJ+dQJ8gl6D/dtOsl4lVRy/BU74dk0fYMiEfiJMYEYuAU0
MCramo3HAeySTP8pxSLFYqJVhcTpL9+NQgbpJBUlx5bLDlJPl7auY077AoGBAMkD
UpJRIv/0gYSz5btVheEyDzcqzOMZUVsngabH7aoQ49VjKrfLzJ9WznzJS5gZF58P
vB7RLuIA8m8Y4FUwxOr4w9WOevzlFh0gyzgNY4gCwrzEryOZqYYqCN+8QLWfq/hL
+gYFYpEW5pJ/lAy2i8kPanC3DyoqiZCsUmlg6JKNAoGBAIdCkf6zgKGhHwKV07cs
DIqx2p0rQEFid6UB3ADkb+zWt2VZ6fAHXeT7shJ1RK0o75ydgomObWR5I8XKWqE7
s1dZjDdx9f9kFuVK1Upd1SxoycNRM4peGJB1nWJydEl8RajcRwZ6U+zeOc+OfWbH
WUFuLadlrEx5212CQ2k+OZlDAoGAdsH2w6kZ83xCFOOv41ioqx5HLQGlYLpxfVg+
2gkeWa523HglIcdPEghYIBNRDQAuG3RRYSeW+kEy+f4Jc2tHu8bS9FWkRcsWoIji
ZzBJ0G5JHPtaub6sEC6/ZWe0F1nJYP2KLop57FxKRt0G2+fxeA0ahpMwa2oMMiQM
4GM3pHUCgYEAj2ZjjsF2MXYA6kuPUG1vyY9pvj1n4fyEEoV/zxY1k56UKboVOtYr
BA/cKaLPqUF+08Tz/9MPBw51UH4GYfppA/x0ktc8998984FeIpfIFX6I2U9yUnoQ
OCCAgsB8g8yTB4qntAYyfofEoDiseKrngQT5DSdxd51A/jw7B8WyBK8=
-----END RSA PRIVATE KEY-----`

// RSAPublicKey for verifying JWT Tokens
// openssl rsa -in alm_rsa -pubout -out alm_rsa.pub
var RSAPublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAnwrjH5iTSErw9xUptp6Q
SFoUfpHUXZ+PaslYSUrpLjw1q27ODSFwmhV4+dAaTMO5chFv/kM36H3ZOyA146nw
xBobS723okFaIkshRrf6qgtD6coTHlVUSBTAcwKEjNn4C9jtEpyOl+eSgxhMzRH3
bwTIFlLlVMiZf7XVE7P3yuOCpqkk2rdYVSpQWQWKU+ZRywJkYcLwjEYjc70AoNpj
O5QnY+Exx98E30iEdPHZpsfNhsjh9Z7IX5TrMYgz7zBTw8+niO/uq3RBaHyIhDbv
enbR9Q59d88lbnEeHKgSMe2RQpFR3rxFRkc/64Rn/bMuL/ptNowPqh1P+9GjYzWm
PwIDAQAB
-----END PUBLIC KEY-----`
