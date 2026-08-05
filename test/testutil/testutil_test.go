package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildTableMap_DefaultTable(t *testing.T) {
	m := buildTableMap(KV{Key: "endpoint", Value: "http://user"})
	assert.Contains(t, m, DefaultTable)
	assert.Equal(t, "http://user", m[DefaultTable]["endpoint"])
}

func TestBuildTableMap_CustomTable(t *testing.T) {
	m := buildTableMap(KV{Key: "token", Value: "abc", Table: "auth"})
	assert.Contains(t, m, "auth")
	assert.Equal(t, "abc", m["auth"]["token"])
}

func TestBuildTableMap_Overwrite(t *testing.T) {
	m := buildTableMap(
		KV{Key: "a", Value: "1"},
		KV{Key: "a", Value: "2"},
	)
	assert.Equal(t, "2", m[DefaultTable]["a"])
}

func TestBuildTableMap_Empty(t *testing.T) {
	m := buildTableMap()
	assert.Empty(t, m)
}
