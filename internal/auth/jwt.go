package auth

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTVerifier handles Bluesky JWT token verification
type JWTVerifier struct {
	publicKeys map[string]*rsa.PublicKey
	client     *http.Client
}

// NewJWTVerifier creates a new JWT verifier
func NewJWTVerifier() *JWTVerifier {
	return &JWTVerifier{
		publicKeys: make(map[string]*rsa.PublicKey),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// BlueSkyJWKS represents the JWK Set from Bluesky
type BlueSkyJWKS struct {
	Keys []struct {
		Kid string `json:"kid"`
		Kty string `json:"kty"`
		Use string `json:"use"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

// ExtractDIDFromToken extracts the DID from a Bluesky JWT token
func (v *JWTVerifier) ExtractDIDFromToken(tokenString string) (string, error) {
	// Remove "Bearer " prefix if present
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	
	// Parse the token without verification first to get the header
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	// Get the kid from token header
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return "", fmt.Errorf("no kid in token header")
	}

	// Get the public key for verification
	publicKey, err := v.getPublicKey(kid)
	if err != nil {
		return "", fmt.Errorf("failed to get public key: %w", err)
	}

	// Verify and parse the token
	verifiedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to verify token: %w", err)
	}

	if !verifiedToken.Valid {
		return "", fmt.Errorf("token is not valid")
	}

	// Extract the DID from claims
	claims, ok := verifiedToken.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}

	// The subject should be the user's DID
	sub, ok := claims["sub"].(string)
	if !ok {
		return "", fmt.Errorf("no sub claim in token")
	}

	// Validate that it's a DID
	if !strings.HasPrefix(sub, "did:") {
		return "", fmt.Errorf("sub is not a valid DID: %s", sub)
	}

	return sub, nil
}

// getPublicKey fetches and caches the public key for a given kid
func (v *JWTVerifier) getPublicKey(kid string) (*rsa.PublicKey, error) {
	// Check cache first
	if key, exists := v.publicKeys[kid]; exists {
		return key, nil
	}

	// Fetch JWKS from Bluesky
	jwks, err := v.fetchJWKS()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	// Find the key with matching kid
	for _, key := range jwks.Keys {
		if key.Kid == kid {
			publicKey, err := v.jwkToRSAPublicKey(key)
			if err != nil {
				return nil, fmt.Errorf("failed to convert JWK to RSA public key: %w", err)
			}
			
			// Cache the key
			v.publicKeys[kid] = publicKey
			return publicKey, nil
		}
	}

	return nil, fmt.Errorf("public key not found for kid: %s", kid)
}

// fetchJWKS fetches the JSON Web Key Set from Bluesky
func (v *JWTVerifier) fetchJWKS() (*BlueSkyJWKS, error) {
	// Bluesky's JWKS endpoint (this is a placeholder - you'll need the actual endpoint)
	// For production, you should get this from Bluesky's documentation
	jwksURL := "https://bsky.social/.well-known/jwks.json"
	
	resp, err := v.client.Get(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS endpoint returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read JWKS response: %w", err)
	}

	var jwks BlueSkyJWKS
	if err := json.Unmarshal(body, &jwks); err != nil {
		return nil, fmt.Errorf("failed to parse JWKS: %w", err)
	}

	return &jwks, nil
}

// jwkToRSAPublicKey converts a JWK to an RSA public key
func (v *JWTVerifier) jwkToRSAPublicKey(jwk struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}) (*rsa.PublicKey, error) {
	// This is a simplified implementation
	// In practice, you'd need to properly decode the base64url encoded N and E values
	// and construct an RSA public key from them
	
	// For now, return an error indicating this needs proper implementation
	return nil, fmt.Errorf("JWK to RSA conversion not implemented - please implement base64url decoding for N and E values")
}

// ValidateToken is a middleware-friendly function that validates a JWT token
func (v *JWTVerifier) ValidateToken(authHeader string) (string, bool) {
	if authHeader == "" {
		return "", false
	}

	did, err := v.ExtractDIDFromToken(authHeader)
	if err != nil {
		log.Printf("JWT validation error: %v", err)
		return "", false
	}

	return did, true
}

// For development/testing purposes - mock JWT validation
type MockJWTVerifier struct{}

func NewMockJWTVerifier() *MockJWTVerifier {
	return &MockJWTVerifier{}
}

func (m *MockJWTVerifier) ValidateToken(authHeader string) (string, bool) {
	// Extract a mock DID from the token for testing
	// In a real scenario, this would be properly parsed from a valid JWT
	if authHeader == "" {
		return "", false
	}
	
	// For testing, return a mock DID
	// You can customize this to return different DIDs for different test tokens
	return "did:plc:test-user-123", true
}

func (m *MockJWTVerifier) ExtractDIDFromToken(tokenString string) (string, error) {
	if tokenString == "" {
		return "", fmt.Errorf("empty token")
	}
	return "did:plc:test-user-123", nil
}
