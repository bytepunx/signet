package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfirmDestructive_SkipTrueBypassesPrompt(t *testing.T) {
	var out bytes.Buffer
	err := confirmDestructive(strings.NewReader(""), &out, "warning text", true)
	require.NoError(t, err)
	assert.Contains(t, out.String(), "warning text")
	assert.NotContains(t, out.String(), "Type 'yes'")
}

func TestConfirmDestructive_TypingYesProceeds(t *testing.T) {
	var out bytes.Buffer
	err := confirmDestructive(strings.NewReader("yes\n"), &out, "warning text", false)
	require.NoError(t, err)
}

func TestConfirmDestructive_TypingAnythingElseAborts(t *testing.T) {
	var out bytes.Buffer
	err := confirmDestructive(strings.NewReader("y\n"), &out, "warning text", false)
	require.Error(t, err)
}

func TestConfirmDestructive_EmptyInputAborts(t *testing.T) {
	var out bytes.Buffer
	err := confirmDestructive(strings.NewReader(""), &out, "warning text", false)
	require.Error(t, err)
}
