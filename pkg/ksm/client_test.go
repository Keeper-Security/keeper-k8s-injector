package ksm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewClient_InvalidConfigReturnsError is the regression test for the nil-client bug:
// the vendored SDK's NewSecretsManager reports an unparseable config by returning a nil
// *SecretsManager (it has no error return), which used to be wrapped into a *Client with a
// nil embedded field and a nil error — every method on it then panicked on first use.
func TestNewClient_InvalidConfigReturnsError(t *testing.T) {
	_, err := NewClient(context.Background(), Config{ConfigJSON: "not a valid ksm config"})

	require.Error(t, err)
}

// TestNewClient_ValidConfigSucceeds is a control: a well-formed, already-bound config (a
// non-empty clientId) must still construct a client normally, so the nil-check doesn't
// reject good input. NewSecretsManager only checks these fields are non-empty strings at
// this stage — no cryptographic parsing happens — so placeholder text works and, unlike a
// base64-looking string, can't be mistaken for a real credential.
func TestNewClient_ValidConfigSucceeds(t *testing.T) {
	client, err := NewClient(context.Background(), Config{
		ConfigJSON: `{"hostname":"keepersecurity.com","clientId":"PLACEHOLDER-NOT-A-REAL-CLIENT-ID","privateKey":"PLACEHOLDER-NOT-A-REAL-PRIVATE-KEY","appKey":"PLACEHOLDER-NOT-A-REAL-APP-KEY","serverPublicKeyId":"10"}`,
	})

	assert.NoError(t, err)
	assert.NotNil(t, client)
}
